package config_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/pedrobarco/mroki/cmd/mroki-proxy/config"
	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func validStandaloneConfig() config.Config {
	var cfg config.Config
	cfg.App.LiveURL = mustURL("http://live:8080")
	cfg.App.ShadowURL = mustURL("http://shadow:8080")
	cfg.App.Port = 8080
	cfg.App.AdminPort = 8081
	cfg.App.SamplingRate = 1.0
	cfg.App.LiveTimeout = 5 * time.Second
	cfg.App.ShadowTimeout = 10 * time.Second
	cfg.App.MaxBodySize = 10485760
	cfg.App.ReadTimeout = 30 * time.Second
	cfg.App.WriteTimeout = 60 * time.Second
	cfg.App.IdleTimeout = 120 * time.Second
	cfg.App.HTTPClient.MaxIdleConns = 100
	cfg.App.HTTPClient.MaxIdleConnsPerHost = 10
	cfg.App.HTTPClient.MaxConnsPerHost = 100
	cfg.App.HTTPClient.IdleConnTimeout = 90 * time.Second
	return cfg
}

func validAPIConfig() config.Config {
	var cfg config.Config
	cfg.App.APIURL = mustURL("http://api:8081")
	cfg.App.GateID = "550e8400-e29b-41d4-a716-446655440000"
	cfg.App.APIKey = "test-api-key-min-16-chars"
	cfg.App.Port = 8080
	cfg.App.AdminPort = 8081
	cfg.App.SamplingRate = 1.0
	cfg.App.LiveTimeout = 5 * time.Second
	cfg.App.ShadowTimeout = 10 * time.Second
	cfg.App.MaxRetries = 3
	cfg.App.RetryDelay = 1 * time.Second
	cfg.App.APITimeout = 30 * time.Second
	cfg.App.CBFailureThreshold = 5
	cfg.App.CBDelay = 1 * time.Minute
	cfg.App.CBSuccessThreshold = 2
	cfg.App.MaxBodySize = 10485760
	cfg.App.ReadTimeout = 30 * time.Second
	cfg.App.WriteTimeout = 60 * time.Second
	cfg.App.IdleTimeout = 120 * time.Second
	cfg.App.HTTPClient.MaxIdleConns = 100
	cfg.App.HTTPClient.MaxIdleConnsPerHost = 10
	cfg.App.HTTPClient.MaxConnsPerHost = 100
	cfg.App.HTTPClient.IdleConnTimeout = 90 * time.Second
	return cfg
}

func TestValidate_valid_standalone(t *testing.T) {
	cfg := validStandaloneConfig()
	require.NoError(t, cfg.Validate())
}

func TestValidate_valid_api_mode(t *testing.T) {
	cfg := validAPIConfig()
	require.NoError(t, cfg.Validate())
}

func TestValidate_no_mode_configured(t *testing.T) {
	var cfg config.Config
	cfg.App.Port = 8080
	cfg.App.LiveTimeout = 5 * time.Second
	cfg.App.ShadowTimeout = 10 * time.Second

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must configure either API mode")
}

func TestValidate_invalid_port(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.Port = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")

	cfg.App.Port = 70000
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
}

func TestValidate_invalid_admin_port(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.AdminPort = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin_port must be between 1 and 65535")

	cfg.App.AdminPort = 70000
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin_port must be between 1 and 65535")
}

func TestValidate_admin_port_equals_port(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.AdminPort = cfg.App.Port
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must differ from port")
}

func TestValidate_invalid_timeouts(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.LiveTimeout = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "live_timeout must be positive")

	cfg = validStandaloneConfig()
	cfg.App.ShadowTimeout = -1 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shadow_timeout must be positive")
}

func TestValidate_invalid_max_body_size(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.MaxBodySize = -1
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_body_size must be non-negative")
}

func TestValidate_zero_max_body_size_is_valid(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.MaxBodySize = 0
	require.NoError(t, cfg.Validate())
}

func TestValidate_sampling_rate(t *testing.T) {
	t.Run("valid rates", func(t *testing.T) {
		for _, rate := range []float64{0.0, 0.5, 1.0} {
			cfg := validStandaloneConfig()
			cfg.App.SamplingRate = rate
			require.NoError(t, cfg.Validate(), "rate %f should be valid", rate)
		}
	})

	t.Run("default is 1.0 (100%)", func(t *testing.T) {
		cfg := validStandaloneConfig()
		// SamplingRate defaults to 1.0 via env tag
		require.NoError(t, cfg.Validate())
	})

	t.Run("negative rate", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.SamplingRate = -0.1
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sampling_rate must be between 0.0 and 1.0")
	})

	t.Run("rate above 1.0", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.SamplingRate = 1.5
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sampling_rate must be between 0.0 and 1.0")
	})
}

