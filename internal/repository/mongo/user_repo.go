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

type UserRepo struct {
	col *gmqb.Collection[model.User]
}

func NewUserRepo(ctx context.Context, db *mongo.Database) (*UserRepo, error) {
	col := db.Collection("users")
	f := gmqb.Field[model.User]

	wrapped := gmqb.Wrap[model.User](col)

	// Create indices
	_, err := wrapped.CreateIndex(ctx, gmqb.NewIndex(gmqb.SortSpec(gmqb.SortRule(f("Username"), 1))).Unique())
	if err != nil {
		return nil, fmt.Errorf("failed to create user indexes: %w", err)
	}

	return &UserRepo{col: wrapped}, nil
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	_, err := r.col.InsertOne(ctx, user)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id bson.ObjectID) (*model.User, error) {
	f := gmqb.Field[model.User]
	return r.col.FindOne(ctx, gmqb.Eq(f("ID"), id))
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	f := gmqb.Field[model.User]
	return r.col.FindOne(ctx, gmqb.Eq(f("Username"), username))
}

func (r *UserRepo) UpdateLastSeen(ctx context.Context, id bson.ObjectID, lastSeen time.Time) error {
	f := gmqb.Field[model.User]
	_, err := r.col.UpdateOne(ctx,
		gmqb.Eq(f("ID"), id),
		gmqb.NewUpdate().
			Set(f("LastSeen"), lastSeen).
			Set(f("UpdatedAt"), time.Now()),
	)
	return err
}
