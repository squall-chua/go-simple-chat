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

type channelDoc struct {
	ID            bson.ObjectID     `bson:"_id,omitempty"`
	Type          model.ChannelType `bson:"type"`
	Participants  []bson.ObjectID   `bson:"participants"`
	Name          string            `bson:"name,omitempty"`
	CreatedAt     time.Time         `bson:"created_at"`
	UpdatedAt     time.Time         `bson:"updated_at"`
	LastMessageID bson.ObjectID     `bson:"last_message_id,omitempty"`
}

func fromChannelModel(m *model.Channel) (*channelDoc, error) {
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

	participants := make([]bson.ObjectID, len(m.Participants))
	for i, p := range m.Participants {
		var err error
		participants[i], err = bson.ObjectIDFromHex(p)
		if err != nil {
			return nil, err
		}
	}

	var lastMsgID bson.ObjectID
	if m.LastMessageID != "" {
		var err error
		lastMsgID, err = bson.ObjectIDFromHex(m.LastMessageID)
		if err != nil {
			return nil, err
		}
	}

	return &channelDoc{
		ID:            id,
		Type:          m.Type,
		Participants:  participants,
		Name:          m.Name,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		LastMessageID: lastMsgID,
	}, nil
}

func toChannelModel(d *channelDoc) *model.Channel {
	if d == nil {
		return nil
	}
	participants := make([]string, len(d.Participants))
	for i, p := range d.Participants {
		participants[i] = p.Hex()
	}
	return &model.Channel{
		ID:            d.ID.Hex(),
		Type:          d.Type,
		Participants:  participants,
		Name:          d.Name,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
		LastMessageID: d.LastMessageID.Hex(),
	}
}

type ChannelRepo struct {
	col *gmqb.Collection[channelDoc]
}

func NewChannelRepo(ctx context.Context, db *mongo.Database) (*ChannelRepo, error) {
	col := db.Collection("channels")
	f := gmqb.Field[channelDoc]

	wrapped := gmqb.Wrap[channelDoc](col)

	// Create index for participants lookup
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(gmqb.SortRule(f("Participants"), 1))))
	if err != nil {
		return nil, fmt.Errorf("failed to create channel indexes: %w", err)
	}

	return &ChannelRepo{col: wrapped}, nil
}

func (r *ChannelRepo) Create(ctx context.Context, channel *model.Channel) error {
	doc, err := fromChannelModel(channel)
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
	channel.ID = res.InsertedID.(bson.ObjectID).Hex()
	return nil
}

func (r *ChannelRepo) GetByID(ctx context.Context, id string) (*model.Channel, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	f := gmqb.Field[channelDoc]
	doc, err := r.col.FindOne(ctx, gmqb.Eq(f("ID"), oid))
	if err != nil {
		return nil, err
	}
	return toChannelModel(doc), nil
}

func (r *ChannelRepo) GetForUser(ctx context.Context, userID string) ([]*model.Channel, error) {
	oid, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	f := gmqb.Field[channelDoc]
	docs, err := r.col.Find(ctx, gmqb.Eq(f("Participants"), oid))
	if err != nil {
		return nil, err
	}

	results := make([]*model.Channel, len(docs))
	for i := range docs {
		results[i] = toChannelModel(&docs[i])
	}
	return results, nil
}

func (r *ChannelRepo) GetDirectChannel(ctx context.Context, user1, user2 string) (*model.Channel, error) {
	o1, err := bson.ObjectIDFromHex(user1)
	if err != nil {
		return nil, err
	}
	o2, err := bson.ObjectIDFromHex(user2)
	if err != nil {
		return nil, err
	}
	f := gmqb.Field[channelDoc]
	doc, err := r.col.FindOne(ctx, gmqb.And(
		gmqb.Eq(f("Type"), model.ChannelDirect),
		gmqb.All(f("Participants"), o1, o2),
	))
	if err != nil {
		return nil, err
	}
	return toChannelModel(doc), nil
}

func (r *ChannelRepo) AddParticipants(ctx context.Context, channelID string, userIDs []string) error {
	oid, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return err
	}
	oids := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		o, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		oids[i] = o
	}

	f := gmqb.Field[channelDoc]
	_, err = r.col.UpdateOne(ctx,
		gmqb.Eq(f("ID"), oid),
		gmqb.NewUpdate().AddToSetEach(f("Participants"), oids...),
	)
	return err
}

func (r *ChannelRepo) UpdateLastMessageID(ctx context.Context, channelID string, messageID string) error {
	oid, err := bson.ObjectIDFromHex(channelID)
	if err != nil {
		return err
	}
	mid, err := bson.ObjectIDFromHex(messageID)
	if err != nil {
		return err
	}
	f := gmqb.Field[channelDoc]
	_, err = r.col.UpdateOne(ctx,
		gmqb.Eq(f("ID"), oid),
		gmqb.NewUpdate().Set(f("LastMessageID"), mid),
	)
	return err
}
