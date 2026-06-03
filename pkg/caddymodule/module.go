package caddymodule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/pedrobarco/mroki/internal/application/services"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/diff"
	diffmetrics "github.com/pedrobarco/mroki/pkg/diff/metrics"
	"github.com/pedrobarco/mroki/pkg/metrics"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

var (
	ErrRequiredLiveURL   = errors.New("live URL is required")
	ErrRequiredShadowURL = errors.New("shadow URL is required")
)

// Default outbound HTTP client connection-pool settings. The Caddy module owns
// these operational defaults (pkg/proxy is unopinionated); they match the
// proxy binary's config defaults and apply unless overridden in the Caddyfile.
const (
	defaultMaxIdleConns        = 100
	defaultMaxIdleConnsPerHost = 10
	defaultMaxConnsPerHost     = 100
	defaultIdleConnTimeout     = 90 * time.Second
)

// defaultMaxConcurrentCallbacks bounds concurrent background callback
// goroutines unless overridden in the Caddyfile. The module owns this
// operational default (pkg/proxy is unopinionated); it matches the proxy
// binary's MROKI_APP_MAX_CONCURRENT_CALLBACKS default.
const defaultMaxConcurrentCallbacks = 200

// Comparison metrics are process-global so multiple mroki_gate instances share
// one recorder and one bridge onto Caddy's metrics registry, avoiding duplicate
// collector registration. The gate label is empty in standalone mode, so a
// shared recorder is semantically correct.
var (
	recorderOnce   sync.Once
	sharedRecorder *diffmetrics.Recorder
	recorderErr    error
)

// provisionRecorder builds, once per process, the shared domain comparison
// recorder, bridging an OTel MeterProvider onto reg so the mroki_* comparison
// metrics surface on Caddy's /metrics endpoint.
func provisionRecorder(reg prometheus.Registerer) (*diffmetrics.Recorder, error) {
	recorderOnce.Do(func() {
		mp, err := metrics.NewMeterProvider(reg)
		if err != nil {
			recorderErr = err
			return
		}
		sharedRecorder, recorderErr = diffmetrics.New(mp)
	})
	return sharedRecorder, recorderErr
}

func init() {
	caddy.RegisterModule(MrokiGate{})
	httpcaddyfile.RegisterHandlerDirective("mroki_gate", parseCaddyfile)
}

type MrokiGate struct {
	// Required
	RawLive   string `json:"live,omitempty"`
	RawShadow string `json:"shadow,omitempty"`

	// Proxy options
	SamplingRate     *string `json:"sampling_rate,omitempty"`
	RawLiveTimeout   *string `json:"live_timeout,omitempty"`
	RawShadowTimeout *string `json:"shadow_timeout,omitempty"`
	RawMaxBodySize   *string `json:"max_body_size,omitempty"`
	RawShadowRules   *string `json:"shadow_rules,omitempty"`

	// RawMaxConcurrentCallbacks bounds concurrent background callback
	// goroutines (0 = unbounded). A nil pointer means the module default
	// (defaultMaxConcurrentCallbacks) applies.
	RawMaxConcurrentCallbacks *string `json:"max_concurrent_callbacks,omitempty"`

	// Outbound HTTP client connection-pool tuning (optional). Grouped under the
	// http_client block to mirror the proxy binary's MROKI_APP_HTTP_CLIENT_*
	// env namespace. A nil pointer means the module defaults apply.
	HTTPClient *HTTPClientOptions `json:"http_client,omitempty"`

	// Diff options
	DiffIgnoredFields  *string `json:"diff_ignored_fields,omitempty"`
	DiffIncludedFields *string `json:"diff_included_fields,omitempty"`
	DiffFloatTolerance *string `json:"diff_float_tolerance,omitempty"`
	DiffSortArrays     *string `json:"diff_sort_arrays,omitempty"`

	// Redaction options
	RedactedFields *string `json:"redacted_fields,omitempty"`

	proxy    *proxy.Proxy
	redactor *traffictesting.Redactor
	logger   *zap.Logger
	recorder *diffmetrics.Recorder
}

