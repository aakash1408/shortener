package store

import (
	"time"
	"context"
)

type User struct{
	ID string
	Username string
	Email string
	PasswordHash string
	CreatedAt time.Time
}

type URL struct{
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ShortCode string     `json:"short_code"`
	LongURL   string     `json:"long_url"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type Click struct{
	ID string
	URLID string
	IPHash string
	UserAgent string
	ClickedAt time.Time
}


type Store interface {
	// Users
	CreateUser(ctx context.Context, username, email, passwordHash string) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)

	// URLs
	CreateURL(ctx context.Context, userID, shortCode, longURL string, expiresAt *time.Time) (URL, error)
	GetURLByCode(ctx context.Context, shortCode string) (URL, error)
	ListURLsByUser(ctx context.Context, userID string) ([]URL, error)
	DeleteURL(ctx context.Context, shortCode, userID string) error
	UpdateURL(ctx context.Context, shortCode, userID, longURL string) (URL, error)

	// Clicks
	RecordClick(ctx context.Context, urlID, ipHash, userAgent string) error
	GetClicksByCode(ctx context.Context, shortCode, userID string) ([]Click, error)
}
