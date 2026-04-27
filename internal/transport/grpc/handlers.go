package grpc

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/internal/metrics"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/service"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatHandler struct {
	chatv1.UnimplementedChatServiceServer
	userService     *service.UserService
	chatService     *service.ChatService
	presenceService *service.PresenceService
	sessionService  *service.SessionService
	uploadService   *service.UploadService
	trustedProxies  []string
}

func NewChatHandler(
	userService *service.UserService,
	chatService *service.ChatService,
	presenceService *service.PresenceService,
	sessionService *service.SessionService,
	uploadService *service.UploadService,
	trustedProxies []string,
) *ChatHandler {
	return &ChatHandler{
		userService:     userService,
		chatService:     chatService,
		presenceService: presenceService,
		sessionService:  sessionService,
		uploadService:   uploadService,
		trustedProxies:  trustedProxies,
	}
}

func (h *ChatHandler) UploadFile(ctx context.Context, req *chatv1.UploadFileRequest) (*chatv1.UploadFileResponse, error) {
	if h.uploadService == nil {
		return nil, status.Error(codes.Unimplemented, "upload service not configured")
	}

	// 1. Authenticate (required for gRPC-Gateway)
	_, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Upload using existing service
	// We wrap the bytes in a reader
	reader := bytes.NewReader(req.Data)
	url, err := h.uploadService.UploadFile(ctx, req.Filename, reader, int64(len(req.Data)), req.ContentType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload: %v", err)
	}

	return &chatv1.UploadFileResponse{
		Url:  url,
		Name: req.Filename,
	}, nil
}

func (h *ChatHandler) getUserIDFromContext(ctx context.Context) (string, time.Time, time.Time, error) {
	var clientCert *x509.Certificate
	var sessionToken string

	// 1. Extract session token from metadata (Web/Bridge client)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		tokens := md.Get("x-session-token")
		if len(tokens) > 0 {
			sessionToken = tokens[0]
		}
	}

	// 2. Extract client certificate from peer info (Direct mTLS)
	if p, ok := peer.FromContext(ctx); ok {
		if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok && len(tlsInfo.State.PeerCertificates) > 0 {
			clientCert = tlsInfo.State.PeerCertificates[0]
		}

		// Support for trusted forwarders (gRPC-Gateway without direct mTLS)
		if clientCert == nil && h.isTrustedProxy(p.Addr) {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				vals := md.Get("x-internal-client-cert")
				if len(vals) > 0 {
					der, err := base64.StdEncoding.DecodeString(vals[0])
					if err == nil {
						cert, err := x509.ParseCertificate(der)
						if err == nil {
							clientCert = cert
						}
					}
				}
			}
		}
	}

	// Case A: Prefer Session Token if present
	if sessionToken != "" && h.sessionService != nil {
		userID, sessionExp, identityExp, err := h.sessionService.ValidateToken(ctx, sessionToken)
		if err == nil {
			return userID, sessionExp, identityExp, nil
		}
	}

	// Case B: Fallback to Certificate Identity
	if clientCert != nil {
		userID := clientCert.Subject.CommonName
		if userID == "" {
			return "", time.Time{}, time.Time{}, status.Error(codes.Unauthenticated, "invalid certificate identity")
		}

		// Verify user exists
		user, err := h.userService.GetUserByID(ctx, userID)
		if err != nil {
			return "", time.Time{}, time.Time{}, status.Error(codes.Unauthenticated, "user not found")
		}

		// Public key pinning
		certPubKey, err := x509.MarshalPKIXPublicKey(clientCert.PublicKey)
		if err == nil && !bytes.Equal(certPubKey, user.PublicKey) {
			return "", time.Time{}, time.Time{}, status.Error(codes.Unauthenticated, "certificate public key mismatch")
		}

		return userID, clientCert.NotAfter, clientCert.NotAfter, nil
	}

	return "", time.Time{}, time.Time{}, status.Error(codes.Unauthenticated, "authentication required")
}

