package logger

import (
	"log/slog"
	"net/http"
	"time"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter 
	responseData        *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 200,
			size:   0,
		}

		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		next.ServeHTTP(lrw, r)

		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", responseData.status,
			"size_bytes", responseData.size,
			"duration", time.Since(start),
		)
	})
}