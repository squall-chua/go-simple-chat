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
	"go-simple-chat/internal/domain"
	"go-simple-chat/internal/presence"
	"go-simple-chat/internal/repository"
	"go-simple-chat/internal/server"
	"go-simple-chat/internal/service"
	chatgrpc "go-simple-chat/internal/transport/grpc"
	chatv1 "go-simple-chat/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Minimal mock repo for integration test
type mockUserRepo struct{ repository.UserRepository }
func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error { return nil }

type mockMsgRepo struct{ repository.MessageRepository }
func (m *mockMsgRepo) Create(ctx context.Context, msg *domain.Message) error { return nil }

type mockChRepo struct{ repository.ChannelRepository }
func (m *mockChRepo) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	return &domain.Channel{ID: id, Participants: []string{"test_user", "system"}}, nil
}

type mockOfflineRepo struct{ repository.OfflineMessageRepository }
func (m *mockOfflineRepo) Create(ctx context.Context, msg *domain.OfflineMessage) error { return nil }

func TestServerIntegration(t *testing.T) {
	// 1. Setup Infra
	conf := &config.Config{Port: "8081", CertDir: t.TempDir()}
	
	ca, err := crypto.NewCA(conf.CertDir)
	require.NoError(t, err)

	testBroker := broker.NewLocalBroker()
	presenceSvc := presence.NewPresenceService(testBroker)
	
	userService := service.NewUserService(&mockUserRepo{}, ca) 
	chatService := service.NewChatService(&mockMsgRepo{}, &mockChRepo{}, &mockOfflineRepo{}, testBroker)

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
		ChannelId: "test",
		Content:   "hello",
	})
	
	require.NoError(t, err)
	assert.NotEmpty(t, resp.MessageId)
}
