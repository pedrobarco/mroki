package handlers

import (
	"net/http"
	"slices"
)

type Chain []func(http.Handler) http.Handler

func (c Chain) ThenFunc(h http.HandlerFunc) http.Handler {
	return c.Then(h)
}

func (c Chain) Then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}
