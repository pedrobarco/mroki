package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pedrobarco/mroki/cmd/mroki-api/config"
	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/events"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/jobs"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/ent"
	"github.com/rs/cors"

	"github.com/pedrobarco/mroki/internal/interfaces/http/handlers"
	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	diffmetrics "github.com/pedrobarco/mroki/pkg/diff/metrics"
	"github.com/pedrobarco/mroki/pkg/dto"
	"github.com/pedrobarco/mroki/pkg/logger"
	"github.com/pedrobarco/mroki/pkg/metrics"
	"github.com/pedrobarco/mroki/pkg/ratelimit"
)

func main() {
	logger := logger.New()

	cfg, err := config.Load()
	if err != nil {
		var verr *config.ValidationError
		if errors.As(err, &verr) {
			for _, w := range verr.Warnings() {
				logger.Warn("Configuration warning", "detail", w.Message)
			}
			if verr.HasErrors() {
				logger.Error("Configuration validation failed", "error", verr.Error())
				os.Exit(1)
			}
		} else {
			logger.Error("Configuration loading failed", "error", err)
			os.Exit(1)
		}
	}

	// Parse pool configuration timeouts (safe after validation)
	maxConnIdleDuration, _ := time.ParseDuration(cfg.App.Database.MaxConnIdle)
	maxConnLifeDuration, _ := time.ParseDuration(cfg.App.Database.MaxConnLife)

	// Open database connection via pgx stdlib driver
	db, err := sql.Open("pgx", cfg.App.Database.URL.String())
	if err != nil {
		panic(fmt.Errorf("failed to open database connection: %w", err))
	}
	db.SetMaxOpenConns(int(cfg.App.Database.MaxConns))
	db.SetMaxIdleConns(int(cfg.App.Database.MinConns))
	db.SetConnMaxIdleTime(maxConnIdleDuration)
	db.SetConnMaxLifetime(maxConnLifeDuration)

	// Create Ent client from the sql.DB connection
	client := ent.NewPostgresClient(db)

	// Ensure client is closed if initialization fails
	var clientClosed bool
	defer func() {
		if !clientClosed && client != nil {
			_ = client.Close()
			logger.Info("Database connection closed (cleanup)")
		}
	}()

	// Infrastructure Layer: Repository implementations
	gateRepo := ent.NewGateRepository(client)
	reqRepo := ent.NewRequestRepository(client)
	statsRepo := ent.NewStatsRepository(client)

	// Application Layer: in-process domain event bus. The CreateRequest handler
	// dispatches RequestCompared events through it after a successful save;
	// subscribers (e.g. the metrics listener wired below) react best-effort
	// without coupling the domain/application layers to their concerns.
	eventBus := events.NewBus(events.WithLogger(logger))

	// Application Layer: Command Handlers (Write operations)
	createGateHandler := commands.NewCreateGateHandler(gateRepo)
	updateGateHandler := commands.NewUpdateGateHandler(gateRepo)
	deleteGateHandler := commands.NewDeleteGateHandler(gateRepo)
	createRequestHandler := commands.NewCreateRequestHandler(reqRepo, gateRepo, commands.WithEventDispatcher(eventBus))

	// Application Layer: Query Handlers (Read operations)
	getGateHandler := queries.NewGetGateHandler(gateRepo, statsRepo)
	listGatesHandler := queries.NewListGatesHandler(gateRepo, statsRepo)
	getRequestHandler := queries.NewGetRequestHandler(reqRepo)
	listRequestsHandler := queries.NewListRequestsHandler(reqRepo)
	getGlobalStatsHandler := queries.NewGetGlobalStatsHandler(statsRepo)

	// Auth error handler maps middleware errors to dto errors
	handleAuthError := func(w http.ResponseWriter, r *http.Request, err error) {
		var dtoErr error

		switch {
		case errors.Is(err, middleware.ErrMissingAuthHeader):
			dtoErr = dto.ErrMissingAuthHeader
		case errors.Is(err, middleware.ErrInvalidAuthFormat):
			dtoErr = dto.ErrInvalidAuthFormat
		case errors.Is(err, middleware.ErrInvalidAPIKey):
			dtoErr = dto.ErrInvalidAPIKey
		default:
			dtoErr = dto.ErrInvalidAPIKey
		}

		// Use AppHandler for automatic RFC 7807 formatting
		handlers.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
			return dtoErr
		}).ServeHTTP(w, r)
	}

	// Rate limit error handler maps to dto error
	handleRateLimitError := func(w http.ResponseWriter, r *http.Request) {
		// Use AppHandler for automatic RFC 7807 formatting
		handlers.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
			return dto.ErrRateLimitExceeded
		}).ServeHTTP(w, r)
	}

	// Create rate limiter with explicit lifecycle management
	rateLimiter := ratelimit.NewLimiter(cfg.App.RateLimit)
	defer func() {
		if err := rateLimiter.Stop(); err != nil {
			logger.Error("Failed to stop rate limiter", "error", err)
		}
	}()

	// Start cleanup job if retention is configured
	if cfg.App.Retention > 0 {
		cleanupJob := jobs.NewCleanupJob(reqRepo, cfg.App.Retention, cfg.App.CleanupInterval, logger)
		cleanupJob.Start()
		defer cleanupJob.Stop()
	}

	// Metrics platform: newAPIMetrics builds an isolated registry holding the
	// runtime/process, build-info and DB-pool collectors with an OTel
	// MeterProvider bridged onto it, plus the shared domain comparison recorder.
	// When disabled the platform and recorder are nil, so per-route
	// instrumentation and comparison recording become no-ops and no endpoint is
	// mounted.
	metricsPlatform, recorder, err := newAPIMetrics(cfg.App.MetricsEnabled, db)
	if err != nil {
		logger.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	if recorder != nil {
		// Subscribe the comparison metrics listener so RequestCompared events
		// recorded by the aggregate are translated into the shared business
		// metrics after each persisted comparison.
		eventBus.Subscribe(traffictesting.EventRequestCompared, newComparisonMetricsListener(recorder))
		logger.Info("Metrics enabled", "endpoint", "/metrics")
	}

	// instrument wraps each API route with server-side otelhttp instrumentation so
	// it lands on the semconv http_server_request_duration_seconds histogram, with
	// the http_route label auto-derived from the matched ServeMux pattern. When
	// metrics are disabled the platform method returns the handler unwrapped.
	instrument := metricsPlatform.InstrumentHandler

	// Middleware
	baseChain := middleware.Chain{
		middleware.RequestID(),
		middleware.Logging(logger),
		middleware.RateLimit(rateLimiter,
			middleware.WithIPExtractor(middleware.ExtractIPWithForwardedFor),
			middleware.WithRateLimitErrorHandler(handleRateLimitError),
		),
		middleware.APIKeyAuth(cfg.App.APIKey,
			middleware.WithAuthErrorHandler(handleAuthError),
		),
	}

	// Middleware chain for POST endpoints with body size limit
	postChain := middleware.Chain{
		middleware.RequestID(),
		middleware.Logging(logger),
		middleware.RateLimit(rateLimiter,
			middleware.WithIPExtractor(middleware.ExtractIPWithForwardedFor),
			middleware.WithRateLimitErrorHandler(handleRateLimitError),
		),
		middleware.APIKeyAuth(cfg.App.APIKey,
			middleware.WithAuthErrorHandler(handleAuthError),
		),
		middleware.MaxBodySize(cfg.App.MaxBodySize),
	}

	// Interface Layer: HTTP Handlers
	createGate := handlers.CreateGate(createGateHandler)
	updateGate := handlers.UpdateGate(updateGateHandler)
	deleteGate := handlers.DeleteGate(deleteGateHandler)
	getGateByID := handlers.GetGateByID(getGateHandler)
	getAllGates := handlers.GetAllGates(listGatesHandler)

	createRequest := handlers.CreateRequest(createRequestHandler)
	getRequestByID := handlers.GetRequestByID(getRequestHandler)
	getAllRequestsByGateID := handlers.GetAllRequestsByGateID(listRequestsHandler)
	getGlobalStats := handlers.GetGlobalStats(getGlobalStatsHandler)

	mux := http.NewServeMux()

	// Health check endpoints (no middleware to avoid logging noise)
	mux.Handle("GET /health/live", handlers.Liveness())
	mux.Handle("GET /health/ready", handlers.Readiness(healthChecker{db: db}))

	// Metrics endpoint (no auth, no middleware) so Prometheus can scrape it the
	// same way as the health probes. Only mounted when metrics are enabled.
	if h := metricsPlatform.MetricsHandler(); h != nil {
		mux.Handle("GET /metrics", h)
	}

	// API endpoints (with middleware). Each route handler is also wrapped with
	// otelhttp, which records the semconv http_server_* metrics and derives the
	// bounded http_route label from the templated ServeMux pattern.
	mux.Handle("GET /stats", instrument("GET /stats", baseChain.Then(getGlobalStats)))
	mux.Handle("GET /gates", instrument("GET /gates", baseChain.Then(getAllGates)))
	mux.Handle("POST /gates", instrument("POST /gates", postChain.Then(createGate)))
	mux.Handle("PATCH /gates/{gate_id}", instrument("PATCH /gates/{gate_id}", postChain.Then(updateGate)))
	mux.Handle("DELETE /gates/{gate_id}", instrument("DELETE /gates/{gate_id}", baseChain.Then(deleteGate)))
	mux.Handle("GET /gates/{gate_id}", instrument("GET /gates/{gate_id}", baseChain.Then(getGateByID)))
	mux.Handle("GET /gates/{gate_id}/requests", instrument("GET /gates/{gate_id}/requests", baseChain.Then(getAllRequestsByGateID)))
	mux.Handle("POST /gates/{gate_id}/requests", instrument("POST /gates/{gate_id}/requests", postChain.Then(createRequest)))
	mux.Handle("GET /gates/{gate_id}/requests/{request_id}", instrument("GET /gates/{gate_id}/requests/{request_id}", baseChain.Then(getRequestByID)))

	// Wrap mux with CORS if configured (before auth/rate-limiting so
	// preflight OPTIONS requests are handled without credentials).
	var handler http.Handler = mux
	if origins := cfg.ParseCORSOrigins(); len(origins) > 0 {
		handler = cors.New(cors.Options{
			AllowedOrigins: origins,
			AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         86400,
		}).Handler(mux)
		logger.Info("CORS enabled", "origins", origins)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      handler,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Starting server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("Server failed to start", "error", err)
		return
	case sig := <-stop:
		logger.Info("Shutting down server", "signal", sig.String())
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", "error", err)
	} else {
		logger.Info("Server stopped")
	}

	// Flush and release the metrics MeterProvider (no-op when disabled).
	if err := metricsPlatform.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down metrics", "error", err)
	}

	// Close database connection
	if client != nil {
		_ = client.Close()
		clientClosed = true
		logger.Info("Database connection closed")
	}
}

// newAPIMetrics builds the API metrics platform (isolated registry with the
// runtime/process, build-info and DB-pool collectors plus an OTel MeterProvider
// bridged onto it) and the shared domain comparison recorder. It returns the
// platform and the recorder. When enabled is false it returns nil, nil so every
// instrumentation seam is a no-op and no endpoint is mounted.
func newAPIMetrics(enabled bool, db *sql.DB) (*metrics.Platform, *diffmetrics.Recorder, error) {
	if !enabled {
		return nil, nil, nil
	}
	platform, err := metrics.NewPlatform(metrics.WithDBStats(db, "mroki"))
	if err != nil {
		return nil, nil, err
	}
	recorder, err := diffmetrics.New(platform.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("create comparison recorder: %w", err)
	}
	return platform, recorder, nil
}

// healthChecker wraps *sql.DB to implement the handlers.HealthChecker interface.
type healthChecker struct {
	db *sql.DB
}

func (h healthChecker) Ping(ctx context.Context) error {
	return h.db.PingContext(ctx)
}
