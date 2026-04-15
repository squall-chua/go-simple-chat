# Antigravity Chat Service

A production-ready, horizontally-scalable chat service in Go using a Hexagonal Modular Monolith architecture.

## Features

- **Multiplexed Port (8080):** gRPC, gRPC-Gateway (REST), WebSockets, and Prometheus metrics on a single port via `cmux`.
- **mTLS Security:** Strict TLS 1.3 with mutual authentication.
- **Scalable Broker:** Pluggable `local` ↔ `Redis` message broker.
- **Persistence:** MongoDB with optimized compound indexes and TTL for offline messages.
- **Monitoring:** Integrated Prometheus metrics + Grafana.
- **Clients:** Go CLI and Nuxt 3 Web Dashboard.

## Quickstart

### 1. Build and Start

```bash
# Register dependencies
go mod tidy

# Build server and client
make build

# Start infrastructure (Mongo, Redis, etc.)
docker-compose up -d

# Start server
./chat-server
```

### 2. mTLS Setup for Browser

To use the Web Demo, you must import the demo certificate:

```bash
# 1. Run the generation script:
go run scripts/p12_gen/main.go

# 2. Import 'web-demo-import-me.p12' into your Browser/OS certificate store.
# Password: password
```

Access the dashboard at: `https://localhost:8080/demo/`

### 3. Demo CLI

```bash
# Run the interactive CLI client
./chat-client --cert testuser.crt --key testuser.key
```

## Architecture

- **Hexagonal Design:** Core logic in `internal/domain` and `internal/service`, decoupled from `internal/repository` and `internal/transport`.
- **Single Source of Truth:** Protobuf definitions in `proto/` drive all API contracts.
- **mTLS:** Every connection (except `/metrics`) requires a client certificate signed by the embedded CA.

## Monitoring

- **Metrics:** `https://localhost:8080/metrics` (Plain HTTP)
- **Grafana:** `http://localhost:3000`
- **Prometheus:** `http://localhost:9090`
