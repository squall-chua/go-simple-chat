package broker

import (
	"context"
	"fmt"
	"sync"

	"github.com/squall-chua/gmqb"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

// PubSubEvent is the document structure stored in the capped collection
type PubSubEvent struct {
	Topic string `bson:"topic"`
	Data  []byte `bson:"data"`
}

type MongoBroker struct {
	ps          *gmqb.TailablePubSub[PubSubEvent]
	mu          sync.RWMutex
	subscribers map[string][]MessageHandler
	logger      *zap.Logger
	cancel      context.CancelFunc
}

func NewMongoBroker(db *mongo.Database, logger *zap.Logger) (*MongoBroker, error) {
	// Initialize TailablePubSub with a shared collection "pubsub_events"
	// Capped at 100MB, which is plenty for transient pubsub messages
	ps, err := gmqb.NewTailablePubSub[PubSubEvent](db, "pubsub_events", gmqb.CappedOpts{
		SizeBytes: 100 * 1024 * 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init tailable pubsub: %w", err)
	}

	b := &MongoBroker{
		ps:          ps,
		subscribers: make(map[string][]MessageHandler),
		logger:      logger,
	}

	// Start global dispatcher
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	
	ch, psCancel := b.ps.Subscribe(ctx)
	
	go func() {
		defer psCancel()
		for event := range ch {
			b.dispatch(event)
		}
	}()

	return b, nil
}

func (b *MongoBroker) Publish(ctx context.Context, topic string, data any) error {
	payload, err := msgpack.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return b.ps.Publish(ctx, PubSubEvent{
		Topic: topic,
		Data:  payload,
	})
}

func (b *MongoBroker) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], handler)
	return nil
}

func (b *MongoBroker) dispatch(event PubSubEvent) {
	b.mu.RLock()
	handlers, ok := b.subscribers[event.Topic]
	b.mu.RUnlock()

	if !ok {
		return
	}

	for _, handler := range handlers {
		go func(h MessageHandler) {
			if err := h(context.Background(), event.Data); err != nil {
				b.logger.Error("handler error in mongo subscribe", zap.Error(err), zap.String("topic", event.Topic))
			}
		}(handler)
	}
}

func (b *MongoBroker) Close() error {
	if b.cancel != nil {
		b.cancel()
	}
	return nil
}
