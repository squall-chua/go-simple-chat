package service

import (
	"context"
	"fmt"
	"time"

	"go-simple-chat/internal/broker"
	"go-simple-chat/internal/domain"
	"go-simple-chat/internal/repository"
)

type ChatService struct {
	msgRepo     repository.MessageRepository
	chRepo      repository.ChannelRepository
	offlineRepo repository.OfflineMessageRepository
	broker      broker.Broker
}

func NewChatService(
	msgRepo repository.MessageRepository,
	chRepo repository.ChannelRepository,
	offlineRepo repository.OfflineMessageRepository,
	broker broker.Broker,
) *ChatService {
	return &ChatService{
		msgRepo:     msgRepo,
		chRepo:      chRepo,
		offlineRepo: offlineRepo,
		broker:      broker,
	}
}

func (s *ChatService) SendMessage(ctx context.Context, msg *domain.Message) error {
	msg.CreatedAt = time.Now()
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("m_%d", time.Now().UnixNano())
	}

	// 1. Persist message
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("failed to persist message: %w", err)
	}

	// 2. Publish to broker (for online users)
	if err := s.broker.Publish(ctx, msg.ChannelID, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// 3. Handle offline messages
	// Get channel participants to identify potential offline recipients
	channel, err := s.chRepo.GetByID(ctx, msg.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get channel for offline delivery: %w", err)
	}

	for _, participantID := range channel.Participants {
		if participantID == msg.SenderID {
			continue
		}
		
		// In a real system, we'd check presence here. For now, we store for everyone
		// and the GetOfflineMessages call (on reconnect) will clear them.
		offlineMsg := &domain.OfflineMessage{
			ID:        fmt.Sprintf("om_%s_%s", participantID, msg.ID),
			UserID:    participantID,
			Message:   *msg,
			ExpiresAt: time.Now().AddDate(0, 0, 30), // 30 days TTL
		}
		_ = s.offlineRepo.Create(ctx, offlineMsg)
	}

	return nil
}

func (s *ChatService) SubscribeToChannel(ctx context.Context, channelID string, handler broker.MessageHandler) error {
	return s.broker.Subscribe(ctx, channelID, handler)
}

func (s *ChatService) GetOfflineMessages(ctx context.Context, userID string) ([]*domain.Message, error) {
	offlineMsgs, err := s.offlineRepo.GetForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var msgs []*domain.Message
	for _, om := range offlineMsgs {
		msgs = append(msgs, &om.Message)
	}

	// Clear once retrieved (reconnect delivery)
	_ = s.offlineRepo.DeleteForUser(ctx, userID)

	return msgs, nil
}
