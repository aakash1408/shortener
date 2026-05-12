package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aakash1408/shortener/internal/apperr"
)

//go:embed migrations
var migrationFiles embed.FS

type PostgresStore struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, username, email, passwordHash string) (User, error) {
	query := `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, created_at`

	var u User
	err := s.db.QueryRow(ctx, query, username, email, passwordHash).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, apperr.ErrConflict
		}
		return User{}, fmt.Errorf("failed to create user: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at
		FROM users WHERE username = $1`

	var u User
	err := s.db.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, apperr.ErrNotFound
		}
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) CreateURL(ctx context.Context, userID, shortCode, longURL string, expiresAt *time.Time) (URL, error) {
	query := `
		INSERT INTO urls (user_id, short_code, long_url, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, short_code, long_url, expires_at, created_at`

	var u URL
	err := s.db.QueryRow(ctx, query, userID, shortCode, longURL, expiresAt).Scan(
		&u.ID,
		&u.UserID,
		&u.ShortCode,
		&u.LongURL,
		&u.ExpiresAt,
		&u.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return URL{}, apperr.ErrConflict
		}
		return URL{}, fmt.Errorf("failed to create url: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) GetURLByCode(ctx context.Context, shortCode string) (URL, error) {
	query := `
		SELECT id, user_id, short_code, long_url, expires_at, created_at
		FROM urls WHERE short_code = $1`

	var u URL
	err := s.db.QueryRow(ctx, query, shortCode).Scan(
		&u.ID,
		&u.UserID,
		&u.ShortCode,
		&u.LongURL,
		&u.ExpiresAt,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return URL{}, apperr.ErrNotFound
		}
		return URL{}, fmt.Errorf("failed to get url: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) ListURLsByUser(ctx context.Context, userID string) ([]URL, error) {
	query := `
		SELECT id, user_id, short_code, long_url, expires_at, created_at
		FROM urls WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list urls: %w", err)
	}
	defer rows.Close()

	var urls []URL
	for rows.Next() {
		var u URL
		if err := rows.Scan(&u.ID, &u.UserID, &u.ShortCode, &u.LongURL, &u.ExpiresAt, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan url: %w", err)
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

func (s *PostgresStore) DeleteURL(ctx context.Context, shortCode, userID string) error {
	query := `DELETE FROM urls WHERE short_code = $1 AND user_id = $2`

	result, err := s.db.Exec(ctx, query, shortCode, userID)
	if err != nil {
		return fmt.Errorf("failed to delete url: %w", err)
	}
	if result.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UpdateURL(ctx context.Context, shortCode, userID, longURL string) (URL, error) {
	query := `
		UPDATE urls SET long_url = $1
		WHERE short_code = $2 AND user_id = $3
		RETURNING id, user_id, short_code, long_url, expires_at, created_at`

	var u URL
	err := s.db.QueryRow(ctx, query, longURL, shortCode, userID).Scan(
		&u.ID, &u.UserID, &u.ShortCode, &u.LongURL, &u.ExpiresAt, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return URL{}, apperr.ErrNotFound
		}
		return URL{}, fmt.Errorf("failed to update url: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) RecordClick(ctx context.Context, urlID, ipHash, userAgent string) error {
	query := `INSERT INTO clicks (url_id, ip_hash, user_agent) VALUES ($1, $2, $3)`

	_, err := s.db.Exec(ctx, query, urlID, ipHash, userAgent)
	if err != nil {
		return fmt.Errorf("failed to record click: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetClicksByCode(ctx context.Context, shortCode, userID string) ([]Click, error) {
	query := `
		SELECT c.id, c.url_id, c.ip_hash, c.user_agent, c.clicked_at
		FROM clicks c
		JOIN urls u ON u.id = c.url_id
		WHERE u.short_code = $1 AND u.user_id = $2
		ORDER BY c.clicked_at DESC`

	rows, err := s.db.Query(ctx, query, shortCode, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks: %w", err)
	}
	defer rows.Close()

	var clicks []Click
	for rows.Next() {
		var c Click
		if err := rows.Scan(&c.ID, &c.URLID, &c.IPHash, &c.UserAgent, &c.ClickedAt); err != nil {
			return nil, fmt.Errorf("failed to scan click: %w", err)
		}
		clicks = append(clicks, c)
	}
	return clicks, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Exec runs a raw SQL statement — used for migrations
func (s *PostgresStore) Exec(ctx context.Context, sql string) error {
	_, err := s.db.Exec(ctx, sql)
	return err
}

// RunMigrations runs all SQL files in the migrations directory in order
func (s *PostgresStore) RunMigrations(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}
		if err := s.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}
