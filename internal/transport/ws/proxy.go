package ws

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// StreamHandler defines how to handle a specific streaming RPC
type StreamHandler struct {
	NewRequest  func() proto.Message
	NewResponse func() proto.Message
	Call        func(ctx context.Context, conn grpc.ClientConnInterface) (grpc.ClientStream, error)
}

type Bridge struct {
	conn     grpc.ClientConnInterface
	handlers map[string]StreamHandler
}

func NewProxy() *Bridge {
	return &Bridge{
		handlers: make(map[string]StreamHandler),
	}
}

func (b *Bridge) SetConn(conn grpc.ClientConnInterface) {
	b.conn = conn
}

func (b *Bridge) RegisterHandler(method string, h StreamHandler) {
	b.handlers[method] = h
}

func (b *Bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if b.conn == nil {
		http.Error(w, "Bridge not initialized", http.StatusInternalServerError)
		return
	}

	// Determine method to route to from URL path
	method := r.URL.Path
	if method == "" || method == "/" {
		http.Error(w, "Missing method in path (use /Service/Method)", http.StatusBadRequest)
		return
	}

	handler, ok := b.handlers[method]
	if !ok {
		http.Error(w, fmt.Sprintf("Unsupported method: %s", method), http.StatusBadRequest)
		return
	}

	// Extract session token BEFORE Upgrade (to allow returning HTTP 401)
	token := r.Header.Get("x-session-token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		if cookie, err := r.Cookie("x-session-token"); err == nil {
			token = cookie.Value
		}
	}

	if token == "" {
		http.Error(w, "Unauthorized: session token required", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Prepare gRPC context
	md := metadata.Pairs("x-session-token", token)
	ctx, cancel := context.WithCancel(metadata.NewOutgoingContext(r.Context(), md))
	defer cancel()

	stream, err := handler.Call(ctx, b.conn)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": "failed to connect to stream: " + err.Error()})
		return
	}

	errCh := make(chan error, 2)

	// Browser -> Server
	go func() {
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			req := handler.NewRequest()
			if err := protojson.Unmarshal(p, req); err != nil {
				continue
			}
			if err := stream.SendMsg(req); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Server -> Browser
	go func() {
		for {
			resp := handler.NewResponse()
			err := stream.RecvMsg(resp)
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			
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
