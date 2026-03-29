package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/pedrobarco/mroki/pkg/client"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/proxy"
)

type ProxyConfig struct {
	Live          *url.URL
	Shadow        *url.URL
	LiveTimeout   time.Duration
	ShadowTimeout time.Duration
	MaxBodySize   int64
	SamplingRate  *proxy.SamplingRate // always set, default 1.0

	// API integration (optional)
	APIClient *client.MrokiClient
	AgentID   string
	Logger    *slog.Logger

	// Diff options for standalone mode
	DiffOptions []diff.Option
}

func Proxy(cfg ProxyConfig) http.HandlerFunc {
	opts := []proxy.Option{
		proxy.WithLiveTimeout(cfg.LiveTimeout),
		proxy.WithShadowTimeout(cfg.ShadowTimeout),
	}

	// Add shadow proxy checks
	var checks []proxy.CheckFunc

	if cfg.MaxBodySize > 0 {
		checks = append(checks, proxy.MaxBodySizeCheck(cfg.MaxBodySize))
	}

	checks = append(checks, proxy.SamplingRateCheck(cfg.SamplingRate))

	if len(checks) > 0 {
		opts = append(opts, proxy.WithShouldProxyToShadow(checks...))
	}

	if cfg.APIClient != nil {
		// API mode: send raw responses to API (diff computed server-side)
		opts = append(opts, proxy.WithCallbackFn(createAPICallback(cfg)))
	} else {
		// Standalone mode: compute and print diff locally
		opts = append(opts, proxy.WithCallbackFn(createStandaloneCallback(cfg)))
	}

	p := proxy.NewProxy(cfg.Live, cfg.Shadow, opts...)

	return func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
}

// createAPICallback creates a callback that sends raw captured requests to the API.
// Diff computation is handled server-side by mroki-api.
func createAPICallback(cfg ProxyConfig) proxy.CallbackFunc {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
		captured := client.ConvertProxyToCapture(req, live, shadow, cfg.AgentID)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := cfg.APIClient.SendRequest(ctx, captured); err != nil {
			logger.Warn("failed to send request to API",
				"error", err,
				"method", req.Method,
				"path", req.Path,
			)
			return nil
		}

		logger.Debug("successfully sent request to API",
			"method", req.Method,
			"path", req.Path,
			"live_status", live.StatusCode,
			"shadow_status", shadow.StatusCode,
		)

		return nil
	}
}

// createStandaloneCallback creates a callback that computes and prints diffs locally.
// Used when no API is configured (standalone agent mode).
func createStandaloneCallback(cfg ProxyConfig) proxy.CallbackFunc {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	differ := proxy.NewProxyResponseDiffer(cfg.DiffOptions...)

	return func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
		ops, err := differ.Diff(live, shadow)
		if err != nil {
			logger.Warn("failed to diff responses",
				slog.String("error", err.Error()),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
			return nil
		}

		if len(ops) > 0 {
			logger.Info("response diff detected",
				slog.String("method", req.Method),
				slog.String("path", req.Path),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
				slog.Int("changes", len(ops)),
			)
			fmt.Println("Diff:")
			fmt.Print(diff.FormatOps(ops))
		} else {
			logger.Debug("responses match",
				slog.String("method", req.Method),
				slog.String("path", req.Path),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
		}

		return nil
	}
}
