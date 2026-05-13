package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/aakash1408/shortener/internal/apperr"
	"github.com/aakash1408/shortener/internal/auth"
	"github.com/aakash1408/shortener/internal/shortcode"
)

var tracer = otel.Tracer("shortener")

// GET /
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

// POST /api/register
func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.register")
	defer span.End()

	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}
	if body.Username == "" || body.Email == "" || body.Password == "" {
		httpError(w, http.StatusBadRequest, fmt.Errorf("username, email and password are required"))
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		httpError(w, http.StatusInternalServerError, fmt.Errorf("failed to hash password"))
		return
	}
	user, err := s.store.CreateUser(ctx, body.Username, body.Email, hash)
	if err != nil {
		httpError(w, apperr.StatusCode(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// POST /api/login
func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.login")
	defer span.End()

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}
	user, err := s.store.GetUserByUsername(ctx, body.Username)
	if err != nil {
		httpError(w, http.StatusUnauthorized, apperr.ErrUnauthorized)
		return
	}
	if !auth.CheckPassword(body.Password, user.PasswordHash) {
		httpError(w, http.StatusUnauthorized, apperr.ErrUnauthorized)
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Username, s.cfg.JWTSecret)
	if err != nil {
		httpError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate token"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// GET /{shortCode}
func (s *server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.redirect")
	defer span.End()

	code := r.PathValue("shortCode")
	url, err := s.store.GetURLByCode(ctx, code)
	if err != nil {
		httpError(w, apperr.StatusCode(err), err)
		return
	}
	if url.ExpiresAt != nil && time.Now().After(*url.ExpiresAt) {
		httpError(w, http.StatusGone, apperr.ErrExpired)
		return
	}
	go func() {
		ipHash := fmt.Sprintf("%x", r.RemoteAddr)
		s.store.RecordClick(r.Context(), url.ID, ipHash, r.UserAgent())
	}()
	http.Redirect(w, r, url.LongURL, http.StatusFound)
}

// POST /api/urls
func (s *server) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.create_url")
	defer span.End()

	userID, _ := ctx.Value(userIDKey).(string)

	var body struct {
		LongURL    string     `json:"long_url"`
		CustomCode string     `json:"custom_code,omitempty"`
		ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}
	if body.LongURL == "" {
		httpError(w, http.StatusBadRequest, fmt.Errorf("long_url is required"))
		return
	}

	code := shortcode.Generate()
	if body.CustomCode != "" {
		if err := shortcode.Validate(body.CustomCode); err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
		code = body.CustomCode
	}

	url, err := s.store.CreateURL(ctx, userID, code, body.LongURL, body.ExpiresAt)
	if err != nil {
		httpError(w, apperr.StatusCode(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, url)
}

// GET /api/urls
func (s *server) handleListURLs(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.list_urls")
	defer span.End()

	userID, _ := ctx.Value(userIDKey).(string)
	urls, err := s.store.ListURLsByUser(ctx, userID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, urls)
}

// DELETE /api/urls/{shortCode}
func (s *server) handleDeleteURL(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.delete_url")
	defer span.End()

	userID, _ := ctx.Value(userIDKey).(string)
	code := r.PathValue("shortCode")
	if err := s.store.DeleteURL(ctx, code, userID); err != nil {
		httpError(w, apperr.StatusCode(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PATCH /api/urls/{shortCode}
func (s *server) handleUpdateURL(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.update_url")
	defer span.End()

	userID, _ := ctx.Value(userIDKey).(string)
	code := r.PathValue("shortCode")

	var body struct {
		LongURL string `json:"long_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}
	if body.LongURL == "" {
		httpError(w, http.StatusBadRequest, fmt.Errorf("long_url is required"))
		return
	}
	url, err := s.store.UpdateURL(ctx, code, userID, body.LongURL)
	if err != nil {
		httpError(w, apperr.StatusCode(err), err)
		return
	}
	writeJSON(w, http.StatusOK, url)
}

// GET /api/urls/{shortCode}/clicks
func (s *server) handleGetClicks(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.get_clicks")
	defer span.End()

	userID, _ := ctx.Value(userIDKey).(string)
	code := r.PathValue("shortCode")
	clicks, err := s.store.GetClicksByCode(ctx, code, userID)
	if err != nil {
		httpError(w, apperr.StatusCode(err), err)
		return
	}
	writeJSON(w, http.StatusOK, clicks)
}
