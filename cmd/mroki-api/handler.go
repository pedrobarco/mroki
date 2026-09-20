package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/pedrobarco/mroki/cmd/mroki-api/config"
	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/events"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/interfaces/http/handlers"
	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/pedrobarco/mroki/pkg/dto"
	"github.com/pedrobarco/mroki/pkg/metrics"
	"github.com/pedrobarco/mroki/pkg/ratelimit"
	"github.com/rs/cors"
)

// handlerDeps bundles the dependencies required to assemble the HTTP handler.
type handlerDeps struct {
	cfg         config.Config
	logger      *slog.Logger
	gateRepo    traffictesting.GateRepository
	requestRepo traffictesting.RequestRepository
	statsRepo   traffictesting.StatsRepository
	dispatcher  events.Dispatcher
	limiter     *ratelimit.Limiter
	metrics     *metrics.Platform
	health      handlers.HealthChecker
}

// newHandler builds the fully assembled HTTP handler: mux, infrastructure routes,
// API middleware chains, optional CORS, and security headers as the outermost wrap.
func newHandler(d handlerDeps) (http.Handler, error) {
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

	// Rate-limit IP extractor. X-Forwarded-For is honored only when a request's
	// direct peer is a configured trusted proxy; otherwise the limiter keys off
	// RemoteAddr so clients cannot spoof their source IP to evade per-IP limits.
	ipExtractor, err := middleware.NewForwardedForExtractor(d.cfg.ParseTrustedProxies())
	if err != nil {
		return nil, err
	}

	// Application Layer: Command Handlers (Write operations)
	createGateHandler := commands.NewCreateGateHandler(d.gateRepo)
	updateGateHandler := commands.NewUpdateGateHandler(d.gateRepo, d.cfg.App.Retention)
	deleteGateHandler := commands.NewDeleteGateHandler(d.gateRepo)
	createRequestHandler := commands.NewCreateRequestHandler(d.requestRepo, d.gateRepo, commands.WithEventDispatcher(d.dispatcher))

	// Application Layer: Query Handlers (Read operations)
	getGateHandler := queries.NewGetGateHandler(d.gateRepo, d.statsRepo)
	listGatesHandler := queries.NewListGatesHandler(d.gateRepo, d.statsRepo)
	getRequestHandler := queries.NewGetRequestHandler(d.requestRepo)
	listRequestsHandler := queries.NewListRequestsHandler(d.requestRepo)
	getGlobalStatsHandler := queries.NewGetGlobalStatsHandler(d.statsRepo)

	// Middleware
	baseChain := middleware.Chain{
		middleware.RequestID(),
		middleware.Logging(d.logger),
		middleware.RateLimit(d.limiter,
			middleware.WithIPExtractor(ipExtractor),
			middleware.WithRateLimitErrorHandler(handleRateLimitError),
		),
		middleware.APIKeyAuth(d.cfg.App.APIKey,
			middleware.WithAuthErrorHandler(handleAuthError),
		),
	}

	// Middleware chain for POST endpoints with body size limit
	postChain := middleware.Chain{
		middleware.RequestID(),
		middleware.Logging(d.logger),
		middleware.RateLimit(d.limiter,
			middleware.WithIPExtractor(ipExtractor),
			middleware.WithRateLimitErrorHandler(handleRateLimitError),
		),
		middleware.APIKeyAuth(d.cfg.App.APIKey,
			middleware.WithAuthErrorHandler(handleAuthError),
		),
		middleware.MaxBodySize(d.cfg.App.MaxBodySize),
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
	getConfig := handlers.GetConfig(d.cfg.App.Retention)

	mux := http.NewServeMux()

	// instrument wraps each API route with server-side otelhttp instrumentation so
	// it lands on the semconv http_server_request_duration_seconds histogram, with
	// the http_route label auto-derived from the matched ServeMux pattern. When
	// metrics are disabled the platform method returns the handler unwrapped.
	instrument := d.metrics.InstrumentHandler

	// Infrastructure endpoints (health probes + metrics) are mounted without the
	// API middleware chain so probes stay quiet and Prometheus can scrape freely.
	mountInfraRoutes(mux, d.health, d.metrics.MetricsHandler())

	// API endpoints (with middleware). Each route handler is also wrapped with
	// otelhttp, which records the semconv http_server_* metrics and derives the
	// bounded http_route label from the templated ServeMux pattern.
	mux.Handle("GET /config", instrument("GET /config", baseChain.Then(getConfig)))
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
	if origins := d.cfg.ParseCORSOrigins(); len(origins) > 0 {
		handler = cors.New(cors.Options{
			AllowedOrigins: origins,
			AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         86400,
		}).Handler(mux)
		d.logger.Info("CORS enabled", "origins", origins)
	}

	// Wrap the whole handler with security headers as the outermost layer so
	// every response — including /health and /metrics — carries them. HSTS is
	// only emitted when explicitly enabled (mroki does not terminate TLS).
	handler = withSecurityHeaders(handler, d.cfg)

	return handler, nil
}

// mountInfraRoutes mounts the unauthenticated infrastructure endpoints — the
// health probes and, when metrics are enabled, the Prometheus scrape endpoint —
// onto mux. These deliberately skip the API middleware chain so probes stay
// quiet and scrapes stay unauthenticated. metricsHandler may be nil (metrics
// disabled), in which case /metrics is not mounted.
func mountInfraRoutes(mux *http.ServeMux, hc handlers.HealthChecker, metricsHandler http.Handler) {
	mux.Handle("GET /health/live", handlers.Liveness())
	mux.Handle("GET /health/ready", handlers.Readiness(hc))
	if metricsHandler != nil {
		mux.Handle("GET /metrics", metricsHandler)
	}
}

// withSecurityHeaders wraps h with the security-headers middleware using the
// configured HSTS settings. It is applied as the outermost layer so every
// response — including the infrastructure endpoints — carries the always-on
// security headers. HSTS is only emitted when explicitly enabled.
func withSecurityHeaders(h http.Handler, cfg config.Config) http.Handler {
	return middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
		HSTSEnabled: cfg.App.HSTSEnabled,
		HSTSMaxAge:  cfg.App.HSTSMaxAge,
	})(h)
}
