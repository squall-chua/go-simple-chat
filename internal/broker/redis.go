package broker

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

type RedisBroker struct {
	client redis.UniversalClient
	logger *zap.Logger
}

func NewRedisBroker(addr string, logger *zap.Logger) *RedisBroker {
	addrs := strings.Split(addr, ",")
	var client redis.UniversalClient
	
	if len(addrs) > 1 {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: addrs,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr: addrs[0],
		})
	}
	return &RedisBroker{
		client: client,
		logger: logger,
	}
}

func (b *RedisBroker) Publish(ctx context.Context, topic string, data any) error {
	payload, err := msgpack.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return b.client.Publish(ctx, topic, payload).Err()
}

func (b *RedisBroker) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	pubsub := b.client.Subscribe(ctx, topic)

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for msg := range ch {
			if err := handler(context.Background(), []byte(msg.Payload)); err != nil {
				b.logger.Error("handler error in redis subscribe", zap.Error(err))
			}
		}
	}()

	return nil
}

func (b *RedisBroker) Close() error {
	return b.client.Close()
}
