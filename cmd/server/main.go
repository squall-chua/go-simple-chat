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
	"go-simple-chat/internal/repository/mongo"
	"go-simple-chat/internal/server"
	"go-simple-chat/internal/service"
	chatgrpc "go-simple-chat/internal/transport/grpc"
	chathttp "go-simple-chat/internal/transport/http"
	chatv1 "go-simple-chat/api/v1"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	cfg := config.LoadConfig()
	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Repositories
	store, err := mongo.NewStore(ctx, cfg.MongoURI)
	if err != nil {
		logger.Fatal("failed to init mongo", zap.Error(err))
	}
	defer store.Close(context.Background())

	userRepo, _ := mongo.NewUserRepo(ctx, store.DB)
	chRepo, _ := mongo.NewChannelRepo(ctx, store.DB)
	msgRepo, _ := mongo.NewMessageRepo(ctx, store.DB)
	readStateRepo, _ := mongo.NewReadStateRepo(ctx, store.DB)
	challengeRepo, _ := mongo.NewChallengeRepo(ctx, store.DB)
	sessionRepo, _ := mongo.NewSessionRepo(ctx, store.DB)

	// 2. Broker
	var b broker.Broker
	if cfg.BrokerType == "redis" {
		b = broker.NewRedisBroker(cfg.RedisAddr, logger)
	} else {
		b = broker.NewLocalBroker(logger)
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
		certPEM, keyPEM, _ := ca.IssueUserCert(cfg.CertCN, nil, cfg.CertDNS)
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

	publicTLSConfig, err := crypto.NewPublicTLSConfig(
		filepath.Join(cfg.CertDir, "ca.crt"),
		serverCertPath,
		serverKeyPath,
	)
	if err != nil {
		logger.Fatal("failed to create public TLS config", zap.Error(err))
	}

	// 4. Services
	userService := service.NewUserService(userRepo, challengeRepo, ca)
	presenceService := service.NewPresenceService(chRepo, userRepo, b)
	chatService := service.NewChatService(msgRepo, chRepo, readStateRepo, userService, b)
	sessionService, _ := service.NewSessionService(sessionRepo, userRepo, ca.GetCACert())
	sessionHandler := chathttp.NewSessionHandler(sessionService)
	uploadService, _ := service.NewUploadService(cfg)

	// 5. gRPC Handler
	handler := chatgrpc.NewChatHandler(userService, chatService, presenceService, sessionService, uploadService, cfg.TrustedProxyAddrs)
	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	chatv1.RegisterChatServiceServer(grpcSrv, handler)

	// 6. Multiplexed Server
	srv := server.NewServer(cfg.Port, cfg.PublicPort, cfg.CertCN, cfg.AllowedOrigins, grpcSrv)
	srv.SetSessionHandler(sessionHandler)
	srv.SetUploadService(uploadService, sessionService)

	go func() {
		logger.Info("secure server starting", zap.String("port", cfg.Port))
		if err := srv.Start(ctx, tlsConfig); err != nil && err != context.Canceled {
			logger.Error("secure server error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("public server starting", zap.String("port", cfg.PublicPort))
		if err := srv.StartPublic(ctx, publicTLSConfig); err != nil && err != context.Canceled {
			logger.Error("public server error", zap.Error(err))
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
