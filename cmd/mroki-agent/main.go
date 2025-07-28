package main

import (
	"log/slog"

	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Import the mroki caddy module
	_ "github.com/pedrobarco/mroki/pkg/caddymodule"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	caddycmd.Main()
}
