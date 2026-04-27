package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReadStateRepository struct {
	pool *pgxpool.Pool
}

func NewReadStateRepository(pool *pgxpool.Pool) *ReadStateRepository {
	return &ReadStateRepository{pool: pool}
}

func (r *ReadStateRepository) Upsert(ctx context.Context, userID, channelID, lastMessageID string) (bool, error) {
	// Check if update is needed (only move forward)
	var currentMid string
	queryCheck := `SELECT last_message_id FROM read_states WHERE user_id = $1 AND channel_id = $2`
	err := r.pool.QueryRow(ctx, queryCheck, userID, channelID).Scan(&currentMid)
	
	if err == nil {
		// Time-ordered UUIDs allow simple string comparison
		if lastMessageID <= currentMid {
			return false, nil
		}
	} else if err != pgx.ErrNoRows {
		return false, err
	}

	query := `
		INSERT INTO read_states (user_id, channel_id, last_message_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, channel_id) 
		DO UPDATE SET last_message_id = EXCLUDED.last_message_id
	`
	_, err = r.pool.Exec(ctx, query, userID, channelID, lastMessageID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *ReadStateRepository) GetForUser(ctx context.Context, userID string) (map[string]string, error) {
	query := `SELECT channel_id, last_message_id FROM read_states WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var cid, mid string
		if err := rows.Scan(&cid, &mid); err != nil {
			return nil, err
		}
		results[cid] = mid
	}
	return results, nil
}
