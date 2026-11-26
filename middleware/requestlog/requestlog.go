//go:build !solution

package requestlog

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func Log(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestId := generateRequestId()

			l.Info(
				"request started",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("request_id", requestId),
			)

			startTime := time.Now()

			wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

			defer func() {
				if rec := recover(); rec != nil {
					l.Info(
						"request panicked",
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("request_id", requestId),
						zap.Duration("duration", time.Since(startTime)),
						zap.Int("status_code", wrapper.statusCode),
					)
					panic(rec)
				} else {
					l.Info(
						"request finished",
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("request_id", requestId),
						zap.Duration("duration", time.Since(startTime)),
						zap.Int("status_code", wrapper.statusCode),
					)
				}
			}()

			next.ServeHTTP(wrapper, r)
		})
	}
}

func generateRequestId() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (w responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}
	w.headerWritten = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
