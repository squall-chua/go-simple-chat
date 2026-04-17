package main

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"log"
	"os"

	chatv1 "go-simple-chat/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	username := flag.String("username", "developer", "Username to register")
	serverAddr := flag.String("addr", "localhost:8080", "Server address")
	caFile := flag.String("ca", "certs/ca.crt", "CA certificate file")
	flag.Parse()

	// 1. Generate local keypair (Ed25519)
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Connect to server
	capool := x509.NewCertPool()
	ca, _ := os.ReadFile(*caFile)
	capool.AppendCertsFromPEM(ca)

	tlsConfig := &tls.Config{
		RootCAs: capool,
	}

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := chatv1.NewChatServiceClient(conn)

	// 3. Register
	pubDER, _ := x509.MarshalPKIXPublicKey(pubKey)
	res, err := client.Register(context.Background(), &chatv1.RegisterRequest{
		Username:  *username,
		PublicKey: pubDER,
	})
	if err != nil {
		log.Fatalf("registration failed: %v", err)
	}

	// 4. Save results
	certFile := *username + ".crt"
	keyFile := *username + ".key"

	os.WriteFile(certFile, res.Certificate, 0644)
	
	// Save the private key we generated locally
	privBytes, _ := x509.MarshalPKCS8PrivateKey(privKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	os.WriteFile(keyFile, keyPEM, 0600)

	log.Printf("Successfully registered %s", *username)
	log.Printf("UserID: %s", res.UserId)
}
