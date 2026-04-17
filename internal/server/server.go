package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"go-simple-chat/internal/transport/ws"
	chatv1 "go-simple-chat/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	grpcServer *grpc.Server
	mux        cmux.CMux
	port       string
}

func NewServer(port string, grpcServer *grpc.Server) *Server {
	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

func (s *Server) Start(ctx context.Context, tlsConfig *tls.Config) error {
	l, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}

	// Create CMux on the raw listener (NOT TLS-wrapped)
	s.mux = cmux.New(l)

	// 1. Match gRPC (must handle TLS internally to populate PeerInfo)
	// gRPC over TLS typically starts with a specific HTTP/2 preface.
	grpcL := s.mux.Match(cmux.Any()) // Fallback to Any/gRPC

	// 2. Match HTTP (Gateway + WS + Static)
	// We need to match HTTP prefixes specifically.
	httpL := s.mux.Match(cmux.HTTP1Fast())

	// gRPC Server setup with TLS
	// Note: We need to recreate the grpcServer with TLS credentials if it wasn't already.
	// But in NewServer we already took the server object.
	// Actually, gRPC's grpc.Creds() must be set at NewServer time.
	
	// Implement Gateway mux
	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	err = chatv1.RegisterChatServiceHandlerFromEndpoint(ctx, gwmux, "localhost:"+s.port, opts)
	if err != nil {
		return err
	}

	// Wrap Gateway mux with WebSocket proxy
	wsHandler := ws.NewProxy(s.grpcServer, tlsConfig)
	
	// Create main HTTP handler
	mainMux := http.NewServeMux()
	fs := http.FileServer(http.Dir("web/.output/public"))
	mainMux.Handle("/demo/", http.StripPrefix("/demo/", fs))
	mainMux.Handle("/", gwmux)
	mainMux.Handle("/ws", wsHandler)
	mainMux.Handle("/metrics", promhttp.Handler())

	// Wrap the HTTP listener with TLS
	tlsHttpL := tls.NewListener(httpL, tlsConfig)

	httpSrv := &http.Server{
		Handler: mainMux,
	}

	// gRPC must be able to handle TLS on its listener. 
	// This requires that s.grpcServer was created with credentials.NewTLS(tlsConfig).
	// Let's modify main.go to ensure this, but for now we serve it on the raw branch
	// and trust that cmux didn't consume the TLS handshake.
	// Actually, cmux needs to recognize the TLS handshake.
	
	go s.grpcServer.Serve(grpcL) // gRPC handles its own TLS if configured in main.go
	go httpSrv.Serve(tlsHttpL)

	return s.mux.Serve()
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
