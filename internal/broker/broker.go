package broker

import (
	"context"
)

type MessageHandler func(ctx context.Context, data []byte) error

type Broker interface {
	Publish(ctx context.Context, topic string, data any) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	Close() error
}
