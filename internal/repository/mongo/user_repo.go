package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/model"
	"github.com/squall-chua/gmqb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UserRepo struct {
	col *gmqb.Collection[model.User]
}

func NewUserRepo(ctx context.Context, db *mongo.Database) (*UserRepo, error) {
	col := db.Collection("users")
	f := gmqb.Field[model.User]

	// Create indices
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: f("Username"), Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user indexes: %w", err)
	}

	return &UserRepo{col: gmqb.Wrap[model.User](col)}, nil
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