func (h *ChatHandler) isTrustedProxy(addr net.Addr) bool {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		host = addr.String()
	}

	ip := net.ParseIP(host)
	for _, tp := range h.trustedProxies {
		if strings.Contains(tp, "/") {
			_, subnet, err := net.ParseCIDR(tp)
			if err == nil && subnet.Contains(ip) {
				return true
			}
		} else {
			if tp == host {
				return true
			}
		}
	}
	return false
}

func (h *ChatHandler) HealthCheck(ctx context.Context, req *chatv1.HealthCheckRequest) (*chatv1.HealthCheckResponse, error) {
	return &chatv1.HealthCheckResponse{Status: "ok"}, nil
}

func (h *ChatHandler) CreateChannel(ctx context.Context, req *chatv1.CreateChannelRequest) (*chatv1.CreateChannelResponse, error) {
	creatorID, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	participants := req.Participants
	for _, uname := range req.ParticipantUsernames {
		u, err := h.userService.GetUserByUsername(ctx, uname)
		if err == nil {
			participants = append(participants, u.ID)
		}
	}

	isGroup := req.Type == chatv1.CreateChannelRequest_TYPE_GROUP
	ch, err := h.chatService.CreateChannel(ctx, creatorID, participants, req.Name, isGroup)
	if err != nil {
		return nil, err
	}

	return &chatv1.CreateChannelResponse{ChannelId: ch.ID}, nil
}

func (h *ChatHandler) ListChannels(ctx context.Context, req *chatv1.ListChannelsRequest) (*chatv1.ListChannelsResponse, error) {
	userID, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	channels, err := h.chatService.GetUserChannels(ctx, userID)
	if err != nil {
		return nil, err
	}

	userCache := make(map[string]string)
	var results []*chatv1.ChannelInfo
	for _, ch := range channels {
		var participants []string
		var usernames []string
		for _, pID := range ch.Participants {
			participants = append(participants, pID)

			if name, ok := userCache[pID]; ok {
				usernames = append(usernames, name)
			} else {
				user, err := h.userService.GetUserByID(ctx, pID)
				if err == nil {
					usernames = append(usernames, user.Username)
					userCache[pID] = user.Username
				} else {
					usernames = append(usernames, "unknown")
				}
			}
		}

		results = append(results, &chatv1.ChannelInfo{
			Id:                   ch.ID,
			Type:                 string(ch.Type),
			Participants:         participants,
			ParticipantUsernames: usernames,
			Name:                 ch.Name,
			LastReadId:           ch.LastReadID,
			LastMessageId:        ch.LastMessageID,
			UnreadCount:          h.chatService.GetUnreadCount(ctx, ch.ID, ch.LastReadID),
		})
	}

	return &chatv1.ListChannelsResponse{Channels: results}, nil
}

func (h *ChatHandler) MarkAsRead(ctx context.Context, req *chatv1.MarkAsReadRequest) (*emptypb.Empty, error) {
	userID, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = h.chatService.MarkAsRead(ctx, userID, req.ChannelId, req.MessageId)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) GetMessages(ctx context.Context, req *chatv1.GetMessagesRequest) (*chatv1.GetMessagesResponse, error) {
	userID, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	msgs, err := h.chatService.GetMessages(ctx, userID, req.ChannelId, int(req.Limit), req.BeforeId)
	if err != nil {
		return nil, err
	}

	// Hydrate usernames for older messages that might not have them persisted
	userCache := make(map[string]string)
	var results []*chatv1.MessageReceived
	for _, m := range msgs {
		if m.SenderUsername == "" {
			if cached, ok := userCache[m.SenderID]; ok {
				m.SenderUsername = cached
			} else {
				user, err := h.userService.GetUserByID(ctx, m.SenderID)
				if err == nil {
					m.SenderUsername = user.Username
					userCache[m.SenderID] = user.Username
				}
			}
		}
		results = append(results, toProtoMsg(m))
	}

	return &chatv1.GetMessagesResponse{Messages: results}, nil
}

