package main

import (
	"fmt"
	"log"
	"os"

	"go-simple-chat/internal/crypto"
)

func main() {
	ca, err := crypto.NewCA("certs")
	if err != nil {
		log.Fatal(err)
	}

	cert, key, err := ca.IssueUserCert("testuser", nil)
	if err != nil {
		log.Fatal(err)
	}

	os.WriteFile("testuser.crt", cert, 0644)
	os.WriteFile("testuser.key", key, 0600)

	fmt.Println("Generated testuser.crt and testuser.key")
}
