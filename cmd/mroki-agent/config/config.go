package config

import (
	"net/url"

	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	LiveURL   *url.URL `env:"LIVE_URL, required"`
	ShadowURL *url.URL `env:"SHADOW_URL, required"`
	Port      int      `env:"PORT, default=8080"`
}]

func Load() Config {
	var cfg Config
	config.Load("cmd/mroki-agent", &cfg)
	return cfg
}
