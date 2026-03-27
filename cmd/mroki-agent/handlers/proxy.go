package handlers

import (
	"context"
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

	// API integration (optional)
	APIClient *client.MrokiClient
	AgentID   string
	Logger    *slog.Logger

	// Diff options from agent config
	DiffOptions []diff.Option
}

func Proxy(cfg ProxyConfig) http.HandlerFunc {
	opts := []proxy.Option{
		proxy.WithLiveTimeout(cfg.LiveTimeout),
		proxy.WithShadowTimeout(cfg.ShadowTimeout),
	}

	// Add max body size check if configured
	if cfg.MaxBodySize > 0 {
		opts = append(opts, proxy.WithShouldProxyToShadow(
			proxy.MaxBodySizeCheck(cfg.MaxBodySize),
		))
	}

	// Configure differ with options from config
	if len(cfg.DiffOptions) > 0 {
		opts = append(opts, proxy.WithDiffOptions(cfg.DiffOptions...))
	}

	// Add API callback if API client is configured
	if cfg.APIClient != nil {
		opts = append(opts, proxy.WithCallbackFn(createAPICallback(cfg)))
	}

	p := proxy.NewProxy(cfg.Live, cfg.Shadow, opts...)

	return func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
}

// createAPICallback creates a callback function that sends captured requests to the API
func createAPICallback(cfg ProxyConfig) proxy.CallbackFunc {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse, diff proxy.DiffResult) error {
		// Handle diff error
		if diff.Error != nil {
			logger.Error("failed to compute diff",
				"error", diff.Error,
				"method", req.Method,
				"path", req.Path,
			)
			return nil
		}

		// Log diff result
		if len(diff.Ops) > 0 {
			logger.Info("diff detected",
				"method", req.Method,
				"path", req.Path,
				"live_status", live.StatusCode,
				"shadow_status", shadow.StatusCode,
				"changes", len(diff.Ops),
			)
		} else {
			logger.Debug("responses match",
				"method", req.Method,
				"path", req.Path,
			)
		}

		// Send to API if configured
		if cfg.APIClient != nil {
			captured := client.ConvertProxyToCapture(req, live, shadow, diff.Ops, cfg.AgentID)

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
				"has_diff", len(diff.Ops) > 0,
			)
		}

		return nil
	}
}
