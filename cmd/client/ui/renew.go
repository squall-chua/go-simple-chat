package ui

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/cmd/client/model"
)

type RenewScreen struct {
	client    chatv1.ChatServiceClient
	userInput textinput.Model
	keyInput  textinput.Model
	focus     int // 0: username, 1: keyPath, 2: submit
	loading   bool
	err       string
	width     int
	height    int
}

func NewRenewScreen(client chatv1.ChatServiceClient, defaultKey string) RenewScreen {
	u := textinput.New()
	u.Placeholder = "Enter username..."
	u.Focus()

	k := textinput.New()
	k.Placeholder = "path/to/existing.key"
	k.SetValue(defaultKey)

	return RenewScreen{
		client:    client,
		userInput: u,
		keyInput:  k,
		focus:     0,
	}
}

func (m RenewScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (m RenewScreen) Update(msg tea.Msg) (RenewScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg {
				return model.MsgSwitchScreen{Screen: model.ScreenLaunch}
			}
		case "tab", "shift+tab":
			if msg.String() == "tab" {
				m.focus++
			} else {
				m.focus--
			}
			if m.focus > 2 {
				m.focus = 0
			} else if m.focus < 0 {
				m.focus = 2
			}
			m.updateFocus()
			return m, nil

		case "enter":
			if m.focus == 2 && !m.loading {
				m.loading = true
				m.err = ""
				return m, m.renewCmd()
			}
			m.focus = 2
			m.updateFocus()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case renewResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		return m, func() tea.Msg {
			return MsgIdentityLoaded{
				CertPath: msg.certPath,
				KeyPath:  m.keyInput.Value(),
			}
		}
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		m.userInput, cmd = m.userInput.Update(msg)
	} else if m.focus == 1 {
		m.keyInput, cmd = m.keyInput.Update(msg)
	}

	return m, cmd
}

func (m *RenewScreen) updateFocus() {
	if m.focus == 0 {
		m.userInput.Focus()
		m.keyInput.Blur()
	} else if m.focus == 1 {
		m.userInput.Blur()
		m.keyInput.Focus()
	} else {
		m.userInput.Blur()
		m.keyInput.Blur()
	}
}

type renewResultMsg struct {
	certPath string
	err      error
}

func (m RenewScreen) renewCmd() tea.Cmd {
	return func() tea.Msg {
		username := m.userInput.Value()
		keyPath := m.keyInput.Value()

		// 1. Load private key
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return renewResultMsg{err: fmt.Errorf("failed to read key: %w", err)}
		}
		
		block, _ := pem.Decode(keyData)
		if block == nil {
			return renewResultMsg{err: errors.New("invalid key format")}
		}
		
		rawKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return renewResultMsg{err: fmt.Errorf("failed to parse key: %w", err)}
		}

		privKey, ok := rawKey.(ed25519.PrivateKey)
		if !ok {
			return renewResultMsg{err: errors.New("only Ed25519 keys are supported for renewal currently")}
		}

		// 2. Get Challenge
		res, err := m.client.GetChallenge(context.Background(), &chatv1.GetChallengeRequest{Username: username})
		if err != nil {
			return renewResultMsg{err: fmt.Errorf("failed to get challenge: %w", err)}
		}

		// 3. Sign Challenge
		challengeData := []byte("RENEW_CERT:" + res.Nonce)
		signature, err := privKey.Sign(rand.Reader, challengeData, crypto.Hash(0))
		if err != nil {
			return renewResultMsg{err: fmt.Errorf("failed to sign: %w", err)}
		}

		// 4. Call Renew
		renewRes, err := m.client.RenewCertificate(context.Background(), &chatv1.RenewCertificateRequest{
			UserId:    res.UserId,
			Signature: signature,
		})
		if err != nil {
			return renewResultMsg{err: fmt.Errorf("renewal failed: %w", err)}
		}

		// 5. Overwrite cert
		certPath := username + ".crt"
		if err := os.WriteFile(certPath, renewRes.Certificate, 0644); err != nil {
			return renewResultMsg{err: fmt.Errorf("failed to save new cert: %w", err)}
		}

		return renewResultMsg{certPath: certPath}
	}
}

func (m RenewScreen) View() string {
	var b strings.Builder

	b.WriteString(model.StyleTitle.Render("Certificate Renewal") + "\n\n")
	b.WriteString("Your certificate has expired or is invalid.\n")
	b.WriteString("Use your existing private key to prove identity and get a new one.\n\n")

	// Username Input
	uStyle := lipgloss.NewStyle()
	if m.focus == 0 { uStyle = uStyle.Foreground(model.ColorHighlight) }
	b.WriteString(uStyle.Render("Username:") + "\n")
	b.WriteString(m.userInput.View() + "\n\n")

	// Key Path Input
	kStyle := lipgloss.NewStyle()
	if m.focus == 1 { kStyle = kStyle.Foreground(model.ColorHighlight) }
	b.WriteString(kStyle.Render("Private Key Path:") + "\n")
	b.WriteString(m.keyInput.View() + "\n\n")

	if m.loading {
		b.WriteString(model.StyleSpecial.Render("Renewing...") + "\n")
	} else {
		subStyle := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.NormalBorder())
		if m.focus == 2 { subStyle = subStyle.BorderForeground(model.ColorHighlight).Foreground(model.ColorHighlight) }
		b.WriteString(subStyle.Render("RENEW CERTIFICATE") + "\n")
	}

	if m.err != "" {
		b.WriteString("\n" + model.StyleStatusError.Render("Error: "+m.err) + "\n")
	}

	b.WriteString("\n\n" + lipgloss.NewStyle().Foreground(model.ColorMuted).Render("Tab to navigate • Esc back to launcher"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}
