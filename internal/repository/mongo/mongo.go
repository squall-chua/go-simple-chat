package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

type Store struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func NewStore(ctx context.Context, uri string) (*Store, error) {
	cs, err := connstring.ParseAndValidate(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mongo uri: %w", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	dbName := cs.Database
	if dbName == "" {
		dbName = "chat_db" // Fallback if not in URI
	}

	db := client.Database(dbName)
	return &Store{
		Client: client,
		DB:     db,
	}, nil
}

func (s *Store) Close(ctx context.Context) error {
	return s.Client.Disconnect(ctx)
}
