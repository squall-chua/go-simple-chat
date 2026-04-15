package broker

import (
	"context"
	"fmt"

	"go-simple-chat/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

type RedisBroker struct {
	client *redis.Client
}

func NewRedisBroker(addr string) *RedisBroker {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisBroker{client: client}
}

func (b *RedisBroker) Publish(ctx context.Context, channelID string, msg *domain.Message) error {
	data, err := msgpack.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return b.client.Publish(ctx, "chat:"+channelID, data).Err()
}

func (b *RedisBroker) Subscribe(ctx context.Context, channelID string, handler MessageHandler) error {
	pubsub := b.client.Subscribe(ctx, "chat:"+channelID)

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for msg := range ch {
			var domainMsg domain.Message
			if err := msgpack.Unmarshal([]byte(msg.Payload), &domainMsg); err != nil {
				// Log error (should ideally pass a logger to the broker)
				continue
			}
			_ = handler(context.Background(), &domainMsg)
		}
	}()

	return nil
}

func (b *RedisBroker) Close() error {
	return b.client.Close()
}
