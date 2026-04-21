package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-simple-chat/internal/repository"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

	// 1. Create TTL index on ExpiresAt
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session indexes: %w", err)
	}

	return &SessionRepo{col: gmqb.Wrap[sessionDoc](col)}, nil
}

func (r *SessionRepo) Store(ctx context.Context, token, userID string, ttl time.Duration, certExpiresAt time.Time) error {
	doc := sessionDoc{
		Token:         token,
		UserID:        userID,
		ExpiresAt:     time.Now().Add(ttl),
		CertExpiresAt: certExpiresAt,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := r.col.Unwrap().ReplaceOne(ctx, bson.M{"_id": token}, doc, opts)
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
