package repository

import (
	"context"

	"go-simple-chat/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, channel *domain.Channel) error
	GetByID(ctx context.Context, id string) (*domain.Channel, error)
	GetForUser(ctx context.Context, userID string) ([]*domain.Channel, error)
	GetDirectChannel(ctx context.Context, user1, user2 string) (*domain.Channel, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *domain.Message) error
	GetByChannel(ctx context.Context, channelID string, limit int, beforeID string) ([]*domain.Message, error)
}

type OfflineMessageRepository interface {
	Create(ctx context.Context, msg *domain.OfflineMessage) error
	GetForUser(ctx context.Context, userID string) ([]*domain.OfflineMessage, error)
	DeleteForUser(ctx context.Context, userID string) error
}
