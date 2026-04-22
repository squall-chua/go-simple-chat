package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/cmd/client/model"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ChatScreen struct {
	client chatv1.ChatServiceClient
	stream chatv1.ChatService_BidiStreamChatClient
	userID string // Self ID

	// UI Components
	sidebar  list.Model
	viewport viewport.Model
	textarea textarea.Model
	delegate channelDelegate

	// State
	sidebarFocused     bool
	participantFocused bool
	activeChannelID    string
	channels           map[string]model.Channel
	messages           map[string][]model.Message
	presenceMap        map[string]model.PresenceInfo
	participantList    list.Model

	width     int
	height    int
	err       error
	status    string
	statusErr bool

	isHistoryLoading bool
	historyExhausted map[string]bool
	newMessagesMarkerID string // ID of the message after which to show "New Messages" marker
}

func NewChatScreen(client chatv1.ChatServiceClient, userID string) ChatScreen {
	presenceMap := make(map[string]model.PresenceInfo)
	delegate := NewChannelDelegate(presenceMap, userID)
	delegate.focused = false // Input focused by default

	// Initialize Sidebar List
	l := list.New([]list.Item{}, delegate, 30, 0)
	l.Title = "Channels"
	l.SetShowHelp(false)
	l.SetStatusBarItemName("channel", "channels")

	// Initialize Participant List
	pDelegate := NewParticipantDelegate(presenceMap)
	pl := list.New([]list.Item{}, pDelegate, 30, 0)
	pl.Title = "Participants"
	pl.SetShowHelp(false)
	pl.SetStatusBarItemName("member", "members")

	// Initialize Viewport
	vp := viewport.New(0, 0)
	vp.SetContent("Select a channel to start chatting...")

	// Initialize Textarea
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.SetHeight(3)
	ta.KeyMap.InsertNewline.SetEnabled(false) // Enter sends message

	return ChatScreen{
		client:           client,
		userID:           userID,
		viewport:         vp,
		textarea:         ta,
		delegate:         delegate,
		channels:         make(map[string]model.Channel),
		messages:         make(map[string][]model.Message),
		presenceMap:      presenceMap,
		historyExhausted: make(map[string]bool),
		sidebar:          l,
		participantList:  pl,
	}
}

func (m ChatScreen) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.loadChannelsCmd(),
		m.initStreamCmd(),
	)
}

