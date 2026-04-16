package mongo

import (
	"context"
	"fmt"
	"time"

	"go-simple-chat/internal/model"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ChallengeRepo struct {
	col *gmqb.Collection[model.AuthChallenge]
}

func NewChallengeRepo(ctx context.Context, db *mongo.Database) (*ChallengeRepo, error) {
	col := db.Collection("challenges")
	f := gmqb.Field[model.AuthChallenge]

	// Create TTL index on CreatedAt
	// Note: We use 0 as the expireAfterSeconds and calculate the expiration in the document insertion
	// or we can set a fixed TTL here. Let's set a fixed TTL of 5 minutes as a fallback.
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: f("CreatedAt"), Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(300), // 5 minutes
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge indexes: %w", err)
	}

	return &ChallengeRepo{col: gmqb.Wrap[model.AuthChallenge](col)}, nil
}

func (r *ChallengeRepo) Store(ctx context.Context, userID, nonce string, ttl time.Duration) error {
	challenge := &model.AuthChallenge{
		UserID:    userID,
		Nonce:     nonce,
		CreatedAt: time.Now(),
	}

	opts := options.Replace().SetUpsert(true)
	_, err := r.col.Unwrap().ReplaceOne(ctx, bson.M{"_id": userID}, challenge, opts)
	return err
}

func (r *ChallengeRepo) GetAndDelete(ctx context.Context, userID string) (string, error) {
	res := r.col.Unwrap().FindOneAndDelete(ctx, bson.M{"_id": userID})
	if res.Err() != nil {
		if res.Err() == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", res.Err()
	}

	var challenge model.AuthChallenge
	if err := res.Decode(&challenge); err != nil {
		return "", err
	}

	return challenge.Nonce, nil
}
