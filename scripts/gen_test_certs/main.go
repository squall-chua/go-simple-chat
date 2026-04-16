package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go-simple-chat/internal/crypto"
)

func main() {
	certDir := flag.String("dir", "certs", "Directory containing CA certificates")
	username := flag.String("username", "testuser", "Username for the certificate")
	flag.Parse()

	// 1. Initialize CA (loads existing or creates new)
	ca, err := crypto.NewCA(*certDir)
	if err != nil {
		log.Fatalf("failed to init CA: %v", err)
	}

	// 2. Issue user certificate
	certPEM, keyPEM, err := ca.IssueUserCert(*username, nil)
	if err != nil {
		log.Fatalf("failed to issue cert: %v", err)
	}

	// 3. Save to files
	certFile := *username + ".crt"
	keyFile := *username + ".key"

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		log.Fatalf("failed to write cert: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		log.Fatalf("failed to write key: %v", err)
	}

	fmt.Printf("Successfully generated certificates for %s:\n", *username)
	fmt.Printf("- %s\n", certFile)
	fmt.Printf("- %s\n", keyFile)
	fmt.Printf("\nYou can now run the client with:\n")
	fmt.Printf("./chat-client --cert %s --key %s\n", certFile, keyFile)
}
