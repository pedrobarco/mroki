package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Import the mroki caddy module
	_ "github.com/pedrobarco/mroki/pkg/caddymodule"
)

func main() {
	caddycmd.Main()
}
