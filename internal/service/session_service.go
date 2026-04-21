package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"go-simple-chat/internal/model"
	"go-simple-chat/internal/repository"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SessionService struct {
	sessionRepo repository.SessionRepository
	userRepo    repository.UserRepository
	caCert      *x509.Certificate
}

func NewSessionService(sessionRepo repository.SessionRepository, userRepo repository.UserRepository, caCertPEM []byte) (*SessionService, error) {
	block, _ := pem.Decode(caCertPEM)
	if block == nil {
		return nil, errors.New("failed to decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return &SessionService{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		caCert:      caCert,
	}, nil
}

func (s *SessionService) CreateChallenge(ctx context.Context) (string, error) {
	nonce := uuid.New().String()
	// Reuse existing challenge repository logic if possible, or use sessionRepo if it supports TTL blobs
	// For simplicity, we use a prefixed key in sessionRepo
	if err := s.sessionRepo.Store(ctx, "challenge:"+nonce, "pending", 5*time.Minute); err != nil {
		return "", err
	}
	return nonce, nil
}

func (s *SessionService) IssueToken(ctx context.Context, certPEM []byte, nonce string, signatureHex string) (string, string, string, error) {
	// 0. Verify Nonce
	status, err := s.sessionRepo.Get(ctx, "challenge:"+nonce)
	if err != nil || status != "pending" {
		return "", "", "", errors.New("invalid or expired challenge")
	}
	_ = s.sessionRepo.Delete(ctx, "challenge:"+nonce)

	// 1. Parse and verify certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", "", "", errors.New("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(s.caCert)
	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return "", "", "", fmt.Errorf("certificate verification failed: %w", err)
	}

	// 2. Extract UserID (CommonName)
	userID := cert.Subject.CommonName
	if userID == "" {
		return "", "", "", errors.New("certificate missing CommonName (UserID)")
	}

	// 3. Optional: Verify user exists
	var user *model.User
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		// If not hex, maybe it's the username (for some older/legacy certs if any)
		user, err = s.userRepo.GetByUsername(ctx, userID)
	} else {
		user, err = s.userRepo.GetByID(ctx, oid)
	}

	if err != nil {
		return "", "", "", fmt.Errorf("user not found: %w", err)
	}

	// 4. Verify public key matches
	// Certificate public key must match the one we have on file for this user
	certPubKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal certificate public key: %w", err)
	}

	if !bytes.Equal(certPubKey, user.PublicKey) {
		return "", "", "", errors.New("certificate public key does not match user's registered public key")
	}

	// 5. Verify Signature (Proof of Possession)
	sig, err := base64.StdEncoding.DecodeString(signatureHex)
	if err != nil {
		return "", "", "", errors.New("invalid signature format")
	}

	// Verify the signature against the nonce using the public key from the certificate
	msg := []byte(nonce)
	pub, ok := cert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return "", "", "", errors.New("unsupported public key type; only Ed25519 is accepted")
	}

	if !ed25519.Verify(pub, msg, sig) {
		return "", "", "", errors.New("signature verification failed")
	}

	username := user.Username

	// 4. Generate token
	token := uuid.New().String()
	ttl := 24 * time.Hour

	// 5. Store session
	if err := s.sessionRepo.Store(ctx, token, user.ID.Hex(), ttl); err != nil {
		return "", "", "", fmt.Errorf("failed to store session: %w", err)
	}

	return token, user.ID.Hex(), username, nil
}

func (s *SessionService) ValidateToken(ctx context.Context, token string) (string, error) {
	return s.sessionRepo.Get(ctx, token)
}
