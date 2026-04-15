package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	chatv1 "go-simple-chat/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	serverAddr := flag.String("addr", "localhost:8080", "Server address")
	caFile := flag.String("ca", "certs/ca.crt", "CA certificate file")
	certFile := flag.String("cert", "testuser.crt", "Client certificate file")
	keyFile := flag.String("key", "testuser.key", "Client key file")
	flag.Parse()

	// 1. Load mTLS credentials
	certificate, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		log.Fatalf("could not load client key pair: %s", err)
	}

	capool := x509.NewCertPool()
	ca, err := os.ReadFile(*caFile)
	if err != nil {
		log.Fatalf("could not read ca certificate: %s", err)
	}
	if ok := capool.AppendCertsFromPEM(ca); !ok {
		log.Fatal("failed to append ca certs")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		RootCAs:      capool,
	}

	// 2. Connect to server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := chatv1.NewChatServiceClient(conn)

	// 3. Start chat stream
	stream, err := client.BidiStreamChat(context.Background())
	if err != nil {
		log.Fatalf("could not start stream: %v", err)
	}

	// 4. Recevier loop
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("stream receive error: %v", err)
				return
			}
			switch payload := res.Payload.(type) {
			case *chatv1.StreamMessageResponse_MessageReceived:
				fmt.Printf("\r[%s] %s: %s\n> ", payload.MessageReceived.ChannelId, payload.MessageReceived.SenderId, payload.MessageReceived.Content)
			}
		}
	}()

	// 5. Sender loop
	fmt.Print("Connected. Type message and press Enter.\n> ")
	for {
		var content string
		fmt.Scanln(&content)
		if content == "" {
			continue
		}

		err := stream.Send(&chatv1.StreamMessageRequest{
			Payload: &chatv1.StreamMessageRequest_SendMessage{
				SendMessage: &chatv1.SendMessageRequest{
					ChannelId: "global",
					Content:   content,
				},
			},
		})
		if err != nil {
			log.Printf("send error: %v", err)
		}
		fmt.Print("> ")
	}
}
