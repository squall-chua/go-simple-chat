package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go-simple-chat/cmd/client/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "server address")
	caPath := flag.String("ca", "certs/ca.crt", "path to CA certificate")
	certPath := flag.String("cert", "user.crt", "path to client certificate")
	keyPath := flag.String("key", "user.key", "path to client private key")
	flag.Parse()

	// Initialize the Root Model which manages the app state and screens
	m := ui.NewRootModel(*addr, *caPath, *certPath, *keyPath)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Optional: Configure logging to a file to avoid messing up the TUI
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	_ = f
}
