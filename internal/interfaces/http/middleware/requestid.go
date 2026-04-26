package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIDKey struct{}

const RequestIDHeader = "X-Request-ID"

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if id == "" {
				id = uuid.New().String()
				r.Header.Set(RequestIDHeader, id)
			}
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
