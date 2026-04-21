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

	"go.mongodb.org/mongo-driver/v2/bson"
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
	_, err := h.getUserIDFromContext(ctx)
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

func (h *ChatHandler) getUserIDFromContext(ctx context.Context) (bson.ObjectID, error) {
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

	// Case A: Prefer Session Token if present (used by WebSocket Bridge and gRPC-Gateway)
	if sessionToken != "" && h.sessionService != nil {
		userID, err := h.sessionService.ValidateToken(ctx, sessionToken)
		if err == nil {
			oid, err := bson.ObjectIDFromHex(userID)
			if err == nil {
				return oid, nil
			}
		}
		// If token was provided but invalid, we could return error, but let's check cert as fallback
	}

	// Case B: Fallback to Certificate Identity
	if clientCert != nil {
		cn := clientCert.Subject.CommonName
		oid, err := bson.ObjectIDFromHex(cn)
		if err != nil {
			// If it's not a valid OID (e.g. server cert), just fail auth if no session token was valid
			return bson.NilObjectID, status.Error(codes.Unauthenticated, "authentication required: no valid certificate or session token found")
		}

		// Verify user exists
		user, err := h.userService.GetUser(ctx, oid.Hex())
		if err != nil {
			return bson.NilObjectID, status.Error(codes.Unauthenticated, "user not found or inactive")
		}

		// Public key pinning
		certPubKey, err := x509.MarshalPKIXPublicKey(clientCert.PublicKey)
		if err == nil && !bytes.Equal(certPubKey, user.PublicKey) {
			return bson.NilObjectID, status.Error(codes.Unauthenticated, "certificate public key mismatch")
		}
		return oid, nil
	}

	return bson.NilObjectID, status.Error(codes.Unauthenticated, "authentication required: no valid certificate or session token found")
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
	creatorOID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var participants []bson.ObjectID
	for _, p := range req.Participants {
		oid, err := bson.ObjectIDFromHex(p)
		if err != nil {
			return nil, fmt.Errorf("invalid participant id %s: %w", p, err)
		}
		participants = append(participants, oid)
	}
	for _, uname := range req.ParticipantUsernames {
		u, err := h.userService.GetUserByUsername(ctx, uname)
		if err == nil {
			participants = append(participants, u.ID)
		}
	}

	isGroup := req.Type == chatv1.CreateChannelRequest_TYPE_GROUP
	ch, err := h.chatService.CreateChannel(ctx, creatorOID, participants, req.Name, isGroup)
	if err != nil {
		return nil, err
	}

	return &chatv1.CreateChannelResponse{ChannelId: ch.ID.Hex()}, nil
}

func (h *ChatHandler) ListChannels(ctx context.Context, req *chatv1.ListChannelsRequest) (*chatv1.ListChannelsResponse, error) {
	uOID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	channels, err := h.chatService.GetUserChannels(ctx, uOID.Hex())
	if err != nil {
		return nil, err
	}

	userCache := make(map[string]string)
	var results []*chatv1.ChannelInfo
	for _, ch := range channels {
		var participants []string
		var usernames []string
		for _, p := range ch.Participants {
			pID := p.Hex()
			participants = append(participants, pID)

			if name, ok := userCache[pID]; ok {
				usernames = append(usernames, name)
			} else {
				user, err := h.userService.GetUserByID(ctx, p)
				if err == nil {
					usernames = append(usernames, user.Username)
					userCache[pID] = user.Username
				} else {
					usernames = append(usernames, "unknown")
				}
			}
		}

		results = append(results, &chatv1.ChannelInfo{
			Id:                   ch.ID.Hex(),
			Type:                 string(ch.Type),
			Participants:         participants,
			ParticipantUsernames: usernames,
			Name:                 ch.Name,
			LastReadId:           ch.LastReadID.Hex(),
			LastMessageId:        ch.LastMessageID.Hex(),
			UnreadCount:          h.chatService.GetUnreadCount(ctx, ch.ID, ch.LastReadID),
		})
	}

	return &chatv1.ListChannelsResponse{Channels: results}, nil
}

func (h *ChatHandler) MarkAsRead(ctx context.Context, req *chatv1.MarkAsReadRequest) (*emptypb.Empty, error) {
	uOID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = h.chatService.MarkAsRead(ctx, uOID.Hex(), req.ChannelId, req.MessageId)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) GetMessages(ctx context.Context, req *chatv1.GetMessagesRequest) (*chatv1.GetMessagesResponse, error) {
	uOID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	msgs, err := h.chatService.GetMessages(ctx, uOID.Hex(), req.ChannelId, int(req.Limit), req.BeforeId)
	if err != nil {
		return nil, err
	}

	// Hydrate usernames for older messages that might not have them persisted
	userCache := make(map[string]string)
	var results []*chatv1.MessageReceived
	for _, m := range msgs {
		if m.SenderUsername == "" {
			if cached, ok := userCache[m.SenderID.Hex()]; ok {
				m.SenderUsername = cached
			} else {
				user, err := h.userService.GetUserByID(ctx, m.SenderID)
				if err == nil {
					m.SenderUsername = user.Username
					userCache[m.SenderID.Hex()] = user.Username
				}
			}
		}
		results = append(results, toProtoMsg(m))
	}

	return &chatv1.GetMessagesResponse{Messages: results}, nil
}

