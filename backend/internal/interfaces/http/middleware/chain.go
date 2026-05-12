package middleware

import "net/http"

// Middleware is the canonical signature.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares in the order they are passed: the leftmost
// middleware sits at the outside of the onion.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
