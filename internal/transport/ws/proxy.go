package ws

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"

	chatv1 "go-simple-chat/api/v1"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Bridge struct {
	client chatv1.ChatServiceClient
}

func NewProxy(grpcServer *grpc.Server, tlsConfig *tls.Config) http.Handler {
	// In our unified server, we can't easily dial ourselves via TLS during bootstrap efficiently,
	// so we'll use the ChatService directly if we can, or just keep the bridge simple.
	return &Bridge{} // We'll initialize the client on the first request or via a setter
}

func (b *Bridge) SetClient(c chatv1.ChatServiceClient) {
	b.client = c
}

func (b *Bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if b.client == nil {
		http.Error(w, "Bridge not initialized", http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Extract session token from headers, cookies, or query params
	token := r.Header.Get("x-session-token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		if cookie, err := r.Cookie("x-session-token"); err == nil {
			token = cookie.Value
		}
	}

	// Prepare gRPC context with metadata
	md := metadata.Pairs("x-session-token", token)
	ctx, cancel := context.WithCancel(metadata.NewOutgoingContext(r.Context(), md))
	defer cancel()

	stream, err := b.client.BidiStreamChat(ctx)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": "failed to connect to chat stream: " + err.Error()})
		return
	}

	// Bidirectional pipe
	errCh := make(chan error, 2)

	// Browser -> Server
	go func() {
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			var msg chatv1.StreamMessageRequest
			if err := protojson.Unmarshal(p, &msg); err != nil {
				// Log or handle unmarshal error?
				continue
			}
			if err := stream.Send(&msg); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Server -> Browser
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			
			// Marshal with protojson to respect snake_case (UseProtoNames: true)
			m := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
			b, err := m.Marshal(resp)
			if err != nil {
				errCh <- err
				return
			}
			
			if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
				errCh <- err
				return
			}
		}
	}()

	<-errCh
}
