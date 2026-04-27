package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type readStateDoc struct {
	UserID    bson.ObjectID `bson:"user_id"`
	ChannelID bson.ObjectID `bson:"channel_id"`
	LastRead  bson.ObjectID `bson:"last_read"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

type ReadStateRepo struct {
	col *gmqb.Collection[readStateDoc]
}

func NewReadStateRepo(ctx context.Context, db *mongo.Database) (*ReadStateRepo, error) {
	col := db.Collection("read_states")
	f := gmqb.Field[readStateDoc]

	wrapped := gmqb.Wrap[readStateDoc](col)

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

func (r *ReadStateRepo) Upsert(ctx context.Context, userID, channelID, lastRead string) (bool, error) {
	uOID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, err
	}
	chOID, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return false, err
	}
	lrOID, err := bson.ObjectIDFromHex(lastRead)
	if err != nil {
		return false, err
	}

	f := gmqb.Field[readStateDoc]
	now := time.Now()

	filter := gmqb.And(
		gmqb.Eq(f("UserID"), uOID),
		gmqb.Eq(f("ChannelID"), chOID),
	)

	update := gmqb.NewUpdate().
		Max(f("LastRead"), lrOID).
		Set(f("UpdatedAt"), now)

	res, err := r.col.UpdateOne(ctx, filter, update, gmqb.WithUpsert(true))
	if err != nil {
		return false, fmt.Errorf("failed to upsert read state: %w", err)
	}

	// If it was newly inserted OR an existing record was modified
	return res.UpsertedCount > 0 || res.ModifiedCount > 0, nil
}

func (r *ReadStateRepo) GetForUser(ctx context.Context, userID string) (map[string]string, error) {
	uOID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	f := gmqb.Field[readStateDoc]
	states, err := r.col.Find(ctx, gmqb.Eq(f("UserID"), uOID))
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range states {
		result[s.ChannelID.Hex()] = s.LastRead.Hex()
	}
	return result, nil
}
