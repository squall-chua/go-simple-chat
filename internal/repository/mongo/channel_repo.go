package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/model"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ChannelRepo struct {
	col *gmqb.Collection[model.Channel]
}

func NewChannelRepo(ctx context.Context, db *mongo.Database) (*ChannelRepo, error) {
	col := db.Collection("channels")
	f := gmqb.Field[model.Channel]

	// Create index for participants lookup
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: f("Participants"), Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create channel indexes: %w", err)
	}

	return &ChannelRepo{col: gmqb.Wrap[model.Channel](col)}, nil
}

func (r *ChannelRepo) Create(ctx context.Context, channel *model.Channel) error {
	if channel.ID.IsZero() {
		channel.ID = bson.NewObjectID()
	}
	_, err := r.col.InsertOne(ctx, channel)
	return err
}

func (r *ChannelRepo) GetByID(ctx context.Context, id bson.ObjectID) (*model.Channel, error) {
	f := gmqb.Field[model.Channel]
	return r.col.FindOne(ctx, gmqb.Eq(f("ID"), id))
}

func (r *ChannelRepo) GetForUser(ctx context.Context, userID bson.ObjectID) ([]*model.Channel, error) {
	f := gmqb.Field[model.Channel]
	channels, err := r.col.Find(ctx, gmqb.Eq(f("Participants"), userID))
	if err != nil {
		return nil, err
	}

	results := make([]*model.Channel, len(channels))
	for i := range channels {
		results[i] = &channels[i]
	}
	return results, nil
}

func (r *ChannelRepo) GetDirectChannel(ctx context.Context, user1, user2 bson.ObjectID) (*model.Channel, error) {
	f := gmqb.Field[model.Channel]
	return r.col.FindOne(ctx, gmqb.And(
		gmqb.Eq(f("Type"), model.ChannelDirect),
		gmqb.All(f("Participants"), user1, user2),
	))
}

func (r *ChannelRepo) AddParticipants(ctx context.Context, channelID bson.ObjectID, userIDs []bson.ObjectID) error {
	f := gmqb.Field[model.Channel]
	filter := bson.M{f("ID"): channelID}
	update := bson.M{
		"$addToSet": bson.M{f("Participants"): bson.M{"$each": userIDs}},
	}
	_, err := r.col.Unwrap().UpdateOne(ctx, filter, update)
	return err
}
