# Antigravity Chat Service

A production-ready, horizontally-scalable chat service in Go using a Hexagonal Modular Monolith architecture.

## Features

- **Multiplexed Port (8080):** gRPC, gRPC-Gateway (REST), WebSockets, and Prometheus metrics on a single port via `cmux`.
- **mTLS Security & Identity:** Strict TLS 1.3 with mutual authentication and application-layer Public Key Pinning.
- **Certificate Renewal:** Secure cryptographic challenge-response protocol for renewing expired certificates without permanent lockout.
- **Multi-Media Support:** Messages support multiple attachments (Image, Video, Audio, File) per message.
- **Read State Sync:** Real-time synchronization of "Last Read" markers across all devices.
- **Scalable Broker:** Horizontal scalability via Redis; cluster-ready challenge storage in MongoDB.
- **Persistence:** MongoDB with optimized compound indexes and TTL-based authentication challenges.
- **Clients:** Go CLI and Nuxt 3 Web Dashboard.

## Quickstart

### 1. Build and Start

```bash
# Register dependencies
go mod tidy

# Build server and client
make build

# Start infrastructure (Mongo, Redis)
docker-compose up -d

# Start server
./chat-server
```

### 2. User Registration & Identity

Identity is tied to mTLS. For new users:

1. **Register**: Submit `username` and `public_key`. Receive an issued certificate.
2. **Authenticate**: Use the certificate for all subsequent mTLS handshakes.
3. **Renew**: If a certificate expires, use the **Challenge-Response** flow to prove identity via the registered public key and obtain a new certificate.

#### Generate Test User Certificates (Manual)

For rapid testing:

```bash
# Creates testuser.crt and testuser.key
go run scripts/gen_test_certs/main.go --username "myuser"
```

### 3. Running the Demo CLI

```bash
# Run the interactive CLI client
./chat-client --cert testuser.crt --key testuser.key
```

## Architecture

- **Hexagonal Modular Monolith:** Core logic decoupled from transport and storage.
- **Unified Identity:** Client certificates are the source of truth for identity (`UserID` in `CommonName`).
- **WebSocket mTLS:** The WebSocket bridge propagates TLS identities into gRPC metadata, allowing browser-based clients to maintain full mTLS security.
- **Public Key Pinning:** Prevents impersonation by verifying the certificate's public key against the user's permanent record in the database.
- **Cluster Readiness:** All short-lived authentication states (challenges) and signaling (broker) are stored in shared repositories.

## Monitoring

- **Metrics:** `https://localhost:8080/metrics`
- **Dashboard:** Grafana on port 3000; Prometheus on 9090.
