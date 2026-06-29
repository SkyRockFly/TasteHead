package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ctxLoggerKey struct{}

var LoggerCtxKey = ctxLoggerKey{}

type middlewareResponseWriter struct {
	w          http.ResponseWriter
	statusCode int
}

func (r *middlewareResponseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *middlewareResponseWriter) Write(w []byte) (int, error) {
	num, err := r.w.Write(w)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}
	return num, nil
}

func (r *middlewareResponseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.w.WriteHeader(statusCode)
}

func LogMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("x-request-id")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		sw := &middlewareResponseWriter{w: w, statusCode: -1}

		subLogger := log.With().Str("requestID", requestID).Logger()

		subLogger.Info().
			Str("path", r.URL.Path).
			Str("method", r.Method).Msg("in")

		ctx := context.WithValue(r.Context(), LoggerCtxKey, subLogger)
		startTime := time.Now()
		next(sw, r.WithContext(ctx))
		duration := time.Since(startTime)
		if sw.statusCode >= 100 && sw.statusCode < 400 {
			subLogger.Info().Int("status", sw.statusCode).Int64("time_ms", duration.Milliseconds()).Msg("out")
		}
		if sw.statusCode >= 400 && sw.statusCode < 500 {
			subLogger.Warn().Int("status", sw.statusCode).Int64("time_ms", duration.Milliseconds()).Msg("out")
		}
		if sw.statusCode >= 500 && sw.statusCode < 600 {
			subLogger.Error().Int("status", sw.statusCode).Int64("time_ms", duration.Milliseconds()).Msg("out")
		}
	}
}
