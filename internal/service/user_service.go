package service

import (
	"context"
	"fmt"
	"time"

	"go-simple-chat/internal/crypto"
	"go-simple-chat/internal/domain"
	"go-simple-chat/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
	ca       *crypto.CA
}

func NewUserService(userRepo repository.UserRepository, ca *crypto.CA) *UserService {
	return &UserService{
		userRepo: userRepo,
		ca:       ca,
	}
}

func (s *UserService) Register(ctx context.Context, username string, publicKey []byte) (*domain.User, []byte, []byte, error) {
	// Simple ID generation for MVP (can use UUID)
	userID := fmt.Sprintf("u_%d", time.Now().UnixNano())

	user := &domain.User{
		ID:        userID,
		Username:  username,
		PublicKey: publicKey,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Issue client certificate
	certPEM, keyPEM, err := s.ca.IssueUserCert(userID, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to issue cert: %w", err)
	}

	return user, certPEM, keyPEM, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