func (h *ChatHandler) AddParticipant(ctx context.Context, req *chatv1.AddParticipantRequest) (*chatv1.AddParticipantResponse, error) {
	userID, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userIDs := req.UserIds
	for _, uname := range req.Usernames {
		user, err := h.userService.GetUserByUsername(ctx, uname)
		if err == nil {
			userIDs = append(userIDs, user.ID)
		}
	}

	err = h.chatService.AddParticipants(ctx, userID, req.ChannelId, userIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &chatv1.AddParticipantResponse{Success: true}, nil
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

func (h *ChatHandler) GetChallenge(ctx context.Context, req *chatv1.GetChallengeRequest) (*chatv1.GetChallengeResponse, error) {
	userID, nonce, err := h.userService.GetChallenge(ctx, req.Username)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &chatv1.GetChallengeResponse{
		UserId: userID,
		Nonce:  nonce,
	}, nil
}

func (h *ChatHandler) RenewCertificate(ctx context.Context, req *chatv1.RenewCertificateRequest) (*chatv1.RenewCertificateResponse, error) {
	cert, err := h.userService.RenewCertificate(ctx, req.UserId, req.Signature)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &chatv1.RenewCertificateResponse{
		Certificate: cert,
	}, nil
}

func (h *ChatHandler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	senderID, _, _, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var medias []model.Media
	for _, m := range req.Medias {
		medias = append(medias, model.Media{
			Type: m.Type,
			URL:  m.Url,
			Name: m.Name,
		})
	}

	sender, err := h.userService.GetUserByID(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sender info: %w", err)
	}

	msg := &model.Message{
		ChannelID:      req.ChannelId,
		SenderID:       senderID,
		SenderUsername: sender.Username,
		Content:        req.Content,
		Medias:         medias,
		ThreadID:       req.ThreadId,
		ParentID:       req.ParentId,
	}

	if err := h.chatService.SendMessage(ctx, msg); err != nil {
		return nil, err
	}

	metrics.TotalMessages.Inc()
	return &chatv1.SendMessageResponse{MessageId: msg.ID}, nil
}

func (h *ChatHandler) BidiStreamChat(stream chatv1.ChatService_BidiStreamChatServer) error {
	ctx := stream.Context()
	userID, sessionExp, identityExp, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. Set user as online
	metrics.OnlineUsers.Inc()
	defer metrics.OnlineUsers.Dec()

	_ = h.presenceService.SetOnline(ctx, userID)
	defer h.presenceService.SetOffline(context.Background(), userID)

	// 2. Subscription Management
	channels, _ := h.chatService.GetUserChannels(ctx, userID)

	// 3. State to protect stream.Send and deduplicate presence
	var stateMu sync.Mutex
	lastPresence := make(map[string]bool)
	forceQuit := make(chan error, 1)

	safeSend := func(res *chatv1.StreamMessageResponse) error {
		stateMu.Lock()
		defer stateMu.Unlock()
		return stream.Send(res)
	}

	// 4. Outgoing Loop (Broker -> Stream)
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	// 4a. Initial subscriptions for existing channels
	for _, ch := range channels {
		h.subscribeToChannel(subCtx, ch.ID, userID, safeSend, &stateMu, lastPresence)
	}

	// 4b. Personal signal topic for dynamic updates (New channels, etc.)
	go func() {
		// 5. Expiration Monitoring
		// hardExpiry is the absolute first point we must drop.
		hardExpiry := sessionExp
		if !identityExp.IsZero() && identityExp.Before(hardExpiry) {
			hardExpiry = identityExp
		}

		expiryTimer := time.NewTimer(time.Until(hardExpiry))
		defer expiryTimer.Stop()

		warnThreshold := 24 * time.Hour
		warned := false

		// 6. Identity Warning (Certs only)
		if !identityExp.IsZero() && time.Until(identityExp) < warnThreshold {
			_ = safeSend(&chatv1.StreamMessageResponse{
				Payload: &chatv1.StreamMessageResponse_IdentityEvent{
					IdentityEvent: &chatv1.IdentityEvent{
						Type:      chatv1.IdentityEvent_TYPE_EXPIRING_SOON,
						ExpiresAt: timestamppb.New(identityExp),
					},
				},
			})
			warned = true
		}

		checkTicker := time.NewTicker(1 * time.Hour)
		defer checkTicker.Stop()
		if warned || identityExp.IsZero() {
			checkTicker.Stop()
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-expiryTimer.C:
				// Determine if this was a session drop or identity drop
				eventType := chatv1.IdentityEvent_TYPE_EXPIRED
				expTime := hardExpiry

				_ = safeSend(&chatv1.StreamMessageResponse{
					Payload: &chatv1.StreamMessageResponse_IdentityEvent{
						IdentityEvent: &chatv1.IdentityEvent{
							Type:      eventType,
							ExpiresAt: timestamppb.New(expTime),
						},
					},
				})
				forceQuit <- status.Error(codes.Unauthenticated, "identity expired")
				return
			case <-checkTicker.C:
				if !identityExp.IsZero() && !warned && time.Until(identityExp) < warnThreshold {
					_ = safeSend(&chatv1.StreamMessageResponse{
						Payload: &chatv1.StreamMessageResponse_IdentityEvent{
							IdentityEvent: &chatv1.IdentityEvent{
								Type:      chatv1.IdentityEvent_TYPE_EXPIRING_SOON,
								ExpiresAt: timestamppb.New(identityExp),
							},
						},
					})
					warned = true
					checkTicker.Stop()
				}
			}
		}
	}()

	// 4c. Personal signal topic for dynamic updates
	go func() {
		_ = h.chatService.SubscribeToSystemSignals(subCtx, userID, func(ctx context.Context, sig *model.SystemSignal) error {
			switch sig.Type {
			case model.SignalNewChannel:
				h.subscribeToChannel(subCtx, sig.ChannelID, userID, safeSend, &stateMu, lastPresence)

				// Notify the user about the new channel
				_ = safeSend(&chatv1.StreamMessageResponse{
					Payload: &chatv1.StreamMessageResponse_ChannelJoined{
						ChannelJoined: &chatv1.ChannelJoined{
							ChannelId: sig.ChannelID,
						},
					},
				})
			case model.SignalRosterUpdate:
				// Someone was added to an existing channel
				_ = safeSend(&chatv1.StreamMessageResponse{
					Payload: &chatv1.StreamMessageResponse_ParticipantAdded{
						ParticipantAdded: &chatv1.ParticipantAdded{
							ChannelId: sig.ChannelID,
						},
					},
				})
			}
			return nil
		})
	}()

	// 5. Incoming Loop (Stream -> Server)
	const heartbeatTimeout = 60 * time.Second
	timer := time.NewTimer(heartbeatTimeout)
	defer timer.Stop()

	recvErr := make(chan error, 1)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				recvErr <- err
				return
			}

			// Reset timeout on any valid activity from client
			timer.Reset(heartbeatTimeout)

			switch payload := req.Payload.(type) {
			case *chatv1.StreamMessageRequest_SendMessage:
				var medias []model.Media
				for _, m := range payload.SendMessage.Medias {
					medias = append(medias, model.Media{
						Type: m.Type,
						URL:  m.Url,
						Name: m.Name,
					})
				}

				msg := &model.Message{
					ChannelID: payload.SendMessage.ChannelId,
					SenderID:  userID,
					Content:   payload.SendMessage.Content,
					Medias:    medias,
					ThreadID:  payload.SendMessage.ThreadId,
					ParentID:  payload.SendMessage.ParentId,
				}
				_ = h.chatService.SendMessage(ctx, msg)
			case *chatv1.StreamMessageRequest_Heartbeat:
				_ = h.presenceService.SetOnline(ctx, userID)
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-forceQuit:
		return err
	case <-timer.C:
		return status.Error(codes.DeadlineExceeded, "heartbeat timeout")
	case err := <-recvErr:
		return err
	}
}

