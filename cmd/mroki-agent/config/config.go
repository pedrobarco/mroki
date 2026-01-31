package config

import (
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

func Load() Config {
	var cfg Config
	config.Load("cmd/mroki-agent", &cfg)
	return cfg
}
