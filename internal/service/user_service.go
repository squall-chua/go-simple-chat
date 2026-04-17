package service

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"fmt"
	"go-simple-chat/internal/model"
	internalcrypto "go-simple-chat/internal/crypto"
	"go-simple-chat/internal/repository"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserService struct {
	userRepo   repository.UserRepository
	ca         *internalcrypto.CA
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
	user := &model.User{
		ID:        bson.NewObjectID(),
		Username:  username,
		PublicKey: publicKey,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Issue client certificate using the provided public key
	var pub any
	if len(publicKey) > 0 {
		var err error
		pub, err = x509.ParsePKIXPublicKey(publicKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid public key: %w", err)
		}
	}

	certPEM, keyPEM, err := s.ca.IssueUserCert(user.ID.Hex(), pub, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to issue cert: %w", err)
	}

	return user, certPEM, keyPEM, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	return s.userRepo.GetByID(ctx, oid)
}

func (s *UserService) GetChallenge(ctx context.Context, username string) (string, string, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", "", fmt.Errorf("user not found")
	}

	nonce := uuid.New().String()
	if err := s.challengeRepo.Store(ctx, user.ID.Hex(), nonce, 2*time.Minute); err != nil {
		return "", "", fmt.Errorf("failed to store challenge: %w", err)
	}

	return user.ID.Hex(), nonce, nil
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
	pub, err := x509.ParsePKIXPublicKey(user.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stored public key: %w", err)
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
func (s *UserService) GetUserByID(ctx context.Context, id bson.ObjectID) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}
