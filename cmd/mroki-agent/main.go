package main

import (
	"flag"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/pedrobarco/mroki/pkg/proxy"
)

var (
	fLive   string
	fShadow string
)

func init() {
	flag.StringVar(&fLive, "live", "", "Live URL to proxy requests to")
	flag.StringVar(&fShadow, "shadow", "", "Shadow URL to proxy requests to")
	flag.Parse()
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	if fLive == "" {
		slog.Error("Live URL is required")
		return
	}

	if fShadow == "" {
		slog.Error("Shadow URL is required")
		return
	}

	live, err := url.Parse(fLive)
	if err != nil {
		slog.Error("Invalid live URL", "error", err)
		return
	}

	shadow, err := url.Parse(fShadow)
	if err != nil {
		slog.Error("Invalid shadow URL", "error", err)
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxy.NewProxy(live, shadow))
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	slog.Info("Started server",
		"live", live.String(),
		"shadow", shadow.String(),
		"address", server.Addr,
	)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
