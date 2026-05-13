package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aakash1408/shortener/internal/apperr"
	"github.com/aakash1408/shortener/internal/auth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

type contextKey string

type spyResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func (w *spyResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *spyResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += n
	return n, err
}

func (s *server) requestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		spy := &spyResponseWriter{ResponseWriter: w}
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(spy, r)

		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", spy.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", spy.bytesWritten,
			"request_id", requestID,
			"ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		spy := &spyResponseWriter{ResponseWriter: w}

		next.ServeHTTP(spy, r)

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", spy.status)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

const userIDKey contextKey = "userID"
const usernameKey contextKey = "username"

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			httpError(w, http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httpError(w, http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(parts[1], s.cfg.JWTSecret)
		if err != nil {
			httpError(w, http.StatusUnauthorized, apperr.ErrUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, usernameKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