func toProtoMsg(m *model.Message) *chatv1.MessageReceived {
	var medias []*chatv1.Media
	for _, med := range m.Medias {
		medias = append(medias, &chatv1.Media{
			Type: med.Type,
			Url:  med.URL,
			Name: med.Name,
		})
	}

	return &chatv1.MessageReceived{
		MessageId:      m.ID,
		ChannelId:      m.ChannelID,
		SenderId:       m.SenderID,
		SenderUsername: m.SenderUsername,
		Content:        m.Content,
		Medias:         medias,
		ThreadId:       m.ThreadID,
		ParentId:       m.ParentID,
		CreatedAt:      timestamppb.New(m.CreatedAt),
	}
}

func (h *ChatHandler) subscribeToChannel(subCtx context.Context, channelID, selfID string, safeSend func(*chatv1.StreamMessageResponse) error, stateMu *sync.Mutex, lastPresence map[string]bool) {
	// 1. Chat messages
	go func() {
		_ = h.chatService.SubscribeToChannel(subCtx, channelID, func(ctx context.Context, msg *model.Message) error {
			return safeSend(&chatv1.StreamMessageResponse{
				Payload: &chatv1.StreamMessageResponse_MessageReceived{
					MessageReceived: toProtoMsg(msg),
				},
			})
		})
	}()

	// 2. Presence events
	go func() {
		pTopic := "presence:" + channelID
		_ = h.presenceService.SubscribeToChannelPresence(subCtx, pTopic, func(ctx context.Context, event *model.PresenceEvent) error {
			if event.UserID == selfID {
				return nil
			}
			stateMu.Lock()
			current, ok := lastPresence[event.UserID]
			if ok && current == event.Online {
				stateMu.Unlock()
				return nil
			}
			lastPresence[event.UserID] = event.Online
			stateMu.Unlock()

			return safeSend(&chatv1.StreamMessageResponse{
				Payload: &chatv1.StreamMessageResponse_PresenceEvent{
					PresenceEvent: &chatv1.PresenceEvent{
						UserId: event.UserID,
						Online: event.Online,
					},
				},
			})
		})
	}()
}

