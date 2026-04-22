package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-simple-chat/internal/repository"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type sessionDoc struct {
	Token         string    `bson:"_id"`
	UserID        string    `bson:"user_id"`
	ExpiresAt     time.Time `bson:"expires_at"`      // Session TTL (TTL Index)
	CertExpiresAt time.Time `bson:"cert_expires_at"` // Identity identity
}

type SessionRepo struct {
	col *gmqb.Collection[sessionDoc]
}

func NewSessionRepo(ctx context.Context, db *mongo.Database) (repository.SessionRepository, error) {
	col := db.Collection("sessions")
	wrapped := gmqb.Wrap[sessionDoc](col)

	// 1. Create TTL index on ExpiresAt
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(gmqb.SortRule("expires_at", 1))).TTL(0))
	if err != nil {
		return nil, fmt.Errorf("failed to create session indexes: %w", err)
	}

	return &SessionRepo{col: wrapped}, nil
}

func (r *SessionRepo) Store(ctx context.Context, token, userID string, ttl time.Duration, certExpiresAt time.Time) error {
	doc := sessionDoc{
		Token:         token,
		UserID:        userID,
		ExpiresAt:     time.Now().Add(ttl),
		CertExpiresAt: certExpiresAt,
	}

	_, err := r.col.ReplaceOne(ctx, gmqb.Eq("_id", token), &doc, gmqb.WithUpsertReplace(true))
	return err
}

func (r *SessionRepo) Get(ctx context.Context, token string) (string, time.Time, time.Time, error) {
	f := gmqb.Field[sessionDoc]
	doc, err := r.col.FindOne(ctx, gmqb.Eq(f("Token"), token))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", time.Time{}, time.Time{}, errors.New("session not found")
		}
		return "", time.Time{}, time.Time{}, err
	}

	// Verification against manual expiration
	if time.Now().After(doc.ExpiresAt) {
		_ = r.Delete(ctx, token)
		return "", time.Time{}, time.Time{}, errors.New("session expired")
	}

	return doc.UserID, doc.ExpiresAt, doc.CertExpiresAt, nil
}

func (r *SessionRepo) Delete(ctx context.Context, token string) error {
	f := gmqb.Field[sessionDoc]
	_, err := r.col.DeleteOne(ctx, gmqb.Eq(f("Token"), token))
	return err
}
