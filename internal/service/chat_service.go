package service

import (
	"context"
	"fmt"
	"time"

	"go-simple-chat/internal/broker"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/repository"

	"github.com/vmihailenco/msgpack/v5"
)

type ChatService struct {
	msgRepo       repository.MessageRepository
	chRepo        repository.ChannelRepository
	readStateRepo repository.ReadStateRepository
	userService   *UserService
	broker        broker.Broker
}

func NewChatService(
	msgRepo repository.MessageRepository,
	chRepo repository.ChannelRepository,
	readStateRepo repository.ReadStateRepository,
	userService *UserService,
	broker broker.Broker,
) *ChatService {
	return &ChatService{
		msgRepo:       msgRepo,
		chRepo:        chRepo,
		readStateRepo: readStateRepo,
		userService:   userService,
		broker:        broker,
	}
}

func (s *ChatService) MarkAsRead(ctx context.Context, userID, channelID, messageID string) error {
	var mID string
	if messageID == "" {
		ch, err := s.chRepo.GetByID(ctx, channelID)
		if err != nil || ch == nil {
			return fmt.Errorf("channel not found for marking read: %w", err)
		}
		mID = ch.LastMessageID
	} else {
		mID = messageID
	}

	if mID == "" {
		return nil // Nothing to mark as read
	}

	// 1. Persist read state
	updated, err := s.readStateRepo.Upsert(ctx, userID, channelID, mID)
	if err != nil {
		return err
	}

	// 2. Signal user's other devices only if the mark actually moved
	if updated {
		signal := &model.SystemSignal{
			Type:      model.SignalReadUpdate,
			ChannelID: channelID,
			MessageID: mID,
		}
		_ = s.broker.Publish(ctx, "signal:"+userID, signal)
	}

	return nil
}

