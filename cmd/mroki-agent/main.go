package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/pedrobarco/mroki/cmd/mroki-agent/config"
	"github.com/pedrobarco/mroki/cmd/mroki-agent/handlers"
	"github.com/pedrobarco/mroki/pkg/client"
	"github.com/pedrobarco/mroki/pkg/logger"
)

func main() {
	cfg := config.Load()

	log := logger.New()

	// Load or generate agent ID
	agentID, err := loadOrGenerateAgentID()
	if err != nil {
		log.Error("Failed to load/generate agent ID", "error", err)
		return
	}
	log.Info("Agent ID loaded", "agent_id", agentID)

	// Configure proxy handler
	proxyConfig := handlers.ProxyConfig{
		Live:          cfg.App.LiveURL,
		Shadow:        cfg.App.ShadowURL,
		LiveTimeout:   cfg.App.LiveTimeout,
		ShadowTimeout: cfg.App.ShadowTimeout,
		MaxBodySize:   cfg.App.MaxBodySize,
		AgentID:       agentID,
		Logger:        log,
	}

	// Create API client if configured
	if cfg.App.APIURL != nil && cfg.App.GateID != "" {
		httpClient := &http.Client{
			Timeout: cfg.App.APITimeout,
		}

		apiClient := client.NewMrokiClient(
			cfg.App.APIURL,
			cfg.App.GateID,
			agentID,
			client.WithHTTPClient(httpClient),
			client.WithMaxRetries(cfg.App.MaxRetries),
			client.WithInitialDelay(cfg.App.RetryDelay),
			client.WithLogger(log),
		)

		proxyConfig.APIClient = apiClient

		log.Info("API integration enabled",
			"api_url", cfg.App.APIURL.String(),
			"gate_id", cfg.App.GateID,
			"max_retries", cfg.App.MaxRetries,
			"retry_delay", cfg.App.RetryDelay,
		)
	} else {
		log.Info("Running in standalone mode (no API integration)")
	}

	mux := http.NewServeMux()
	mux.Handle("/", handlers.Proxy(proxyConfig))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,  // Time to read request
		WriteTimeout: 60 * time.Second,  // Time to write response
		IdleTimeout:  120 * time.Second, // Keep-alive timeout
	}

	log.Info("Started server",
		"live", cfg.App.LiveURL.String(),
		"shadow", cfg.App.ShadowURL.String(),
		"address", server.Addr,
		"agent_id", agentID,
	)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
