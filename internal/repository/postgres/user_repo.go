package postgres

import (
	"context"
	"fmt"
	"go-simple-chat/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	if user.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate uuid: %w", err)
		}
		user.ID = id.String()
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO users (id, username, public_key, last_seen, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Username,
		user.PublicKey,
		user.LastSeen,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT id, username, public_key, last_seen, created_at, updated_at FROM users WHERE id = $1`
	var u model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.PublicKey,
		&u.LastSeen,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, public_key, last_seen, created_at, updated_at FROM users WHERE username = $1`
	var u model.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.PublicKey,
		&u.LastSeen,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error {
	query := `UPDATE users SET last_seen = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, lastSeen, time.Now(), id)
	return err
}
