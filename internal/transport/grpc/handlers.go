package grpc

import (
	"context"
	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/internal/domain"
	"go-simple-chat/internal/metrics"
	"go-simple-chat/internal/presence"
	"go-simple-chat/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatHandler struct {
	chatv1.UnimplementedChatServiceServer
	userService     *service.UserService
	chatService     *service.ChatService
	presenceService *presence.PresenceService
}

func NewChatHandler(
	userService *service.UserService,
	chatService *service.ChatService,
	presenceService *presence.PresenceService,
) *ChatHandler {
	return &ChatHandler{
		userService:     userService,
		chatService:     chatService,
		presenceService: presenceService,
	}
}

func (h *ChatHandler) HealthCheck(ctx context.Context, req *chatv1.HealthCheckRequest) (*chatv1.HealthCheckResponse, error) {
	return &chatv1.HealthCheckResponse{Status: "ok"}, nil
}

func (h *ChatHandler) Register(ctx context.Context, req *chatv1.RegisterRequest) (*chatv1.RegisterResponse, error) {
	user, cert, key, err := h.userService.Register(ctx, req.Username, req.PublicKey)
	if err != nil {
		return nil, err
	}

	return &chatv1.RegisterResponse{
		UserId:      user.ID,
		Certificate: cert,
		PrivateKey:  key,
	}, nil
}

func (h *ChatHandler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	metrics.TotalMessages.Inc()
	msg := &domain.Message{
		ChannelID: req.ChannelId,
		SenderID:  "system", // In a real app, extract from mTLS context
		Content:   req.Content,
		MediaType: req.MediaType,
		MediaURL:  req.MediaUrl,
		ThreadID:  req.ThreadId,
		ParentID:  req.ParentId,
	}

	if err := h.chatService.SendMessage(ctx, msg); err != nil {
		return nil, err
	}

	return &chatv1.SendMessageResponse{MessageId: msg.ID}, nil
}

func (h *ChatHandler) BidiStreamChat(stream chatv1.ChatService_BidiStreamChatServer) error {
	// 1. Identification (In a real app, extract from mTLS context)
	userID := "test_user" // Placeholder
	ctx := stream.Context()

	// 2. Set user as online
	metrics.OnlineUsers.Inc()
	defer metrics.OnlineUsers.Dec()

	_ = h.presenceService.SetOnline(ctx, userID)
	defer h.presenceService.SetOffline(context.Background(), userID)

	// 3. Deliver offline messages immediately
	offlineMsgs, _ := h.chatService.GetOfflineMessages(ctx, userID)
	for _, m := range offlineMsgs {
		_ = stream.Send(&chatv1.StreamMessageResponse{
			Payload: &chatv1.StreamMessageResponse_MessageReceived{
				MessageReceived: toProtoMsg(m),
			},
		})
	}

	// 4. Handle incoming messages and subscriptions
	
	// Subscriber for all user's channels (MVP: subscribe to a single global channel or specific ones)
	// For demo, we'll just subscribe to any message published.
	// In a real app, you'd iterate over user's channels.
	
	return stream.Context().Err() // Blocking for now to keep the stream open
}

func toProtoMsg(m *domain.Message) *chatv1.MessageReceived {
	return &chatv1.MessageReceived{
		MessageId: m.ID,
		ChannelId: m.ChannelID,
		SenderId:  m.SenderID,
		Content:   m.Content,
		MediaType: m.MediaType,
		MediaUrl:  m.MediaURL,
		ThreadId:  m.ThreadID,
		ParentId:  m.ParentID,
		CreatedAt: timestamppb.New(m.CreatedAt),
	}
}
