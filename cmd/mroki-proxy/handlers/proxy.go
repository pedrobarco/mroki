package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"

	"github.com/pedrobarco/mroki/internal/application/services"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
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
	ShadowRules   []proxy.ShadowRule  // shadow matching rules

	// MaxConcurrentCallbacks bounds concurrent background callback goroutines
	// (0 = unbounded). Passed through to proxy.WithMaxConcurrentCallbacks.
	MaxConcurrentCallbacks int

	// HTTPClient holds outbound connection-pool tuning. When non-zero, the
	// proxy is built with a client from these values; the zero value falls back
	// to NewProxy's default client (net/http pool semantics, since pkg/proxy
	// holds no operational defaults).
	HTTPClient proxy.HTTPClientConfig

	// API integration (optional)
	APIClient  *client.MrokiClient
	APITimeout time.Duration // overall deadline for API calls (incl. retries)
	Logger     *slog.Logger

	// Diff options for standalone mode
	DiffOptions []diff.Option

	// Redactor for standalone mode (redacts headers + body fields)
	Redactor *traffictesting.Redactor
}

func Proxy(cfg ProxyConfig) http.HandlerFunc {
	opts := []proxy.Option{
		proxy.WithLiveTimeout(cfg.LiveTimeout),
		proxy.WithShadowTimeout(cfg.ShadowTimeout),
		proxy.WithMaxConcurrentCallbacks(cfg.MaxConcurrentCallbacks),
	}

	if cfg.Logger != nil {
		opts = append(opts, proxy.WithLogger(cfg.Logger))
	}

	if cfg.HTTPClient != (proxy.HTTPClientConfig{}) {
		opts = append(opts, proxy.WithHTTPClient(proxy.NewHTTPClient(cfg.HTTPClient)))
	}

	// Add shadow proxy checks
	var checks []proxy.CheckFunc

	if cfg.MaxBodySize > 0 {
		checks = append(checks, proxy.MaxBodySizeCheck(cfg.MaxBodySize))
	}

	checks = append(checks, proxy.SamplingRateCheck(cfg.SamplingRate))

	if len(cfg.ShadowRules) > 0 {
		checks = append(checks, proxy.ShadowRulesCheck(cfg.ShadowRules))
	}

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
		reqLogger := logger.With(
			slog.String("request.id", req.Headers.Get("X-Request-ID")),
			slog.String("request.method", req.Method),
			slog.String("request.path", req.Path),
		)

		captured := client.ConvertProxyToCapture(req, live, shadow)

		timeout := cfg.APITimeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := cfg.APIClient.SendRequest(ctx, captured); err != nil {
			if errors.Is(err, circuitbreaker.ErrOpen) {
				reqLogger.Warn("circuit breaker open, skipping API request")
				return nil
			}
			reqLogger.Warn("failed to send request to API",
				slog.String("error", err.Error()),
			)
			return nil
		}

		reqLogger.Debug("successfully sent request to API",
			slog.Int("live_status", live.StatusCode),
			slog.Int("shadow_status", shadow.StatusCode),
		)

		return nil
	}
}

// createStandaloneCallback creates a callback that computes and prints diffs locally.
// Used when no API is configured (standalone proxy mode).
func createStandaloneCallback(cfg ProxyConfig) proxy.CallbackFunc {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	differ := proxy.NewProxyResponseDiffer(cfg.DiffOptions...)
	redactor := cfg.Redactor

	return func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
		reqLogger := logger.With(
			slog.String("request.id", req.Headers.Get("X-Request-ID")),
			slog.String("request.method", req.Method),
			slog.String("request.path", req.Path),
		)

		// Optimized path: redact + diff via ResponseComparer
		if redactor != nil {
			comparer := services.NewResponseComparer(redactor, cfg.DiffOptions)
			result, err := comparer.Compare(
				services.ResponseData{Headers: req.Headers, Body: req.Body},
				services.ResponseData{StatusCode: live.StatusCode, Headers: live.Response.Header, Body: live.Body},
				services.ResponseData{StatusCode: shadow.StatusCode, Headers: shadow.Response.Header, Body: shadow.Body},
			)
			if err != nil {
				reqLogger.Error("failed to redact, skipping diff", slog.String("error", err.Error()))
				return nil
			}

			// Apply redacted data back for logging
			req.Headers = result.Request.Headers
			req.Body = result.Request.Body
			live.Response.Header = result.Live.Headers
			live.Body = result.Live.Body
			shadow.Response.Header = result.Shadow.Headers
			shadow.Body = result.Shadow.Body

			logDiffResult(reqLogger, live, shadow, result.Ops)
			return nil
		}

		// Fallback: byte-level diff (no redactor)
		ops, err := differ.Diff(live, shadow)
		if err != nil {
			reqLogger.Warn("failed to diff responses",
				slog.String("error", err.Error()),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
			return nil
		}
		logDiffResult(reqLogger, live, shadow, ops)
		return nil
	}
}

// logDiffResult logs the diff outcome and prints the ops if any.
func logDiffResult(logger *slog.Logger, live, shadow proxy.ProxyResponse, ops []diff.PatchOp) {
	if len(ops) > 0 {
		logger.Info("response diff detected",
			slog.Int("live_status", live.StatusCode),
			slog.Int("shadow_status", shadow.StatusCode),
			slog.Int("changes", len(ops)),
		)
		fmt.Println("Diff:")
		fmt.Print(diff.FormatOps(ops))
	} else {
		logger.Debug("responses match",
			slog.Int("live_status", live.StatusCode),
			slog.Int("shadow_status", shadow.StatusCode),
		)
	}
}
