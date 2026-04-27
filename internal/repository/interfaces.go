package repository

import (
	"context"
	"time"

	"go-simple-chat/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error
}

type ChannelRepository interface {
	Create(ctx context.Context, channel *model.Channel) error
	GetByID(ctx context.Context, id string) (*model.Channel, error)
	GetForUser(ctx context.Context, userID string) ([]*model.Channel, error)
	GetDirectChannel(ctx context.Context, user1, user2 string) (*model.Channel, error)
	AddParticipants(ctx context.Context, channelID string, userIDs []string) error
	UpdateLastMessageID(ctx context.Context, channelID string, messageID string) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *model.Message) error
	GetByChannel(ctx context.Context, channelID string, limit int, beforeID string) ([]*model.Message, error)
	CountAfter(ctx context.Context, channelID string, afterID string) (int64, error)
}

type ReadStateRepository interface {
	Upsert(ctx context.Context, userID, channelID, lastRead string) (bool, error)
	GetForUser(ctx context.Context, userID string) (map[string]string, error)
}

type ChallengeRepository interface {
	Store(ctx context.Context, userID, nonce string, ttl time.Duration) error
	GetAndDelete(ctx context.Context, userID string) (string, error)
}

type SessionRepository interface {
	Store(ctx context.Context, key, value string, ttl time.Duration, certExpiresAt time.Time) error
	Get(ctx context.Context, key string) (string, time.Time, time.Time, error)
	Delete(ctx context.Context, token string) error
}
