package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pedrobarco/mroki/cmd/mroki-proxy/config"
	"github.com/pedrobarco/mroki/cmd/mroki-proxy/handlers"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/client"
	"github.com/pedrobarco/mroki/pkg/client/transport"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/logger"
	"github.com/pedrobarco/mroki/pkg/proxy"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		var verr *config.ValidationError
		if errors.As(err, &verr) {
			for _, w := range verr.Warnings() {
				log.Warn("Configuration warning", "detail", w.Message)
			}
			if verr.HasErrors() {
				log.Error("Configuration validation failed", "error", verr.Error())
				os.Exit(1)
			}
		} else {
			log.Error("Configuration loading failed", "error", err)
			os.Exit(1)
		}
	}

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

	// Exclude the shadow identification header from diff comparison so it never
	// shows up as a difference. It is not redacted — its value stays visible for
	// reference in stored request data.
	diffOpts = append(diffOpts, diff.WithIgnoredFields("headers."+proxy.ShadowHeader))

	if len(cfg.App.DiffIgnoredFields) > 0 {
		diffOpts = append(diffOpts, diff.WithIgnoredFields(cfg.App.DiffIgnoredFields...))
	}

	if len(cfg.App.DiffIncludedFields) > 0 {
		diffOpts = append(diffOpts, diff.WithIncludedFields(cfg.App.DiffIncludedFields...))
	}

	if cfg.App.DiffFloatTolerance > 0 {
		diffOpts = append(diffOpts, diff.WithFloatTolerance(cfg.App.DiffFloatTolerance))
	}

	if cfg.App.DiffSortArrays {
		diffOpts = append(diffOpts, diff.WithSortArrays(true))
	}

	// Build redactor from config (adds to default redacted list)
	redactedFieldsCfg, err := traffictesting.NewRedactedFields(cfg.App.RedactedFields)
	if err != nil {
		log.Error("Invalid REDACTED_FIELDS configuration", "error", err)
		os.Exit(1)
	}
	redactor := traffictesting.NewRedactor(redactedFieldsCfg.AllFields())

	// Add redacted fields as ignored diff fields (prevents diff noise from redacted values)
	for _, f := range redactedFieldsCfg.AllFields() {
		diffOpts = append(diffOpts, diff.WithIgnoredFields(f))
	}

	log.Debug("Diff options configured",
		"ignored_fields", cfg.App.DiffIgnoredFields,
		"included_fields", cfg.App.DiffIncludedFields,
		"float_tolerance", cfg.App.DiffFloatTolerance,
		"sort_arrays", cfg.App.DiffSortArrays,
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
		DiffOptions:   diffOpts,           // Only used in standalone mode
		Redactor:      redactor,           // Only used in standalone mode
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

	// Admin server hosts the health endpoints on a dedicated port so they never
	// collide with proxied traffic on the main listener. ready signals whether
	// the proxy should receive traffic; it is flipped on once startup completes
	// and back off when graceful shutdown begins.
	var ready atomic.Bool
	adminMux := http.NewServeMux()
	adminMux.Handle("/health/live", handlers.Liveness())
	adminMux.Handle("/health/ready", handlers.Readiness(&ready))

	adminServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.AdminPort),
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// serverErrors is buffered for both listeners so a failing goroutine never
	// blocks on send.
	serverErrors := make(chan error, 2)

	go func() {
		log.Info("Admin server started", "address", adminServer.Addr)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("admin server: %w", err)
		}
	}()

	// Start server in goroutine
	go func() {
		log.Info("Proxy server started",
			"address", server.Addr,
			"mode", getModeString(apiClient),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("proxy server: %w", err)
		}
	}()

	// Configuration is loaded and both listeners are starting; report ready.
	ready.Store(true)

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

	// Stop reporting ready so readiness probes drain this pod before the
	// listeners close.
	ready.Store(false)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Error during shutdown", "error", err)
	} else {
		log.Info("Server stopped")
	}

	if err := adminServer.Shutdown(shutdownCtx); err != nil {
		log.Error("Error during admin server shutdown", "error", err)
	} else {
		log.Info("Admin server stopped")
	}
}

func getModeString(apiClient *client.MrokiClient) string {
	if apiClient != nil {
		return "api"
	}
	return "standalone"
}
