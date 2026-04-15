package presence

import (
	"context"
	"sync"
	"time"

	"go-simple-chat/internal/broker"
)

type PresenceService struct {
	mu          sync.RWMutex
	onlineUsers map[string]time.Time
	broker      broker.Broker
}

func NewPresenceService(broker broker.Broker) *PresenceService {
	return &PresenceService{
		onlineUsers: make(map[string]time.Time),
		broker:      broker,
	}
}

func (s *PresenceService) SetOnline(ctx context.Context, userID string) error {
	s.mu.Lock()
	s.onlineUsers[userID] = time.Now()
	s.mu.Unlock()

	// Notify others via broker
	// We'll use a special channel ID for presence or broadcast to all user's channels
	// For MVP, we'll just publish a generic presence event to a global topic if needed,
	// but usually users subscribe to presence of their contacts.
	
	// Implementation detail: For now, we just track it locally.
	// In Task 8, we'll integrate this with the streaming handlers.
	return nil
}

func (s *PresenceService) SetOffline(ctx context.Context, userID string) error {
	s.mu.Lock()
	delete(s.onlineUsers, userID)
	s.mu.Unlock()
	return nil
}

func (s *PresenceService) IsOnline(ctx context.Context, userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.onlineUsers[userID]
	return ok
}
