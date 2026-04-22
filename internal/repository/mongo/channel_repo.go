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

	wrapped := gmqb.Wrap[model.Channel](col)

	// Create index for participants lookup
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(gmqb.SortRule(f("Participants"), 1))))
	if err != nil {
		return nil, fmt.Errorf("failed to create channel indexes: %w", err)
	}

	return &ChannelRepo{col: wrapped}, nil
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
	ifaces := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		ifaces[i] = id
	}

	_, err := r.col.UpdateOne(ctx,
		gmqb.Eq(f("ID"), channelID),
		gmqb.NewUpdate().AddToSetEach(f("Participants"), ifaces...),
	)
	return err
}

func (r *ChannelRepo) UpdateLastMessageID(ctx context.Context, channelID bson.ObjectID, messageID bson.ObjectID) error {
	f := gmqb.Field[model.Channel]
	_, err := r.col.UpdateOne(ctx,
		gmqb.Eq(f("ID"), channelID),
		gmqb.NewUpdate().Set(f("LastMessageID"), messageID),
	)
	return err
}
