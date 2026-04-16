package ws

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
)

// NewProxy creates a new WebSocket to gRPC proxy handler.
// It uses the wsproxy middleware to wrap the gRPC-Gateway mux or gRPC server directly.
func NewProxy(grpcServer *grpc.Server, tlsConfig *tls.Config) http.Handler {
	p := wsproxy.WebsocketProxy(grpcServer,
		wsproxy.WithTokenCookieName("auth_token"),
		wsproxy.WithForwardedHeaders(func(header string) bool {
			h := strings.ToLower(header)
			return h == "x-client-cert" || h == "x-internal-client-cert"
		}),
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If we have a client certificate, forward it in a header so the gRPC handler can see it.
		// Use a dedicated header that we trust internally.
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			cert := r.TLS.PeerCertificates[0]
			// We can pass the raw DER in base64
			encoded := base64.StdEncoding.EncodeToString(cert.Raw)
			r.Header.Set("X-Internal-Client-Cert", encoded)
		}
		p.ServeHTTP(w, r)
	})
}

// Note: In Task 8, we will integrate this into the main cmux listener.
// The wsproxy will handle the "Upgrade: websocket" header and bridge it
// to the gRPC streams automatically.
