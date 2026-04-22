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

	// Build message content
	content := msg.Content

	// Append media if any
	if len(msg.Medias) > 0 {
		var mediaStrs []string
		for _, m := range msg.Medias {
			icon := "[📎 File]"
			switch strings.ToLower(m.Type) {
			case "image":
				icon = "[📷 Image]"
			case "video":
				icon = "[🎥 Video]"
			case "audio":
				icon = "[🎵 Audio]"
			case "pdf", "text", "document":
				icon = "[📄 Doc]"
			}
			
			name := m.Name
			if name == "" {
				name = "Attachment"
			}
			
			// Format: [icon] Name (URL)
			detail := fmt.Sprintf("%s %s %s", 
				model.StyleSpecial.Render(icon),
				model.StyleBold.Render(name),
				model.StyleMuted.Render("("+m.Url+")"),
			)
			mediaStrs = append(mediaStrs, detail)
		}
		if content != "" {
			content += "\n  "
		}
		content += strings.Join(mediaStrs, "\n  ")
	}

	var res string
	if msg.SenderID == selfID {
		res = model.StyleOwnMsg.Render(
			fmt.Sprintf("%s\n  %s",
				headerStyle.Render(model.StyleMuted.Render(header)),
				content,
			),
		)
	} else {
		res = model.StyleOtherMsg.Render(
			fmt.Sprintf("%s\n  %s",
				headerStyle.Render(model.StyleSender.Render(sender)+" "+model.StyleTimestamp.Render("["+timestamp+"]")),
				content,
			),
		)
	}

	return res + "\n"
}

// RefreshViewport updates the viewport model with formatted messages with semi-virtualization.
func RefreshViewport(vp *viewport.Model, messages []model.Message, selfID string, width int, markerID string) {
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
	
	if markerID == "MARKER_TOP" {
		b.WriteString(renderMarker(width) + "\n\n")
	}

	for i := start; i < len(messages); i++ {
		b.WriteString(FormatMessage(messages[i], selfID, width) + "\n")
		if messages[i].ID == markerID && markerID != "" && markerID != "MARKER_TOP" {
			b.WriteString("\n" + renderMarker(width) + "\n\n")
		}
	}

	vp.SetContent(b.String())
}

func renderMarker(width int) string {
	const label = " New Messages "
	sideLen := (width - len(label)) / 2
	if sideLen < 0 {
		sideLen = 0
	}
	line := strings.Repeat("─", sideLen)
	return model.StyleNewMessagesMarker.Render(line + label + line)
}
