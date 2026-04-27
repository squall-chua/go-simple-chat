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

type messageDoc struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	ChannelID      bson.ObjectID `bson:"channel_id"`
	SenderID       bson.ObjectID `bson:"sender_id"`
	SenderUsername string        `bson:"sender_username,omitempty"`
	Content        string        `bson:"content"`
	Medias         []model.Media `bson:"medias,omitempty"`
	ThreadID       string        `bson:"thread_id,omitempty"`
	ParentID       string        `bson:"parent_id,omitempty"`
	CreatedAt      time.Time     `bson:"created_at"`
}

func fromMessageModel(m *model.Message) (*messageDoc, error) {
	if m == nil {
		return nil, nil
	}
	var id bson.ObjectID
	if m.ID != "" {
		var err error
		id, err = bson.ObjectIDFromHex(m.ID)
		if err != nil {
			return nil, err
		}
	}

	cid, err := bson.ObjectIDFromHex(m.ChannelID)
	if err != nil {
		return nil, err
	}
	sid, err := bson.ObjectIDFromHex(m.SenderID)
	if err != nil {
		return nil, err
	}

	return &messageDoc{
		ID:             id,
		ChannelID:      cid,
		SenderID:       sid,
		SenderUsername: m.SenderUsername,
		Content:        m.Content,
		Medias:         m.Medias,
		ThreadID:       m.ThreadID,
		ParentID:       m.ParentID,
		CreatedAt:      m.CreatedAt,
	}, nil
}

func toMessageModel(d *messageDoc) *model.Message {
	if d == nil {
		return nil
	}
	return &model.Message{
		ID:             d.ID.Hex(),
		ChannelID:      d.ChannelID.Hex(),
		SenderID:       d.SenderID.Hex(),
		SenderUsername: d.SenderUsername,
		Content:        d.Content,
		Medias:         d.Medias,
		ThreadID:       d.ThreadID,
		ParentID:       d.ParentID,
		CreatedAt:      d.CreatedAt,
	}
}

type MessageRepo struct {
	col *gmqb.Collection[messageDoc]
}

func NewMessageRepo(ctx context.Context, db *mongo.Database) (*MessageRepo, error) {
	col := db.Collection("messages")
	f := gmqb.Field[messageDoc]

	wrapped := gmqb.Wrap[messageDoc](col)

	// Create compound index for channel ordering
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(
		gmqb.SortRule(f("ChannelID"), 1),
		gmqb.SortRule(f("CreatedAt"), -1),
	)))
	if err != nil {
		return nil, fmt.Errorf("failed to create message indexes: %w", err)
	}

	return &MessageRepo{col: wrapped}, nil
}

func (r *MessageRepo) Create(ctx context.Context, msg *model.Message) error {
	doc, err := fromMessageModel(msg)
	if err != nil {
		return err
	}
	if doc.ID.IsZero() {
		doc.ID = bson.NewObjectID()
	}
	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		return err
	}
	msg.ID = res.InsertedID.(bson.ObjectID).Hex()
	return nil
}

func (r *MessageRepo) GetByChannel(ctx context.Context, channelID string, limit int, beforeID string) ([]*model.Message, error) {
	cid, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return nil, err
	}
	var bid bson.ObjectID
	if beforeID != "" {
		bid, _ = bson.ObjectIDFromHex(beforeID)
	}

	f := gmqb.Field[messageDoc]
	filter := gmqb.Eq(f("ChannelID"), cid)
	if !bid.IsZero() {
		filter = gmqb.And(filter, gmqb.Lt(f("ID"), bid))
	}

	docs, err := r.col.Find(ctx, filter,
		gmqb.WithLimit(int64(limit)),
		gmqb.WithSort(gmqb.SortSpec(gmqb.SortRule(f("CreatedAt"), -1))),
	)
	if err != nil {
		return nil, err
	}

	results := make([]*model.Message, len(docs))
	for i := range docs {
		results[i] = toMessageModel(&docs[i])
	}
	return results, nil
}

func (r *MessageRepo) CountAfter(ctx context.Context, channelID string, afterID string) (int64, error) {
	cid, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return 0, err
	}
	var aid bson.ObjectID
	if afterID != "" {
		aid, _ = bson.ObjectIDFromHex(afterID)
	}

	f := gmqb.Field[messageDoc]
	filter := gmqb.Eq(f("ChannelID"), cid)
	if !aid.IsZero() {
		filter = gmqb.And(filter, gmqb.Gt(f("ID"), aid))
	}
	return r.col.CountDocuments(ctx, filter)
}
