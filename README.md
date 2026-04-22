# Go Simple Chat

A high-performance, horizontally-scalable chat service architecture in Go implementing Hexagonal Modular Monolith principles and strict mTLS security.

## Features

- **Dual-Port Security Model:** Serving strict mTLS for direct clients on port 8080 and a hardened session-based gateway for browsers on port 8081.
- **Strict mTLS Identity:** TLS 1.3 with mutual authentication and application-layer Public Key Pinning (PKP) to prevent impersonation.
- **Cryptographic Certificate Renewal:** A secure challenge-response protocol using signed nonces for safe identity rotation.
- **Unified Media Metadata:** Server-side support for diverse media payloads (Image, Video, Audio, Document) with unified storage logic.
- **Real-Time Read State Sync:** Consistent participant read-state tracking across multiple sessions and devices.
- **Scalable Pub/Sub Broker:** Topology-agnostic horizontal scalability powered by a Redis-backed message broker.
- **Persistent Session Store:** Production-grade MongoDB-backed session management with automatic TTL indexing and dual-expiration validation.
- **Proactive Security Alerts:** Real-time monitoring of identity/session life with automated "Expiring Soon" warnings and immediate connection lockdowns.
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
- **Security Alerts:**
  - **Amber Warning:** A persistent "SECURITY ALERT" in the status bar notifies you when your certificate expires in under 24 hours.
  - **Red Lockdown:** Upon expiration, the UI locks and displays a fatal error requiring a restart and identity renewal.

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

### 3. Hardened Identity Lifecycle

Go Simple Chat implements a robust **Proof-of-Possession (PoP)** mechanism to manage user identities via modern **Ed25519** cryptography:

1. **Challenge Request:** Both the Web and TUI clients fetch a unique, time-limited nonce from the server.
2. **Local Signature:** The client signs this nonce using their **Private Key** locally in the browser (via `SubtleCrypto`) or terminal—private keys never leave the client device.
3. **Verification & Issue:** The server verifies the signature against the nonce using the **Pinned Public Key** stored in the database.
4. **Renewal/Session:** Upon success, the server either issues a fresh certificate (Renewal) or stores an `HttpOnly` session token (Web Login).
5. **Real-time Enforcement:** The server monitors the `NotAfter` date of your certificate (and session TTL) in a dedicated background goroutine. If your identity expires during an active session, the server sends an `IdentityEvent` signal and force-terminates the connection to ensure non-repudiation.

### 4. Web Session Bridge

The web client securely bridges standard browser environments with the mTLS-strict backend via the **Public Gateway (Port 8081)**:

1. **Replay Protection:** The client performs a cryptographic challenge-response handshake (`POST /api/session`) to prove possession of the private key, preventing attackers from replaying captured public certificates.
2. **Secure Token Issuance:** The server verifies the PoP and issues a short-lived, **`HttpOnly`**, **`Secure`** session cookie (`x-session-token`). This protects the token from Cross-Site Scripting (XSS) attacks by preventing JavaScript access.
3. **Authorization:** Subsequent API and WebSocket requests are authenticated via this secure cookie. The gateway validates the token and proxies the request to the internal mTLS core (Port 8080) using a loopback-trusted identity bridge.
4. **Seamless Re-authentication:** Both the Web and TUI clients implement background re-authentication. If a session token expires while the certificate is still valid, the client silently performs a new PoP handshake to obtain a fresh session without interrupting the user's workflow.

## ⚙️ Configuration

The server supports the following environment variables for security hardening:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Secure gRPC/mTLS port. |
| `PUBLIC_PORT` | `8081` | Public REST/WebSocket gateway port. |
| `TRUSTED_PROXY_ADDRS` | `127.0.0.1,::1` | IPs allowed to inject internal identity headers. |
| `ALLOWED_ORIGINS` | `*` | Allowed CORS origins (comma-separated). |
| `CERT_CN` | `localhost` | Common Name for automatically generated server certs. |

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
- **Trusted Bridge Model:** The public gateway validates web sessions and proxies them to the internal gRPC core using service-level certificates.
- **WebSocket Gateway:** Real-time streams are authenticated via the same secure `HttpOnly` session cookies as the REST API, eliminating sensitive tokens from URLs.
- **Bulk Presence Tracking:** The `GetPresence` API supports batch lookups, allowing clients to synchronize the online status of dozens of participants in a single efficient network call.
- **Public Key Pinning:** Prevents impersonation by verifying the certificate's public key against the user's permanent database record.

## Monitoring

- **Metrics:** `https://localhost:8081/metrics`
- **Dashboards:** Grafana (port 3002) and Prometheus (port 9090) are included in the stack.
