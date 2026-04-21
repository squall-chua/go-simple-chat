package ui

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/cmd/client/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type RootModel struct {
	serverAddr string
	caFile     string
	
	conn       *grpc.ClientConn
	client     chatv1.ChatServiceClient
	
	screen     model.Screen
	
	// Child Models
	launch   LaunchScreen
	register RegisterScreen
	renew    RenewScreen
	chat     ChatScreen
	
	width  int
	height int
}

func NewRootModel(serverAddr, caFile, defaultCert, defaultKey string) RootModel {
	return RootModel{
		serverAddr: serverAddr,
		caFile:     caFile,
		screen:     model.ScreenLaunch,
		launch:     NewLaunchScreen(defaultCert, defaultKey),
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.launch.Init()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Dimensions will be propagated to the active screen in the switch below

	case model.MsgSwitchScreen:
		m.screen = msg.Screen
		return m.handleScreenSwitch()

	case MsgIdentityLoaded:
		// User has chosen an identity (from launch, register, or renew)
		// Now we dial with mTLS and switch to ChatScreen
		return m.dialAndConnectChat(msg.CertPath, msg.KeyPath)

	}

	// Route updates to the active screen
	switch m.screen {
	case model.ScreenLaunch:
		m.launch, cmd = m.launch.Update(msg)
	case model.ScreenRegister:
		m.register, cmd = m.register.Update(msg)
	case model.ScreenRenew:
		m.renew, cmd = m.renew.Update(msg)
	case model.ScreenChat:
		m.chat, cmd = m.chat.Update(msg)
	}

	return m, cmd
}

func (m RootModel) View() string {
	switch m.screen {
	case model.ScreenLaunch:
		return m.launch.View()
	case model.ScreenRegister:
		return m.register.View()
	case model.ScreenRenew:
		return m.renew.View()
	case model.ScreenChat:
		return m.chat.View()
	default:
		return "Unknown screen state"
	}
}

func (m *RootModel) handleScreenSwitch() (tea.Model, tea.Cmd) {
	// Initialize the connection if not already present for registration/renewal
	if (m.screen == model.ScreenRegister || m.screen == model.ScreenRenew) && m.client == nil {
		err := m.initBasicConnection()
		if err != nil {
			// fallback or error msg
			return m, func() tea.Msg { return model.MsgError{Err: err} }
		}
	}

	var cmd tea.Cmd
	switch m.screen {
	case model.ScreenLaunch:
		cmd = m.launch.Init()
	case model.ScreenRegister:
		m.register = NewRegisterScreen(m.client)
		m.register.width, m.register.height = m.width, m.height
		cmd = m.register.Init()
	case model.ScreenRenew:
		m.renew = NewRenewScreen(m.client, m.launch.keyInput.Value())
		m.renew.width, m.renew.height = m.width, m.height
		cmd = m.renew.Init()
	}
	return m, cmd
}

func (m *RootModel) initBasicConnection() error {
	capool := x509.NewCertPool()
	ca, err := os.ReadFile(m.caFile)
	if err != nil { return err }
	capool.AppendCertsFromPEM(ca)

	host, _, _ := net.SplitHostPort(m.serverAddr)
	tlsConfig := &tls.Config{
		RootCAs:    capool,
		ServerName: host,
		MinVersion: tls.VersionTLS13,
	}

	conn, err := grpc.Dial(m.serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil { return err }
	
	m.conn = conn
	m.client = chatv1.NewChatServiceClient(conn)
	return nil
}

func (m *RootModel) dialAndConnectChat(certPath, keyPath string) (tea.Model, tea.Cmd) {
	capool := x509.NewCertPool()
	ca, _ := os.ReadFile(m.caFile)
	capool.AppendCertsFromPEM(ca)

	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return m, func() tea.Msg { return model.MsgError{Err: err} }
	}

	host, _, _ := net.SplitHostPort(m.serverAddr)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		RootCAs:      capool,
		ServerName:   host,
		MinVersion:   tls.VersionTLS13,
	}

	conn, err := grpc.Dial(m.serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return m, func() tea.Msg { return model.MsgError{Err: err} }
	}

	// Extract UserID from cert
	x509Cert, _ := x509.ParseCertificate(certificate.Certificate[0])
	userID := x509Cert.Subject.CommonName

	m.conn = conn
	m.client = chatv1.NewChatServiceClient(conn)
	m.chat = NewChatScreen(m.client, userID)
	m.chat.width = m.width
	m.chat.height = m.height
	m.chat.UpdateLayout()
	m.screen = model.ScreenChat

	return m, m.chat.Init()
}
