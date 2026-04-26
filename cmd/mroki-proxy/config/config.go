package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/config"
)

// ValidationError is a type alias for config.ValidationError so that
// consumers don't need to import internal/config directly.
type ValidationError = config.ValidationError

type Config config.Config[struct {
	// URLs - optional if API mode configured (proxy fetches from API)
	LiveURL   *url.URL `env:"LIVE_URL"`
	ShadowURL *url.URL `env:"SHADOW_URL"`

	Port          int           `env:"PORT, default=8080"`
	MaxBodySize   int64         `env:"MAX_BODY_SIZE, default=10485760"` // 10MB, 0=unlimited
	SamplingRate  float64       `env:"SAMPLING_RATE, default=1.0"`       // 0.0-1.0, default=1.0 (100%)
	LiveTimeout   time.Duration `env:"LIVE_TIMEOUT, default=5s"`
	ShadowTimeout time.Duration `env:"SHADOW_TIMEOUT, default=10s"`
	ReadTimeout   time.Duration `env:"READ_TIMEOUT, default=30s"`
	WriteTimeout  time.Duration `env:"WRITE_TIMEOUT, default=60s"`
	IdleTimeout   time.Duration `env:"IDLE_TIMEOUT, default=120s"`

	// API integration (optional - if not set, proxy runs in standalone mode)
	APIURL     *url.URL      `env:"API_URL"`
	APIKey     string        `env:"API_KEY"`
	GateID     string        `env:"GATE_ID"`
	MaxRetries int           `env:"MAX_RETRIES, default=3"`
	RetryDelay time.Duration `env:"RETRY_DELAY, default=1s"`
	APITimeout time.Duration `env:"API_TIMEOUT, default=30s"`

	// Circuit breaker
	CBFailureThreshold int           `env:"CB_FAILURE_THRESHOLD, default=5"`
	CBDelay            time.Duration `env:"CB_DELAY, default=1m"`
	CBSuccessThreshold int           `env:"CB_SUCCESS_THRESHOLD, default=2"`

	// Diff options (optional, works in both API and standalone modes)
	DiffIgnoredFields  []string `env:"DIFF_IGNORED_FIELDS"`  // Comma-separated
	DiffIncludedFields []string `env:"DIFF_INCLUDED_FIELDS"` // Comma-separated
	DiffFloatTolerance float64  `env:"DIFF_FLOAT_TOLERANCE, default=0"`
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

	// Check if API mode or standalone mode
	hasAPIConfig := c.App.APIURL != nil && c.App.GateID != "" && c.App.APIKey != ""
	hasStandaloneConfig := c.App.LiveURL != nil && c.App.ShadowURL != nil

	if !hasAPIConfig && !hasStandaloneConfig {
		verr.Add(config.SeverityError, "must configure either API mode (API_URL+GATE_ID+API_KEY) or standalone mode (LIVE_URL+SHADOW_URL)")
	}

	// Validate API mode configuration
	if hasAPIConfig {
		// Validate API URL scheme
		if c.App.APIURL.Scheme != "http" && c.App.APIURL.Scheme != "https" {
			verr.Add(config.SeverityError, fmt.Sprintf("api_url must use http or https scheme, got %q", c.App.APIURL.Scheme))
		}

		// Validate gate ID is a valid UUID
		if _, err := uuid.Parse(c.App.GateID); err != nil {
			verr.Add(config.SeverityError, fmt.Sprintf("gate_id must be a valid UUID: %v", err))
		}

		// API key validation
		if c.App.APIKey == "" {
			verr.Add(config.SeverityError, "api_key is required when api integration is configured")
		}

		// Validate retry config
		if c.App.MaxRetries < 0 {
			verr.Add(config.SeverityError, fmt.Sprintf("max_retries must be non-negative, got %d", c.App.MaxRetries))
		}
		if c.App.RetryDelay <= 0 {
			verr.Add(config.SeverityError, fmt.Sprintf("retry_delay must be positive, got %s", c.App.RetryDelay))
		}
		if c.App.APITimeout <= 0 {
			verr.Add(config.SeverityError, fmt.Sprintf("api_timeout must be positive, got %s", c.App.APITimeout))
		}
		if c.App.CBFailureThreshold < 1 {
			verr.Add(config.SeverityError, fmt.Sprintf("cb_failure_threshold must be positive, got %d", c.App.CBFailureThreshold))
		}
		if c.App.CBDelay <= 0 {
			verr.Add(config.SeverityError, fmt.Sprintf("cb_delay must be positive, got %s", c.App.CBDelay))
		}
		if c.App.CBSuccessThreshold < 1 {
			verr.Add(config.SeverityError, fmt.Sprintf("cb_success_threshold must be positive, got %d", c.App.CBSuccessThreshold))
		}
	}

	// Validate standalone mode configuration (only when not in API mode)
	if !hasAPIConfig && hasStandaloneConfig {
		// Validate live URL
		if c.App.LiveURL.Scheme != "http" && c.App.LiveURL.Scheme != "https" {
			verr.Add(config.SeverityError, fmt.Sprintf("live_url must use http or https scheme, got %q", c.App.LiveURL.Scheme))
		}

		// Validate shadow URL
		if c.App.ShadowURL.Scheme != "http" && c.App.ShadowURL.Scheme != "https" {
			verr.Add(config.SeverityError, fmt.Sprintf("shadow_url must use http or https scheme, got %q", c.App.ShadowURL.Scheme))
		}
	}

	// Validate timeouts (both modes)
	if c.App.LiveTimeout <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("live_timeout must be positive, got %s", c.App.LiveTimeout))
	}
	if c.App.ShadowTimeout <= 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("shadow_timeout must be positive, got %s", c.App.ShadowTimeout))
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

	// Cross-validate write timeout covers live request lifecycle
	if c.App.WriteTimeout > 0 && c.App.LiveTimeout > 0 && c.App.WriteTimeout < c.App.LiveTimeout {
		verr.Add(config.SeverityError, fmt.Sprintf("write_timeout (%s) must be >= live_timeout (%s); the proxy needs enough time to write the live response back to the client",
			c.App.WriteTimeout, c.App.LiveTimeout))
	}

	// Validate max body size
	if c.App.MaxBodySize < 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("max_body_size must be non-negative (0=unlimited), got %d", c.App.MaxBodySize))
	}

	// Validate sampling rate
	if c.App.SamplingRate < 0 || c.App.SamplingRate > 1 {
		verr.Add(config.SeverityError, fmt.Sprintf("sampling_rate must be between 0.0 and 1.0, got %f", c.App.SamplingRate))
	}

	// Validate diff float tolerance
	if c.App.DiffFloatTolerance < 0 {
		verr.Add(config.SeverityError, fmt.Sprintf("diff_float_tolerance must be non-negative, got %f", c.App.DiffFloatTolerance))
	}

	// --- Warnings (non-fatal) ---

	// tlsHandshakeTimeout is the hardcoded TLS handshake safety net used in
	// pkg/proxy.newDefaultHTTPClient.
	const tlsHandshakeTimeout = 5 * time.Second

	// Warn if live timeout is shorter than the hardcoded TLS handshake
	// safety net — TLS failures will surface as generic context errors
	// instead of clear TLS-specific errors.
	if c.App.LiveTimeout > 0 && c.App.LiveTimeout < tlsHandshakeTimeout {
		verr.Add(config.SeverityWarning, fmt.Sprintf(
			"live_timeout (%s) is less than the TLS handshake safety net (%s); TLS errors to HTTPS backends will appear as generic context deadline errors",
			c.App.LiveTimeout, tlsHandshakeTimeout))
	}

	// Warn if the retry budget could exceed the API timeout.
	// Worst-case backoff: InitialDelay + 2*InitialDelay + 4*InitialDelay + ...
	// for MaxRetries attempts = InitialDelay * (2^MaxRetries - 1).
	if c.App.APIURL != nil && c.App.APITimeout > 0 && c.App.MaxRetries > 0 && c.App.RetryDelay > 0 {
		worstCaseBackoff := c.App.RetryDelay * time.Duration((1<<c.App.MaxRetries)-1)
		if worstCaseBackoff >= c.App.APITimeout {
			verr.Add(config.SeverityWarning, fmt.Sprintf(
				"retry budget (worst-case backoff %s for %d retries with %s initial delay) may exceed api_timeout (%s); retries could be cancelled before completing",
				worstCaseBackoff, c.App.MaxRetries, c.App.RetryDelay, c.App.APITimeout))
		}
	}

	if verr.HasEntries() {
		return verr
	}
	return nil
}

// Load reads configuration from environment and .env files, validates it,
// and returns the config along with any validation error.
func Load() (Config, error) {
	var cfg Config
	config.Load("cmd/mroki-proxy", &cfg)
	return cfg, cfg.Validate()
}