func TestValidate_api_mode_invalid_gate_id(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.GateID = "not-a-uuid"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gate_id must be a valid UUID")
}

func TestValidate_api_mode_invalid_url_scheme(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.APIURL = mustURL("ftp://api:8081")
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_url must use http or https scheme")
}

func TestValidate_api_mode_negative_max_retries(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.MaxRetries = -1
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_retries must be non-negative")
}

func TestValidate_api_mode_invalid_retry_delay(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.RetryDelay = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry_delay must be positive")
}

func TestValidate_api_mode_invalid_api_timeout(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.APITimeout = -1 * time.Second
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_timeout must be positive")
}

func TestValidate_standalone_invalid_url_scheme(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.LiveURL = mustURL("ftp://live:8080")
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "live_url must use http or https scheme")

	cfg = validStandaloneConfig()
	cfg.App.ShadowURL = mustURL("ftp://shadow:8080")
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shadow_url must use http or https scheme")
}

func TestValidate_diff_float_tolerance(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.DiffFloatTolerance = -0.001
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "diff_float_tolerance must be non-negative")

	cfg = validStandaloneConfig()
	cfg.App.DiffFloatTolerance = 0.001
	require.NoError(t, cfg.Validate())
}

func TestValidate_diff_sort_arrays(t *testing.T) {
	// Default is false
	cfg := validStandaloneConfig()
	assert.False(t, cfg.App.DiffSortArrays)
	require.NoError(t, cfg.Validate())

	// Explicitly set to true is valid
	cfg = validStandaloneConfig()
	cfg.App.DiffSortArrays = true
	require.NoError(t, cfg.Validate())
}

func TestValidate_shadow_rules(t *testing.T) {
	t.Run("empty is valid (uses defaults)", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.ShadowRules = ""
		require.NoError(t, cfg.Validate())
	})

	t.Run("valid rules", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.ShadowRules = "deny *:/health/*,allow POST:/api/v1/search,deny POST:*"
		require.NoError(t, cfg.Validate())
	})

	t.Run("invalid rule format", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.ShadowRules = "deny POST"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid shadow_rules")
	})

	t.Run("invalid action", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.ShadowRules = "block POST:*"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid shadow_rules")
	})

	t.Run("invalid path pattern", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.ShadowRules = "deny GET:/api/["
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid shadow_rules")
	})
}

func TestValidate_invalid_server_timeouts(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.ReadTimeout = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout must be positive")

	cfg = validStandaloneConfig()
	cfg.App.WriteTimeout = -1 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write_timeout must be positive")

	cfg = validStandaloneConfig()
	cfg.App.IdleTimeout = 0
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "idle_timeout must be positive")
}

func TestValidate_write_timeout_less_than_live_timeout(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.LiveTimeout = 10 * time.Second
	cfg.App.WriteTimeout = 5 * time.Second
	cfg.App.IdleTimeout = 120 * time.Second
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write_timeout (5s) must be >= live_timeout (10s)")
}

func TestValidate_server_timeout_ordering(t *testing.T) {
	// ReadTimeout >= WriteTimeout
	cfg := validStandaloneConfig()
	cfg.App.ReadTimeout = 60 * time.Second
	cfg.App.WriteTimeout = 30 * time.Second
	cfg.App.IdleTimeout = 120 * time.Second
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout (1m0s) must be less than write_timeout (30s)")

	// WriteTimeout >= IdleTimeout
	cfg = validStandaloneConfig()
	cfg.App.ReadTimeout = 30 * time.Second
	cfg.App.WriteTimeout = 120 * time.Second
	cfg.App.IdleTimeout = 60 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write_timeout (2m0s) must be less than idle_timeout (1m0s)")

	// Equal values are also invalid
	cfg = validStandaloneConfig()
	cfg.App.ReadTimeout = 30 * time.Second
	cfg.App.WriteTimeout = 30 * time.Second
	cfg.App.IdleTimeout = 120 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout (30s) must be less than write_timeout (30s)")
}

