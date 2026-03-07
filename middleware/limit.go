package middleware

import (
	"net/http"

	"github.com/qinyul/flash-sale/utils"
)

func BodyLimit(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.ContentLength > maxBytes {
				utils.Error(w, http.StatusRequestEntityTooLarge, "Request body too large")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}
