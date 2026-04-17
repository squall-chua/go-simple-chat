package model

import (
	"time"

	chatv1 "go-simple-chat/api/v1"
)

// Screen represents the different states of the TUI application.
type Screen int

const (
	ScreenLaunch Screen = iota
	ScreenRegister
	ScreenRenew
	ScreenChat
)

// Message represents a chat message in the UI.
type Message struct {
	ID             string
	ChannelID      string
	SenderID       string
	SenderUsername string
	Content        string
	Medias         []*chatv1.Media
	CreatedAt      time.Time
}

// Channel represents a chat channel in the UI.
type Channel struct {
	ID                   string
	Name                 string
	Type                 string
	Participants         []string
	ParticipantUsernames []string
	UnreadCount          int
	LastReadID           string
	LastMessageID        string
}

// PresenceInfo represents the online status of a user.
type PresenceInfo struct {
	UserID   string
	Online   bool
	LastSeen time.Time
}

// --- Bubble Tea Messages ---

// MsgChannelsLoaded is sent when the channel list is successfully fetched.
type MsgChannelsLoaded struct {
	Channels []Channel
}

// MsgHistoryLoaded is sent when message history is fetched.
type MsgHistoryLoaded struct {
	ChannelID string
	Messages  []Message
	Appended  bool // true if these were loaded via pagination (prepend)
}

// MsgStreamEvent is sent when an event is received on the Bidi stream.
type MsgStreamEvent struct {
	Response *chatv1.StreamMessageResponse
}

// MsgPresenceLoaded is sent when presence info for a user is fetched.
type MsgPresenceUpdate struct {
	Event *chatv1.PresenceEvent
}

type MsgStreamStarted struct {
	Stream chatv1.ChatService_BidiStreamChatClient
}

// MsgError is sent for toast notifications.
type MsgError struct {
	Err error
}

// MsgStatusClear is sent to auto-dismiss the status bar/toast.
type MsgStatusClear struct{}

// MsgSwitchScreen is sent to navigate between screens.
type MsgSwitchScreen struct {
	Screen Screen
}
