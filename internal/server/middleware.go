package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/aakash1408/shortener/internal/apperr"
	"github.com/aakash1408/shortener/internal/auth"
)

type contextKey string

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
