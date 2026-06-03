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
	diffmetrics "github.com/pedrobarco/mroki/pkg/diff/metrics"
	"github.com/pedrobarco/mroki/pkg/logger"
	"github.com/pedrobarco/mroki/pkg/metrics"
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

	// Metrics platform: newProxyMetrics builds an isolated registry with the
	// runtime/process collectors and an OTel MeterProvider bridged onto it, plus
	// the shared domain comparison recorder. The scrape handler is mounted on the
	// admin port. Created before the clients so their transports can be
	// instrumented with otelhttp. When disabled, the platform and recorder are
	// nil so every instrumentation seam is a no-op and no endpoint is mounted.
	metricsPlatform, recorder, err := newProxyMetrics(cfg.App.MetricsEnabled)
	if err != nil {
		log.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	if metricsPlatform != nil {
		log.Info("Metrics enabled", "endpoint", "/metrics", "port", cfg.App.AdminPort)
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

		// Instrument the API client transport (mroki.target="api") so outbound
		// calls to the API are timed alongside live/shadow traffic on the shared
		// semconv http_client_request_duration_seconds histogram. No-op when
		// metrics are disabled.
		httpClient.Transport = metricsPlatform.InstrumentClient("api", httpClient.Transport)

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

	// Parse shadow rules. User-supplied rules (if any) are evaluated first; the
	// base rules (deny non-idempotent methods) are always appended as the final
	// catch-all so the write-protection cannot be accidentally dropped.
	var userShadowRules []proxy.ShadowRule
	if cfg.App.ShadowRules != "" {
		userShadowRules, err = proxy.ParseShadowRules(cfg.App.ShadowRules)
		if err != nil {
			log.Error("Invalid SHADOW_RULES configuration", "error", err)
			os.Exit(1)
		}
		log.Info("Custom shadow rules configured", slog.Int("count", len(userShadowRules)))
	}
	baseShadowRules := proxy.BaseShadowRules()
	shadowRules := append(userShadowRules, baseShadowRules...)
	log.Info("Shadow rules active",
		slog.Int("user_rules", len(userShadowRules)),
		slog.Int("base_rules", len(baseShadowRules)),
	)

	// Configure proxy handler
	// Configure sampling rate
	samplingRate, err := proxy.NewSamplingRate(cfg.App.SamplingRate)
	if err != nil {
		panic(fmt.Errorf("invalid sampling rate: %w", err))
	}
	log.Info("Sampling rate configured", slog.Float64("rate", cfg.App.SamplingRate))

	// instrumentUpstream wraps the outbound live/shadow client transport with
	// client-side otelhttp, resolving the mroki.target attribute per request from
	// the outbound host (live, shadow, or unknown). The platform method is a no-op
	// when metrics are disabled, returning the transport unwrapped.
	var liveHost, shadowHost string
	if liveURL != nil {
		liveHost = liveURL.Host
	}
	if shadowURL != nil {
		shadowHost = shadowURL.Host
	}
	targetFn := func(req *http.Request) string {
		switch req.URL.Host {
		case liveHost:
			return "live"
		case shadowHost:
			return "shadow"
		default:
			return "unknown"
		}
	}
	instrumentUpstream := func(rt http.RoundTripper) http.RoundTripper {
		return metricsPlatform.InstrumentClientFunc(targetFn, rt)
	}

	proxyConfig := handlers.ProxyConfig{
		Live:                   liveURL,
		Shadow:                 shadowURL,
		LiveTimeout:            cfg.App.LiveTimeout,
		ShadowTimeout:          cfg.App.ShadowTimeout,
		MaxBodySize:            cfg.App.MaxBodySize,
		SamplingRate:           samplingRate,
		ShadowRules:            shadowRules,
		MaxConcurrentCallbacks: cfg.App.MaxConcurrentCallbacks,
		HTTPClient: proxy.HTTPClientConfig{
			MaxIdleConns:        cfg.App.HTTPClient.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.App.HTTPClient.MaxIdleConnsPerHost,
			MaxConnsPerHost:     cfg.App.HTTPClient.MaxConnsPerHost,
			IdleConnTimeout:     cfg.App.HTTPClient.IdleConnTimeout,
		},
		Logger:              log,
		APIClient:           apiClient,          // nil if standalone mode
		APITimeout:          cfg.App.APITimeout, // overall deadline for API calls
		DiffOptions:         diffOpts,           // Only used in standalone mode
		Redactor:            redactor,           // Only used in standalone mode
		Recorder:            recorder,           // shared domain comparison metrics; nil if disabled
		InstrumentTransport: instrumentUpstream, // nil if metrics disabled
	}

	// Inbound server metrics: wrap the transparent proxy handler with server-side
	// otelhttp so every request lands on the semconv
	// http_server_request_duration_seconds histogram. The proxy is a catch-all
	// mirror, so it is served directly without a ServeMux — that leaves r.Pattern
	// empty, so otelhttp records no http_route and the unbounded request paths
	// never become a metric label. No-op when metrics are disabled.
	proxyHandler := metricsPlatform.InstrumentHandler("proxy", handlers.Proxy(proxyConfig))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      proxyHandler,
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

	// Metrics endpoint lives on the admin port so scrape traffic never collides
	// with proxied traffic on the main listener. Only mounted when enabled.
	if h := metricsPlatform.MetricsHandler(); h != nil {
		adminMux.Handle("/metrics", h)
	}

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

	// Flush and release the metrics MeterProvider (no-op when disabled).
	if err := metricsPlatform.Shutdown(shutdownCtx); err != nil {
		log.Error("Error shutting down metrics", "error", err)
	}
}

// newProxyMetrics builds the proxy metrics platform (isolated registry with the
// runtime/process collectors plus an OTel MeterProvider bridged onto it) and the
// shared domain comparison recorder. It returns the platform and the recorder.
// When enabled is false it returns nil, nil so every instrumentation seam is a
// no-op and no endpoint is mounted.
func newProxyMetrics(enabled bool) (*metrics.Platform, *diffmetrics.Recorder, error) {
	if !enabled {
		return nil, nil, nil
	}
	platform, err := metrics.NewPlatform()
	if err != nil {
		return nil, nil, err
	}
	recorder, err := diffmetrics.New(platform.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("create comparison recorder: %w", err)
	}
	return platform, recorder, nil
}

func getModeString(apiClient *client.MrokiClient) string {
	if apiClient != nil {
		return "api"
	}
	return "standalone"
}
