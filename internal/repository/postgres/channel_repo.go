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

type ChannelRepository struct {
	pool *pgxpool.Pool
}

func NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{pool: pool}
}

func (r *ChannelRepository) Create(ctx context.Context, ch *model.Channel) error {
	if ch.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate uuid: %w", err)
		}
		ch.ID = id.String()
	}

	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = time.Now()
	}
	if ch.UpdatedAt.IsZero() {
		ch.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO channels (id, type, name, participants, last_message_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		ch.ID,
		ch.Type,
		ch.Name,
		ch.Participants,
		ch.LastMessageID,
		ch.CreatedAt,
		ch.UpdatedAt,
	)
	return err
}

func (r *ChannelRepository) GetByID(ctx context.Context, id string) (*model.Channel, error) {
	query := `SELECT id, type, name, participants, last_message_id, created_at, updated_at FROM channels WHERE id = $1`
	var ch model.Channel
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ch.ID,
		&ch.Type,
		&ch.Name,
		&ch.Participants,
		&ch.LastMessageID,
		&ch.CreatedAt,
		&ch.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *ChannelRepository) GetForUser(ctx context.Context, userID string) ([]*model.Channel, error) {
	// Use ANY operator for GIN index compatibility on participants array
	query := `SELECT id, type, name, participants, last_message_id, created_at, updated_at 
	          FROM channels 
	          WHERE $1 = ANY(participants)
	          ORDER BY updated_at DESC`
	
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.Channel
	for rows.Next() {
		var ch model.Channel
		err := rows.Scan(
			&ch.ID,
			&ch.Type,
			&ch.Name,
			&ch.Participants,
			&ch.LastMessageID,
			&ch.CreatedAt,
			&ch.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &ch)
	}
	return results, nil
}

func (r *ChannelRepository) GetDirectChannel(ctx context.Context, userA, userB string) (*model.Channel, error) {
	query := `SELECT id, type, name, participants, last_message_id, created_at, updated_at 
	          FROM channels 
	          WHERE type = 'direct' 
	          AND participants @> $1::text[] 
	          AND array_length(participants, 1) = 2`
	
	participants := []string{userA, userB}
	var ch model.Channel
	err := r.pool.QueryRow(ctx, query, participants).Scan(
		&ch.ID,
		&ch.Type,
		&ch.Name,
		&ch.Participants,
		&ch.LastMessageID,
		&ch.CreatedAt,
		&ch.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *ChannelRepository) UpdateLastMessageID(ctx context.Context, channelID, mid string) error {
	query := `UPDATE channels SET last_message_id = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, mid, time.Now(), channelID)
	return err
}

func (r *ChannelRepository) AddParticipants(ctx context.Context, channelID string, userIDs []string) error {
	// Use array_cat or array_append with unique check
	query := `UPDATE channels SET participants = ARRAY(SELECT DISTINCT unnest(participants || $1::text[])), updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, userIDs, time.Now(), channelID)
	return err
}
