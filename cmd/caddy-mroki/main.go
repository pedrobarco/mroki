package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Plug in Caddy's standard modules and the mroki gate handler so this
	// binary is a self-contained Caddy build (no xcaddy required).
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	_ "github.com/pedrobarco/mroki/pkg/caddymodule"
)

func main() {
	caddycmd.Main()
}
