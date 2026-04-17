package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go-simple-chat/cmd/client/model"
)

type LaunchScreen struct {
	certInput textinput.Model
	keyInput  textinput.Model
	focus     int // 0: cert, 1: key, 2: submit
	err       string
	width     int
	height    int
}

func NewLaunchScreen(defaultCert, defaultKey string) LaunchScreen {
	c := textinput.New()
	c.Placeholder = "path/to/user.crt"
	c.SetValue(defaultCert)
	c.Focus()

	k := textinput.New()
	k.Placeholder = "path/to/user.key"
	k.SetValue(defaultKey)

	return LaunchScreen{
		certInput: c,
		keyInput:  k,
		focus:     0,
	}
}

func (m LaunchScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (m LaunchScreen) Update(msg tea.Msg) (LaunchScreen, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.focus--
			} else {
				m.focus++
			}

			if m.focus > 2 {
				m.focus = 0
			} else if m.focus < 0 {
				m.focus = 2
			}

			cmds := make([]tea.Cmd, 2)
			if m.focus == 0 {
				cmds[0] = m.certInput.Focus()
				m.keyInput.Blur()
			} else if m.focus == 1 {
				m.certInput.Blur()
				cmds[0] = m.keyInput.Focus()
			} else {
				m.certInput.Blur()
				m.keyInput.Blur()
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if m.focus == 2 {
				return m.validateAndContinue()
			}
			m.focus = 2
			m.certInput.Blur()
			m.keyInput.Blur()

		case "r":
			if m.focus != 0 && m.focus != 1 {
				return m, func() tea.Msg {
					return model.MsgSwitchScreen{Screen: model.ScreenRegister}
				}
			}
		case "n":
			if m.focus != 0 && m.focus != 1 {
				return m, func() tea.Msg {
					return model.MsgSwitchScreen{Screen: model.ScreenRenew}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.focus == 0 {
		m.certInput, cmd = m.certInput.Update(msg)
	} else if m.focus == 1 {
		m.keyInput, cmd = m.keyInput.Update(msg)
	}

	return m, cmd
}

func (m LaunchScreen) validateAndContinue() (LaunchScreen, tea.Cmd) {
	certPath := m.certInput.Value()
	keyPath := m.keyInput.Value()

	if certPath == "" || keyPath == "" {
		m.err = "Both paths are required."
		return m, nil
	}

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		m.err = fmt.Sprintf("Cert file not found: %s", certPath)
		return m, nil
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		m.err = fmt.Sprintf("Key file not found: %s", keyPath)
		return m, nil
	}

	// Success - navigate to Chat
	return m, func() tea.Msg {
		// We'll need a way to pass these paths to the Root model
		// For now, Root model will have access to these from the msg if we define it so
		return MsgIdentityLoaded{CertPath: certPath, KeyPath: keyPath}
	}
}

type MsgIdentityLoaded struct {
	CertPath string
	KeyPath  string
}

func (m LaunchScreen) View() string {
	var b strings.Builder

	b.WriteString(model.StyleTitle.Render("Go-Simple-Chat Launcher") + "\n\n")
	b.WriteString("Please provide your mTLS identity files:\n\n")

	// Cert Input
	certStyle := lipgloss.NewStyle().PaddingLeft(2)
	if m.focus == 0 {
		certStyle = certStyle.Foreground(model.ColorHighlight)
	}
	b.WriteString(certStyle.Render("Certificate Path:") + "\n")
	b.WriteString(m.certInput.View() + "\n\n")

	// Key Input
	keyStyle := lipgloss.NewStyle().PaddingLeft(2)
	if m.focus == 1 {
		keyStyle = keyStyle.Foreground(model.ColorHighlight)
	}
	b.WriteString(keyStyle.Render("Private Key Path:") + "\n")
	b.WriteString(m.keyInput.View() + "\n\n")

	// Submit Button
	submitStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.NormalBorder()).
		MarginTop(1)
	if m.focus == 2 {
		submitStyle = submitStyle.BorderForeground(model.ColorHighlight).Foreground(model.ColorHighlight)
	}
	b.WriteString(submitStyle.Render("CONNECT") + "\n\n")

	if m.err != "" {
		b.WriteString(model.StyleStatusError.Render("Error: "+m.err) + "\n\n")
	}

	footer := lipgloss.NewStyle().Foreground(model.ColorMuted).MarginTop(2)
	b.WriteString(footer.Render("Tab/Arrows to navigate • [r] register new user • [n] renew certificates • [Esc] quit"))

	// Center everything
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}
