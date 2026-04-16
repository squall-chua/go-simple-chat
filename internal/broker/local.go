package broker

import (
	"context"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

type LocalBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]MessageHandler
	logger      *zap.Logger
}

func NewLocalBroker(logger *zap.Logger) *LocalBroker {
	return &LocalBroker{
		subscribers: make(map[string][]MessageHandler),
		logger:      logger,
	}
}

func (b *LocalBroker) Publish(ctx context.Context, topic string, data any) error {
	b.mu.RLock()
	handlers, ok := b.subscribers[topic]
	b.mu.RUnlock()

	if !ok {
		return nil
	}

	// For local consistency with Redis, we marshal to []byte
	payload, err := msgpack.Marshal(data)
	if err != nil {
		return err
	}

	for _, handler := range handlers {
		go func(h MessageHandler) {
			_ = h(context.Background(), payload)
		}(handler)
	}

	return nil
}

func (b *LocalBroker) Subscribe(ctx context.Context, channelID string, handler MessageHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[channelID] = append(b.subscribers[channelID], handler)
	return nil
}

func (b *LocalBroker) Close() error {
	return nil
}