func (m ChatScreen) Update(msg tea.Msg) (ChatScreen, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, tea.Quit // Or maybe back to launcher?
		case "tab":
			m.sidebarFocused = !m.sidebarFocused
			if m.sidebarFocused {
				m.textarea.Blur()
			} else {
				m.textarea.Focus()
			}
			m.delegate.focused = m.sidebarFocused
			m.sidebar.SetDelegate(m.delegate)
			return m, nil
		case "enter":
			if m.sidebarFocused && m.sidebar.FilterState() == list.Filtering {
				// If filter is empty, dismiss it on enter
				if m.sidebar.FilterInput.Value() == "" {
					m.sidebar.ResetFilter()
					return m, nil
				}
			}

			if m.textarea.Focused() {
				// Send message or handle command
				content := strings.TrimSpace(m.textarea.Value())
				if content != "" {
					if strings.HasPrefix(content, "/") {
						cmd = HandleCommand(m, content)
					} else if m.activeChannelID != "" {
						cmd = m.sendMessageCmd(content)
						m.newMessagesMarkerID = "" // Clear marker on interaction
					}
					cmds = append(cmds, cmd)
					m.textarea.Reset()
				}
				return m, tea.Batch(cmds...)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Dimensions will be propagated to children in the active screen switch below

	case model.MsgChannelsLoaded:
		m.channels = make(map[string]model.Channel)
		var pIDs []string
		pSet := make(map[string]struct{})

		for _, ch := range msg.Channels {
			m.channels[ch.ID] = ch
			for _, pID := range ch.Participants {
				if _, ok := pSet[pID]; !ok {
					pSet[pID] = struct{}{}
					pIDs = append(pIDs, pID)
				}
			}
		}

		// Check presence for all known participants in one bulk request
		if len(pIDs) > 0 {
			cmds = append(cmds, m.getPresenceCmd(pIDs))
		}

		items := make([]list.Item, len(msg.Channels))
		for i, ch := range msg.Channels {
			items[i] = channelListItem{ch}
		}
		m.sidebar.SetItems(items)
		m.delegate.focused = m.sidebarFocused
		m.sidebar.SetDelegate(m.delegate)

		if m.activeChannelID == "" && len(msg.Channels) > 0 {
			m.activeChannelID = msg.Channels[0].ID
			cmds = append(cmds, m.loadHistoryCmd(m.activeChannelID, ""))
		}

	case model.MsgHistoryLoaded:
		m.isHistoryLoading = false
		if len(msg.Messages) == 0 {
			m.historyExhausted[msg.ChannelID] = true
		}

		if msg.Appended {
			// Loading older messages (PREPEND)
			m.messages[msg.ChannelID] = mergeMessages(msg.Messages, m.messages[msg.ChannelID])
		} else {
			// Initial channel switch (REPLACE with hydration)
			m.messages[msg.ChannelID] = mergeMessages(m.messages[msg.ChannelID], msg.Messages)
		}

		if msg.ChannelID == m.activeChannelID {
			m.refreshParticipantList()
			m.refreshViewportContents()
			if !msg.Appended {
				m.viewport.GotoBottom()
			}
		}

	case model.MsgPresenceUpdate:
		pe := msg.Event
		m.presenceMap[pe.UserId] = model.PresenceInfo{
			UserID: pe.UserId,
			Online: pe.Online,
		}
		m.refreshParticipantList()
		return m, nil

	case model.MsgBulkPresenceUpdate:
		for _, pe := range msg.Events {
			m.presenceMap[pe.UserId] = model.PresenceInfo{
				UserID: pe.UserId,
				Online: pe.Online,
			}
		}
		m.refreshParticipantList()
		return m, nil

	case model.MsgStreamEvent:
		res := msg.Response
		switch payload := res.Payload.(type) {
		case *chatv1.StreamMessageResponse_MessageReceived:
			pm := payload.MessageReceived
			cmds = append(cmds, m.handleIncomingMessage(pm))
		case *chatv1.StreamMessageResponse_PresenceEvent:
			pe := payload.PresenceEvent
			m.presenceMap[pe.UserId] = model.PresenceInfo{
				UserID: pe.UserId,
				Online: pe.Online,
			}
		case *chatv1.StreamMessageResponse_ChannelJoined:
			// A new channel was assigned to us, reload the list
			m.status = "Added to a new channel!"
			m.statusErr = false
			cmds = append(cmds, m.loadChannelsCmd())
			cmds = append(cmds, tea.Tick(time.Second*5, func(t time.Time) tea.Msg { return model.MsgStatusClear{} }))
		case *chatv1.StreamMessageResponse_ParticipantAdded:
			// Roster changed in one of the channels, reload to get latest metadata
			cmds = append(cmds, m.loadChannelsCmd())
		case *chatv1.StreamMessageResponse_IdentityEvent:
			ie := payload.IdentityEvent
			switch ie.Type {
			case chatv1.IdentityEvent_TYPE_EXPIRING_SOON:
				m.status = fmt.Sprintf("SECURITY ALERT: Certificate expires soon (%s). Please renew.", ie.ExpiresAt.AsTime().Format(time.Kitchen))
				m.statusErr = true
			case chatv1.IdentityEvent_TYPE_EXPIRED:
				m.err = fmt.Errorf("IDENTITY EXPIRED: Connection terminated. Please restart with a renewed certificate.")
			}
		case *chatv1.StreamMessageResponse_Error:
			cmds = append(cmds, func() tea.Msg { return model.MsgError{Err: fmt.Errorf("stream error: %s", payload.Error.Message)} })
		}
		// Continue listening
		cmds = append(cmds, m.recvStreamCmd())

	case model.MsgStreamStarted:
		m.stream = msg.Stream
		// Mark self as online locally
		m.presenceMap[m.userID] = model.PresenceInfo{
			UserID:   m.userID,
			Online:   true,
			LastSeen: time.Now(),
		}
		cmds = append(cmds, m.recvStreamCmd(), m.heartbeatCmd())

	case model.MsgError:
		if msg.Err != nil {
			m.status = msg.Err.Error()
			m.statusErr = true
			return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg { return model.MsgStatusClear{} })
		}

	case model.MsgStatusClear:
		m.status = ""
		m.statusErr = false

	case heartbeatMsg:
		return m, m.heartbeatCmd()
	}

	// Update children
	if m.sidebarFocused {
		m.sidebar, cmd = m.sidebar.Update(msg)
		cmds = append(cmds, cmd)

		// Selection change detection
		if it := m.sidebar.SelectedItem(); it != nil {
			id := it.(channelListItem).ch.ID
			if id != m.activeChannelID {
				m.activeChannelID = id

				// Clear unread on focus
				if ch, ok := m.channels[id]; ok {
					// Set marker if there are unread messages
					if ch.UnreadCount > 0 {
						m.newMessagesMarkerID = ch.LastReadID
						if m.newMessagesMarkerID == "" {
							m.newMessagesMarkerID = "MARKER_TOP"
						}
					} else {
						m.newMessagesMarkerID = ""
					}

					ch.LastReadID = ch.LastMessageID
					ch.UnreadCount = 0
					m.channels[id] = ch
					cmds = append(cmds, m.markAsReadCmd(id, ""))

					// Update sidebar items state
					items := m.sidebar.Items()
					for i, item := range items {
						cli := item.(channelListItem)
						if cli.ch.ID == id {
							cli.ch = ch
							cmds = append(cmds, m.sidebar.SetItem(i, cli))
							break
						}
					}
				}
				m.refreshParticipantList()
				cmds = append(cmds, m.loadHistoryCmd(id, ""))
			}
		}
	} else {
		prevLen := len(m.textarea.Value())
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)

		// If we started/stopped typing a command, re-layout
		currVal := m.textarea.Value()
		if (strings.HasPrefix(currVal, "/") && !strings.HasPrefix(m.textarea.Placeholder, "/")) ||
			(len(currVal) == 0 && prevLen > 0) || (len(currVal) > 0 && prevLen == 0) {
			m.UpdateLayout()
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Lazy load history on scroll to top - only if NOT already loading and NOT exhausted
	if m.viewport.AtTop() && m.activeChannelID != "" && !m.isHistoryLoading && !m.historyExhausted[m.activeChannelID] {
		msgs := m.messages[m.activeChannelID]
		if len(msgs) > 0 {
			m.isHistoryLoading = true
			cmds = append(cmds, m.loadHistoryCmd(m.activeChannelID, msgs[0].ID))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ChatScreen) UpdateLayout() {
	sidebarWidth := 30
	mainWidth := m.width - sidebarWidth - 1 // -1 for separator

	m.sidebar.SetSize(sidebarWidth, m.height/2-2)
	m.participantList.SetSize(sidebarWidth, m.height/2-2)

	m.viewport.Width = mainWidth

	reservedHeight := 4 + 3 // Header + Input
	if strings.HasPrefix(m.textarea.Value(), "/") {
		reservedHeight += 3 // Command help overlay
	}
	m.viewport.Height = m.height - reservedHeight

	m.textarea.SetWidth(mainWidth)
}

func (m *ChatScreen) refreshParticipantList() {
	ch, ok := m.channels[m.activeChannelID]
	if !ok {
		m.participantList.SetItems([]list.Item{})
		return
	}

	items := make([]list.Item, 0, len(ch.Participants))
	for i, pID := range ch.Participants {
		username := pID
		if i < len(ch.ParticipantUsernames) {
			username = ch.ParticipantUsernames[i]
		}

		online := false
		if info, ok := m.presenceMap[pID]; ok {
			online = info.Online
		}

		items = append(items, participantListItem{
			userID:   pID,
			username: username,
			online:   online,
		})
	}
	m.participantList.SetItems(items)
}

func (m ChatScreen) View() string {
	if m.err != nil {
		return model.StyleStatusError.Render(fmt.Sprintf("Fatal Error: %v\nPress Ctrl+C to quit.", m.err))
	}

	sidebarWidth := 25
	// Sidebar (Channels + Participants)
	sidebarTop := RenderSidebarContainer(m.sidebar.View(), m.sidebarFocused, sidebarWidth)
	sidebarBottom := RenderSidebarContainer(m.participantList.View(), false, sidebarWidth)
	sidebar := lipgloss.JoinVertical(lipgloss.Left, sidebarTop, sidebarBottom)

	// Header
	activeName := "No Channel Selected"
	if ch, ok := m.channels[m.activeChannelID]; ok {
		activeName = ch.Name
	}
	header := model.StyleHeader.Copy().
		Width(m.width).
		Render(fmt.Sprintf(" Go-Simple-Chat %s %s", model.Divider, activeName))

	// Footer Help
	footer := model.StyleSidebar.Copy().
		Width(m.width).
		Render(" ↑↓ channels • Tab switch focus • PgUp/PgDn scroll • Enter send • Ctrl+C quit")

	// Input Area + Command Overlay
	inputArea := model.StyleInputContainer.Render(m.textarea.View())

	// Command Overlay
	if strings.HasPrefix(m.textarea.Value(), "/") {
		helpBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(model.ColorHighlight).
			Padding(0, 1).
			Background(model.ColorSubtle).
			Render("Commands: /create <name> <users> | /dm <user> | /add <users> | /presence <user> | /read | /help")

		// Overlay it above the input
		inputArea = lipgloss.JoinVertical(lipgloss.Left, helpBox, inputArea)
	}

	// Status Bar
	statusBar := ""
	if m.status != "" {
		st := model.StyleStatusOK
		if m.statusErr {
			st = model.StyleStatusError
		}
		statusBar = "\n" + st.Render(" "+m.status)
	}

	mainWidth := m.width - sidebarWidth - 1

	main := lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		lipgloss.NewStyle().Width(mainWidth).Render(inputArea),
		lipgloss.NewStyle().Width(mainWidth).Render(statusBar),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	// Final assembly with full-screen height enforcement
	finalView := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	// Ensure the entire application fills the screen exactly
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		MaxWidth(m.width).
		MaxHeight(m.height).
		Align(lipgloss.Left, lipgloss.Top).
		Render(finalView)
}

// --- Commands ---

func (m ChatScreen) loadChannelsCmd() tea.Cmd {
	return func() tea.Msg {
		res, err := m.client.ListChannels(context.Background(), &chatv1.ListChannelsRequest{})
		if err != nil {
			return model.MsgError{Err: err}
		}

		channels := make([]model.Channel, len(res.Channels))
		for i, c := range res.Channels {
			name := c.Name
			if c.Type == "direct" || c.Type == "TYPE_DIRECT" {
				for j, pID := range c.Participants {
					if pID != m.userID && j < len(c.ParticipantUsernames) {
						name = c.ParticipantUsernames[j]
						break
					}
				}
			}

			channels[i] = model.Channel{
				ID:                   c.Id,
				Name:                 name,
				Type:                 c.Type,
				Participants:         c.Participants,
				ParticipantUsernames: c.ParticipantUsernames,
				LastReadID:           c.LastReadId,
				LastMessageID:        c.LastMessageId,
				UnreadCount:          int(c.UnreadCount),
			}
		}
		return model.MsgChannelsLoaded{Channels: channels}
	}
}

func (m ChatScreen) loadHistoryCmd(channelID string, beforeID string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.client.GetMessages(context.Background(), &chatv1.GetMessagesRequest{
			ChannelId: channelID,
			Limit:     50,
			BeforeId:  beforeID,
		})
		if err != nil {
			return model.MsgError{Err: err}
		}

		messages := make([]model.Message, len(res.Messages))
		for i, rm := range res.Messages {
			messages[i] = model.Message{
				ID:             rm.MessageId,
				ChannelID:      rm.ChannelId,
				SenderID:       rm.SenderId,
				SenderUsername: rm.SenderUsername,
				Content:        rm.Content,
				Medias:         rm.Medias,
				CreatedAt:      rm.CreatedAt.AsTime(),
			}
		}
		return model.MsgHistoryLoaded{
			ChannelID: channelID,
			Messages:  messages,
			Appended:  beforeID != "",
		}
	}
}

func (m ChatScreen) initStreamCmd() tea.Cmd {
	return func() tea.Msg {
		stream, err := m.client.BidiStreamChat(context.Background())
		if err != nil {
			return model.MsgError{Err: err}
		}
		return model.MsgStreamStarted{Stream: stream}
	}
}

type heartbeatMsg struct{}

func (m ChatScreen) heartbeatCmd() tea.Cmd {
	return tea.Tick(time.Second*30, func(t time.Time) tea.Msg {
		if m.stream != nil {
			_ = m.stream.Send(&chatv1.StreamMessageRequest{
				Payload: &chatv1.StreamMessageRequest_Heartbeat{
					Heartbeat: &chatv1.Heartbeat{},
				},
			})
		}
		return heartbeatMsg{}
	})
}


func (m ChatScreen) getPresenceCmd(userIDs []string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.client.GetPresence(context.Background(), &chatv1.GetPresenceRequest{
			UserIds: userIDs,
		})
		if err != nil || len(res.Presences) == 0 {
			return nil // Silent fail for presence
		}
		
		var events []*chatv1.PresenceEvent
		for _, p := range res.Presences {
			events = append(events, &chatv1.PresenceEvent{
				UserId: p.UserId,
				Online: p.Online,
			})
		}
		return model.MsgBulkPresenceUpdate{Events: events}
	}
}

