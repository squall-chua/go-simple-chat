package ui

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/cmd/client/model"
)

type RegisterScreen struct {
	client    chatv1.ChatServiceClient
	userInput textinput.Model
	loading   bool
	err       string
	width     int
	height    int
}

func NewRegisterScreen(client chatv1.ChatServiceClient) RegisterScreen {
	u := textinput.New()
	u.Placeholder = "Enter desired username..."
	u.Focus()

	return RegisterScreen{
		client:    client,
		userInput: u,
	}
}

func (m RegisterScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (m RegisterScreen) Update(msg tea.Msg) (RegisterScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg {
				return model.MsgSwitchScreen{Screen: model.ScreenLaunch}
			}
		case "enter":
			if m.userInput.Value() != "" && !m.loading {
				m.loading = true
				m.err = ""
				return m, m.registerCmd()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case regResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		// Success! Switch back to launch or chat with new paths
		return m, func() tea.Msg {
			return MsgIdentityLoaded{
				CertPath: msg.certPath,
				KeyPath:  msg.keyPath,
			}
		}
	}

	var cmd tea.Cmd
	m.userInput, cmd = m.userInput.Update(msg)
	return m, cmd
}

type regResultMsg struct {
	certPath string
	keyPath  string
	err      error
}

func (m RegisterScreen) registerCmd() tea.Cmd {
	return func() tea.Msg {
		username := m.userInput.Value()

		// 1. Generate local keypair
		pubKey, privKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return regResultMsg{err: fmt.Errorf("failed to generate keys: %w", err)}
		}

		pubDER, _ := x509.MarshalPKIXPublicKey(pubKey)

		// 2. Call Register RPC
		res, err := m.client.Register(context.Background(), &chatv1.RegisterRequest{
			Username:  username,
			PublicKey: pubDER,
		})
		if err != nil {
			return regResultMsg{err: fmt.Errorf("registration failed: %w", err)}
		}

		// 3. Save results
		certFile := username + ".crt"
		keyFile := username + ".key"

		if err := os.WriteFile(certFile, res.Certificate, 0644); err != nil {
			return regResultMsg{err: fmt.Errorf("failed to save cert: %w", err)}
		}

		privBytes, _ := x509.MarshalPKCS8PrivateKey(privKey)
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
		if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
			return regResultMsg{err: fmt.Errorf("failed to save key: %w", err)}
		}

		return regResultMsg{certPath: certFile, keyPath: keyFile}
	}
}

func (m RegisterScreen) View() string {
	var b strings.Builder

	b.WriteString(model.StyleTitle.Render("User Registration") + "\n\n")
	b.WriteString("Choose a username to join Go-Simple-Chat.\n")
	b.WriteString("We will generate your secure mTLS identity locally.\n\n")

	if m.loading {
		b.WriteString(model.StyleSpecial.Render("Registering...") + "\n")
	} else {
		b.WriteString(m.userInput.View() + "\n")
	}

	if m.err != "" {
		b.WriteString("\n" + model.StyleStatusError.Render("Error: "+m.err) + "\n")
	}

	b.WriteString("\n\n" + lipgloss.NewStyle().Foreground(model.ColorMuted).Render("Enter to register • Esc back to launcher"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}
