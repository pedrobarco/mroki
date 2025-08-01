package handlers

import (
	"net/http"
	"net/url"

	"github.com/pedrobarco/mroki/pkg/proxy"
)

func Proxy(live, shadow *url.URL) http.HandlerFunc {
	proxy := proxy.NewProxy(live, shadow)
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}
