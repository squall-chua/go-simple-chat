package model

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorSubtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	ColorHighlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	ColorSpecial   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	ColorError     = lipgloss.AdaptiveColor{Light: "#FF4D4D", Dark: "#FF6666"}
	ColorMuted     = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}

	// Text Styles
	StyleTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	StyleSpecial = lipgloss.NewStyle().
			Foreground(ColorSpecial)

	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			PaddingBottom(1)

	StyleSender = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	StyleTimestamp = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Italic(true)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleUnread = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSpecial)

	// Layout Styles
	StyleApp = lipgloss.NewStyle().
			Padding(1, 2)

	StyleSidebar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, true, false, false).
			BorderForeground(ColorSubtle).
			PaddingRight(2)

	StyleMainContent = lipgloss.NewStyle().
				PaddingLeft(1)

	StyleInputContainer = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(ColorSubtle).
				PaddingTop(1)

	// Presence Icons
	StyleOnline  = lipgloss.NewStyle().Foreground(ColorSpecial).SetString("●")
	StyleOffline = lipgloss.NewStyle().Foreground(ColorMuted).SetString("○")

	// Status/Toast Styles
	StyleStatusOK    = lipgloss.NewStyle().Foreground(ColorSpecial)
	StyleStatusError = lipgloss.NewStyle().Foreground(ColorError)

	// Message Alignment
	StyleOwnMsg = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, false, true).
			BorderForeground(ColorHighlight).
			PaddingLeft(1)

	StyleOtherMsg = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, false, true).
			BorderForeground(ColorSubtle).
			PaddingLeft(1)

	StyleHighlight = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	// Misc
	Divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(ColorSubtle).
		String()
)
