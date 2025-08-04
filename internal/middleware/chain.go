package middleware

import (
	"net/http"
	"slices"
)

type Middleware func(http.Handler) http.Handler

type Chain []Middleware

func (c Chain) ThenFunc(h http.HandlerFunc) http.Handler {
	return c.Then(h)
}

func (c Chain) Then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}
