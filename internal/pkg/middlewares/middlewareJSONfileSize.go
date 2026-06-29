package middlewares

import "net/http"

const (
	maxBodySize = 10 << 20 // 10 MB
)

func JSONReqSizeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		next.ServeHTTP(w, r)
	}
}
