package handlers

import (
	"net/http"
	"sync/atomic"
)

// Liveness returns an HTTP handler that always responds with 200 OK.
//
// This endpoint is intended for Kubernetes liveness probes to determine
// if the proxy process is running and responsive. It performs no external
// dependency checks, as failures here indicate the process itself is
// unhealthy and should be restarted.
//
// Response:
//   - 200 OK with body "OK" if the process is alive
//
// Kubernetes liveness probe example:
//
//	livenessProbe:
//	  httpGet:
//	    path: /health/live
//	    port: admin
//	  periodSeconds: 10
//	  failureThreshold: 3
func Liveness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			// Log error but don't fail - response is already sent
			return
		}
	})
}

// Readiness returns an HTTP handler that reports whether the proxy is ready
// to serve traffic, based on the supplied atomic flag.
//
// This endpoint is intended for Kubernetes readiness probes. The proxy flips
// the flag to true once configuration is loaded and startup completes, and back
// to false when graceful shutdown begins so the pod is removed from the service
// load balancer while connections drain.
//
// Response:
//   - 200 OK with body "OK" when ready
//   - 503 Service Unavailable with body "NOT READY" otherwise
//
// Kubernetes readiness probe example:
//
//	readinessProbe:
//	  httpGet:
//	    path: /health/ready
//	    port: admin
//	  periodSeconds: 5
//	  failureThreshold: 2
func Readiness(ready *atomic.Bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if !ready.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.Write([]byte("NOT READY")); err != nil {
				// Log error but don't fail - response is already sent
				return
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			// Log error but don't fail - response is already sent
			return
		}
	})
}
