package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	LiveURL       *url.URL      `env:"LIVE_URL, required"`
	ShadowURL     *url.URL      `env:"SHADOW_URL, required"`
	Port          int           `env:"PORT, default=8080"`
	LiveTimeout   time.Duration `env:"LIVE_TIMEOUT, default=5s"`
	ShadowTimeout time.Duration `env:"SHADOW_TIMEOUT, default=10s"`
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
