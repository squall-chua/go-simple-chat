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

	filter := gmqb.And(
		gmqb.Eq(f("UserID"), userID),
		gmqb.Eq(f("ChannelID"), channelID),
	)

	// Single atomic call using an aggregation pipeline update (requires MongoDB 4.2+)
	// 1. last_read = max(existing, new)
	// 2. updated_at = if last_read increased then now else existing
	pipeline := mongo.Pipeline{
		{{Key: "$set", Value: bson.M{
			f("LastRead"): bson.M{"$max": bson.A{"$" + f("LastRead"), lastRead}},
			f("UpdatedAt"): bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$lt": bson.A{"$" + f("LastRead"), lastRead}},
					"then": now,
					"else": bson.M{"$ifNull": bson.A{"$" + f("UpdatedAt"), now}},
				},
			},
		}}},
	}

	opts := options.UpdateOne().SetUpsert(true)
	res, err := r.col.Unwrap().UpdateOne(ctx, filter, pipeline, opts)
	if err != nil {
		return false, fmt.Errorf("failed to upsert read state: %w", err)
	}

	// If it was newly inserted OR an existing record was modified (meaning last_read increased)
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
