package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pedrobarco/mroki/internal/config"
)

// ValidationError is a type alias for config.ValidationError so that
// consumers don't need to import internal/config directly.
type ValidationError = config.ValidationError

type Config config.Config[struct {
	Port            int           `env:"PORT, default=8090"`
	MaxBodySize     int64         `env:"MAX_BODY_SIZE, default=10485760"` // 10MB
	RateLimit       int           `env:"RATE_LIMIT, default=1000"`        // requests per minute per IP
	APIKey          string        `env:"API_KEY, required"`
	CORSOrigins     string        `env:"CORS_ORIGINS"`            // comma-separated allowed origins, empty = disabled
	TrustedProxies  string        `env:"TRUSTED_PROXIES"`         // comma-separated CIDRs/IPs allowed to set X-Forwarded-For, empty = XFF ignored
	Retention       time.Duration `env:"RETENTION, default=720h"` // global request retention floor; must be > 0, e.g. 168h = 7 days
	CleanupInterval time.Duration `env:"CLEANUP_INTERVAL, default=1h"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT, default=15s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT, default=30s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT, default=60s"`
	MetricsEnabled  bool          `env:"METRICS_ENABLED, default=true"` // expose /metrics for Prometheus scraping
	// HSTS (Strict-Transport-Security). Off by default because mroki does not
	// terminate TLS; only enable once a TLS-terminating reverse proxy is in
	// front of the API. When enabled, HSTSMaxAge must be > 0.
	HSTSEnabled bool          `env:"HSTS_ENABLED, default=false"`
	HSTSMaxAge  time.Duration `env:"HSTS_MAX_AGE, default=8760h"` // max-age for the HSTS header (default 365d)
	// Logging: level is one of debug, info, warn, error; format is text or json.
	// When left empty the effective defaults are derived from APP_ENV via
	// EffectiveLogLevel/EffectiveLogFormat (development: debug/text,
	// production: info/json).
	LogLevel  string `env:"LOG_LEVEL"`
	LogFormat string `env:"LOG_FORMAT"`
	Database  struct {
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
		verr.Add(config.SeverityError, fmt.Sprintf("port must be between 1 and 65535, got %d", c.App.Port))
	}

	// Validate max body size
	if c.App.MaxBodySize <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("max_body_size must be positive, got %d", c.App.MaxBodySize))
	}

	// Validate rate limit
	if c.App.RateLimit <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("rate_limit must be positive, got %d", c.App.RateLimit))
	}

	// Reasonable upper bound to prevent misconfiguration
	if c.App.RateLimit > 100000 {
		verr.Add(config.SeverityError, fmt.Sprintf("rate_limit too high (max 100000), got %d", c.App.RateLimit))
	}

	// Validate API key
	if c.App.APIKey == "" {
		verr.Add(config.SeverityError, "api_key is required")
	} else if len(c.App.APIKey) < 16 {
		verr.Add(config.SeverityError, fmt.Sprintf("api_key must be at least 16 characters, got %d", len(c.App.APIKey)))
	}

	// Validate retention. Keep-forever (0) is no longer supported: retention is
	// the global floor applied to every gate, so it must be a positive duration.
	if c.App.Retention <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("retention must be positive (keep-forever is no longer supported), got %s", c.App.Retention))
	}

	// Validate cleanup interval
	if c.App.CleanupInterval <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("cleanup_interval must be positive, got %s", c.App.CleanupInterval))
	}

	// Validate server timeouts
	if c.App.ReadTimeout <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("read_timeout must be positive, got %s", c.App.ReadTimeout))
	}
	if c.App.WriteTimeout <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("write_timeout must be positive, got %s", c.App.WriteTimeout))
	}
	if c.App.IdleTimeout <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("idle_timeout must be positive, got %s", c.App.IdleTimeout))
	}

	// Cross-validate server timeout ordering: Read < Write < Idle
	if c.App.ReadTimeout > 0 && c.App.WriteTimeout > 0 && c.App.ReadTimeout >= c.App.WriteTimeout {
		verr.Add(config.SeverityError, fmt.Sprintf("read_timeout (%s) must be less than write_timeout (%s)",
			c.App.ReadTimeout, c.App.WriteTimeout))
	}
	if c.App.WriteTimeout > 0 && c.App.IdleTimeout > 0 && c.App.WriteTimeout >= c.App.IdleTimeout {
		verr.Add(config.SeverityError, fmt.Sprintf("write_timeout (%s) must be less than idle_timeout (%s)",
			c.App.WriteTimeout, c.App.IdleTimeout))
	}

	// Validate HSTS. The max-age must be positive when HSTS is enabled so the
	// emitted Strict-Transport-Security header is meaningful. When disabled the
	// value is ignored and the header is never sent.
	if c.App.HSTSEnabled && c.App.HSTSMaxAge <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("hsts_max_age must be positive when hsts is enabled, got %s", c.App.HSTSMaxAge))
	}

	// Validate CORS origins. The API always allows the Authorization header
	// (see the CORS setup in cmd/mroki-api/main.go), so a wildcard origin is
	// rejected to avoid an unsafe credentialed cross-origin policy.
	if c.App.CORSOrigins != "" {
		config.ValidateCORSOrigins(verr, c.ParseCORSOrigins(), true)
	}

	// Validate trusted proxies. Each entry must be a valid CIDR or bare IP.
	// Empty means X-Forwarded-For is never trusted and per-IP rate limiting
	// always keys off the direct peer (RemoteAddr).
	if c.App.TrustedProxies != "" {
		config.ValidateTrustedProxies(verr, c.ParseTrustedProxies())
	}

	// Validate database URL scheme
	if c.App.Database.URL == nil {
		verr.Add(config.SeverityError, "database.url is required")
	} else if c.App.Database.URL.Scheme != "postgres" && c.App.Database.URL.Scheme != "postgresql" {
		verr.Add(config.SeverityError, fmt.Sprintf("database.url must be a valid postgresql URL, got scheme %q", c.App.Database.URL.Scheme))
	}

	// Validate max connections
	if c.App.Database.MaxConns <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("database.max_conns must be greater than 0, got %d", c.App.Database.MaxConns))
	}

	// Validate min connections
	if c.App.Database.MinConns <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("database.min_conns must be greater than 0, got %d", c.App.Database.MinConns))
	}

	// Validate min <= max
	if c.App.Database.MinConns > c.App.Database.MaxConns {
		verr.Add(config.SeverityError, fmt.Sprintf("database.min_conns (%d) must be <= database.max_conns (%d)",
			c.App.Database.MinConns, c.App.Database.MaxConns))
	}

	// Validate max_conn_idle duration
	if maxConnIdle, err := time.ParseDuration(c.App.Database.MaxConnIdle); err != nil {
		verr.Add(config.SeverityError, fmt.Sprintf("database.max_conn_idle must be a valid duration (e.g., \"5m\"), got %q",
			c.App.Database.MaxConnIdle))
	} else if maxConnIdle <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("database.max_conn_idle must be positive, got %s", c.App.Database.MaxConnIdle))
	}

	// Validate max_conn_life duration
	if maxConnLife, err := time.ParseDuration(c.App.Database.MaxConnLife); err != nil {
		verr.Add(config.SeverityError, fmt.Sprintf("database.max_conn_life must be a valid duration (e.g., \"1h\"), got %q",
			c.App.Database.MaxConnLife))
	} else if maxConnLife <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("database.max_conn_life must be positive, got %s", c.App.Database.MaxConnLife))
	}

	// Validate logging settings
	config.ValidateLogSettings(verr, c.App.LogLevel, c.App.LogFormat)

	// Validate application environment
	config.ValidateAppEnv(verr, c.AppEnv)

	if verr.HasEntries() {
		return verr
	}
	return nil
}

// Load reads configuration from environment and .env files, validates it,
// and returns the config along with any validation error.
func Load() (Config, error) {
	var cfg Config
	config.Load("cmd/mroki-api", &cfg)
	return cfg, cfg.Validate()
}

// EffectiveLogLevel returns the log level to use. An explicit LogLevel always
// wins; otherwise the level is derived from APP_ENV (production: info,
// development: debug).
func (c Config) EffectiveLogLevel() string {
	return c.AppEnv.EffectiveLogLevel(c.App.LogLevel)
}

// EffectiveLogFormat returns the log format to use. An explicit LogFormat always
// wins; otherwise the format is derived from APP_ENV (production: json,
// development: text).
func (c Config) EffectiveLogFormat() string {
	return c.AppEnv.EffectiveLogFormat(c.App.LogFormat)
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

// ParseTrustedProxies splits the comma-separated TrustedProxies string into a
// slice of trimmed, non-empty entries (CIDRs or bare IPs). Returns nil if empty.
func (c Config) ParseTrustedProxies() []string {
	if c.App.TrustedProxies == "" {
		return nil
	}
	parts := strings.Split(c.App.TrustedProxies, ",")
	proxies := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			proxies = append(proxies, trimmed)
		}
	}
	return proxies
}