func (m ChatScreen) recvStreamCmd() tea.Cmd {
	return func() tea.Msg {
		if m.stream == nil {
			return nil
		}
		res, err := m.stream.Recv()
		if err != nil {
			return model.MsgError{Err: err}
		}
		return model.MsgStreamEvent{Response: res}
	}
}

func (m *ChatScreen) handleIncomingMessage(pm *chatv1.MessageReceived) tea.Cmd {
	var cmds []tea.Cmd
	newMsg := model.Message{
		ID:             pm.MessageId,
		ChannelID:      pm.ChannelId,
		SenderID:       pm.SenderId,
		SenderUsername: pm.SenderUsername,
		Content:        pm.Content,
		Medias:         pm.Medias,
		CreatedAt:      pm.CreatedAt.AsTime(),
	}

	m.messages[pm.ChannelId] = mergeMessages(m.messages[pm.ChannelId], []model.Message{newMsg})

	if pm.ChannelId == m.activeChannelID {
		m.refreshViewportContents()
		m.viewport.GotoBottom()
		// Auto-mark as read if we are actively viewing this channel
		cmds = append(cmds, m.markAsReadCmd(pm.ChannelId, pm.MessageId))

		if ch, ok := m.channels[pm.ChannelId]; ok {
			ch.LastReadID = pm.MessageId
			ch.LastMessageID = pm.MessageId
			m.channels[pm.ChannelId] = ch
		}
	} else {
		if ch, ok := m.channels[pm.ChannelId]; ok {
			ch.LastMessageID = pm.MessageId
			ch.UnreadCount++
			m.channels[pm.ChannelId] = ch

			// Update sidebar item
			items := m.sidebar.Items()
			for i, item := range items {
				cli := item.(channelListItem)
				if cli.ch.ID == pm.ChannelId {
					cli.ch = ch // sync updated channel with IDs and count
					cmds = append(cmds, m.sidebar.SetItem(i, cli))
					break
				}
			}
		}
	}
	return tea.Batch(cmds...)
}

