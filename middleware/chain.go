package middleware

import "net/http"

// define the standard function for middlewares
type Middleware func(http.Handler) http.Handler

// chain helper for handing multiple middlewares
func Chain(middlewares ...Middleware) Middleware {

	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}

}
