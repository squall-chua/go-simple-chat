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

type ReadStateRepo struct {
	col *gmqb.Collection[model.ReadState]
}

func NewReadStateRepo(ctx context.Context, db *mongo.Database) (*ReadStateRepo, error) {
	col := db.Collection("read_states")
	f := gmqb.Field[model.ReadState]

	// Index: user_id + channel_id unique
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: f("UserID"), Value: 1},
			{Key: f("ChannelID"), Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create read_state indexes: %w", err)
	}

	return &ReadStateRepo{col: gmqb.Wrap[model.ReadState](col)}, nil
}

func (r *ReadStateRepo) Upsert(ctx context.Context, userID, channelID, lastRead bson.ObjectID) (bool, error) {
	f := gmqb.Field[model.ReadState]
	now := time.Now()

	// Use literal bson.M for filter to ensure MongoDB can extract fields for upsert
	filter := bson.M{
		f("UserID"):    userID,
		f("ChannelID"): channelID,
	}

	// Use standard $max and $set
	// Note: We don't necessarily need $setOnInsert if these fields are in the filter
	update := bson.M{
		"$max": bson.M{f("LastRead"): lastRead},
		"$set": bson.M{f("UpdatedAt"): now},
	}

	opts := options.UpdateOne().SetUpsert(true)
	res, err := r.col.Unwrap().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, fmt.Errorf("failed to upsert read state: %w", err)
	}

	// If it was newly inserted OR an existing record was modified
	return res.UpsertedCount > 0 || res.ModifiedCount > 0, nil
}

func (r *ReadStateRepo) GetForUser(ctx context.Context, userID bson.ObjectID) (map[bson.ObjectID]bson.ObjectID, error) {
	f := gmqb.Field[model.ReadState]
	states, err := r.col.Find(ctx, gmqb.Eq(f("UserID"), userID))
	if err != nil {
		return nil, err
	}

	result := make(map[bson.ObjectID]bson.ObjectID)
	for _, s := range states {
		result[s.ChannelID] = s.LastRead
	}
	return result, nil
}
