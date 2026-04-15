package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ChannelRepo struct {
	col *mongo.Collection
}

func NewChannelRepo(ctx context.Context, db *mongo.Database) (*ChannelRepo, error) {
	col := db.Collection("channels")

	// Create index for participants lookup
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "participants", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create channel indexes: %w", err)
	}

	return &ChannelRepo{col: col}, nil
}

func (r *ChannelRepo) Create(ctx context.Context, channel *domain.Channel) error {
	_, err := r.col.InsertOne(ctx, channel)
	return err
}

func (r *ChannelRepo) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	var ch domain.Channel
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&ch)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *ChannelRepo) GetForUser(ctx context.Context, userID string) ([]*domain.Channel, error) {
	cursor, err := r.col.Find(ctx, bson.M{"participants": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var channels []*domain.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *ChannelRepo) GetDirectChannel(ctx context.Context, user1, user2 string) (*domain.Channel, error) {
	var ch domain.Channel
	filter := bson.M{
		"type": domain.ChannelDirect,
		"participants": bson.M{
			"$all": []string{user1, user2},
		},
	}
	err := r.col.FindOne(ctx, filter).Decode(&ch)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}
