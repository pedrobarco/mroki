package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
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

	conn, err := pgx.Connect(ctx, cfg.App.Database.URL.String())
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}
	defer func() {
		if err := conn.Close(ctx); err != nil {
			slog.Error("failed to close database connection", "error", err)
		}
	}()

	queries := db.New(conn)

	gateRepo := postgres.NewGateRepository(queries)
	gateSvc := diffing.NewGateService(gateRepo)

	reqRepo := postgres.NewRequestRepository(queries, conn)
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