// HTTPClientOptions holds the raw, unparsed outbound connection-pool directives
// from the http_client block. Values are parsed and validated in
// buildHTTPClientConfig; a value of 0 follows net/http semantics (no limit, or
// no timeout for idle_conn_timeout).
type HTTPClientOptions struct {
	RawMaxIdleConns        *string `json:"max_idle_conns,omitempty"`
	RawMaxIdleConnsPerHost *string `json:"max_idle_conns_per_host,omitempty"`
	RawMaxConnsPerHost     *string `json:"max_conns_per_host,omitempty"`
	RawIdleConnTimeout     *string `json:"idle_conn_timeout,omitempty"`
}

var (
	_ caddy.Provisioner           = (*MrokiGate)(nil)
	_ caddy.Validator             = (*MrokiGate)(nil)
	_ caddyhttp.MiddlewareHandler = (*MrokiGate)(nil)
	_ caddyfile.Unmarshaler       = (*MrokiGate)(nil)
)

func (MrokiGate) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.mroki_gate",
		New: func() caddy.Module { return new(MrokiGate) },
	}
}

func (m *MrokiGate) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger()

	// Domain comparison metrics. A failure here must not break Caddy startup, so
	// it is logged and the gate continues without recording (a nil recorder is a
	// no-op), preserving the best-effort, never-fail-live-traffic contract.
	recorder, err := provisionRecorder(ctx.GetMetricsRegistry())
	if err != nil {
		m.logger.Warn("failed to initialise comparison metrics; continuing without them", zap.Error(err))
	} else {
		m.recorder = recorder
	}
	return nil
}

