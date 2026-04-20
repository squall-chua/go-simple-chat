package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"go-simple-chat/internal/transport/ws"
	"go-simple-chat/internal/service"
	chatv1 "go-simple-chat/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

type Server struct {
	grpcServer     *grpc.Server
	port           string
	sessionHandler http.Handler
	uploadService  *service.UploadService
	sessionService *service.SessionService
	httpServer     *http.Server
}

func NewServer(port string, grpcServer *grpc.Server) *Server {
	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

func (s *Server) SetSessionHandler(h http.Handler) {
	s.sessionHandler = h
}

func (s *Server) SetUploadService(us *service.UploadService, ss *service.SessionService) {
	s.uploadService = us
	s.sessionService = ss
}

func (s *Server) Start(ctx context.Context, tlsConfig *tls.Config) error {
	l, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}

	// 1. Setup gRPC Gateway
	gwmux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			case "x-session-token":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	
	// Ensure the gateway client doesn't present the server cert as its own identity
	gwDialTLS := tlsConfig.Clone()
	gwDialTLS.Certificates = nil
	gwDialTLS.InsecureSkipVerify = true
	
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(gwDialTLS))}
	err = chatv1.RegisterChatServiceHandlerFromEndpoint(ctx, gwmux, "localhost:"+s.port, opts)
	if err != nil {
		return err
	}

	// 2. Setup WebSocket Proxy
	// The bridge needs a gRPC client to talk to the local server
	wsBridge := ws.NewProxy(nil, tlsConfig).(*ws.Bridge)
	
	// 3. Main HTTP Router
	mainMux := http.NewServeMux()
	if s.sessionHandler != nil {
		mainMux.Handle("/api/session", s.sessionHandler)
	}
	mainMux.Handle("/metrics", promhttp.Handler())
	mainMux.Handle("/", gwmux)

	// 4. Multiplexing Handler
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle WebSockets First
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			// Lazily initialize the client if needed
			// We dial localhost using the same TLS config we're serving with
			// but we MUST clear the Certificates so the bridge doesn't present the 
			// server cert as its own identity. We also skip verification for the self-dial.
			dialTLS := tlsConfig.Clone()
			dialTLS.Certificates = nil
			dialTLS.InsecureSkipVerify = true 
			
			conn, err := grpc.Dial("localhost:"+s.port, grpc.WithTransportCredentials(credentials.NewTLS(dialTLS)))
			if err == nil {
				wsBridge.SetClient(chatv1.NewChatServiceClient(conn))
			}
			
			wsBridge.ServeHTTP(w, r)
			return
		}

		// Handle gRPC Second
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			s.grpcServer.ServeHTTP(w, r)
			return
		}

		// Handle Standard HTTP Third
		allowCORS(mainMux).ServeHTTP(w, r)
	})

	s.httpServer = &http.Server{
		Addr:      ":" + s.port,
		Handler:   rootHandler,
		TLSConfig: tlsConfig,
	}

	// Start serving on the TLS listener
	tlsListener := tls.NewListener(l, tlsConfig)
	return s.httpServer.Serve(tlsListener)
}

func (s *Server) Stop() {
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
}

func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-session-token")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}
