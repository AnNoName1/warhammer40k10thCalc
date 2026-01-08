package middleware

import (
	"log"
	"net/http"
)

func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				reqID := GetRequestID(r.Context())

				log.Printf("[%s] PANIC: %v", reqID, rec)

				http.Error(
					w,
					"Internal Server Error",
					http.StatusInternalServerError,
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
