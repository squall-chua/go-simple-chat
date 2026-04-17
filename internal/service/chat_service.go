package service

import (
	"context"
	"fmt"
	"time"

	"go-simple-chat/internal/broker"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/repository"

	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	uOID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	chOID, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return err
	}

	var mOID bson.ObjectID
	if messageID == "" {
		ch, err := s.chRepo.GetByID(ctx, chOID)
		if err != nil || ch == nil {
			return fmt.Errorf("channel not found for marking read: %w", err)
		}
		mOID = ch.LastMessageID
	} else {
		mOID, err = bson.ObjectIDFromHex(messageID)
		if err != nil {
			return err
		}
	}

	if mOID.IsZero() {
		return nil // Nothing to mark as read
	}

	// 1. Persist read state
	updated, err := s.readStateRepo.Upsert(ctx, uOID, chOID, mOID)
	if err != nil {
		return err
	}

	// 2. Signal user's other devices only if the mark actually moved
	if updated {
		signal := &model.SystemSignal{
			Type:      model.SignalReadUpdate,
			ChannelID: chOID,
			MessageID: mOID,
		}
		_ = s.broker.Publish(ctx, "signal:"+userID, signal)
	}

	return nil
}

func (s *ChatService) SendMessage(ctx context.Context, msg *model.Message) error {
	msg.CreatedAt = time.Now()
	if msg.ID.IsZero() {
		msg.ID = bson.NewObjectID()
	}

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
	// Broker uses string as routing key
	if err := s.broker.Publish(ctx, "chat:"+msg.ChannelID.Hex(), msg); err != nil {
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
	uOID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	channels, err := s.chRepo.GetForUser(ctx, uOID)
	if err != nil {
		return nil, err
	}

	readStates, _ := s.readStateRepo.GetForUser(ctx, uOID)

	for _, ch := range channels {
		if lastRead, ok := readStates[ch.ID]; ok {
			ch.LastReadID = lastRead
		}
	}

	return channels, nil
}

func (s *ChatService) CreateChannel(ctx context.Context, creatorID bson.ObjectID, participants []bson.ObjectID, name string, isGroup bool) (*model.Channel, error) {
	// Deduplicate participants and ensure creator is included
	pMap := map[bson.ObjectID]bool{creatorID: true}
	for _, p := range participants {
		pMap[p] = true
	}

	var finalParticipants []bson.ObjectID
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
		ID:           bson.NewObjectID(),
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
		_, _ = s.readStateRepo.Upsert(ctx, pID, ch.ID, bson.NilObjectID)
	}

	// Signal participants to update their dynamic subscriptions
	for _, pID := range finalParticipants {
		// Signal topic: signal:USER_ID
		signal := &model.SystemSignal{
			Type:      model.SignalNewChannel,
			ChannelID: ch.ID,
		}
		_ = s.broker.Publish(ctx, "signal:"+pID.Hex(), signal)
	}

	return ch, nil
}

func (s *ChatService) GetMessages(ctx context.Context, userID string, channelID string, limit int, beforeID string) ([]*model.Message, error) {
	uOID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	chOID, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return nil, err
	}

	// 1. Authorization: Verify participant
	channel, err := s.chRepo.GetByID(ctx, chOID)
	if err != nil {
		return nil, err
	}

	isParticipant := false
	for _, p := range channel.Participants {
		if p == uOID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, fmt.Errorf("user not in channel")
	}

	// 2. Query history
	var bOID bson.ObjectID
	if beforeID != "" {
		bOID, _ = bson.ObjectIDFromHex(beforeID)
	}

	if limit <= 0 {
		limit = 50
	}

	return s.msgRepo.GetByChannel(ctx, chOID, limit, bOID)
}

func (s *ChatService) AddParticipants(ctx context.Context, operatorID, channelID string, targetUserIDs []string) error {
	oOID, _ := bson.ObjectIDFromHex(operatorID)
	chOID, _ := bson.ObjectIDFromHex(channelID)

	var tOIDs []bson.ObjectID
	for _, id := range targetUserIDs {
		oid, err := bson.ObjectIDFromHex(id)
		if err == nil {
			tOIDs = append(tOIDs, oid)
		}
	}

	// 1. Verify channel
	channel, err := s.chRepo.GetByID(ctx, chOID)
	if err != nil {
		return err
	}
	if channel.Type != model.ChannelGroup {
		return fmt.Errorf("cannot add participants to non-group channel")
	}

	// 2. Verify operator
	existing := make(map[bson.ObjectID]bool)
	isParticipant := false
	for _, p := range channel.Participants {
		existing[p] = true
		if p == oOID {
			isParticipant = true
		}
	}
	if !isParticipant {
		return fmt.Errorf("permission denied")
	}

	// 3. Filter only new participants
	var filteredOIDs []bson.ObjectID
	var filteredIDs []string
	for i, tOID := range tOIDs {
		if !existing[tOID] {
			filteredOIDs = append(filteredOIDs, tOID)
			filteredIDs = append(filteredIDs, targetUserIDs[i])
		}
	}

	if len(filteredOIDs) == 0 {
		return nil // All users already in channel
	}

	// 4. Update DB
	if err := s.chRepo.AddParticipants(ctx, chOID, filteredOIDs); err != nil {
		return err
	}

	// Initialize read state for new participants to current tip
	channel, _ = s.chRepo.GetByID(ctx, chOID)
	for _, tOID := range filteredOIDs {
		_, _ = s.readStateRepo.Upsert(ctx, tOID, chOID, channel.LastMessageID)
	}

	// 5. Signal only newly added users
	for _, tID := range filteredIDs {
		signal := &model.SystemSignal{
			Type:      model.SignalNewChannel,
			ChannelID: chOID,
		}
		_ = s.broker.Publish(ctx, "signal:"+tID, signal)
	}

	// 6. Notify existing participants of roster update
	for _, pID := range channel.Participants {
		signal := &model.SystemSignal{
			Type:      model.SignalRosterUpdate,
			ChannelID: chOID,
		}
		_ = s.broker.Publish(ctx, "signal:"+pID.Hex(), signal)
	}

	return nil
}

func (s *ChatService) GetUnreadCount(ctx context.Context, channelID, lastReadID bson.ObjectID) int32 {
	if lastReadID.IsZero() {
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