func TestValidate_http_client_pool(t *testing.T) {
	t.Run("zero values are valid (net/http semantics)", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.HTTPClient.MaxIdleConns = 0
		cfg.App.HTTPClient.MaxIdleConnsPerHost = 0
		cfg.App.HTTPClient.MaxConnsPerHost = 0
		cfg.App.HTTPClient.IdleConnTimeout = 0
		require.NoError(t, cfg.Validate())
	})

	t.Run("negative max_idle_conns rejected", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.HTTPClient.MaxIdleConns = -1
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_idle_conns must be non-negative")
	})

	t.Run("negative max_idle_conns_per_host rejected", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.HTTPClient.MaxIdleConnsPerHost = -1
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_idle_conns_per_host must be non-negative")
	})

	t.Run("negative max_conns_per_host rejected", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.HTTPClient.MaxConnsPerHost = -1
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_conns_per_host must be non-negative")
	})

	t.Run("negative idle_conn_timeout rejected", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.HTTPClient.IdleConnTimeout = -1 * time.Second
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "idle_conn_timeout must be non-negative")
	})
}

// TestDefaults_http_client_pool guards the config layer's connection-pool
// `default=` env tags. The config layer is the sole owner of these operational
// defaults (pkg/proxy holds none), so these literals are the source of truth.
// It loads the config with an empty environment so only the struct-tag defaults
// apply.
func TestDefaults_http_client_pool(t *testing.T) {
	var cfg config.Config
	err := envconfig.ProcessWith(context.Background(), &envconfig.Config{
		Lookuper: envconfig.MapLookuper(map[string]string{}),
		Target:   &cfg,
	})
	require.NoError(t, err)

	assert.Equal(t, 100, cfg.App.HTTPClient.MaxIdleConns)
	assert.Equal(t, 10, cfg.App.HTTPClient.MaxIdleConnsPerHost)
	assert.Equal(t, 100, cfg.App.HTTPClient.MaxConnsPerHost)
	assert.Equal(t, 90*time.Second, cfg.App.HTTPClient.IdleConnTimeout)
}

func asValidationError(t *testing.T, err error) *config.ValidationError {
	t.Helper()
	verr, ok := err.(*config.ValidationError)
	require.True(t, ok, "expected *config.ValidationError, got %T", err)
	return verr
}

func TestValidate_warning_live_timeout_below_tls(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.LiveTimeout = 3 * time.Second
	err := cfg.Validate()
	require.Error(t, err)
	verr := asValidationError(t, err)
	require.False(t, verr.HasErrors())
	require.True(t, verr.HasWarnings())
	warnings := verr.Warnings()
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Message, "live_timeout (3s) is less than the TLS handshake safety net (5s)")
}

func TestValidate_no_warning_live_timeout_at_or_above_tls(t *testing.T) {
	cfg := validStandaloneConfig()
	cfg.App.LiveTimeout = 5 * time.Second
	require.NoError(t, cfg.Validate())
}

func TestValidate_warning_retry_budget_exceeds_api_timeout(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.MaxRetries = 5
	cfg.App.RetryDelay = 2 * time.Second
	cfg.App.APITimeout = 10 * time.Second
	err := cfg.Validate()
	require.Error(t, err)
	verr := asValidationError(t, err)
	require.False(t, verr.HasErrors())
	require.True(t, verr.HasWarnings())
	warnings := verr.Warnings()
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Message, "retry budget")
	assert.Contains(t, warnings[0].Message, "may exceed api_timeout")
}

func TestValidate_no_warning_retry_budget_within_api_timeout(t *testing.T) {
	cfg := validAPIConfig()
	cfg.App.MaxRetries = 3
	cfg.App.RetryDelay = 1 * time.Second
	cfg.App.APITimeout = 30 * time.Second
	require.NoError(t, cfg.Validate())
}

func TestValidate_no_warnings_standalone(t *testing.T) {
	cfg := validStandaloneConfig()
	require.NoError(t, cfg.Validate())
}

func TestValidate_multiple_errors(t *testing.T) {
	var cfg config.Config
	cfg.App.Port = 0
	cfg.App.LiveTimeout = 0
	cfg.App.ShadowTimeout = 0
	cfg.App.MaxBodySize = -1

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
	assert.Contains(t, err.Error(), "live_timeout must be positive")
	assert.Contains(t, err.Error(), "shadow_timeout must be positive")
	assert.Contains(t, err.Error(), "max_body_size must be non-negative")
	assert.Contains(t, err.Error(), "read_timeout must be positive")
	assert.Contains(t, err.Error(), "write_timeout must be positive")
	assert.Contains(t, err.Error(), "idle_timeout must be positive")
}
