package config

import (
	"net/url"

	"github.com/pedrobarco/mroki/internal/config"
)

type Config config.Config[struct {
	Port     int `env:"PORT, default=8090"`
	Database struct {
		URL         *url.URL `env:"URL, default=postgres://postgres:postgres@localhost:5432/postgres"`
		MaxConns    int32    `env:"MAX_CONNS, default=25"`
		MinConns    int32    `env:"MIN_CONNS, default=5"`
		MaxConnIdle string   `env:"MAX_CONN_IDLE, default=5m"`
		MaxConnLife string   `env:"MAX_CONN_LIFE, default=1h"`
	} `envPrefix:"DATABASE_"`
}]

func Load() Config {
	var cfg Config
	config.Load("cmd/mroki-api", &cfg)
	return cfg
}
