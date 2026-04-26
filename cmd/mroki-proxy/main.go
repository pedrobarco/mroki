package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pedrobarco/mroki/cmd/mroki-proxy/config"
	"github.com/pedrobarco/mroki/cmd/mroki-proxy/handlers"
	"github.com/pedrobarco/mroki/pkg/client"
	"github.com/pedrobarco/mroki/pkg/client/transport"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/logger"
	"github.com/pedrobarco/mroki/pkg/proxy"
)

func main() {
	cfg := config.Load()

	slog.SetLogLoggerLevel(slog.LevelDebug)
	log := logger.New()

	// Determine mode and configure URLs
	var liveURL, shadowURL *url.URL
	var apiClient *client.MrokiClient

	if cfg.App.APIURL != nil && cfg.App.GateID != "" && cfg.App.APIKey != "" {
		// API Mode: Fetch gate configuration from API
		log.Info("Starting in API mode",
			"api_url", cfg.App.APIURL.String(),
			"gate_id", cfg.App.GateID,
		)

		// Create resilient HTTP client with retry + circuit breaker
		httpClient := transport.NewHTTPClient(transport.Config{
			APIKey:             cfg.App.APIKey,
			MaxRetries:         cfg.App.MaxRetries,
			InitialDelay:       cfg.App.RetryDelay,
			CBFailureThreshold: cfg.App.CBFailureThreshold,
			CBDelay:            cfg.App.CBDelay,
			CBSuccessThreshold: cfg.App.CBSuccessThreshold,
			Logger:             log,
		})

		apiClient = client.NewMrokiClient(
			cfg.App.APIURL,
			cfg.App.GateID,
			client.WithHTTPClient(httpClient),
			client.WithLogger(log),
		)

		// Fetch gate configuration (retries handled by the resilient HTTP client)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.App.APITimeout)
		gate, err := apiClient.GetGate(ctx)
		cancel()
		if err != nil {
			log.Error("Failed to fetch gate configuration", "error", err)
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
	// Configure sampling rate
	samplingRate, err := proxy.NewSamplingRate(cfg.App.SamplingRate)
	if err != nil {
		panic(fmt.Errorf("invalid sampling rate: %w", err))
	}
	log.Info("Sampling rate configured", slog.Float64("rate", cfg.App.SamplingRate))

	proxyConfig := handlers.ProxyConfig{
		Live:          liveURL,
		Shadow:        shadowURL,
		LiveTimeout:   cfg.App.LiveTimeout,
		ShadowTimeout: cfg.App.ShadowTimeout,
		MaxBodySize:   cfg.App.MaxBodySize,
		SamplingRate:  samplingRate,
		Logger:        log,
		APIClient:     apiClient,          // nil if standalone mode
		APITimeout:    cfg.App.APITimeout, // overall deadline for API calls
		DiffOptions:   diffOpts,  // Only used in standalone mode
	}

	mux := http.NewServeMux()
	mux.Handle("/", handlers.Proxy(proxyConfig))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      mux,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
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

func getModeString(apiClient *client.MrokiClient) string {
	if apiClient != nil {
		return "api"
	}
	return "standalone"
}