func (h *ChatHandler) GetPresence(ctx context.Context, req *chatv1.GetPresenceRequest) (*chatv1.GetPresenceResponse, error) {
	// 1. Resolve all unique user IDs
	uniqueIDs := make(map[string]struct{})
	for _, id := range req.UserIds {
		if id != "" {
			uniqueIDs[id] = struct{}{}
		}
	}

	for _, uname := range req.Usernames {
		if uname == "" {
			continue
		}
		user, err := h.userService.GetUserByUsername(ctx, uname)
		if err == nil {
			uniqueIDs[user.ID] = struct{}{}
		}
	}

	if len(uniqueIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one user_id or username is required")
	}

	// 2. Fetch presence for each resolved ID
	var results []*chatv1.UserPresence
	for id := range uniqueIDs {
		online, lastSeen, err := h.presenceService.GetPresence(ctx, id)
		if err != nil {
			continue // Skip errors for individual users
		}

		// Hydrate username for the response if possible
		username := ""
		user, err := h.userService.GetUserByID(ctx, id)
		if err == nil {
			username = user.Username
		}

		results = append(results, &chatv1.UserPresence{
			UserId:   id,
			Username: username,
			Online:   online,
			LastSeen: timestamppb.New(lastSeen),
		})
	}

	return &chatv1.GetPresenceResponse{
		Presences: results,
	}, nil
}