func (h *ChatHandler) AddParticipant(ctx context.Context, req *chatv1.AddParticipantRequest) (*chatv1.AddParticipantResponse, error) {
	uOID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userIDs := req.UserIds
	for _, uname := range req.Usernames {
		user, err := h.userService.GetUserByUsername(ctx, uname)
		if err == nil {
			userIDs = append(userIDs, user.ID.Hex())
		}
	}

	err = h.chatService.AddParticipants(ctx, uOID.Hex(), req.ChannelId, userIDs)
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
		UserId:      user.ID.Hex(),
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
	channelOID, err := bson.ObjectIDFromHex(req.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("invalid channel id: %w", err)
	}

	senderOID, err := h.getUserIDFromContext(ctx)
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

	sender, err := h.userService.GetUserByID(ctx, senderOID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sender info: %w", err)
	}

	msg := &model.Message{
		ChannelID:      channelOID,
		SenderID:       senderOID,
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
	return &chatv1.SendMessageResponse{MessageId: msg.ID.Hex()}, nil
}

func (h *ChatHandler) BidiStreamChat(stream chatv1.ChatService_BidiStreamChatServer) error {
	ctx := stream.Context()
	userID, err := h.getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	// 1. Set user as online
	metrics.OnlineUsers.Inc()
	defer metrics.OnlineUsers.Dec()

	_ = h.presenceService.SetOnline(ctx, userID.Hex())
	defer h.presenceService.SetOffline(context.Background(), userID.Hex())

	// 2. Subscription Management
	channels, _ := h.chatService.GetUserChannels(ctx, userID.Hex())

	// 3. State to protect stream.Send and deduplicate presence
	var stateMu sync.Mutex
	lastPresence := make(map[string]bool)

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
		h.subscribeToChannel(subCtx, ch.ID.Hex(), userID.Hex(), safeSend, &stateMu, lastPresence)
	}

	// 4b. Personal signal topic for dynamic updates (New channels, etc.)
	go func() {
		_ = h.chatService.SubscribeToSystemSignals(subCtx, userID.Hex(), func(ctx context.Context, sig *model.SystemSignal) error {
			switch sig.Type {
			case model.SignalNewChannel:
				h.subscribeToChannel(subCtx, sig.ChannelID.Hex(), userID.Hex(), safeSend, &stateMu, lastPresence)

				// Notify the user about the new channel
				_ = safeSend(&chatv1.StreamMessageResponse{
					Payload: &chatv1.StreamMessageResponse_ChannelJoined{
						ChannelJoined: &chatv1.ChannelJoined{
							ChannelId: sig.ChannelID.Hex(),
						},
					},
				})
			case model.SignalRosterUpdate:
				// Someone was added to an existing channel
				_ = safeSend(&chatv1.StreamMessageResponse{
					Payload: &chatv1.StreamMessageResponse_ParticipantAdded{
						ParticipantAdded: &chatv1.ParticipantAdded{
							ChannelId: sig.ChannelID.Hex(),
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
				channelOID, err := bson.ObjectIDFromHex(payload.SendMessage.ChannelId)
				if err != nil {
					continue
				}

				var medias []model.Media
				for _, m := range payload.SendMessage.Medias {
					medias = append(medias, model.Media{
						Type: m.Type,
						URL:  m.Url,
						Name: m.Name,
					})
				}

				msg := &model.Message{
					ChannelID: channelOID,
					SenderID:  userID,
					Content:   payload.SendMessage.Content,
					Medias:    medias,
					ThreadID:  payload.SendMessage.ThreadId,
					ParentID:  payload.SendMessage.ParentId,
				}
				_ = h.chatService.SendMessage(ctx, msg)
			case *chatv1.StreamMessageRequest_Heartbeat:
				_ = h.presenceService.SetOnline(ctx, userID.Hex())
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
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
		MessageId:      m.ID.Hex(),
		ChannelId:      m.ChannelID.Hex(),
		SenderId:       m.SenderID.Hex(),
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
	id := req.UserId
	if id == "" && req.Username != "" {
		user, err := h.userService.GetUserByUsername(ctx, req.Username)
		if err != nil {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		id = user.ID.Hex()
	}

	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id or username is required")
	}

	online, lastSeen, err := h.presenceService.GetPresence(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get presence: %v", err)
	}

	return &chatv1.GetPresenceResponse{
		Online:   online,
		LastSeen: timestamppb.New(lastSeen),
	}, nil
}
