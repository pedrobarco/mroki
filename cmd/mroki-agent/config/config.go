package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	// URLs - optional if API mode configured (agent fetches from API)
	LiveURL   *url.URL `env:"LIVE_URL"`
	ShadowURL *url.URL `env:"SHADOW_URL"`

	Port          int           `env:"PORT, default=8080"`
	MaxBodySize   int64         `env:"MAX_BODY_SIZE, default=10485760"` // 10MB, 0=unlimited
	SamplingRate  float64       `env:"SAMPLING_RATE, default=1.0"`       // 0.0-1.0, default=1.0 (100%)
	LiveTimeout   time.Duration `env:"LIVE_TIMEOUT, default=5s"`
	ShadowTimeout time.Duration `env:"SHADOW_TIMEOUT, default=10s"`

	// API integration (optional - if not set, agent runs in standalone mode)
	APIURL     *url.URL      `env:"API_URL"`
	APIKey     string        `env:"API_KEY"`
	GateID     string        `env:"GATE_ID"`
	MaxRetries int           `env:"MAX_RETRIES, default=3"`
	RetryDelay time.Duration `env:"RETRY_DELAY, default=1s"`
	APITimeout time.Duration `env:"API_TIMEOUT, default=30s"`

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
		verr.Add(fmt.Errorf("port must be between 1 and 65535, got %d", c.App.Port))
	}

	// Check if API mode or standalone mode
	hasAPIConfig := c.App.APIURL != nil && c.App.GateID != "" && c.App.APIKey != ""
	hasStandaloneConfig := c.App.LiveURL != nil && c.App.ShadowURL != nil

	if !hasAPIConfig && !hasStandaloneConfig {
		verr.Add(fmt.Errorf("must configure either API mode (API_URL+GATE_ID+API_KEY) or standalone mode (LIVE_URL+SHADOW_URL)"))
	}

	// Validate API mode configuration
	if hasAPIConfig {
		// Validate API URL scheme
		if c.App.APIURL.Scheme != "http" && c.App.APIURL.Scheme != "https" {
			verr.Add(fmt.Errorf("api_url must use http or https scheme, got %q", c.App.APIURL.Scheme))
		}

		// Validate gate ID is a valid UUID
		if _, err := uuid.Parse(c.App.GateID); err != nil {
			verr.Add(fmt.Errorf("gate_id must be a valid UUID: %w", err))
		}

		// API key validation
		if c.App.APIKey == "" {
			verr.Add(fmt.Errorf("api_key is required when api integration is configured"))
		}

		// Validate retry config
		if c.App.MaxRetries < 0 {
			verr.Add(fmt.Errorf("max_retries must be non-negative, got %d", c.App.MaxRetries))
		}
		if c.App.RetryDelay <= 0 {
			verr.Add(fmt.Errorf("retry_delay must be positive, got %s", c.App.RetryDelay))
		}
		if c.App.APITimeout <= 0 {
			verr.Add(fmt.Errorf("api_timeout must be positive, got %s", c.App.APITimeout))
		}
	}

	// Validate standalone mode configuration (only when not in API mode)
	if !hasAPIConfig && hasStandaloneConfig {
		// Validate live URL
		if c.App.LiveURL.Scheme != "http" && c.App.LiveURL.Scheme != "https" {
			verr.Add(fmt.Errorf("live_url must use http or https scheme, got %q", c.App.LiveURL.Scheme))
		}

		// Validate shadow URL
		if c.App.ShadowURL.Scheme != "http" && c.App.ShadowURL.Scheme != "https" {
			verr.Add(fmt.Errorf("shadow_url must use http or https scheme, got %q", c.App.ShadowURL.Scheme))
		}
	}

	// Validate timeouts (both modes)
	if c.App.LiveTimeout <= 0 {
		verr.Add(fmt.Errorf("live_timeout must be positive, got %s", c.App.LiveTimeout))
	}
	if c.App.ShadowTimeout <= 0 {
		verr.Add(fmt.Errorf("shadow_timeout must be positive, got %s", c.App.ShadowTimeout))
	}

	// Validate max body size
	if c.App.MaxBodySize < 0 {
		verr.Add(fmt.Errorf("max_body_size must be non-negative (0=unlimited), got %d", c.App.MaxBodySize))
	}

	// Validate sampling rate
	if c.App.SamplingRate < 0 || c.App.SamplingRate > 1 {
		verr.Add(fmt.Errorf("sampling_rate must be between 0.0 and 1.0, got %f", c.App.SamplingRate))
	}

	// Validate diff float tolerance
	if c.App.DiffFloatTolerance < 0 {
		verr.Add(fmt.Errorf("diff_float_tolerance must be non-negative, got %f", c.App.DiffFloatTolerance))
	}

	if verr.HasErrors() {
		return verr
	}
	return nil
}

func Load() Config {
	var cfg Config
	config.Load("cmd/mroki-agent", &cfg)

	// Validate configuration before returning
	if err := cfg.Validate(); err != nil {
		panic(fmt.Errorf("configuration validation failed: %w", err))
	}

	return cfg
}
