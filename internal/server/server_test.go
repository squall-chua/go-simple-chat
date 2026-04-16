package server_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Minimal mock repo for integration test
type mockUserRepo struct{ repository.UserRepository }

func (m *mockUserRepo) Create(ctx context.Context, u *model.User) error { return nil }

type mockMsgRepo struct{ repository.MessageRepository }

func (m *mockMsgRepo) Create(ctx context.Context, msg *model.Message) error { return nil }

type mockChRepo struct{ repository.ChannelRepository }

func (m *mockChRepo) GetByID(ctx context.Context, id bson.ObjectID) (*model.Channel, error) {
	return &model.Channel{ID: id, Participants: []bson.ObjectID{bson.NewObjectID()}}, nil
}

type mockReadStateRepo struct{ repository.ReadStateRepository }

func (m *mockReadStateRepo) Upsert(ctx context.Context, u, c, l bson.ObjectID) (bool, error) {
	return true, nil
}
func (m *mockReadStateRepo) GetForUser(ctx context.Context, u bson.ObjectID) (map[bson.ObjectID]bson.ObjectID, error) {
	return nil, nil
}

type mockChallengeRepo struct{ repository.ChallengeRepository }

func (m *mockChallengeRepo) Store(ctx context.Context, userID, nonce string, ttl time.Duration) error {
	return nil
}
func (m *mockChallengeRepo) GetAndDelete(ctx context.Context, userID string) (string, error) {
	return "", nil
}


func TestServerIntegration(t *testing.T) {
	// 1. Setup Infra
	conf := &config.Config{Port: "8081", CertDir: t.TempDir()}
	
	ca, err := crypto.NewCA(conf.CertDir)
	require.NoError(t, err)

	testBroker := broker.NewLocalBroker(zap.NewNop())
	presenceSvc := service.NewPresenceService(&mockChRepo{}, testBroker)
	
	userService := service.NewUserService(&mockUserRepo{}, &mockChallengeRepo{}, ca) 
	chatService := service.NewChatService(&mockMsgRepo{}, &mockChRepo{}, &mockReadStateRepo{}, testBroker)

	// Setup gRPC Server
	grpcServer := grpc.NewServer()
	chatv1.RegisterChatServiceServer(grpcServer, chatgrpc.NewChatHandler(userService, chatService, presenceSvc))

	srv := server.NewServer(conf.Port, grpcServer)

	// 2. Start Server
	caPath := filepath.Join(conf.CertDir, "ca.crt")
	caKeyPath := filepath.Join(conf.CertDir, "ca.key")
	tlsConfig, err := crypto.NewServerTLSConfig(caPath, caPath, caKeyPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Start(ctx, tlsConfig)
	defer srv.Stop()

	time.Sleep(300 * time.Millisecond) // Wait for start

	// 3. Client Setup (mTLS)
	certPEM, keyPEM, err := ca.IssueUserCert("test_user", []string{"localhost", "127.0.0.1"})
	require.NoError(t, err)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM(ca.GetCACert())

	clientTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      cp,
		ServerName:   "localhost",
		InsecureSkipVerify: true, // For localhost tests
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
		ChannelId: bson.NewObjectID().Hex(),
		Content:   "hello",
	})
	
	require.NoError(t, err)
	assert.NotEmpty(t, resp.MessageId)
}
