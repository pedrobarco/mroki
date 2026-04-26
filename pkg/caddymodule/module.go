package caddymodule

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/pedrobarco/mroki/pkg/diff"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

var (
	ErrRequiredLiveURL   = errors.New("live URL is required")
	ErrRequiredShadowURL = errors.New("shadow URL is required")
)

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

	// Diff options
	DiffIgnoredFields  *string `json:"diff_ignored_fields,omitempty"`
	DiffIncludedFields *string `json:"diff_included_fields,omitempty"`
	DiffFloatTolerance *string `json:"diff_float_tolerance,omitempty"`

	proxy  *proxy.Proxy
	logger *zap.Logger
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

	// Bridge Caddy's zap logger to slog for the proxy
	var logger *slog.Logger
	if m.logger != nil {
		logger = slog.New(zapslog.NewHandler(m.logger.Core(), nil))
	} else {
		logger = slog.Default()
	}
	opts = append(opts, proxy.WithLogger(logger))

	// Standalone mode: compute and print diffs locally
	callback, err := m.createDiffCallback(logger)
	if err != nil {
		return err
	}
	opts = append(opts, proxy.WithCallbackFn(callback))

	m.proxy = proxy.NewProxy(live, shadow, opts...)
	return nil
}

// createDiffCallback builds a callback that computes and prints diffs locally.
func (m *MrokiGate) createDiffCallback(logger *slog.Logger) (proxy.CallbackFunc, error) {
	// Build diff options
	var diffOpts []diff.Option

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
			return nil, fmt.Errorf("invalid diff_float_tolerance: %w", err)
		}
		diffOpts = append(diffOpts, diff.WithFloatTolerance(tolerance))
	}

	differ := proxy.NewProxyResponseDiffer(diffOpts...)

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
	}, nil
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
//	    [diff_ignored_fields  <comma-separated>]
//	    [diff_included_fields <comma-separated>]
//	    [diff_float_tolerance <float>]
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
