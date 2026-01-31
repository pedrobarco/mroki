package caddymodule

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/pedrobarco/mroki/pkg/proxy"
	"go.uber.org/zap"
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
	RawLive          string  `json:"live,omitempty"`
	RawShadow        string  `json:"shadow,omitempty"`
	SamplingRate     *string `json:"sampling_rate,omitempty"`
	RawLiveTimeout   *string `json:"live_timeout,omitempty"`
	RawShadowTimeout *string `json:"shadow_timeout,omitempty"`

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

	if m.SamplingRate != nil {
		rate, err := strconv.ParseFloat(*m.SamplingRate, 64)
		if err != nil {
			return fmt.Errorf("invalid sampling rate: %w", err)
		}

		sr, err := proxy.NewSamplingRate(rate)
		if err != nil {
			return fmt.Errorf("failed to create sampling rate: %w", err)
		}

		opts = append(opts, proxy.WithSamplingRate(sr))
	}

	// Live timeout
	if m.RawLiveTimeout != nil {
		timeout, err := time.ParseDuration(*m.RawLiveTimeout)
		if err != nil {
			return fmt.Errorf("invalid live timeout: %w", err)
		}
		opts = append(opts, proxy.WithLiveTimeout(timeout))
	}

	// Shadow timeout
	if m.RawShadowTimeout != nil {
		timeout, err := time.ParseDuration(*m.RawShadowTimeout)
		if err != nil {
			return fmt.Errorf("invalid shadow timeout: %w", err)
		}
		opts = append(opts, proxy.WithShadowTimeout(timeout))
	}

	m.proxy = proxy.NewProxy(live, shadow, opts...)
	return nil
}

func (m MrokiGate) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	m.proxy.ServeHTTP(w, r)
	// return next.ServeHTTP(w, r)
	return nil
}

// UnmarshalCaddyfile parses the Caddyfile tokens into the MrokiGate struct.
// It expects the following format:
//
//	mroki_gate {
//	    live <live_url>
//	    shadow <shadow_url>
//	    [sampling_rate <rate>]
//	    [live_timeout <duration>]
//	    [shadow_timeout <duration>]
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
