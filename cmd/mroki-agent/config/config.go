package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	LiveURL       *url.URL      `env:"LIVE_URL, required"`
	ShadowURL     *url.URL      `env:"SHADOW_URL, required"`
	Port          int           `env:"PORT, default=8080"`
	MaxBodySize   int64         `env:"MAX_BODY_SIZE, default=10485760"` // 10MB, 0=unlimited
	LiveTimeout   time.Duration `env:"LIVE_TIMEOUT, default=5s"`
	ShadowTimeout time.Duration `env:"SHADOW_TIMEOUT, default=10s"`

	// API integration (optional - if not set, agent runs in standalone mode)
	APIURL     *url.URL      `env:"API_URL"`
	APIKey     string        `env:"API_KEY"`
	GateID     string        `env:"GATE_ID"`
	MaxRetries int           `env:"MAX_RETRIES, default=3"`
	RetryDelay time.Duration `env:"RETRY_DELAY, default=1s"`
	APITimeout time.Duration `env:"API_TIMEOUT, default=30s"`
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

	// Validate live URL
	if c.App.LiveURL == nil {
		verr.Add(fmt.Errorf("live_url is required"))
	} else if c.App.LiveURL.Scheme != "http" && c.App.LiveURL.Scheme != "https" {
		verr.Add(fmt.Errorf("live_url must use http or https scheme, got %q", c.App.LiveURL.Scheme))
	}

	// Validate shadow URL
	if c.App.ShadowURL == nil {
		verr.Add(fmt.Errorf("shadow_url is required"))
	} else if c.App.ShadowURL.Scheme != "http" && c.App.ShadowURL.Scheme != "https" {
		verr.Add(fmt.Errorf("shadow_url must use http or https scheme, got %q", c.App.ShadowURL.Scheme))
	}

	// Validate live timeout
	if c.App.LiveTimeout <= 0 {
		verr.Add(fmt.Errorf("live_timeout must be positive, got %s", c.App.LiveTimeout))
	}

	// Validate shadow timeout
	if c.App.ShadowTimeout <= 0 {
		verr.Add(fmt.Errorf("shadow_timeout must be positive, got %s", c.App.ShadowTimeout))
	}

	// Validate max body size (0 is allowed for unlimited)
	if c.App.MaxBodySize < 0 {
		verr.Add(fmt.Errorf("max_body_size must be non-negative (0=unlimited), got %d", c.App.MaxBodySize))
	}

	// Validate API configuration (optional, but if one is set, all must be set)
	hasAPIURL := c.App.APIURL != nil
	hasGateID := c.App.GateID != ""
	hasAPIKey := c.App.APIKey != ""

	if hasAPIURL || hasGateID || hasAPIKey {
		// If any API config is set, all must be set
		if !hasAPIURL {
			verr.Add(fmt.Errorf("api_url is required when api integration is configured"))
		}
		if !hasGateID {
			verr.Add(fmt.Errorf("gate_id is required when api integration is configured"))
		}
		if !hasAPIKey {
			verr.Add(fmt.Errorf("api_key is required when api integration is configured"))
		}
	}

	if hasAPIURL {
		// Validate API URL scheme
		if c.App.APIURL.Scheme != "http" && c.App.APIURL.Scheme != "https" {
			verr.Add(fmt.Errorf("api_url must use http or https scheme, got %q", c.App.APIURL.Scheme))
		}

		// Validate gate ID is a valid UUID
		if _, err := uuid.Parse(c.App.GateID); err != nil {
			verr.Add(fmt.Errorf("gate_id must be a valid UUID: %w", err))
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
