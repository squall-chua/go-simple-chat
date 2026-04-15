package ws

import (
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
)

// NewProxy creates a new WebSocket to gRPC proxy handler.
// It uses the wsproxy middleware to wrap the gRPC-Gateway mux or gRPC server directly.
func NewProxy(grpcServer *grpc.Server, tlsConfig *tls.Config) http.Handler {
	// The proxy needs to know how to connect to the gRPC server.
	// Since we are running in the same process, we can use the wsproxy's ability
	// to wrap a http.Handler.
	
	return wsproxy.WebsocketProxy(grpcServer, 
		wsproxy.WithTokenCookieName("auth_token"),
		wsproxy.WithForwardedHeaders(func(header string) bool {
			return strings.ToLower(header) == "x-client-cert"
		}),
	)
}

// Note: In Task 8, we will integrate this into the main cmux listener.
// The wsproxy will handle the "Upgrade: websocket" header and bridge it
// to the gRPC streams automatically.
