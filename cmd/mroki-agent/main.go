package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pedrobarco/mroki/cmd/mroki-agent/config"
	"github.com/pedrobarco/mroki/cmd/mroki-agent/handlers"
	"github.com/pedrobarco/mroki/pkg/client"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/dto"
	"github.com/pedrobarco/mroki/pkg/logger"
)

func main() {
	cfg := config.Load()

	slog.SetLogLoggerLevel(slog.LevelDebug)
	log := logger.New()

	// Load or generate agent ID
	agentID, err := loadOrGenerateAgentID()
	if err != nil {
		log.Error("Failed to load/generate agent ID", "error", err)
		return
	}
	log.Info("Agent ID loaded", "agent_id", agentID)

	// Determine mode and configure URLs
	var liveURL, shadowURL *url.URL
	var apiClient *client.MrokiClient

	if cfg.App.APIURL != nil && cfg.App.GateID != "" && cfg.App.APIKey != "" {
		// API Mode: Fetch gate configuration from API
		log.Info("Starting in API mode",
			"api_url", cfg.App.APIURL.String(),
			"gate_id", cfg.App.GateID,
		)

		// Create API client
		httpClient := &http.Client{
			Timeout: cfg.App.APITimeout,
		}

		apiClient = client.NewMrokiClient(
			cfg.App.APIURL,
			cfg.App.GateID,
			agentID,
			cfg.App.APIKey,
			client.WithHTTPClient(httpClient),
			client.WithMaxRetries(cfg.App.MaxRetries),
			client.WithInitialDelay(cfg.App.RetryDelay),
			client.WithLogger(log),
		)

		// Fetch gate configuration with retry
		gate, err := fetchGateWithRetry(apiClient, cfg.App.MaxRetries, cfg.App.RetryDelay, log)
		if err != nil {
			log.Error("Failed to fetch gate configuration after retries", "error", err)
			return
		}

		// Parse gate URLs
		liveURL, err = url.Parse(gate.LiveURL)
		if err != nil {
			log.Error("Invalid live URL received from API", "error", err, "url", gate.LiveURL)
			return
		}

		shadowURL, err = url.Parse(gate.ShadowURL)
		if err != nil {
			log.Error("Invalid shadow URL received from API", "error", err, "url", gate.ShadowURL)
			return
		}

		log.Info("Gate configuration loaded",
			"gate_id", gate.ID,
			"live_url", liveURL.String(),
			"shadow_url", shadowURL.String(),
		)

	} else {
		// Standalone Mode: Use URLs from .env
		log.Info("Starting in standalone mode")

		liveURL = cfg.App.LiveURL
		shadowURL = cfg.App.ShadowURL

		log.Debug("Using URLs from environment",
			"live_url", liveURL.String(),
			"shadow_url", shadowURL.String(),
		)
	}

	// Build diff options from config (works in both modes)
	var diffOpts []diff.Option

	if len(cfg.App.DiffIgnoredFields) > 0 {
		diffOpts = append(diffOpts, diff.WithIgnoredFields(cfg.App.DiffIgnoredFields...))
	}

	if len(cfg.App.DiffIncludedFields) > 0 {
		diffOpts = append(diffOpts, diff.WithIncludedFields(cfg.App.DiffIncludedFields...))
	}

	if cfg.App.DiffFloatTolerance > 0 {
		diffOpts = append(diffOpts, diff.WithFloatTolerance(cfg.App.DiffFloatTolerance))
	}

	log.Debug("Diff options configured",
		"ignored_fields", cfg.App.DiffIgnoredFields,
		"included_fields", cfg.App.DiffIncludedFields,
		"float_tolerance", cfg.App.DiffFloatTolerance,
	)

	// Configure proxy handler
	proxyConfig := handlers.ProxyConfig{
		Live:          liveURL,
		Shadow:        shadowURL,
		LiveTimeout:   cfg.App.LiveTimeout,
		ShadowTimeout: cfg.App.ShadowTimeout,
		MaxBodySize:   cfg.App.MaxBodySize,
		AgentID:       agentID,
		Logger:        log,
		APIClient:     apiClient, // nil if standalone mode
		DiffOptions:   diffOpts,  // NEW: Pass diff options
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
		log.Info("Proxy server started",
			"address", server.Addr,
			"mode", getModeString(apiClient),
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

// fetchGateWithRetry fetches gate config with exponential backoff retry
func fetchGateWithRetry(client *client.MrokiClient, maxRetries int, initialDelay time.Duration, log *slog.Logger) (*dto.Gate, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			delay := initialDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			log.Warn("Retrying gate fetch after failure",
				"attempt", attempt+1,
				"max_attempts", maxRetries+1,
				"delay", delay,
			)
			time.Sleep(delay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		gate, err := client.GetGate(ctx)
		cancel()

		if err == nil {
			if attempt > 0 {
				log.Info("Gate fetch succeeded after retry", "attempts", attempt+1)
			}
			return gate, nil
		}

		lastErr = err
		log.Debug("Gate fetch attempt failed",
			"attempt", attempt+1,
			"max_attempts", maxRetries+1,
			"error", err,
		)
	}

	return nil, fmt.Errorf("failed to fetch gate after %d attempts: %w", maxRetries+1, lastErr)
}

func getModeString(apiClient *client.MrokiClient) string {
	if apiClient != nil {
		return "api"
	}
	return "standalone"
}
