# Go Simple Chat

A high-performance, horizontally-scalable chat service architecture in Go implementing Hexagonal Modular Monolith principles and strict mTLS security.

## Features

- **Multiplexed gRPC & REST:** Serving gRPC, gRPC-Gateway (REST), and WebSockets on a single port (8080) via `cmux`.
- **Strict mTLS Identity:** TLS 1.3 with mutual authentication and application-layer Public Key Pinning (PKP) to prevent impersonation.
- **Cryptographic Certificate Renewal:** A secure challenge-response protocol using signed nonces for safe identity rotation.
- **Unified Media Metadata:** Server-side support for diverse media payloads (Image, Video, Audio, Document) with unified storage logic.
- **Real-Time Read State Sync:** Consistent participant read-state tracking across multiple sessions and devices.
- **Scalable Pub/Sub Broker:** Topology-agnostic horizontal scalability powered by a Redis-backed message broker.
- **Cloud-Ready Persistence:** Optimized MongoDB storage with compound indexing and TTL-based authentication challenges.
- **Observability:** Built-in Prometheus instrumentation for real-time monitoring of service health and RPC performance.

## 🚀 Premium TUI Client

Go Simple Chat features a high-performance terminal interface built with **Bubble Tea** and **Lipgloss**.

### 1. Launching the Client

After building (`make build`), start the client with your issued credentials:

```bash
./chat-client --cert <username>.crt --key <username>.key
```

### 2. UI Layout & Navigation

The interface is divided into three main zones:

- **Left Sidebar:** A dual-pane list showcasing your **Channels** (top) and **Participants** in the active channel (bottom).
- **Main Chat:** The scrollable message history for the selected conversation.
- **Message Input:** The bottom area where you type messages or execute commands.

**Keyboard Shortcuts:**

| Key | Action |
| :--- | :--- |
| `Tab` | **Switch Focus** between the Sidebar and the Message Input. |
| `Arrows (↑↓)` | Navigate the Channel list (when Sidebar is focused) or move text cursor. |
| `PgUp / PgDn` | **Scroll History** in the active channel. |
| `Enter` | **Send** your message or execute a `/command`. |
| `Ctrl+C / ESC` | Safely exit the application. |

### 3. Understanding the Sidebar

- **Presence Indicators:**
  - `●` (Green): User is currently online and active.
  - `○` (Gray): User is offline (last-seen time available via `/presence`).
  - **Group Count:** Groups show the number of online participants (e.g., `3●`).
- **Unread Markers:**
  - Conversations with new messages are highlighted in **bold green**.
  - Numerical badges (e.g., `[5]`) indicate exactly how many messages you've missed.
  - Clicking/Focusing a channel automatically synchronizes your read-state to the server.

### 4. Interactive Command Palette

Type these slash-commands into the message input field:

- `/dm <username>` — Instantly start a private 1-on-1 conversation.
- `/create <name> <u1> <u2>...` — Create a new group channel with multiple members.
- `/add <usernames...>` — Add new participants to an existing group.
- `/presence <username>` — View detailed status and metadata for any user.
- `/read` — Manually clear all unread markers for the active channel.
- `/help` — Display the interactive command guide.

### 5. Media & Alignment

- **Rich Previews:** Attachment details (Icon, Filename, and URL) are rendered directly in the history.
- **Unified Layout:** Media entries inherit the same styled borders and padding as the message text, ensuring a perfectly aligned and professional appearance.

## 🌈 Nuxt 3 Web Client

The web client provides a modern, accessible interface for chat and secure account lifecycle management.

### 1. Midnight Design System

Optimized for long-term focus and OLED displays, the web client uses a custom **"Dark Dim" (Midnight Slate)** theme. This reduces glare and retinal strain while maintaining high-contrast ratios for crisp legibility.

### 2. Intelligent Chat Flow

- **Smart Scroll:** When opening a channel, the view automatically centers on the **"New Messages"** marker, preserving your reading context.
- **Context-Aware Icons:** Media attachments are tagged with type-specific icons (🎥, 🎵, 📄, 📦) for instant identification.

### 3. Secure Renewal Lifecycle

If your mTLS certificate expires, the web client handles the cryptographic recovery seamlessly:

1. **Challenge Request:** The client fetches a unique nonce from the server.
2. **Signature Generation:** Your private key signs the challenge locally (Ed25519 or ECDSA).
3. **Verification & Issue:** Upon successful signature verification, the server issues a fresh certificate.
4. **Download:** The dashboard provides an instant `.crt` download to restore your identity.

### 4. Web Session Bridge

The web client uses a session-based authentication flow to bridge browser environments with the mTLS-strict backend:

1. **Certificate Exchange:** The client sends its public certificate to the `/api/session` endpoint.
2. **Token Issuance:** The server verifies the certificate's authenticity (and its public key pinning) and issues a short-lived, stateless **Session Token**.
3. **Authorization:** Subsequent API and WebSocket requests are authenticated via the `x-session-token` header or cookie. This allows the web client to maintain mTLS-level identity assurance without requiring the browser to manage client certificates for every individual RPC call.

## Quickstart

### 1. Build and Start

```bash
# Register dependencies
go mod tidy

# Build server and client
make build

# Start services (Server, Mongo, Redis, Prometheus)
docker-compose up -d --build
```

> [!NOTE]
> On the first run, the server automatically generates the root CA (`ca.crt`) and its own server certificates (`server.crt`, `server.key`) in the `certs/` directory if they do not exist.

### 2. User Registration & Identity

Identity is tied to mTLS. For new users:

1. **Register**: Use the included script to register and receive your credentials.

   ```bash
   go run scripts/register_user/register_user.go --username "alice"
   ```

2. **Authenticate**: Use the issued certificate for all subsequent connections.
3. **Renew**: If your certificate expires, use the **Challenge-Response** flow (built into the Web and TUI clients) to obtain a new one using your pinned private key.

### 3. Running the Clients

#### TUI Client (Go)

```bash
# Start the TUI client
./chat-client --cert alice.crt --key alice.key
```

#### Web Client (Nuxt 3)

```bash
# Navigate to web directory
cd web

# Install dependencies
npm install

# Start development server
npm run dev
```

## Architecture

- **Hexagonal Modular Monolith:** Core logic decoupled from transport and storage.
- **Real-Time Roster Sync:** Invisible system signals trigger automatic UI refreshes when participants are added.
- **WebSocket mTLS:** The WebSocket bridge propagates TLS identities into gRPC metadata.
- **Public Key Pinning:** Prevents impersonation by verifying the certificate's public key against the user's permanent database record.

## Monitoring

- **Metrics:** `https://localhost:8080/metrics`
- **Dashboards:** Grafana (port 3000) and Prometheus (port 9090) are included in the stack.
