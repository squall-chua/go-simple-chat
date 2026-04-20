package ui

import (
	"fmt"
	"io"

	"go-simple-chat/cmd/client/model"

	"github.com/charmbracelet/bubbles/list"
)

// channelListItem adapts our model.Channel to bubbles/list.Item
type channelListItem struct {
	ch model.Channel
}

func (i channelListItem) Title() string       { return i.ch.Name }
func (i channelListItem) Description() string { return i.ch.ID }
func (i channelListItem) FilterValue() string { return i.ch.Name }

// participantListItem adapts user info for the participant list
type participantListItem struct {
	userID   string
	username string
	online   bool
}

func (i participantListItem) Title() string       { return i.username }
func (i participantListItem) Description() string { return i.userID }
func (i participantListItem) FilterValue() string { return i.username }

// RenderSidebarContainer wraps a list in the standard sidebar style with a fixed width
func RenderSidebarContainer(view string, focused bool, width int) string {
	style := model.StyleSidebar.Copy().Width(width)
	if focused {
		style = style.BorderForeground(model.ColorHighlight)
	}
	return style.Render(view)
}

// In a more advanced implementation, we would implement list.DefaultDelegate to show
// the presence and unreads inside the list items themselves.
// Let's do that for a premium feel.

type channelDelegate struct {
	list.DefaultDelegate
	presenceMap map[string]model.PresenceInfo
	selfID      string
	focused     bool // List-level focus
}

func NewChannelDelegate(presenceMap map[string]model.PresenceInfo, selfID string) channelDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	return channelDelegate{
		DefaultDelegate: d,
		presenceMap:     presenceMap,
		selfID:          selfID,
	}
}

func (d channelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	chItem, ok := item.(channelListItem)
	if !ok {
		return
	}

	ch := chItem.ch

	// Calculate presence for this channel (Normalized to 4 chars)
	presenceStr := "    "
	if ch.Type == "direct" || ch.Type == "TYPE_DIRECT" {
		for _, pID := range ch.Participants {
			if pID == d.selfID {
				continue
			}
			if info, found := d.presenceMap[pID]; found {
				dot := model.StyleOffline.String()
				if info.Online {
					dot = model.StyleOnline.String()
				}
				presenceStr = fmt.Sprintf("  %s ", dot)
				break
			}
		}
	} else {
		onlineCount := 0
		for _, pID := range ch.Participants {
			if pID == d.selfID {
				continue
			}
			if info, found := d.presenceMap[pID]; found && info.Online {
				onlineCount++
			}
		}
		if onlineCount > 0 {
			presenceStr = fmt.Sprintf("%2d%s ", onlineCount, model.StyleOnline.String())
		}
	}

	// Unread badge logic
	isUnread := ch.LastReadID != ch.LastMessageID && ch.LastMessageID != ""

	icon := "#"
	if ch.Type == "direct" || ch.Type == "TYPE_DIRECT" {
		icon = "@"
	}

	titleText := fmt.Sprintf("%-2s%s", icon, ch.Name)
	if ch.UnreadCount > 0 {
		titleText = fmt.Sprintf("%s [%d]", titleText, ch.UnreadCount)
	}

	var title string
	if isUnread {
		title = model.StyleUnread.Render(titleText)
	} else {
		title = titleText
	}

	if index == m.Index() {
		marker := "> "
		if d.focused {
			_, _ = fmt.Fprint(w, model.StyleHighlight.Render(marker+presenceStr+title))
		} else {
			_, _ = fmt.Fprint(w, model.StyleMuted.Render(marker)+presenceStr+title)
		}
	} else {
		_, _ = fmt.Fprint(w, "  "+presenceStr+title)
	}
}

type participantDelegate struct {
	list.DefaultDelegate
	presenceMap map[string]model.PresenceInfo
}

func NewParticipantDelegate(presenceMap map[string]model.PresenceInfo) participantDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	return participantDelegate{
		DefaultDelegate: d,
		presenceMap:     presenceMap,
	}
}

func (d participantDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pItem, ok := item.(participantListItem)
	if !ok {
		return
	}

	presenceStr := model.StyleOffline.String() + " "
	if info, found := d.presenceMap[pItem.userID]; found && info.Online {
		presenceStr = model.StyleOnline.String() + " "
	}

	if index == m.Index() {
		_, _ = fmt.Fprint(w, model.StyleHighlight.Render("  "+presenceStr+"  "+pItem.username))
	} else {
		_, _ = fmt.Fprint(w, "  "+presenceStr+"  "+pItem.username)
	}
}
