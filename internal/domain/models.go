package domain

import (
	"time"
)

type User struct {
	ID        string    `bson:"_id"`
	Username  string    `bson:"username"`
	PublicKey []byte    `bson:"public_key"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

type ChannelType string

const (
	ChannelDirect ChannelType = "direct"
	ChannelGroup  ChannelType = "group"
)

type Channel struct {
	ID           string      `bson:"_id"`
	Type         ChannelType `bson:"type"`
	Participants []string    `bson:"participants"` // User IDs
	Name         string      `bson:"name,omitempty"`
	CreatedAt    time.Time   `bson:"created_at"`
	UpdatedAt    time.Time   `bson:"updated_at"`
}

type Message struct {
	ID        string    `bson:"_id"`
	ChannelID string    `bson:"channel_id"`
	SenderID  string    `bson:"sender_id"`
	Content   string    `bson:"content"`
	MediaType string    `bson:"media_type,omitempty"` // text, image, voice, etc.
	MediaURL  string    `bson:"media_url,omitempty"`
	ThreadID  string    `bson:"thread_id,omitempty"`
	ParentID  string    `bson:"parent_id,omitempty"`
	CreatedAt time.Time `bson:"created_at"`
}

type OfflineMessage struct {
	ID        string    `bson:"_id"`
	UserID    string    `bson:"user_id"`
	Message   Message   `bson:"message"`
	ExpiresAt time.Time `bson:"expires_at"` // For TTL indexing
}
