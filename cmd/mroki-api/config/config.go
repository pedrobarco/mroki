package config

import (
	"net/url"

	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	Port     int `env:"PORT, default=8090"`
	Database struct {
		URL *url.URL `env:"URL, default=postgres://postgres:postgres@localhost:5432/postgres"`
	} `envPrefix:"DATABASE_"`
}]

func Load() Config {
	var cfg Config
	config.Load("cmd/mroki-api", &cfg)
	return cfg
}
