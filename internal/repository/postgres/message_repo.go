package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"go-simple-chat/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(ctx context.Context, msg *model.Message) error {
	if msg.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate uuid: %w", err)
		}
		msg.ID = id.String()
	}

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	mediasJSON, err := json.Marshal(msg.Medias)
	if err != nil {
		return fmt.Errorf("failed to marshal medias: %w", err)
	}

	query := `
		INSERT INTO messages (id, channel_id, sender_id, sender_username, content, medias, thread_id, parent_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = r.pool.Exec(ctx, query,
		msg.ID,
		msg.ChannelID,
		msg.SenderID,
		msg.SenderUsername,
		msg.Content,
		mediasJSON,
		msg.ThreadID,
		msg.ParentID,
		msg.CreatedAt,
	)
	return err
}

func (r *MessageRepository) GetByChannel(ctx context.Context, channelID string, limit int, beforeID string) ([]*model.Message, error) {
	var query string
	var args []any

	if beforeID != "" {
		// Time-ordered UUIDs allow us to use ID comparison for pagination
		query = `SELECT id, channel_id, sender_id, sender_username, content, medias, thread_id, parent_id, created_at 
		         FROM messages 
		         WHERE channel_id = $1 AND id < $2
		         ORDER BY id DESC 
		         LIMIT $3`
		args = append(args, channelID, beforeID, limit)
	} else {
		query = `SELECT id, channel_id, sender_id, sender_username, content, medias, thread_id, parent_id, created_at 
		         FROM messages 
		         WHERE channel_id = $1 
		         ORDER BY id DESC 
		         LIMIT $2`
		args = append(args, channelID, limit)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.Message
	for rows.Next() {
		var m model.Message
		var mediasJSON []byte
		err := rows.Scan(
			&m.ID,
			&m.ChannelID,
			&m.SenderID,
			&m.SenderUsername,
			&m.Content,
			&mediasJSON,
			&m.ThreadID,
			&m.ParentID,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(mediasJSON) > 0 {
			_ = json.Unmarshal(mediasJSON, &m.Medias)
		}
		results = append(results, &m)
	}

	// Flip back to chronological order (services expect older -> newer)
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, nil
}

func (r *MessageRepository) CountAfter(ctx context.Context, channelID, lastReadID string) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE channel_id = $1 AND id > $2`
	var count int64
	err := r.pool.QueryRow(ctx, query, channelID, lastReadID).Scan(&count)
	return count, err
}
