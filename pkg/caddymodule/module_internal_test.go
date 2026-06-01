package caddymodule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// strptr is a small helper for building optional *string Caddyfile fields.
func strptr(s string) *string { return &s }

// TestBuildHTTPClientConfig_defaults verifies the module applies its own
// operational defaults when no connection-pool directives are set. The Caddy
// module owns these defaults because pkg/proxy is unopinionated.
func TestBuildHTTPClientConfig_defaults(t *testing.T) {
	m := &MrokiGate{}

	cfg, err := m.buildHTTPClientConfig()
	require.NoError(t, err)

	assert.Equal(t, defaultMaxIdleConns, cfg.MaxIdleConns)
	assert.Equal(t, defaultMaxIdleConnsPerHost, cfg.MaxIdleConnsPerHost)
	assert.Equal(t, defaultMaxConnsPerHost, cfg.MaxConnsPerHost)
	assert.Equal(t, defaultIdleConnTimeout, cfg.IdleConnTimeout)
}

// TestBuildHTTPClientConfig_overrides verifies that set directives override the
// defaults and land in the resulting config verbatim (including a 0 override,
// which is valid and follows net/http "unlimited" semantics).
func TestBuildHTTPClientConfig_overrides(t *testing.T) {
	m := &MrokiGate{
		HTTPClient: &HTTPClientOptions{
			RawMaxIdleConns:        strptr("250"),
			RawMaxIdleConnsPerHost: strptr("25"),
			RawMaxConnsPerHost:     strptr("0"),
			RawIdleConnTimeout:     strptr("45s"),
		},
	}

	cfg, err := m.buildHTTPClientConfig()
	require.NoError(t, err)

	assert.Equal(t, 250, cfg.MaxIdleConns)
	assert.Equal(t, 25, cfg.MaxIdleConnsPerHost)
	assert.Equal(t, 0, cfg.MaxConnsPerHost)
	assert.Equal(t, 45*time.Second, cfg.IdleConnTimeout)
}

// TestBuildHTTPClientConfig_partial_override verifies that unset fields keep the
// module defaults while only the provided field is overridden.
func TestBuildHTTPClientConfig_partial_override(t *testing.T) {
	m := &MrokiGate{HTTPClient: &HTTPClientOptions{RawMaxIdleConnsPerHost: strptr("50")}}

	cfg, err := m.buildHTTPClientConfig()
	require.NoError(t, err)

	assert.Equal(t, defaultMaxIdleConns, cfg.MaxIdleConns)
	assert.Equal(t, 50, cfg.MaxIdleConnsPerHost)
	assert.Equal(t, defaultMaxConnsPerHost, cfg.MaxConnsPerHost)
	assert.Equal(t, defaultIdleConnTimeout, cfg.IdleConnTimeout)
}

// TestBuildHTTPClientConfig_invalid confirms parse/validation errors surface and
// are not silently swallowed.
func TestBuildHTTPClientConfig_invalid(t *testing.T) {
	t.Run("non-numeric int", func(t *testing.T) {
		m := &MrokiGate{HTTPClient: &HTTPClientOptions{RawMaxIdleConns: strptr("abc")}}
		_, err := m.buildHTTPClientConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid max_idle_conns")
	})

	t.Run("negative int", func(t *testing.T) {
		m := &MrokiGate{HTTPClient: &HTTPClientOptions{RawMaxConnsPerHost: strptr("-1")}}
		_, err := m.buildHTTPClientConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_conns_per_host must be non-negative")
	})

	t.Run("invalid duration", func(t *testing.T) {
		m := &MrokiGate{HTTPClient: &HTTPClientOptions{RawIdleConnTimeout: strptr("nope")}}
		_, err := m.buildHTTPClientConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid idle_conn_timeout")
	})

	t.Run("negative duration", func(t *testing.T) {
		m := &MrokiGate{HTTPClient: &HTTPClientOptions{RawIdleConnTimeout: strptr("-1s")}}
		_, err := m.buildHTTPClientConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "idle_conn_timeout must be non-negative")
	})
}

// TestDefaultPoolConsts_match_expected guards the module's default constants
// against accidental drift; they must mirror the proxy binary's config-layer
// defaults (100 / 10 / 100 / 90s).
func TestDefaultPoolConsts_match_expected(t *testing.T) {
	assert.Equal(t, 100, defaultMaxIdleConns)
	assert.Equal(t, 10, defaultMaxIdleConnsPerHost)
	assert.Equal(t, 100, defaultMaxConnsPerHost)
	assert.Equal(t, 90*time.Second, defaultIdleConnTimeout)
}
