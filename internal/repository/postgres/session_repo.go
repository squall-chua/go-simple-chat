package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Store(ctx context.Context, key, value string, ttl time.Duration, certExpiresAt time.Time) error {
	query := `
		INSERT INTO sessions (token, user_id, expires_at, cert_expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, expires_at = EXCLUDED.expires_at, cert_expires_at = EXCLUDED.cert_expires_at
	`
	_, err := r.pool.Exec(ctx, query, key, value, time.Now().Add(ttl), certExpiresAt)
	return err
}

func (r *SessionRepository) Get(ctx context.Context, key string) (string, time.Time, time.Time, error) {
	var userID string
	var expiresAt, certExpiresAt time.Time

	query := `SELECT user_id, expires_at, cert_expires_at FROM sessions WHERE token = $1`
	err := r.pool.QueryRow(ctx, query, key).Scan(&userID, &expiresAt, &certExpiresAt)
	if err == pgx.ErrNoRows {
		return "", time.Time{}, time.Time{}, fmt.Errorf("session not found")
	}
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}

	if time.Now().After(expiresAt) {
		return "", time.Time{}, time.Time{}, fmt.Errorf("session expired")
	}

	return userID, expiresAt, certExpiresAt, nil
}

func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, token)
	return err
}
