package broker

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

// PostgresBroker implements Broker using PostgreSQL LISTEN/NOTIFY
type PostgresBroker struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	
	// Subscription management
	mu            sync.RWMutex
	subscriptions map[string][]MessageHandler
	topicToChan   map[string]string
	chanToTopic   map[string]string
	
	// Listener connection
	listenConn *pgx.Conn
	connString string
	
	stopChan chan struct{}
}

func NewPostgresBroker(connString string, pool *pgxpool.Pool, logger *zap.Logger) (*PostgresBroker, error) {
	b := &PostgresBroker{
		pool:          pool,
		logger:        logger,
		subscriptions: make(map[string][]MessageHandler),
		topicToChan:   make(map[string]string),
		chanToTopic:   make(map[string]string),
		connString:    connString,
		stopChan:      make(chan struct{}),
	}

	// Connect the listener
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect listener: %w", err)
	}
	b.listenConn = conn

	go b.listenLoop()

	return b, nil
}

func (b *PostgresBroker) Publish(ctx context.Context, topic string, msg interface{}) error {
	data, err := msgpack.Marshal(msg)
	if err != nil {
		return err
	}

	payload := base64.StdEncoding.EncodeToString(data)
	if len(payload) > 7999 {
		return fmt.Errorf("payload too large for Postgres NOTIFY (%d bytes)", len(payload))
	}

	channelName := b.getChannelName(topic)
	query := fmt.Sprintf("SELECT pg_notify(%s, $1)", pgx.Identifier{channelName}.Sanitize())
	_, err = b.pool.Exec(ctx, query, payload)
	return err
}

func (b *PostgresBroker) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	channelName := b.getChannelName(topic)
	
	// If first subscription for this topic, tell PG to LISTEN
	if _, ok := b.subscriptions[topic]; !ok {
		query := fmt.Sprintf("LISTEN %s", pgx.Identifier{channelName}.Sanitize())
		_, err := b.listenConn.Exec(context.Background(), query)
		if err != nil {
			return fmt.Errorf("failed to LISTEN on %s: %w", topic, err)
		}
	}

	b.subscriptions[topic] = append(b.subscriptions[topic], handler)
	return nil
}

func (b *PostgresBroker) getChannelName(topic string) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if name, ok := b.topicToChan[topic]; ok {
		return name
	}

	// Create a safe channel name
	// Postgres identifiers are limited in length and character set
	// We'll use a prefix + hash or just replace characters
	name := "ps_" + strings.ReplaceAll(strings.ReplaceAll(topic, ":", "_"), "-", "_")
	if len(name) > 63 {
		name = name[:63]
	}
	
	b.topicToChan[topic] = name
	b.chanToTopic[name] = topic
	return name
}

func (b *PostgresBroker) listenLoop() {
	for {
		select {
		case <-b.stopChan:
			return
		default:
			// WaitForNotification will block until a notification arrives or context is cancelled
			notification, err := b.listenConn.WaitForNotification(context.Background())
			if err != nil {
				select {
				case <-b.stopChan:
					return
				default:
					b.logger.Error("Postgres listener error", zap.Error(err))
					time.Sleep(1 * time.Second)
					if b.listenConn == nil || b.listenConn.IsClosed() {
						b.reconnect()
					}
					continue
				}
			}

			b.handleNotification(notification)
		}
	}
}

func (b *PostgresBroker) handleNotification(n *pgconn.Notification) {
	b.mu.RLock()
	topic, ok := b.chanToTopic[n.Channel]
	handlers := b.subscriptions[topic]
	b.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return
	}

	data, err := base64.StdEncoding.DecodeString(n.Payload)
	if err != nil {
		b.logger.Warn("Failed to decode postgres notification payload", zap.String("channel", n.Channel), zap.Error(err))
		return
	}

	for _, h := range handlers {
		go func(handler MessageHandler) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := handler(ctx, data); err != nil {
				b.logger.Error("Postgres sub handler error", zap.String("topic", topic), zap.Error(err))
			}
		}(h)
	}
}

func (b *PostgresBroker) reconnect() {
	b.logger.Info("Attempting to reconnect Postgres listener")
	for {
		select {
		case <-b.stopChan:
			return
		default:
			conn, err := pgx.Connect(context.Background(), b.connString)
			if err == nil {
				b.mu.Lock()
				b.listenConn = conn
				// Re-subscribe to all active topics
				for topic := range b.subscriptions {
					channelName := b.topicToChan[topic]
					query := fmt.Sprintf("LISTEN %s", pgx.Identifier{channelName}.Sanitize())
					_, _ = b.listenConn.Exec(context.Background(), query)
				}
				b.mu.Unlock()
				b.logger.Info("Postgres listener reconnected")
				return
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (b *PostgresBroker) Close() error {
	close(b.stopChan)
	if b.listenConn != nil {
		return b.listenConn.Close(context.Background())
	}
	return nil
}
