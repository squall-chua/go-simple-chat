package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/model"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MessageRepo struct {
	col *gmqb.Collection[model.Message]
}

func NewMessageRepo(ctx context.Context, db *mongo.Database) (*MessageRepo, error) {
	col := db.Collection("messages")
	f := gmqb.Field[model.Message]

	// Create compound index for channel ordering
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: f("ChannelID"), Value: 1},
			{Key: f("CreatedAt"), Value: -1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create message indexes: %w", err)
	}

	return &MessageRepo{col: gmqb.Wrap[model.Message](col)}, nil
}

func (r *MessageRepo) Create(ctx context.Context, msg *model.Message) error {
	if msg.ID.IsZero() {
		msg.ID = bson.NewObjectID()
	}
	_, err := r.col.InsertOne(ctx, msg)
	return err
}

func (r *MessageRepo) GetByChannel(ctx context.Context, channelID bson.ObjectID, limit int, beforeID bson.ObjectID) ([]*model.Message, error) {
	f := gmqb.Field[model.Message]
	filter := gmqb.Eq(f("ChannelID"), channelID)
	if !beforeID.IsZero() {
		filter = gmqb.And(filter, gmqb.Lt(f("ID"), beforeID))
	}

	msgs, err := r.col.Find(ctx, filter,
		gmqb.WithLimit(int64(limit)),
		gmqb.WithSort(gmqb.SortSpec(gmqb.SortRule(f("CreatedAt"), -1))),
	)
	if err != nil {
		return nil, err
	}

	// Convert to slice of pointers as required by signature
	results := make([]*model.Message, len(msgs))
	for i := range msgs {
		results[i] = &msgs[i]
	}
	return results, nil
}
