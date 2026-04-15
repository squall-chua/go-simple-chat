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

	// Use TLS listener for all muxed traffic except metrics if we want,
	// but cmux works best on a raw listener and then we wrap if needed.
	// Actually, for mTLS, we wrap the whole listener.
	tlsListener := tls.NewListener(l, tlsConfig)
	s.mux = cmux.New(tlsListener)

	// 1. Match gRPC
	grpcL := s.mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))

	// 2. Match HTTP/1.1 (Gateway + WS)
	httpL := s.mux.Match(cmux.HTTP1Fast())

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
	
	// Serve static web files
	fs := http.FileServer(http.Dir("web/.output/public"))
	mainMux.Handle("/demo/", http.StripPrefix("/demo/", fs))
	
	mainMux.Handle("/", gwmux)
	mainMux.Handle("/ws", wsHandler) // WS endpoint
	mainMux.Handle("/metrics", promhttp.Handler())

	httpSrv := &http.Server{
		Handler: mainMux,
	}

	// Start servers
	go s.grpcServer.Serve(grpcL)
	go httpSrv.Serve(httpL)

	return s.mux.Serve()
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
