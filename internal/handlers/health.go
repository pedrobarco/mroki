package handlers

import (
	"context"
	"net/http"
	"time"
)

// HealthChecker defines the interface for performing health checks on external dependencies.
// This interface enables testing without requiring a real database connection.
type HealthChecker interface {
	// Ping verifies connectivity to the underlying resource.
	// It should return an error if the resource is unreachable or unhealthy.
	Ping(ctx context.Context) error
}

// Liveness returns an HTTP handler that always responds with 200 OK.
//
// This endpoint is intended for Kubernetes liveness probes to determine
// if the application process is running and responsive. It performs no
// external dependency checks, as failures here indicate the process
// itself is unhealthy and should be restarted.
//
// Response:
//   - 200 OK with body "OK" if the process is alive
//
// Kubernetes liveness probe example:
//
//	livenessProbe:
//	  httpGet:
//	    path: /health/live
//	    port: 8090
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

// Readiness returns an HTTP handler that checks database connectivity.
//
// This endpoint is intended for Kubernetes readiness probes to determine
// if the application can handle traffic. It verifies that the database
// is reachable within a 1-second timeout. If the database is unreachable,
// the pod will be removed from the service load balancer until it recovers.
//
// This same endpoint can also be used for startup probes to allow sufficient
// time for initial database connection during application startup.
//
// Response:
//   - 200 OK with body "OK" if the database is reachable
//   - 503 Service Unavailable with error details if the database check fails
//
// Kubernetes readiness probe example:
//
//	readinessProbe:
//	  httpGet:
//	    path: /health/ready
//	    port: 8090
//	  periodSeconds: 5
//	  failureThreshold: 2
//
// Kubernetes startup probe example:
//
//	startupProbe:
//	  httpGet:
//	    path: /health/ready
//	    port: 8090
//	  periodSeconds: 5
//	  failureThreshold: 12
func Readiness(db HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use 1-second timeout for quick failure detection
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		// Ping database to verify connectivity
		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, writeErr := w.Write([]byte("database health check failed: " + err.Error())); writeErr != nil {
				// Log error but don't fail - response is already sent
				return
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			// Log error but don't fail - response is already sent
			return
		}
	})
}
