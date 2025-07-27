package caddymodule

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
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
	RawLive   string `json:"live,omitempty"`
	RawShadow string `json:"shadow,omitempty"`

	live   *url.URL
	shadow *url.URL
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

	m.live = live
	m.shadow = shadow
	return nil
}

func (m MrokiGate) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	req := r.Clone(r.Context())
	b, err := httputil.DumpRequest(req, true)
	if err != nil {
		return fmt.Errorf("failed to dump request: %w", err)
	}

	m.logger.Info("mroki request",
		zap.String("live_url", m.live.String()),
		zap.String("shadow_url", m.shadow.String()),
	)

	fmt.Printf("Request: %s\n", string(b))

	live := m.live.String()
	http.Redirect(w, r, live, http.StatusTemporaryRedirect)
	return nil
	// return next.ServeHTTP(w, req)
}

// UnmarshalCaddyfile parses the Caddyfile tokens into the MrokiGate struct.
// It expects the following format:
//
// ```
//
//	mroki_gate {
//	    live <live_url>
//	    shadow <shadow_url>
//	}
//
// ```
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