func (s *ChatService) SendMessage(ctx context.Context, msg *model.Message) error {
	msg.CreatedAt = time.Now()

	// 0. Hydrate sender username
	if msg.SenderUsername == "" {
		if sender, err := s.userService.GetUserByID(ctx, msg.SenderID); err == nil {
			msg.SenderUsername = sender.Username
		}
	}

	// 1. Persist message
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("failed to persist message: %w", err)
	}

	// 2. Update Channel last message pointer
	_ = s.chRepo.UpdateLastMessageID(ctx, msg.ChannelID, msg.ID)

	// 3. Publish to broker (for online users)
	if err := s.broker.Publish(ctx, "chat:"+msg.ChannelID, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (s *ChatService) SubscribeToChannel(ctx context.Context, channelID string, handler func(context.Context, *model.Message) error) error {
	return s.broker.Subscribe(ctx, "chat:"+channelID, func(ctx context.Context, data []byte) error {
		var msg model.Message
		if err := msgpack.Unmarshal(data, &msg); err != nil {
			return nil
		}
		return handler(ctx, &msg)
	})
}

func (s *ChatService) GetUserChannels(ctx context.Context, userID string) ([]*model.Channel, error) {
	channels, err := s.chRepo.GetForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	readStates, _ := s.readStateRepo.GetForUser(ctx, userID)

	for _, ch := range channels {
		if lastRead, ok := readStates[ch.ID]; ok {
			ch.LastReadID = lastRead
		}
	}

	return channels, nil
}

func (s *ChatService) CreateChannel(ctx context.Context, creatorID string, participants []string, name string, isGroup bool) (*model.Channel, error) {
	// Deduplicate participants and ensure creator is included
	pMap := map[string]bool{creatorID: true}
	for _, p := range participants {
		pMap[p] = true
	}

	var finalParticipants []string
	for id := range pMap {
		finalParticipants = append(finalParticipants, id)
	}

	// Idempotency for Direct channels
	if !isGroup && len(finalParticipants) == 2 {
		existing, _ := s.chRepo.GetDirectChannel(ctx, finalParticipants[0], finalParticipants[1])
		if existing != nil {
			return existing, nil
		}
	}

	ch := &model.Channel{
		Type:         model.ChannelDirect,
		Participants: finalParticipants,
		Name:         name,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if isGroup {
		ch.Type = model.ChannelGroup
	}

	if err := s.chRepo.Create(ctx, ch); err != nil {
		return nil, err
	}

	// Initialize read state for all participants to empty (zero unreads)
	for _, pID := range finalParticipants {
		_, _ = s.readStateRepo.Upsert(ctx, pID, ch.ID, "")
	}

	// Signal participants to update their dynamic subscriptions
	for _, pID := range finalParticipants {
		signal := &model.SystemSignal{
			Type:      model.SignalNewChannel,
			ChannelID: ch.ID,
		}
		_ = s.broker.Publish(ctx, "signal:"+pID, signal)
	}

	return ch, nil
}

func (s *ChatService) GetMessages(ctx context.Context, userID string, channelID string, limit int, beforeID string) ([]*model.Message, error) {
	// 1. Authorization: Verify participant
	channel, err := s.chRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	isParticipant := false
	for _, p := range channel.Participants {
		if p == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, fmt.Errorf("user not in channel")
	}

	// 2. Query history
	if limit <= 0 {
		limit = 50
	}

	return s.msgRepo.GetByChannel(ctx, channelID, limit, beforeID)
}

func (s *ChatService) AddParticipants(ctx context.Context, operatorID, channelID string, targetUserIDs []string) error {
	// 1. Verify channel
	channel, err := s.chRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel.Type != model.ChannelGroup {
		return fmt.Errorf("cannot add participants to non-group channel")
	}

	// 2. Verify operator
	existing := make(map[string]bool)
	isParticipant := false
	for _, p := range channel.Participants {
		existing[p] = true
		if p == operatorID {
			isParticipant = true
		}
	}
	if !isParticipant {
		return fmt.Errorf("permission denied")
	}

	// 3. Filter only new participants
	var filteredIDs []string
	for _, tID := range targetUserIDs {
		if !existing[tID] {
			filteredIDs = append(filteredIDs, tID)
		}
	}

	if len(filteredIDs) == 0 {
		return nil // All users already in channel
	}

	// 4. Update DB
	if err := s.chRepo.AddParticipants(ctx, channelID, filteredIDs); err != nil {
		return err
	}

	// Initialize read state for new participants to current tip
	channel, _ = s.chRepo.GetByID(ctx, channelID)
	for _, tID := range filteredIDs {
		_, _ = s.readStateRepo.Upsert(ctx, tID, channelID, channel.LastMessageID)
	}

	// 5. Signal only newly added users
	for _, tID := range filteredIDs {
		signal := &model.SystemSignal{
			Type:      model.SignalNewChannel,
			ChannelID: channelID,
		}
		_ = s.broker.Publish(ctx, "signal:"+tID, signal)
	}

	// 6. Notify existing participants of roster update
	for _, pID := range channel.Participants {
		signal := &model.SystemSignal{
			Type:      model.SignalRosterUpdate,
			ChannelID: channelID,
		}
		_ = s.broker.Publish(ctx, "signal:"+pID, signal)
	}

	return nil
}

func (s *ChatService) GetUnreadCount(ctx context.Context, channelID, lastReadID string) int32 {
	if lastReadID == "" {
		return 0
	}
	count, _ := s.msgRepo.CountAfter(ctx, channelID, lastReadID)
	return int32(count)
}

func (s *ChatService) SubscribeToSystemSignals(ctx context.Context, userID string, handler func(context.Context, *model.SystemSignal) error) error {
	topic := "signal:" + userID
	return s.broker.Subscribe(ctx, topic, func(ctx context.Context, data []byte) error {
		var sig model.SystemSignal
		if err := msgpack.Unmarshal(data, &sig); err != nil {
			return nil
		}
		return handler(ctx, &sig)
	})
}
