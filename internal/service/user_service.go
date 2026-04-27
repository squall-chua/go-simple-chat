package service

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"fmt"
	internalcrypto "go-simple-chat/internal/crypto"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/repository"
	"time"

	"github.com/google/uuid"
)

type UserService struct {
	userRepo      repository.UserRepository
	ca            *internalcrypto.CA
	challengeRepo repository.ChallengeRepository
}

func NewUserService(userRepo repository.UserRepository, challengeRepo repository.ChallengeRepository, ca *internalcrypto.CA) *UserService {
	return &UserService{
		userRepo:      userRepo,
		challengeRepo: challengeRepo,
		ca:            ca,
	}
}

func (s *UserService) Register(ctx context.Context, username string, publicKey []byte) (*model.User, []byte, []byte, error) {
	// Parse the provided public key (could be raw Ed25519 or PKIX)
	var pub any
	var err error
	if len(publicKey) == 32 {
		// Treat as raw Ed25519 public key (common for browser clients)
		pub = ed25519.PublicKey(publicKey)
	} else if len(publicKey) > 0 {
		pub, err = x509.ParsePKIXPublicKey(publicKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid public key: %w", err)
		}
	} else {
		return nil, nil, nil, fmt.Errorf("public key is required")
	}

	// Normalize to PKIX for consistent storage and cross-client compatibility (E2E)
	normalizedKey, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to normalize public key: %w", err)
	}

	user := &model.User{
		Username:  username,
		PublicKey: normalizedKey, // Always store PKIX
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Issue client certificate using the parsed public key
	certPEM, keyPEM, err := s.ca.IssueUserCert(user.ID, pub, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to issue cert: %w", err)
	}

	return user, certPEM, keyPEM, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetChallenge(ctx context.Context, username string) (string, string, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", "", fmt.Errorf("user not found")
	}

	nonce := uuid.New().String()
	if err := s.challengeRepo.Store(ctx, user.ID, nonce, 2*time.Minute); err != nil {
		return "", "", fmt.Errorf("failed to store challenge: %w", err)
	}

	return user.ID, nonce, nil
}

func (s *UserService) RenewCertificate(ctx context.Context, userID string, signature []byte) ([]byte, error) {
	nonce, err := s.challengeRepo.GetAndDelete(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve challenge: %w", err)
	}
	if nonce == "" {
		return nil, fmt.Errorf("no active challenge for user")
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Verify signature
	var pub any
	if len(user.PublicKey) == 32 {
		pub = ed25519.PublicKey(user.PublicKey)
	} else {
		var err error
		pub, err = x509.ParsePKIXPublicKey(user.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stored public key: %w", err)
		}
	}

	challengeData := []byte("RENEW_CERT:" + nonce)

	switch pk := pub.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(pk, challengeData, signature) {
			return nil, fmt.Errorf("signature verification failed")
		}
	default:
		return nil, fmt.Errorf("unsupported key type for renewal")
	}

	// Issue fresh certificate using the pinned public key
	certPEM, _, err := s.ca.IssueUserCert(userID, pub, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to issue cert: %w", err)
	}

	return certPEM, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}
