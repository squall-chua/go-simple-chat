package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go-simple-chat/internal/broker"
	"go-simple-chat/internal/config"
	"go-simple-chat/internal/crypto"
	"go-simple-chat/internal/presence"
	"go-simple-chat/internal/repository/mongo"
	"go-simple-chat/internal/server"
	"go-simple-chat/internal/service"
	chatgrpc "go-simple-chat/internal/transport/grpc"
	chatv1 "go-simple-chat/api/v1"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig()
	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Repositories
	store, err := mongo.NewStore(ctx, cfg.MongoURI, "chat")
	if err != nil {
		logger.Fatal("failed to init mongo", zap.Error(err))
	}
	defer store.Close(context.Background())

	userRepo, _ := mongo.NewUserRepo(ctx, store.DB)
	chRepo, _ := mongo.NewChannelRepo(ctx, store.DB)
	msgRepo, _ := mongo.NewMessageRepo(ctx, store.DB)
	offlineRepo, _ := mongo.NewOfflineMessageRepo(ctx, store.DB)

	// 2. Broker
	var b broker.Broker
	if cfg.BrokerType == "redis" {
		b = broker.NewRedisBroker(cfg.RedisAddr)
	} else {
		b = broker.NewLocalBroker()
	}
	defer b.Close()

	// 3. Crypto/CA
	ca, err := crypto.NewCA(cfg.CertDir)
	if err != nil {
		logger.Fatal("failed to init CA", zap.Error(err))
	}

	serverCertPath := filepath.Join(cfg.CertDir, "server.crt")
	serverKeyPath := filepath.Join(cfg.CertDir, "server.key")
	if _, err := os.Stat(serverCertPath); os.IsNotExist(err) {
		certPEM, keyPEM, _ := ca.IssueUserCert("localhost", []string{"localhost", "127.0.0.1"})
		os.WriteFile(serverCertPath, certPEM, 0644)
		os.WriteFile(serverKeyPath, keyPEM, 0600)
	}

	tlsConfig, err := crypto.NewServerTLSConfig(
		filepath.Join(cfg.CertDir, "ca.crt"),
		serverCertPath,
		serverKeyPath,
	)
	if err != nil {
		logger.Fatal("failed to create TLS config", zap.Error(err))
	}

	// 4. Services
	userService := service.NewUserService(userRepo, ca)
	presenceService := presence.NewPresenceService(b)
	chatService := service.NewChatService(msgRepo, chRepo, offlineRepo, b)

	// 5. gRPC Handler
	handler := chatgrpc.NewChatHandler(userService, chatService, presenceService)
	grpcSrv := grpc.NewServer()
	chatv1.RegisterChatServiceServer(grpcSrv, handler)

	// 6. Multiplexed Server
	srv := server.NewServer(cfg.Port, grpcSrv)
	
	go func() {
		logger.Info("server starting", zap.String("port", cfg.Port))
		if err := srv.Start(ctx, tlsConfig); err != nil && err != context.Canceled {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")
	srv.Stop()
	logger.Info("shutdown complete")
}

func initLogger(level string) *zap.Logger {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = zap.InfoLevel
	}
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(l)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := config.Build()
	return logger
}
