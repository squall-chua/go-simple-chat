package model

import (
	"time"
)

type User struct {
	ID        string
	Username  string
	PublicKey []byte
	LastSeen  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ChannelType string

const (
	ChannelDirect ChannelType = "direct"
	ChannelGroup  ChannelType = "group"
)

type Channel struct {
	ID            string
	Type          ChannelType
	Participants  []string // User IDs
	Name          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastMessageID string
	LastReadID    string // User-specific state
}

type Media struct {
	Type string
	URL  string
	Name string
}

type Message struct {
	ID             string
	ChannelID      string
	SenderID       string
	SenderUsername string
	Content        string
	Medias         []Media
	ThreadID       string
	ParentID       string
	CreatedAt      time.Time
}

type PresenceEvent struct {
	UserID string
	Online bool
}

type SignalType string

const (
	SignalNewChannel   SignalType = "new_channel"
	SignalReadUpdate   SignalType = "read_update"
	SignalRosterUpdate SignalType = "roster_update"
)

type SystemSignal struct {
	Type      SignalType
	ChannelID string
	MessageID string
	UserID    string
	Username  string
}

type ReadState struct {
	UserID    string
	ChannelID string
	LastRead  string
	UpdatedAt time.Time
}

type AuthChallenge struct {
	UserID    string
	Nonce     string
	CreatedAt time.Time
}
