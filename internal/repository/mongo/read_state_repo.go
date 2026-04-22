package mongo

import (
	"context"
	"fmt"
	"time"

	"go-simple-chat/internal/model"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ReadStateRepo struct {
	col *gmqb.Collection[model.ReadState]
}

func NewReadStateRepo(ctx context.Context, db *mongo.Database) (*ReadStateRepo, error) {
	col := db.Collection("read_states")
	f := gmqb.Field[model.ReadState]

	wrapped := gmqb.Wrap[model.ReadState](col)

	// Index: user_id + channel_id unique
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(
		gmqb.SortRule(f("UserID"), 1),
		gmqb.SortRule(f("ChannelID"), 1),
	)).Unique())
	if err != nil {
		return nil, fmt.Errorf("failed to create read_state indexes: %w", err)
	}

	return &ReadStateRepo{col: wrapped}, nil
}

func (r *ReadStateRepo) Upsert(ctx context.Context, userID, channelID, lastRead bson.ObjectID) (bool, error) {
	f := gmqb.Field[model.ReadState]
	now := time.Now()

	filter := gmqb.And(
		gmqb.Eq(f("UserID"), userID),
		gmqb.Eq(f("ChannelID"), channelID),
	)

	update := gmqb.NewUpdate().
		Max(f("LastRead"), lastRead).
		Set(f("UpdatedAt"), now)

	res, err := r.col.UpdateOne(ctx, filter, update, gmqb.WithUpsert(true))
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
