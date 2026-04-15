package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MessageRepo struct {
	col *mongo.Collection
}

func NewMessageRepo(ctx context.Context, db *mongo.Database) (*MessageRepo, error) {
	col := db.Collection("messages")

	// Create compound index for channel ordering
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "channel_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create message indexes: %w", err)
	}

	return &MessageRepo{col: col}, nil
}

func (r *MessageRepo) Create(ctx context.Context, msg *domain.Message) error {
	_, err := r.col.InsertOne(ctx, msg)
	return err
}

func (r *MessageRepo) GetByChannel(ctx context.Context, channelID string, limit int, beforeID string) ([]*domain.Message, error) {
	filter := bson.M{"channel_id": channelID}
	if beforeID != "" {
		filter["_id"] = bson.M{"$lt": beforeID} // Using ID for pagination (assuming lexicographical ID ordering or sort)
	}

	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*domain.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}