func (m ChatScreen) markAsReadCmd(channelID, messageID string) tea.Cmd {
	return func() tea.Msg {
		id := messageID
		if id == "" {
			if ch, ok := m.channels[channelID]; ok {
				id = ch.LastMessageID
			}
		}
		if id == "" {
			return nil
		}

		_, _ = m.client.MarkAsRead(context.Background(), &chatv1.MarkAsReadRequest{
			ChannelId: channelID,
			MessageId: id,
		})
		return nil
	}
}

func (m ChatScreen) sendMessageCmd(content string) tea.Cmd {
	return func() tea.Msg {
		// Optimistically send message. Handled via stream later.
		_, err := m.client.SendMessage(context.Background(), &chatv1.SendMessageRequest{
			ChannelId: m.activeChannelID,
			Content:   content,
		})
		if err != nil {
			return model.MsgError{Err: err}
		}
		return nil
	}
}

func (m *ChatScreen) refreshViewportContents() {
	RefreshViewport(&m.viewport, m.messages[m.activeChannelID], m.userID, m.viewport.Width, m.newMessagesMarkerID)
	m.viewport.GotoBottom()
}

// --- Helpers ---

func mergeMessages(base, incoming []model.Message) []model.Message {
	if len(incoming) == 0 {
		return base
	}

	seen := make(map[string]bool)
	var uniqueBase []model.Message
	for _, m := range base {
		if m.ID != "" {
			seen[m.ID] = true
		}
		uniqueBase = append(uniqueBase, m)
	}

	needsSort := false
	for _, m := range incoming {
		if m.ID != "" {
			if seen[m.ID] {
				continue
			}
			seen[m.ID] = true
		}
		uniqueBase = append(uniqueBase, m)
		needsSort = true
	}

	if !needsSort {
		return base
	}

	// Only sort if we actually added something new.
	// Optimization: If we know incoming is LATEST, we could just append,
	// but a sort here ensures sanity across history loads/streams.
	sort.Slice(uniqueBase, func(i, j int) bool {
		return uniqueBase[i].CreatedAt.Before(uniqueBase[j].CreatedAt)
	})

	return uniqueBase
}
