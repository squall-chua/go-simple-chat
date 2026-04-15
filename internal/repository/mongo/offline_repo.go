package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type OfflineMessageRepo struct {
	col *mongo.Collection
}

func NewOfflineMessageRepo(ctx context.Context, db *mongo.Database) (*OfflineMessageRepo, error) {
	col := db.Collection("offline_messages")

	// Create user_id index
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create offline message user index: %w", err)
	}

	// Create TTL index on expires_at (30 days default set at creation)
	_, err = col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0), // Expire at the specific time in the field
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create offline message TTL index: %w", err)
	}

	return &OfflineMessageRepo{col: col}, nil
}

func (r *OfflineMessageRepo) Create(ctx context.Context, msg *domain.OfflineMessage) error {
	_, err := r.col.InsertOne(ctx, msg)
	return err
}

func (r *OfflineMessageRepo) GetForUser(ctx context.Context, userID string) ([]*domain.OfflineMessage, error) {
	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var msgs []*domain.OfflineMessage
	if err := cursor.All(ctx, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *OfflineMessageRepo) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}
