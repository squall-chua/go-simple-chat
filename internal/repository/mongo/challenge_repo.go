package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-simple-chat/internal/model"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ChallengeRepo struct {
	col *gmqb.Collection[model.AuthChallenge]
}

func NewChallengeRepo(ctx context.Context, db *mongo.Database) (*ChallengeRepo, error) {
	col := db.Collection("challenges")
	f := gmqb.Field[model.AuthChallenge]

	wrapped := gmqb.Wrap[model.AuthChallenge](col)

	// Create TTL index on CreatedAt
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(gmqb.SortRule(f("CreatedAt"), 1))).TTL(300))
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge indexes: %w", err)
	}

	return &ChallengeRepo{col: wrapped}, nil
}

func (r *ChallengeRepo) Store(ctx context.Context, userID, nonce string, ttl time.Duration) error {
	challenge := &model.AuthChallenge{
		UserID:    userID,
		Nonce:     nonce,
		CreatedAt: time.Now(),
	}

	_, err := r.col.ReplaceOne(ctx, gmqb.Eq("_id", userID), challenge, gmqb.WithUpsertReplace(true))
	return err
}

func (r *ChallengeRepo) GetAndDelete(ctx context.Context, userID string) (string, error) {
	challenge, err := r.col.FindOneAndDelete(ctx, gmqb.Eq("_id", userID))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", nil
		}
		return "", err
	}

	return challenge.Nonce, nil
}
