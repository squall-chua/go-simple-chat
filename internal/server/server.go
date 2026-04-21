package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	chatv1 "go-simple-chat/api/v1"
	"go-simple-chat/internal/service"
	"go-simple-chat/internal/transport/ws"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	grpcServer     *grpc.Server
	port           string
	publicPort     string
	serverName     string
	allowedOrigins []string
	sessionHandler http.Handler
	uploadService  *service.UploadService
	sessionService *service.SessionService
	httpServer     *http.Server
	publicServer   *http.Server
	wsConn         *grpc.ClientConn
	wsBridge       *ws.Bridge
}

func NewServer(port, publicPort, serverName string, allowedOrigins []string, grpcServer *grpc.Server) *Server {
	return &Server{
		grpcServer:     grpcServer,
		port:           port,
		publicPort:     publicPort,
		serverName:     serverName,
		allowedOrigins: allowedOrigins,
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

	// Fix TASK-04: Internal gRPC dial uses a named server cert instead of InsecureSkipVerify
	// We extract the server cert from tlsConfig
	var serverCert tls.Certificate
	if len(tlsConfig.Certificates) > 0 {
		serverCert = tlsConfig.Certificates[0]
	}

	gwDialTLS := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		RootCAs:      tlsConfig.RootCAs,
		ServerName:   s.serverName,
		MinVersion:   tls.VersionTLS13,
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(gwDialTLS))}
	err = chatv1.RegisterChatServiceHandlerFromEndpoint(ctx, gwmux, "localhost:"+s.port, opts)
	if err != nil {
		return err
	}

	s.wsBridge = ws.NewProxy()
	s.wsBridge.RegisterHandler(chatv1.ChatService_BidiStreamChat_FullMethodName, ws.StreamHandler{
		NewRequest:  func() proto.Message { return &chatv1.StreamMessageRequest{} },
		NewResponse: func() proto.Message { return &chatv1.StreamMessageResponse{} },
		Call: func(ctx context.Context, conn grpc.ClientConnInterface) (grpc.ClientStream, error) {
			return chatv1.NewChatServiceClient(conn).BidiStreamChat(ctx)
		},
	})

	wsConn, err := grpc.Dial("localhost:"+s.port, grpc.WithTransportCredentials(credentials.NewTLS(gwDialTLS)))
	if err != nil {
		return err
	}
	s.wsConn = wsConn
	s.wsBridge.SetConn(wsConn)

	// 3. Main HTTP Router
	mainMux := http.NewServeMux()
	mainMux.Handle("/", gwmux)

	// 4. Multiplexing Handler
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			s.wsBridge.ServeHTTP(w, r)
			return
		}

		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			s.grpcServer.ServeHTTP(w, r)
			return
		}

		s.allowCORS(mainMux).ServeHTTP(w, r)
	})

	s.httpServer = &http.Server{
		Addr:      ":" + s.port,
		Handler:   rootHandler,
		TLSConfig: tlsConfig,
	}

	tlsListener := tls.NewListener(l, tlsConfig)
	return s.httpServer.Serve(tlsListener)
}

func (s *Server) StartPublic(ctx context.Context, tlsConfig *tls.Config) error {
	l, err := net.Listen("tcp", ":"+s.publicPort)
	if err != nil {
		return err
	}

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

	// Dial the local port (internal communication)
	// We use the same dial config as Start()
	internalDialTLS := &tls.Config{
		Certificates: tlsConfig.Certificates,
		RootCAs:      tlsConfig.RootCAs,
		ServerName:   s.serverName,
		MinVersion:   tls.VersionTLS13,
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(internalDialTLS))}
	err = chatv1.RegisterChatServiceHandlerFromEndpoint(ctx, gwmux, "localhost:"+s.port, opts)
	if err != nil {
		return err
	}

	publicMux := http.NewServeMux()
	// Register HTTP Session Handler (Standard TLS)
	// We register both so it handles exact /api/session and subpaths like /api/session/challenge
	publicMux.Handle("/api/session", s.sessionHandler)
	publicMux.Handle("/api/session/", s.sessionHandler)
	publicMux.Handle("/metrics", promhttp.Handler())
	publicMux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" && s.wsBridge != nil {
			s.wsBridge.ServeHTTP(w, r)
			return
		}

		// Allow REST API (/v1/) and metrics
		if strings.HasPrefix(r.URL.Path, "/v1/") || r.URL.Path == "/metrics" || strings.HasPrefix(r.URL.Path, "/chat.v1.ChatService/") {
			gwmux.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	}))

	s.publicServer = &http.Server{
		Addr:      ":" + s.publicPort,
		Handler:   s.allowCORS(publicMux),
		TLSConfig: tlsConfig,
	}

	tlsListener := tls.NewListener(l, tlsConfig)
	return s.publicServer.Serve(tlsListener)
}

func (s *Server) Stop() {
	if s.wsConn != nil {
		s.wsConn.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	if s.publicServer != nil {
		s.publicServer.Shutdown(context.Background())
	}
}

func (s *Server) allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false

		if origin != "" {
			for _, o := range s.allowedOrigins {
				trimmed := strings.TrimSpace(o)
				if trimmed == "*" || trimmed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					allowed = true
					break
				}
			}
		}

		// Support fallback for internal/direct requests if needed, but usually origin is present in browsers
		if !allowed && len(s.allowedOrigins) > 0 && s.allowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-session-token, Accept, Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}
