package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pedrobarco/mroki/cmd/mroki-api/config"
	"github.com/pedrobarco/mroki/internal/domain/diffing"
	"github.com/pedrobarco/mroki/internal/handlers"
	"github.com/pedrobarco/mroki/internal/middleware"
	"github.com/pedrobarco/mroki/internal/storage/postgres"
	"github.com/pedrobarco/mroki/internal/storage/postgres/db"
	"github.com/pedrobarco/mroki/pkg/logger"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	logger := logger.New()

	// Parse pool configuration timeouts
	maxConnIdleDuration, err := time.ParseDuration(cfg.App.Database.MaxConnIdle)
	if err != nil {
		panic(fmt.Errorf("failed to parse MAX_CONN_IDLE: %w", err))
	}

	maxConnLifeDuration, err := time.ParseDuration(cfg.App.Database.MaxConnLife)
	if err != nil {
		panic(fmt.Errorf("failed to parse MAX_CONN_LIFE: %w", err))
	}

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

	gateRepo := postgres.NewGateRepository(queries)
	gateSvc := diffing.NewGateService(gateRepo)

	reqRepo := postgres.NewRequestRepository(queries, pool)
	reqSvc := diffing.NewRequestService(reqRepo)

	baseChain := middleware.Chain{
		middleware.Logging(logger),
	}

	getAllGates := handlers.GetAllGates(gateSvc)
	createGate := handlers.CreateGate(gateSvc)
	getGateByID := handlers.GetGateByID(gateSvc)

	getAllRequestsByGateID := handlers.GetAllRequestsByGateID(reqSvc)
	createRequest := handlers.CreateRequest(reqSvc)
	getRequestByID := handlers.GetRequestByID(reqSvc)

	mux := http.NewServeMux()

	// Health check endpoints (no middleware to avoid logging noise)
	mux.Handle("GET /health/live", handlers.Liveness())
	mux.Handle("GET /health/ready", handlers.Readiness(pool))

	// API endpoints (with middleware)
	mux.Handle("GET /gates", baseChain.Then(getAllGates))
	mux.Handle("POST /gates", baseChain.Then(createGate))
	mux.Handle("GET /gates/{gate_id}", baseChain.Then(getGateByID))
	mux.Handle("GET /gates/{gate_id}/requests", baseChain.Then(getAllRequestsByGateID))
	mux.Handle("POST /gates/{gate_id}/requests", baseChain.Then(createRequest))
	mux.Handle("GET /gates/{gate_id}/requests/{request_id}", baseChain.Then(getRequestByID))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: mux,
	}

	logger.Info("Started server",
		"address", server.Addr,
	)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
