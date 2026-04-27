package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChallengeRepository struct {
	pool *pgxpool.Pool
}

func NewChallengeRepository(pool *pgxpool.Pool) *ChallengeRepository {
	return &ChallengeRepository{pool: pool}
}

func (r *ChallengeRepository) Store(ctx context.Context, userID, nonce string, ttl time.Duration) error {
	query := `
		INSERT INTO auth_challenges (user_id, nonce, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET nonce = EXCLUDED.nonce, expires_at = EXCLUDED.expires_at
	`
	_, err := r.pool.Exec(ctx, query, userID, nonce, time.Now().Add(ttl))
	return err
}

func (r *ChallengeRepository) GetAndDelete(ctx context.Context, userID string) (string, error) {
	var nonce string
	var expiresAt time.Time
	
	query := `DELETE FROM auth_challenges WHERE user_id = $1 RETURNING nonce, expires_at`
	err := r.pool.QueryRow(ctx, query, userID).Scan(&nonce, &expiresAt)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("challenge not found")
	}
	if err != nil {
		return "", err
	}

	if time.Now().After(expiresAt) {
		return "", fmt.Errorf("challenge expired")
	}

	return nonce, nil
}
