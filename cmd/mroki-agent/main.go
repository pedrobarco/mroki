package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pedrobarco/mroki/cmd/mroki-agent/config"
	"github.com/pedrobarco/mroki/cmd/mroki-agent/handlers"
	"github.com/pedrobarco/mroki/pkg/logger"
)

func main() {
	cfg := config.Load()

	logger := logger.New()

	mux := http.NewServeMux()
	mux.Handle("/", handlers.Proxy(cfg.App.LiveURL, cfg.App.ShadowURL))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: mux,
	}

	logger.Info("Started server",
		"live", cfg.App.LiveURL.String(),
		"shadow", cfg.App.ShadowURL.String(),
		"address", server.Addr,
	)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
