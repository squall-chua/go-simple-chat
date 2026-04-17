package service

import (
	"context"
	"sync"
	"time"

	"go-simple-chat/internal/broker"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/repository"

	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PresenceService struct {
	mu          sync.RWMutex
	onlineUsers map[string]time.Time
	chRepo      repository.ChannelRepository
	userRepo    repository.UserRepository
	broker      broker.Broker
}

func NewPresenceService(chRepo repository.ChannelRepository, userRepo repository.UserRepository, broker broker.Broker) *PresenceService {
	return &PresenceService{
		onlineUsers: make(map[string]time.Time),
		chRepo:      chRepo,
		userRepo:    userRepo,
		broker:      broker,
	}
}

func (s *PresenceService) SetOnline(ctx context.Context, userID string) error {
	s.mu.Lock()
	_, alreadyOnline := s.onlineUsers[userID]
	s.onlineUsers[userID] = time.Now()
	s.mu.Unlock()

	if alreadyOnline {
		return nil
	}

	return s.publishPresenceChange(ctx, userID, true)
}

func (s *PresenceService) SetOffline(ctx context.Context, userID string) error {
	s.mu.Lock()
	_, wasOnline := s.onlineUsers[userID]
	delete(s.onlineUsers, userID)
	s.mu.Unlock()

	if !wasOnline {
		return nil
	}

	// Persist last seen
	oid, _ := bson.ObjectIDFromHex(userID)
	_ = s.userRepo.UpdateLastSeen(ctx, oid, time.Now())

	return s.publishPresenceChange(ctx, userID, false)
}

func (s *PresenceService) GetPresence(ctx context.Context, userID string) (bool, time.Time, error) {
	s.mu.RLock()
	lastActive, online := s.onlineUsers[userID]
	s.mu.RUnlock()

	if online {
		return true, lastActive, nil
	}

	// For offline users, get last seen from DB
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, time.Time{}, err
	}
	user, err := s.userRepo.GetByID(ctx, oid)
	if err != nil {
		return false, time.Time{}, err
	}

	return false, user.LastSeen, nil
}

func (s *PresenceService) publishPresenceChange(ctx context.Context, userID string, online bool) error {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	channels, err := s.chRepo.GetForUser(ctx, oid)
	if err != nil {
		return err
	}

	event := &model.PresenceEvent{
		UserID: userID,
		Online: online,
	}

	for _, ch := range channels {
		topic := "presence:" + ch.ID.Hex()
		_ = s.broker.Publish(ctx, topic, event)
	}

	return nil
}

func (s *PresenceService) IsOnline(ctx context.Context, userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.onlineUsers[userID]
	return ok
}

func (s *PresenceService) SubscribeToChannelPresence(ctx context.Context, topic string, handler func(context.Context, *model.PresenceEvent) error) error {
	return s.broker.Subscribe(ctx, topic, func(ctx context.Context, data []byte) error {
		var event model.PresenceEvent
		if err := msgpack.Unmarshal(data, &event); err != nil {
			return nil
		}
		return handler(ctx, &event)
	})
}
