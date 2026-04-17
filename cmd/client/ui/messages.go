package ui

import (
	"fmt"
	"strings"
	"time"

	"go-simple-chat/cmd/client/model"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Message Alignment
	StyleOwnMsg = lipgloss.NewStyle()

	StyleOtherMsg = lipgloss.NewStyle()

	headerStyle    = lipgloss.NewStyle().PaddingLeft(2)
	mediaIconStyle = lipgloss.NewStyle() // Placeholder for special
)

// RenderMessages renders the message viewport.
func RenderMessages(vp viewport.Model, messages []model.Message, selfID string, width int) string {
	return vp.View()
}

// FormatMessage applies styles to a single message based on sender.
func FormatMessage(msg model.Message, selfID string, width int) string {
	now := time.Now()
	isToday := msg.CreatedAt.Year() == now.Year() && msg.CreatedAt.YearDay() == now.YearDay()

	format := "03:04 PM"
	if !isToday {
		format = "2004-12-31 03:04 PM"
	}
	timestamp := msg.CreatedAt.Format(format)
	sender := msg.SenderUsername
	if sender == "" {
		sender = msg.SenderID
	}

	header := fmt.Sprintf("%s [%s]", sender, timestamp)

	var res string
	if msg.SenderID == selfID {
		res = model.StyleOwnMsg.Render(
			fmt.Sprintf("%s\n  %s",
				headerStyle.Render(model.StyleMuted.Render(header)),
				msg.Content,
			),
		)
	} else {
		res = model.StyleOtherMsg.Render(
			fmt.Sprintf("%s\n  %s",
				headerStyle.Render(model.StyleSender.Render(sender)+" "+model.StyleTimestamp.Render("["+timestamp+"]")),
				msg.Content,
			),
		)
	}

	// Handle media attachments
	if len(msg.Medias) > 0 {
		var mediaStrs []string
		for _, m := range msg.Medias {
			icon := "[📎 File]"
			switch m.Type {
			case "image":
				icon = "[📷 Image]"
			case "video":
				icon = "[🎥 Video]"
			case "audio":
				icon = "[🎵 Audio]"
			}
			mediaStrs = append(mediaStrs, model.StyleSpecial.Render(icon))
		}
		res += "\n" + strings.Join(mediaStrs, " ")
	}

	return res + "\n"
}

// RefreshViewport updates the viewport model with formatted messages with semi-virtualization.
func RefreshViewport(vp *viewport.Model, messages []model.Message, selfID string, width int) {
	if len(messages) == 0 {
		vp.SetContent("No messages yet...")
		return
	}

	// Limit rendering to prevent huge history from spiking CPU
	const maxRender = 150
	start := 0
	if len(messages) > maxRender {
		start = len(messages) - maxRender
	}

	var b strings.Builder
	for i := start; i < len(messages); i++ {
		b.WriteString(FormatMessage(messages[i], selfID, width) + "\n")
	}

	vp.SetContent(b.String())
}
