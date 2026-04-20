package repository

import (
	"context"
	"time"

	"go-simple-chat/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id bson.ObjectID) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateLastSeen(ctx context.Context, id bson.ObjectID, lastSeen time.Time) error
}

type ChannelRepository interface {
	Create(ctx context.Context, channel *model.Channel) error
	GetByID(ctx context.Context, id bson.ObjectID) (*model.Channel, error)
	GetForUser(ctx context.Context, userID bson.ObjectID) ([]*model.Channel, error)
	GetDirectChannel(ctx context.Context, user1, user2 bson.ObjectID) (*model.Channel, error)
	AddParticipants(ctx context.Context, channelID bson.ObjectID, userIDs []bson.ObjectID) error
	UpdateLastMessageID(ctx context.Context, channelID bson.ObjectID, messageID bson.ObjectID) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *model.Message) error
	GetByChannel(ctx context.Context, channelID bson.ObjectID, limit int, beforeID bson.ObjectID) ([]*model.Message, error)
	CountAfter(ctx context.Context, channelID bson.ObjectID, afterID bson.ObjectID) (int64, error)
}

type ReadStateRepository interface {
	Upsert(ctx context.Context, userID, channelID, lastRead bson.ObjectID) (bool, error)
	GetForUser(ctx context.Context, userID bson.ObjectID) (map[bson.ObjectID]bson.ObjectID, error)
}

type ChallengeRepository interface {
	Store(ctx context.Context, userID, nonce string, ttl time.Duration) error
	GetAndDelete(ctx context.Context, userID string) (string, error)
}

type SessionRepository interface {
	Store(ctx context.Context, token, userID string, ttl time.Duration) error
	Get(ctx context.Context, token string) (string, error)
	Delete(ctx context.Context, token string) error
}

