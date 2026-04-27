package server_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-simple-chat/internal/broker"
	"go-simple-chat/internal/config"
	"go-simple-chat/internal/crypto"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/repository"
	"go-simple-chat/internal/server"
	"go-simple-chat/internal/service"
	chatgrpc "go-simple-chat/internal/transport/grpc"
	chatv1 "go-simple-chat/api/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Minimal mock repo for integration test
type mockUserRepo struct {
	repository.UserRepository
	PublicKey []byte
}

func (m *mockUserRepo) Create(ctx context.Context, u *model.User) error { 
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil 
}
func (m *mockUserRepo) UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error {
	return nil
}
func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	return &model.User{ID: id, Username: "test_user", PublicKey: m.PublicKey}, nil
}
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return &model.User{ID: uuid.New().String(), Username: username, PublicKey: m.PublicKey}, nil
}

type mockMsgRepo struct{ repository.MessageRepository }

func (m *mockMsgRepo) Create(ctx context.Context, msg *model.Message) error { 
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	return nil 
}

type mockChRepo struct {
	repository.ChannelRepository
	ParticipantID string
}

func (m *mockChRepo) GetByID(ctx context.Context, id string) (*model.Channel, error) {
	return &model.Channel{ID: id, Participants: []string{m.ParticipantID}}, nil
}
func (m *mockChRepo) UpdateLastMessageID(ctx context.Context, id, mid string) error {
	return nil
}
func (m *mockChRepo) GetForUser(ctx context.Context, id string) ([]*model.Channel, error) {
	return nil, nil
}

type mockReadStateRepo struct{ repository.ReadStateRepository }

func (m *mockReadStateRepo) Upsert(ctx context.Context, u, c, l string) (bool, error) {
	return true, nil
}
func (m *mockReadStateRepo) GetForUser(ctx context.Context, u string) (map[string]string, error) {
	return make(map[string]string), nil
}

type mockChallengeRepo struct{ repository.ChallengeRepository }

func (m *mockChallengeRepo) Store(ctx context.Context, userID, nonce string, ttl time.Duration) error {
	return nil
}
func (m *mockChallengeRepo) GetAndDelete(ctx context.Context, userID string) (string, error) {
	return "", nil
}

type mockSessionRepo struct {
	repository.SessionRepository
	sessions sync.Map
}

func (m *mockSessionRepo) Store(ctx context.Context, key, value string, ttl time.Duration, certExpiresAt time.Time) error {
	m.sessions.Store(key, value)
	return nil
}
func (m *mockSessionRepo) Get(ctx context.Context, key string) (string, time.Time, time.Time, error) {
	v, ok := m.sessions.Load(key)
	if !ok {
		return "", time.Time{}, time.Time{}, errors.New("not found")
	}
	return v.(string), time.Now().Add(time.Hour), time.Now().Add(time.Hour), nil
}
func (m *mockSessionRepo) Delete(ctx context.Context, key string) error {
	m.sessions.Delete(key)
	return nil
}

func TestServerIntegration(t *testing.T) {
	// 1. Setup Infra
	conf := &config.Config{Port: "18081", CertDir: t.TempDir()}
	
	ca, err := crypto.NewCA(conf.CertDir)
	require.NoError(t, err)

	testBroker := broker.NewLocalBroker(zap.NewNop())
	uRepo := &mockUserRepo{}
	testUserID := uuid.New().String()
	chRepo := &mockChRepo{ParticipantID: testUserID}
	presenceSvc := service.NewPresenceService(chRepo, uRepo, testBroker)
	
	userService := service.NewUserService(uRepo, &mockChallengeRepo{}, ca) 
	chatService := service.NewChatService(&mockMsgRepo{}, chRepo, &mockReadStateRepo{}, userService, testBroker)

	// Setup gRPC Server
	grpcServer := grpc.NewServer()
	
	sessionRepo := &mockSessionRepo{}
	sessionService, _ := service.NewSessionService(sessionRepo, uRepo, ca.GetCACert())
	
	chatv1.RegisterChatServiceServer(grpcServer, chatgrpc.NewChatHandler(userService, chatService, presenceSvc, sessionService, nil, []string{"127.0.0.1"}))

	srv := server.NewServer(conf.Port, "18082", "localhost", []string{"*"}, grpcServer)

	// 2. Start Server
	caPath := filepath.Join(conf.CertDir, "ca.crt")
	
	// Issue a server cert for the test
	serverCertPEM, serverKeyPEM, err := ca.IssueUserCert("localhost", nil, []string{"localhost", "127.0.0.1"})
	require.NoError(t, err)
	
	serverCertPath := filepath.Join(conf.CertDir, "server.crt")
	serverKeyPath := filepath.Join(conf.CertDir, "server.key")
	os.WriteFile(serverCertPath, serverCertPEM, 0644)
	os.WriteFile(serverKeyPath, serverKeyPEM, 0600)

	tlsConfig, err := crypto.NewServerTLSConfig(caPath, serverCertPath, serverKeyPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Start(ctx, tlsConfig)
	defer srv.Stop()

	time.Sleep(300 * time.Millisecond) // Wait for start

	// 3. Client Setup (mTLS)
	certPEM, keyPEM, err := ca.IssueUserCert(testUserID, nil, []string{"localhost", "127.0.0.1"})
	require.NoError(t, err)

	// Update mock repo with the issued public key for pinning check
	block, _ := pem.Decode(certPEM)
	c, _ := x509.ParseCertificate(block.Bytes)
	pubKey, _ := x509.MarshalPKIXPublicKey(c.PublicKey)
	uRepo.PublicKey = pubKey

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM(ca.GetCACert())

	clientTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      cp,
		ServerName:   "localhost",
		InsecureSkipVerify: true, // Localhost test environment
		NextProtos:   []string{"h2", "http/1.1"},
	}

	// Connect to gRPC-gateway port or gRPC port (they are the same)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)),
	}
	conn, err := grpc.Dial("localhost:"+conf.Port, opts...)
	require.NoError(t, err)
	defer conn.Close()

	client := chatv1.NewChatServiceClient(conn)

	// 4. Test RPC
	opCtx, opCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer opCancel()

	resp, err := client.SendMessage(opCtx, &chatv1.SendMessageRequest{
		ChannelId: uuid.New().String(),
		Content:   "hello",
	})
	
	require.NoError(t, err)
	assert.NotEmpty(t, resp.MessageId)
}
