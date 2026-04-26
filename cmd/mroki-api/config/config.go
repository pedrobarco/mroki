package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	Port            int           `env:"PORT, default=8090"`
	MaxBodySize     int64         `env:"MAX_BODY_SIZE, default=10485760"` // 10MB
	RateLimit       int           `env:"RATE_LIMIT, default=1000"`        // requests per minute per IP
	APIKey          string        `env:"API_KEY, required"`
	CORSOrigins     string        `env:"CORS_ORIGINS"`         // comma-separated allowed origins, empty = disabled
	Retention       time.Duration `env:"RETENTION, default=0"` // 0 = keep forever, e.g. 168h = 7 days
	CleanupInterval time.Duration `env:"CLEANUP_INTERVAL, default=1h"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT, default=15s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT, default=30s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT, default=60s"`
	Database        struct {
		URL         *url.URL `env:"URL, default=postgres://postgres:postgres@localhost:5432/postgres"`
		MaxConns    int32    `env:"MAX_CONNS, default=25"`
		MinConns    int32    `env:"MIN_CONNS, default=5"`
		MaxConnIdle string   `env:"MAX_CONN_IDLE, default=5m"`
		MaxConnLife string   `env:"MAX_CONN_LIFE, default=1h"`
	} `env:", prefix=DATABASE_"`
}]

// Validate checks all configuration values and returns a ValidationError
// containing all issues found. This allows users to see all configuration
// problems at once rather than fixing them one at a time.
func (c Config) Validate() error {
	verr := &config.ValidationError{}

	// Validate port range
	if c.App.Port < 1 || c.App.Port > 65535 {
		verr.Add(fmt.Errorf("port must be between 1 and 65535, got %d", c.App.Port))
	}

	// Validate max body size
	if c.App.MaxBodySize <= 0 {
		verr.Add(fmt.Errorf("max_body_size must be positive, got %d", c.App.MaxBodySize))
	}

	// Validate rate limit
	if c.App.RateLimit <= 0 {
		verr.Add(fmt.Errorf("rate_limit must be positive, got %d", c.App.RateLimit))
	}

	// Reasonable upper bound to prevent misconfiguration
	if c.App.RateLimit > 100000 {
		verr.Add(fmt.Errorf("rate_limit too high (max 100000), got %d", c.App.RateLimit))
	}

	// Validate API key
	if c.App.APIKey == "" {
		verr.Add(fmt.Errorf("api_key is required"))
	}

	if len(c.App.APIKey) < 16 {
		verr.Add(fmt.Errorf("api_key must be at least 16 characters, got %d", len(c.App.APIKey)))
	}

	// Validate retention
	if c.App.Retention < 0 {
		verr.Add(fmt.Errorf("retention must be non-negative, got %s", c.App.Retention))
	}

	// Validate cleanup interval
	if c.App.CleanupInterval <= 0 {
		verr.Add(fmt.Errorf("cleanup_interval must be positive, got %s", c.App.CleanupInterval))
	}

	// Validate server timeouts
	if c.App.ReadTimeout <= 0 {
		verr.Add(fmt.Errorf("read_timeout must be positive, got %s", c.App.ReadTimeout))
	}
	if c.App.WriteTimeout <= 0 {
		verr.Add(fmt.Errorf("write_timeout must be positive, got %s", c.App.WriteTimeout))
	}
	if c.App.IdleTimeout <= 0 {
		verr.Add(fmt.Errorf("idle_timeout must be positive, got %s", c.App.IdleTimeout))
	}

	// Validate database URL scheme
	if c.App.Database.URL == nil {
		verr.Add(fmt.Errorf("database.url is required"))
	} else if c.App.Database.URL.Scheme != "postgres" && c.App.Database.URL.Scheme != "postgresql" {
		verr.Add(fmt.Errorf("database.url must be a valid postgresql URL, got scheme %q", c.App.Database.URL.Scheme))
	}

	// Validate max connections
	if c.App.Database.MaxConns <= 0 {
		verr.Add(fmt.Errorf("database.max_conns must be greater than 0, got %d", c.App.Database.MaxConns))
	}

	// Validate min connections
	if c.App.Database.MinConns <= 0 {
		verr.Add(fmt.Errorf("database.min_conns must be greater than 0, got %d", c.App.Database.MinConns))
	}

	// Validate min <= max
	if c.App.Database.MinConns > c.App.Database.MaxConns {
		verr.Add(fmt.Errorf("database.min_conns (%d) must be <= database.max_conns (%d)",
			c.App.Database.MinConns, c.App.Database.MaxConns))
	}

	// Validate max_conn_idle duration
	if maxConnIdle, err := time.ParseDuration(c.App.Database.MaxConnIdle); err != nil {
		verr.Add(fmt.Errorf("database.max_conn_idle must be a valid duration (e.g., \"5m\"), got %q",
			c.App.Database.MaxConnIdle))
	} else if maxConnIdle <= 0 {
		verr.Add(fmt.Errorf("database.max_conn_idle must be positive, got %s", c.App.Database.MaxConnIdle))
	}

	// Validate max_conn_life duration
	if maxConnLife, err := time.ParseDuration(c.App.Database.MaxConnLife); err != nil {
		verr.Add(fmt.Errorf("database.max_conn_life must be a valid duration (e.g., \"1h\"), got %q",
			c.App.Database.MaxConnLife))
	} else if maxConnLife <= 0 {
		verr.Add(fmt.Errorf("database.max_conn_life must be positive, got %s", c.App.Database.MaxConnLife))
	}

	if verr.HasErrors() {
		return verr
	}
	return nil
}

func Load() Config {
	var cfg Config
	config.Load("cmd/mroki-api", &cfg)

	// Validate configuration before returning
	if err := cfg.Validate(); err != nil {
		panic(fmt.Errorf("configuration validation failed: %w", err))
	}

	return cfg
}

// ParseCORSOrigins splits the comma-separated CORSOrigins string into
// a slice of trimmed, non-empty origin strings. Returns nil if empty.
func (c Config) ParseCORSOrigins() []string {
	if c.App.CORSOrigins == "" {
		return nil
	}
	parts := strings.Split(c.App.CORSOrigins, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
