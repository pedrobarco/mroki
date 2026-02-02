package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting server",
			"live", cfg.App.LiveURL.String(),
			"shadow", cfg.App.ShadowURL.String(),
			"address", server.Addr,
			"agent_id", agentID,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("Server failed to start", "error", err)
		return
	case sig := <-stop:
		log.Info("Shutting down server", "signal", sig.String())
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Error during shutdown", "error", err)
	} else {
		log.Info("Server stopped")
	}
}
