package mongo

import (
	"context"
	"fmt"

	"go-simple-chat/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UserRepo struct {
	col *mongo.Collection
}

func NewUserRepo(ctx context.Context, db *mongo.Database) (*UserRepo, error) {
	col := db.Collection("users")

	// Create indices
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user indexes: %w", err)
	}

	return &UserRepo{col: col}, nil
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := r.col.InsertOne(ctx, user)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
