package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pedrobarco/mroki/cmd/mroki-api/config"
	"github.com/pedrobarco/mroki/internal/application/commands"
	appqueries "github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres/db"
	"github.com/pedrobarco/mroki/internal/interfaces/http/handlers"
	"github.com/pedrobarco/mroki/internal/interfaces/http/middleware"
	"github.com/pedrobarco/mroki/pkg/logger"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	logger := logger.New()

	// Parse pool configuration timeouts (safe after validation)
	maxConnIdleDuration, _ := time.ParseDuration(cfg.App.Database.MaxConnIdle)
	maxConnLifeDuration, _ := time.ParseDuration(cfg.App.Database.MaxConnLife)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.App.Database.URL.String())
	if err != nil {
		panic(fmt.Errorf("failed to parse database URL: %w", err))
	}

	poolConfig.MaxConns = cfg.App.Database.MaxConns
	poolConfig.MinConns = cfg.App.Database.MinConns
	poolConfig.MaxConnIdleTime = maxConnIdleDuration
	poolConfig.MaxConnLifetime = maxConnLifeDuration

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create connection pool: %w", err))
	}
	defer pool.Close()

	queries := db.New(pool)

	// Infrastructure Layer: Repository implementations
	gateRepo := postgres.NewGateRepository(queries)
	reqRepo := postgres.NewRequestRepository(queries, pool)

	// Application Layer: Command Handlers (Write operations)
	createGateHandler := commands.NewCreateGateHandler(gateRepo)
	createRequestHandler := commands.NewCreateRequestHandler(reqRepo)

	// Application Layer: Query Handlers (Read operations)
	getGateHandler := appqueries.NewGetGateHandler(gateRepo)
	listGatesHandler := appqueries.NewListGatesHandler(gateRepo)
	getRequestHandler := appqueries.NewGetRequestHandler(reqRepo)
	listRequestsHandler := appqueries.NewListRequestsHandler(reqRepo)

	// Middleware
	baseChain := middleware.Chain{
		middleware.Logging(logger),
	}

	// Middleware chain for POST endpoints with body size limit
	postChain := middleware.Chain{
		middleware.Logging(logger),
		middleware.MaxBodySize(cfg.App.MaxBodySize),
	}

	// Interface Layer: HTTP Handlers
	createGate := handlers.CreateGate(createGateHandler)
	getGateByID := handlers.GetGateByID(getGateHandler)
	getAllGates := handlers.GetAllGates(listGatesHandler)

	createRequest := handlers.CreateRequest(createRequestHandler)
	getRequestByID := handlers.GetRequestByID(getRequestHandler)
	getAllRequestsByGateID := handlers.GetAllRequestsByGateID(listRequestsHandler)

	mux := http.NewServeMux()

	// Health check endpoints (no middleware to avoid logging noise)
	mux.Handle("GET /health/live", handlers.Liveness())
	mux.Handle("GET /health/ready", handlers.Readiness(pool))

	// API endpoints (with middleware)
	mux.Handle("GET /gates", baseChain.Then(getAllGates))
	mux.Handle("POST /gates", postChain.Then(createGate))
	mux.Handle("GET /gates/{gate_id}", baseChain.Then(getGateByID))
	mux.Handle("GET /gates/{gate_id}/requests", baseChain.Then(getAllRequestsByGateID))
	mux.Handle("POST /gates/{gate_id}/requests", postChain.Then(createRequest))
	mux.Handle("GET /gates/{gate_id}/requests/{request_id}", baseChain.Then(getRequestByID))
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second, // Time to read request
		WriteTimeout: 30 * time.Second, // Time to write response
		IdleTimeout:  60 * time.Second, // Keep-alive timeout
	}

	logger.Info("Started server",
		"address", server.Addr,
	)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
