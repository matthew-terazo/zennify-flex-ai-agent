package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	LoggerKey contextKey = "logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		requestID := GetRequestID(r.Context())
		logger := log.With().Str("request_id", requestID).Logger()

		ctx := r.Context()
		ctx = context.WithValue(ctx, LoggerKey, logger)
		r = r.WithContext(ctx)

		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("body", string(requestBody)).
			Msg("Incoming request")

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rw.statusCode).
			Str("body", rw.body.String()).
			Msg("Response sent")
	})
}

func GetLogger(ctx context.Context) zerolog.Logger {
	logger, ok := ctx.Value(LoggerKey).(zerolog.Logger)
	if !ok {
		return zerolog.New(nil)
	}
	return logger
}
