package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go-simple-chat/internal/config"
	"go-simple-chat/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type userDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Username  string        `bson:"username"`
	PublicKey []byte        `bson:"public_key"`
}

func main() {
	cfg := config.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("go-simple-chat")
	userCol := db.Collection("users")

	fmt.Println("--- User Certificate Migration Validator ---")
	
	cursor, err := userCol.Find(ctx, bson.M{})
	if err != nil {
		log.Fatalf("failed to find users: %v", err)
	}
	defer cursor.Close(ctx)

	var docs []userDoc
	if err := cursor.All(ctx, &docs); err != nil {
		log.Fatalf("failed to decode users: %v", err)
	}

	var users []model.User
	for _, d := range docs {
		users = append(users, model.User{
			ID:        d.ID.Hex(),
			Username:  d.Username,
			PublicKey: d.PublicKey,
		})
	}

	validCount := 0
	errorCount := 0

	for _, u := range users {
		fmt.Printf("Checking User: %s (%s)\n", u.Username, u.ID)
		
		if len(u.PublicKey) == 0 {
			fmt.Printf("  [!] ERROR: Missing public key in database\n")
			errorCount++
			continue
		}

		// In a real scenario, we might try to find a .crt file if it's on disk
		// but usually the server only has the pinning.
		
		fmt.Printf("  [+] PASS: Public key pinned (%d bytes)\n", len(u.PublicKey))
		validCount++
	}

	fmt.Println("--- Summary ---")
	fmt.Printf("Total Users: %d\n", len(users))
	fmt.Printf("Valid:       %d\n", validCount)
	fmt.Printf("Errors:      %d\n", errorCount)

	if errorCount > 0 {
		fmt.Println("\n[!] WARNING: Some users have missing public keys and will NOT be able to connect via mTLS.")
		os.Exit(1)
	}
	fmt.Println("\nAll users are ready for strict mTLS enforcement.")
}
