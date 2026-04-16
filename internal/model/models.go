package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Username  string        `bson:"username"`
	PublicKey []byte        `bson:"public_key"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

type ChannelType string

const (
	ChannelDirect ChannelType = "direct"
	ChannelGroup  ChannelType = "group"
)

type Channel struct {
	ID           bson.ObjectID   `bson:"_id,omitempty"`
	Type         ChannelType     `bson:"type"`
	Participants []bson.ObjectID `bson:"participants"` // User IDs
	Name         string          `bson:"name,omitempty"`
	CreatedAt    time.Time       `bson:"created_at"`
	UpdatedAt    time.Time       `bson:"updated_at"`
	LastReadID   bson.ObjectID   `bson:"-"` // User-specific state
}

type Media struct {
	Type string `bson:"type"`
	URL  string `bson:"url"`
}

type Message struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	ChannelID bson.ObjectID `bson:"channel_id"`
	SenderID  bson.ObjectID `bson:"sender_id"`
	Content   string        `bson:"content"`
	Medias    []Media       `bson:"medias,omitempty"`
	ThreadID  string        `bson:"thread_id,omitempty"`
	ParentID  string        `bson:"parent_id,omitempty"`
	CreatedAt time.Time     `bson:"created_at"`
}

type PresenceEvent struct {
	UserID string `bson:"user_id"`
	Online bool   `bson:"online"`
}

type SignalType string

const (
	SignalNewChannel SignalType = "new_channel"
	SignalReadUpdate SignalType = "read_update"
)

type SystemSignal struct {
	Type      SignalType    `bson:"type"`
	ChannelID bson.ObjectID `bson:"channel_id"`
	MessageID bson.ObjectID `bson:"message_id,omitempty"`
}

type ReadState struct {
	UserID    bson.ObjectID `bson:"user_id"`
	ChannelID bson.ObjectID `bson:"channel_id"`
	LastRead  bson.ObjectID `bson:"last_read"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

type AuthChallenge struct {
	UserID    string    `bson:"_id"`
	Nonce     string    `bson:"nonce"`
	CreatedAt time.Time `bson:"created_at"`
}
