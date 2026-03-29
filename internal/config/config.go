package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

// Config represents the application configuration.
// It uses generics to allow for different types of application configurations.
// It provides an utility struct that holds the application environment and the
// application-specific configuration.
type Config[T any] struct {
	AppEnv AppEnv `env:"APP_ENV, default=development"`
	App    T
}

// Load loads the configuration from environment variables and optionally from a
// .env file.
// It uses the APP_ENV environment variable to determine if it should load the
// .env file. If APP_ENV is not set, it defaults to "development".
// The cmd parameter is used to determine the directory where the .env file
// should be loaded from. It is expected to be the path to the command that is
// being executed (e.g., "cmd/mroki-proxy" or "cmd/caddy-mroki").
// The cfg parameter is a pointer to a struct that will be populated with the
// configuration values. The struct should have fields tagged with `env` to
// specify the environment variable names.
func Load(cmd string, cfg any) {
	env := AppEnv(os.Getenv("APP_ENV"))
	if env == "" {
		env = appEnvDevelopment
	}

	if env.IsDevelopment() {
		loadEnvFile(cmd)
	}

	if err := envconfig.ProcessWith(context.Background(), &envconfig.Config{
		Lookuper: envconfig.PrefixLookuper("MROKI_APP_", envconfig.OsLookuper()),
		Target:   cfg,
	}); err != nil {
		panic(fmt.Errorf("error processing environment variables: %w", err))
	}
}

func loadEnvFile(cmd string) {
	dir, err := filepath.Abs(cmd)
	if err != nil {
		panic(fmt.Errorf("error getting absolute path: %w", err))
	}

	f := filepath.Join(dir, ".env")
	if err := godotenv.Load(f); err != nil && !os.IsNotExist(err) {
		panic(fmt.Errorf("error loading .env file: %w", err))
	}
}
