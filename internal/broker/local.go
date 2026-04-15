package broker

import (
	"context"
	"sync"

	"go-simple-chat/internal/domain"
)

type LocalBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]MessageHandler
}

func NewLocalBroker() *LocalBroker {
	return &LocalBroker{
		subscribers: make(map[string][]MessageHandler),
	}
}

func (b *LocalBroker) Publish(ctx context.Context, channelID string, msg *domain.Message) error {
	b.mu.RLock()
	handlers, ok := b.subscribers[channelID]
	b.mu.RUnlock()

	if !ok {
		return nil
	}

	for _, handler := range handlers {
		go func(h MessageHandler) {
			_ = h(context.Background(), msg)
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
