package broker

import (
	"context"

	"go-simple-chat/internal/domain"
)

type MessageHandler func(ctx context.Context, msg *domain.Message) error

type Broker interface {
	Publish(ctx context.Context, channelID string, msg *domain.Message) error
	Subscribe(ctx context.Context, channelID string, handler MessageHandler) error
	Close() error
}
