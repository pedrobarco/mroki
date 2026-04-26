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

func validConfig() Config {
	var cfg Config
	cfg.App.Port = 8090
	cfg.App.MaxBodySize = 10485760
	cfg.App.RateLimit = 1000
	cfg.App.APIKey = "test-api-key-min-16-chars"
	cfg.App.Retention = 0
	cfg.App.CleanupInterval = 1 * time.Hour
	cfg.App.ReadTimeout = 15 * time.Second
	cfg.App.WriteTimeout = 30 * time.Second
	cfg.App.IdleTimeout = 60 * time.Second
	cfg.App.Database.URL = mustURL("postgres://postgres:postgres@localhost:5432/postgres")
	cfg.App.Database.MaxConns = 25
	cfg.App.Database.MinConns = 5
	cfg.App.Database.MaxConnIdle = "5m"
	cfg.App.Database.MaxConnLife = "1h"
	return cfg
}

func TestValidate_valid(t *testing.T) {
	cfg := validConfig()
	require.NoError(t, cfg.Validate())
}

func TestValidate_invalid_port(t *testing.T) {
	cfg := validConfig()
	cfg.App.Port = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")

	cfg = validConfig()
	cfg.App.Port = 70000
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
}

func TestValidate_invalid_api_key(t *testing.T) {
	cfg := validConfig()
	cfg.App.APIKey = "short"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key must be at least 16 characters")
}

func TestValidate_invalid_rate_limit(t *testing.T) {
	cfg := validConfig()
	cfg.App.RateLimit = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limit must be positive")

	cfg = validConfig()
	cfg.App.RateLimit = 200000
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limit too high")
}

func TestValidate_invalid_server_timeouts(t *testing.T) {
	cfg := validConfig()
	cfg.App.ReadTimeout = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout must be positive")

	cfg = validConfig()
	cfg.App.WriteTimeout = -1 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write_timeout must be positive")

	cfg = validConfig()
	cfg.App.IdleTimeout = 0
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "idle_timeout must be positive")
}

func TestValidate_server_timeout_ordering(t *testing.T) {
	// ReadTimeout >= WriteTimeout
	cfg := validConfig()
	cfg.App.ReadTimeout = 30 * time.Second
	cfg.App.WriteTimeout = 15 * time.Second
	cfg.App.IdleTimeout = 60 * time.Second
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout (30s) must be less than write_timeout (15s)")

	// WriteTimeout >= IdleTimeout
	cfg = validConfig()
	cfg.App.ReadTimeout = 15 * time.Second
	cfg.App.WriteTimeout = 60 * time.Second
	cfg.App.IdleTimeout = 30 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write_timeout (1m0s) must be less than idle_timeout (30s)")

	// Equal values are also invalid
	cfg = validConfig()
	cfg.App.ReadTimeout = 30 * time.Second
	cfg.App.WriteTimeout = 30 * time.Second
	cfg.App.IdleTimeout = 60 * time.Second
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout (30s) must be less than write_timeout (30s)")
}

func TestValidate_cors_origins(t *testing.T) {
	t.Run("valid origins", func(t *testing.T) {
		cfg := validConfig()
		cfg.App.CORSOrigins = "http://localhost:5173, https://example.com"
		require.NoError(t, cfg.Validate())
	})

	t.Run("empty is valid", func(t *testing.T) {
		cfg := validConfig()
		cfg.App.CORSOrigins = ""
		require.NoError(t, cfg.Validate())
	})

	t.Run("invalid scheme", func(t *testing.T) {
		cfg := validConfig()
		cfg.App.CORSOrigins = "ftp://example.com"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cors_origins entry \"ftp://example.com\" must use http or https scheme")
	})

	t.Run("mixed valid and invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.App.CORSOrigins = "https://good.com, ftp://bad.com"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cors_origins entry \"ftp://bad.com\" must use http or https scheme")
	})
}

func TestValidate_invalid_retention(t *testing.T) {
	cfg := validConfig()
	cfg.App.Retention = -1 * time.Hour
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retention must be non-negative")
}

func TestValidate_invalid_cleanup_interval(t *testing.T) {
	cfg := validConfig()
	cfg.App.CleanupInterval = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup_interval must be positive")
}

func TestValidate_invalid_database_url(t *testing.T) {
	cfg := validConfig()
	cfg.App.Database.URL = mustURL("ftp://localhost/db")
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database.url must be a valid postgresql URL")
}

func TestValidate_multiple_errors(t *testing.T) {
	var cfg Config
	cfg.App.Port = 0
	cfg.App.APIKey = ""
	cfg.App.Database.MaxConnIdle = "5m"
	cfg.App.Database.MaxConnLife = "1h"

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
	assert.Contains(t, err.Error(), "max_body_size must be positive")
	assert.Contains(t, err.Error(), "rate_limit must be positive")
	assert.Contains(t, err.Error(), "api_key is required")
	assert.Contains(t, err.Error(), "read_timeout must be positive")
	assert.Contains(t, err.Error(), "write_timeout must be positive")
	assert.Contains(t, err.Error(), "idle_timeout must be positive")
	assert.Contains(t, err.Error(), "cleanup_interval must be positive")
}

func TestParseCORSOrigins(t *testing.T) {
	cfg := validConfig()
	cfg.App.CORSOrigins = ""
	assert.Nil(t, cfg.ParseCORSOrigins())

	cfg.App.CORSOrigins = "http://localhost:5173"
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.ParseCORSOrigins())

	cfg.App.CORSOrigins = "http://a.com, http://b.com"
	assert.Equal(t, []string{"http://a.com", "http://b.com"}, cfg.ParseCORSOrigins())
}
