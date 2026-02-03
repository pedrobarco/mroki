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
		proxy.WithMaxBodySize(cfg.MaxBodySize),
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

	// Create differ with configured diff options
	differ := proxy.NewProxyResponseDiffer(cfg.DiffOptions...)

	return func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
		// Skip non-JSON responses
		if !isJSONContent(live.Response) || !isJSONContent(shadow.Response) {
			logger.Debug("skipping non-JSON response",
				"method", req.Method,
				"path", req.Path,
				"live_content_type", live.Response.Header.Get("Content-Type"),
				"shadow_content_type", shadow.Response.Header.Get("Content-Type"),
			)
			return nil
		}

		// Compute diff using differ with pre-configured options
		diffContent, err := differ.Diff(live, shadow)
		if err != nil {
			logger.Error("failed to compute diff",
				"error", err,
				"method", req.Method,
				"path", req.Path,
			)
			return nil // Don't fail the callback
		}

		// Log if diff found
		if len(diffContent) > 0 {
			logger.Info("diff detected",
				"method", req.Method,
				"path", req.Path,
				"live_status", live.StatusCode,
				"shadow_status", shadow.StatusCode,
			)
		} else {
			logger.Debug("responses match",
				"method", req.Method,
				"path", req.Path,
			)
		}

		// If API client configured, send to API
		if cfg.APIClient != nil {
			// Convert to API format
			captured := client.ConvertProxyToCapture(req, live, shadow, diffContent, cfg.AgentID)

			// Send to API with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := cfg.APIClient.SendRequest(ctx, captured); err != nil {
				logger.Warn("failed to send request to API",
					"error", err,
					"method", req.Method,
					"path", req.Path,
				)
				return nil // Log but don't fail - best effort delivery
			}

			logger.Debug("successfully sent request to API",
				"method", req.Method,
				"path", req.Path,
				"has_diff", len(diffContent) > 0,
			)
		}

		return nil
	}
}

// isJSONContent checks if the response has JSON content type
func isJSONContent(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	contentType := resp.Header.Get("Content-Type")
	return contentType == "application/json" ||
		(len(contentType) > 16 && contentType[:16] == "application/json")
}
