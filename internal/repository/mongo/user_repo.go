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

type userDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Username  string        `bson:"username"`
	PublicKey []byte        `bson:"public_key"`
	LastSeen  time.Time     `bson:"last_seen,omitempty"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

func fromUserModel(u *model.User) (*userDoc, error) {
	if u == nil {
		return nil, nil
	}
	var id bson.ObjectID
	if u.ID != "" {
		var err error
		id, err = bson.ObjectIDFromHex(u.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid object id: %w", err)
		}
	}
	return &userDoc{
		ID:        id,
		Username:  u.Username,
		PublicKey: u.PublicKey,
		LastSeen:  u.LastSeen,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}

func toUserModel(d *userDoc) *model.User {
	if d == nil {
		return nil
	}
	return &model.User{
		ID:        d.ID.Hex(),
		Username:  d.Username,
		PublicKey: d.PublicKey,
		LastSeen:  d.LastSeen,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

type UserRepo struct {
	col *gmqb.Collection[userDoc]
}

func NewUserRepo(ctx context.Context, db *mongo.Database) (*UserRepo, error) {
	col := db.Collection("users")
	f := gmqb.Field[userDoc]

	wrapped := gmqb.Wrap[userDoc](col)

	// Create indices
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(gmqb.SortRule(f("Username"), 1))).Unique())
	if err != nil {
		return nil, fmt.Errorf("failed to create user indexes: %w", err)
	}

	return &UserRepo{col: wrapped}, nil
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	doc, err := fromUserModel(user)
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
	user.ID = res.InsertedID.(bson.ObjectID).Hex()
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	f := gmqb.Field[userDoc]
	doc, err := r.col.FindOne(ctx, gmqb.Eq(f("ID"), oid))
	if err != nil {
		return nil, err
	}
	return toUserModel(doc), nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	f := gmqb.Field[userDoc]
	doc, err := r.col.FindOne(ctx, gmqb.Eq(f("Username"), username))
	if err != nil {
		return nil, err
	}
	return toUserModel(doc), nil
}

func (r *UserRepo) UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	f := gmqb.Field[userDoc]
	_, err = r.col.UpdateOne(ctx,
		gmqb.Eq(f("ID"), oid),
		gmqb.NewUpdate().
			Set(f("LastSeen"), lastSeen).
			Set(f("UpdatedAt"), time.Now()),
	)
	return err
}