func (m *MrokiGate) Validate() error {
	if m.RawLive == "" {
		return ErrRequiredLiveURL
	}

	if m.RawShadow == "" {
		return ErrRequiredShadowURL
	}

	live, err := url.Parse(m.RawLive)
	if err != nil {
		return fmt.Errorf("invalid live URL: %w", err)
	}

	shadow, err := url.Parse(m.RawShadow)
	if err != nil {
		return fmt.Errorf("invalid shadow URL: %w", err)
	}

	var opts []proxy.Option

	// Shadow proxy checks
	var checks []proxy.CheckFunc

	// Default sampling rate to 1.0 (100%) when not specified
	samplingRateValue := 1.0
	if m.SamplingRate != nil {
		rate, err := strconv.ParseFloat(*m.SamplingRate, 64)
		if err != nil {
			return fmt.Errorf("invalid sampling rate: %w", err)
		}
		samplingRateValue = rate
	}

	sr, err := proxy.NewSamplingRate(samplingRateValue)
	if err != nil {
		return fmt.Errorf("failed to create sampling rate: %w", err)
	}
	checks = append(checks, proxy.SamplingRateCheck(sr))

	if m.RawMaxBodySize != nil {
		maxBytes, err := strconv.ParseInt(*m.RawMaxBodySize, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max_body_size: %w", err)
		}
		checks = append(checks, proxy.MaxBodySizeCheck(maxBytes))
	}

	// Shadow matching rules. User-supplied rules (if any) are evaluated first;
	// BaseShadowRules (deny non-idempotent methods) are always appended as the
	// final catch-all so the write-protection cannot be accidentally dropped —
	// matching the proxy binary's behavior.
	var userShadowRules []proxy.ShadowRule
	if m.RawShadowRules != nil {
		parsed, err := proxy.ParseShadowRules(*m.RawShadowRules)
		if err != nil {
			return fmt.Errorf("invalid shadow_rules: %w", err)
		}
		userShadowRules = parsed
	}
	shadowRules := append(userShadowRules, proxy.BaseShadowRules()...)
	checks = append(checks, proxy.ShadowRulesCheck(shadowRules))

	if len(checks) > 0 {
		opts = append(opts, proxy.WithShouldProxyToShadow(checks...))
	}

	// Timeouts
	if m.RawLiveTimeout != nil {
		timeout, err := time.ParseDuration(*m.RawLiveTimeout)
		if err != nil {
			return fmt.Errorf("invalid live timeout: %w", err)
		}
		opts = append(opts, proxy.WithLiveTimeout(timeout))
	}

	if m.RawShadowTimeout != nil {
		timeout, err := time.ParseDuration(*m.RawShadowTimeout)
		if err != nil {
			return fmt.Errorf("invalid shadow timeout: %w", err)
		}
		opts = append(opts, proxy.WithShadowTimeout(timeout))
	}

	// Outbound HTTP client connection-pool tuning (module defaults applied when
	// directives are unset).
	clientCfg, err := m.buildHTTPClientConfig()
	if err != nil {
		return err
	}
	opts = append(opts, proxy.WithHTTPClient(proxy.NewHTTPClient(clientCfg)))

	// Bounded callback concurrency. The module default applies unless overridden;
	// 0 disables the limit (unbounded).
	maxCallbacks := defaultMaxConcurrentCallbacks
	if m.RawMaxConcurrentCallbacks != nil {
		v, err := strconv.Atoi(*m.RawMaxConcurrentCallbacks)
		if err != nil {
			return fmt.Errorf("invalid max_concurrent_callbacks: %w", err)
		}
		if v < 0 {
			return fmt.Errorf("max_concurrent_callbacks must be non-negative (0=unbounded), got %d", v)
		}
		maxCallbacks = v
	}
	opts = append(opts, proxy.WithMaxConcurrentCallbacks(maxCallbacks))

	// Bridge Caddy's zap logger to slog for the proxy
	var logger *slog.Logger
	if m.logger != nil {
		logger = slog.New(zapslog.NewHandler(m.logger.Core(), nil))
	} else {
		logger = slog.Default()
	}
	opts = append(opts, proxy.WithLogger(logger))

	// Build diff options
	var diffOpts []diff.Option

	// Exclude the shadow identification header from diff comparison so it never
	// shows up as a difference. It is not redacted — its value stays visible for
	// reference in stored request data.
	diffOpts = append(diffOpts, diff.WithIgnoredFields("headers."+proxy.ShadowHeader))

	if m.DiffIgnoredFields != nil {
		fields := strings.Split(*m.DiffIgnoredFields, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		diffOpts = append(diffOpts, diff.WithIgnoredFields(fields...))
	}

	if m.DiffIncludedFields != nil {
		fields := strings.Split(*m.DiffIncludedFields, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		diffOpts = append(diffOpts, diff.WithIncludedFields(fields...))
	}

	if m.DiffFloatTolerance != nil {
		tolerance, err := strconv.ParseFloat(*m.DiffFloatTolerance, 64)
		if err != nil {
			return fmt.Errorf("invalid diff_float_tolerance: %w", err)
		}
		diffOpts = append(diffOpts, diff.WithFloatTolerance(tolerance))
	}

	if m.DiffSortArrays != nil {
		v, err := strconv.ParseBool(*m.DiffSortArrays)
		if err != nil {
			return fmt.Errorf("invalid diff_sort_arrays: %w", err)
		}
		if v {
			diffOpts = append(diffOpts, diff.WithSortArrays(true))
		}
	}

	// Build redactor from config (adds to default redacted list)
	var additionalFields []string
	if m.RedactedFields != nil {
		fields := strings.Split(*m.RedactedFields, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		additionalFields = fields
	}

	redactedFieldsCfg, err := traffictesting.NewRedactedFields(additionalFields)
	if err != nil {
		return fmt.Errorf("invalid redacted_fields: %w", err)
	}

	allFields := redactedFieldsCfg.AllFields()
	m.redactor = traffictesting.NewRedactor(allFields)

	// Auto-add redacted fields as diff ignored fields (prevents diff noise)
	for _, f := range allFields {
		diffOpts = append(diffOpts, diff.WithIgnoredFields(f))
	}

	// Standalone mode: compute and print diffs locally
	callback, err := m.createDiffCallback(logger, diffOpts)
	if err != nil {
		return err
	}
	opts = append(opts, proxy.WithCallbackFn(callback))

	m.proxy = proxy.NewProxy(live, shadow, opts...)
	return nil
}

// buildHTTPClientConfig returns the outbound HTTP client connection-pool config,
// starting from the module's defaults and overriding with any values set in the
// Caddyfile. Negative values are rejected; 0 follows net/http semantics.
func (m *MrokiGate) buildHTTPClientConfig() (proxy.HTTPClientConfig, error) {
	cfg := proxy.HTTPClientConfig{
		MaxIdleConns:        defaultMaxIdleConns,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		MaxConnsPerHost:     defaultMaxConnsPerHost,
		IdleConnTimeout:     defaultIdleConnTimeout,
	}

	// No http_client block — the module defaults apply verbatim.
	if m.HTTPClient == nil {
		return cfg, nil
	}

	parseConn := func(name string, raw *string, dst *int) error {
		if raw == nil {
			return nil
		}
		v, err := strconv.Atoi(*raw)
		if err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
		if v < 0 {
			return fmt.Errorf("%s must be non-negative, got %d", name, v)
		}
		*dst = v
		return nil
	}

	if err := parseConn("max_idle_conns", m.HTTPClient.RawMaxIdleConns, &cfg.MaxIdleConns); err != nil {
		return cfg, err
	}
	if err := parseConn("max_idle_conns_per_host", m.HTTPClient.RawMaxIdleConnsPerHost, &cfg.MaxIdleConnsPerHost); err != nil {
		return cfg, err
	}
	if err := parseConn("max_conns_per_host", m.HTTPClient.RawMaxConnsPerHost, &cfg.MaxConnsPerHost); err != nil {
		return cfg, err
	}
	if m.HTTPClient.RawIdleConnTimeout != nil {
		d, err := time.ParseDuration(*m.HTTPClient.RawIdleConnTimeout)
		if err != nil {
			return cfg, fmt.Errorf("invalid idle_conn_timeout: %w", err)
		}
		if d < 0 {
			return cfg, fmt.Errorf("idle_conn_timeout must be non-negative, got %s", d)
		}
		cfg.IdleConnTimeout = d
	}

	return cfg, nil
}

// createDiffCallback builds a callback that computes and prints diffs locally.
func (m *MrokiGate) createDiffCallback(logger *slog.Logger, diffOpts []diff.Option) (proxy.CallbackFunc, error) {
	differ := proxy.NewProxyResponseDiffer(diffOpts...)
	redactor := m.redactor

	return func(req proxy.ProxyRequest, live, shadow proxy.ProxyResponse) error {
		reqLogger := logger.With(
			slog.String("request.id", req.Headers.Get("X-Request-ID")),
			slog.String("request.method", req.Method),
			slog.String("request.path", req.Path),
		)

		// Optimized path: redact + diff via ResponseComparer
		if redactor != nil {
			comparer := services.NewResponseComparer(redactor, diffOpts)
			result, err := comparer.Compare(
				services.ResponseData{Headers: req.Headers, Body: req.Body},
				services.ResponseData{StatusCode: live.StatusCode, Headers: live.Response.Header, Body: live.Body},
				services.ResponseData{StatusCode: shadow.StatusCode, Headers: shadow.Response.Header, Body: shadow.Body},
			)
			if err != nil {
				m.recorder.Observe(context.Background(), "", nil, err)
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

			m.recorder.Observe(context.Background(), "", result.Ops, nil)
			m.logDiffResult(reqLogger, live, shadow, result.Ops)
			return nil
		}

		// Fallback: byte-level diff (no redactor). Unreachable in practice since
		// Validate() always initialises m.redactor; retained for structural
		// symmetry with createStandaloneCallback in handlers/proxy.go.
		ops, err := differ.Diff(live, shadow)
		if err != nil {
			m.recorder.Observe(context.Background(), "", nil, err)
			reqLogger.Warn("failed to diff responses",
				slog.String("error", err.Error()),
				slog.Int("live_status", live.StatusCode),
				slog.Int("shadow_status", shadow.StatusCode),
			)
			return nil
		}
		m.recorder.Observe(context.Background(), "", ops, nil)
		m.logDiffResult(reqLogger, live, shadow, ops)
		return nil
	}, nil
}

// logDiffResult logs the diff outcome and prints the ops if any.
func (m *MrokiGate) logDiffResult(logger *slog.Logger, live, shadow proxy.ProxyResponse, ops []diff.PatchOp) {
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

// ServeHTTP is a terminating handler — it writes the full live response via
// proxy.ServeHTTP, then fires shadow async. next is intentionally not called
// because the response is already written.
func (m MrokiGate) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	m.proxy.ServeHTTP(w, r)
	return nil
}

// UnmarshalCaddyfile parses the Caddyfile tokens into the MrokiGate struct.
// It expects the following format:
//
//	mroki_gate {
//	    live                 <url>
//	    shadow               <url>
//	    [sampling_rate       <float>]
//	    [live_timeout        <duration>]
//	    [shadow_timeout      <duration>]
//	    [max_body_size       <bytes>]
//	    [shadow_rules        <comma-separated ACTION METHOD:path>]
//	    [max_concurrent_callbacks <int>]
//	    [http_client {
//	        [max_idle_conns          <int>]
//	        [max_idle_conns_per_host <int>]
//	        [max_conns_per_host      <int>]
//	        [idle_conn_timeout       <duration>]
//	    }]
//	    [diff_ignored_fields  <comma-separated>]
//	    [diff_included_fields <comma-separated>]
//	    [diff_float_tolerance <float>]
//	    [diff_sort_arrays     <bool>]
//	    [redacted_fields      <comma-separated>]
//	}
func (m *MrokiGate) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "live":
			if !d.NextArg() {
				return d.ArgErr()
			}
			m.RawLive = d.Val()
		case "shadow":
			if !d.NextArg() {
				return d.ArgErr()
			}
			m.RawShadow = d.Val()
		case "sampling_rate":
			if !d.NextArg() {
				return d.ArgErr()
			}
			rate := d.Val()
			m.SamplingRate = &rate
		case "live_timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			timeout := d.Val()
			m.RawLiveTimeout = &timeout
		case "shadow_timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			timeout := d.Val()
			m.RawShadowTimeout = &timeout
		case "max_body_size":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.RawMaxBodySize = &val
		case "shadow_rules":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.RawShadowRules = &val
		case "max_concurrent_callbacks":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.RawMaxConcurrentCallbacks = &val
		case "http_client":
			if m.HTTPClient == nil {
				m.HTTPClient = &HTTPClientOptions{}
			}
			for nesting := d.Nesting(); d.NextBlock(nesting); {
				switch d.Val() {
				case "max_idle_conns":
					if !d.NextArg() {
						return d.ArgErr()
					}
					val := d.Val()
					m.HTTPClient.RawMaxIdleConns = &val
				case "max_idle_conns_per_host":
					if !d.NextArg() {
						return d.ArgErr()
					}
					val := d.Val()
					m.HTTPClient.RawMaxIdleConnsPerHost = &val
				case "max_conns_per_host":
					if !d.NextArg() {
						return d.ArgErr()
					}
					val := d.Val()
					m.HTTPClient.RawMaxConnsPerHost = &val
				case "idle_conn_timeout":
					if !d.NextArg() {
						return d.ArgErr()
					}
					val := d.Val()
					m.HTTPClient.RawIdleConnTimeout = &val
				default:
					return d.Errf("unknown http_client property '%s'", d.Val())
				}
			}
		case "diff_ignored_fields":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.DiffIgnoredFields = &val
		case "diff_included_fields":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.DiffIncludedFields = &val
		case "diff_float_tolerance":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.DiffFloatTolerance = &val
		case "diff_sort_arrays":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.DiffSortArrays = &val
		case "redacted_fields":
			if !d.NextArg() {
				return d.ArgErr()
			}
			val := d.Val()
			m.RedactedFields = &val
		default:
			return d.Errf("unknown property '%s'", d.Val())
		}
	}

	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m MrokiGate
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}
