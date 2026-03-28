package config

import (
	"net/url"
	"testing"
	"time"

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

func float64Ptr(f float64) *float64 {
	return &f
}

func validStandaloneConfig() Config {
	var cfg Config
	cfg.App.LiveURL = mustURL("http://live:8080")
	cfg.App.ShadowURL = mustURL("http://shadow:8080")
	cfg.App.Port = 8080
	cfg.App.LiveTimeout = 5 * time.Second
	cfg.App.ShadowTimeout = 10 * time.Second
	cfg.App.MaxBodySize = 10485760
	return cfg
}

func validAPIConfig() Config {
	var cfg Config
	cfg.App.APIURL = mustURL("http://api:8081")
	cfg.App.GateID = "550e8400-e29b-41d4-a716-446655440000"
	cfg.App.APIKey = "test-api-key-min-16-chars"
	cfg.App.Port = 8080
	cfg.App.LiveTimeout = 5 * time.Second
	cfg.App.ShadowTimeout = 10 * time.Second
	cfg.App.MaxRetries = 3
	cfg.App.RetryDelay = 1 * time.Second
	cfg.App.APITimeout = 30 * time.Second
	cfg.App.MaxBodySize = 10485760
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
	var cfg Config
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
			cfg.App.SamplingRate = float64Ptr(rate)
			require.NoError(t, cfg.Validate(), "rate %f should be valid", rate)
		}
	})

	t.Run("nil is valid (defaults to 100%)", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.SamplingRate = nil
		require.NoError(t, cfg.Validate())
	})

	t.Run("negative rate", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.SamplingRate = float64Ptr(-0.1)
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sampling_rate must be between 0.0 and 1.0")
	})

	t.Run("rate above 1.0", func(t *testing.T) {
		cfg := validStandaloneConfig()
		cfg.App.SamplingRate = float64Ptr(1.5)
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

func TestValidate_multiple_errors(t *testing.T) {
	var cfg Config
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
}