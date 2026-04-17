# Go Simple Chat

A production-ready, horizontally-scalable chat service in Go using a Hexagonal Modular Monolith architecture.

## Features

- **Multiplexed Port (8080):** gRPC, gRPC-Gateway (REST), WebSockets, and Prometheus metrics on a single port via `cmux`.
- **mTLS Security & Identity:** Strict TLS 1.3 with mutual authentication and application-layer Public Key Pinning.
- **Certificate Renewal:** Secure cryptographic challenge-response protocol for renewing expired certificates without permanent lockout.
- **Multi-Media Support:** Messages support multiple attachments (Image, Video, Audio, File) per message.
- **Read State Sync:** Real-time synchronization of "Last Read" markers across all devices.
- **Scalable Broker:** Horizontal scalability via Redis; cluster-ready challenge storage in MongoDB.
- **Persistence:** MongoDB with optimized compound indexes and TTL-based authentication challenges.
- **Premium TUI Client:** A Go-based terminal interface with dual-pane layout, real-time presence, and smart timestamps.
- **Nuxt 3 Dashboard:** Web-based administrative and chat dashboard.

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

### 5. Identity & Lifecycle Management

**Go Simple Chat** enforces strict identity via mTLS. The TUI client manages this lifecycle seamlessly:

- **🔐 Registration:** New users must first register to receive their unique certificate and key. This single-command flow anchors your identity into the persistent database.
- **🕒 Automated Renewal:** If your certificate expires, the TUI client will automatically initiate the **Challenge-Response** flow. You will be prompted with a cryptographic nonce; once signed with your pinned private key, the server issues a fresh certificate instantly, ensuring you are never locked out of your conversations.

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
3. **Renew**: If your certificate expires, the client will prompt you to use the **Challenge-Response** flow to obtain a new one using your pinned private key.

### 3. Running the Client

```bash
# Start the TUI client
./chat-client --cert alice.crt --key alice.key
```

## Architecture

- **Hexagonal Modular Monolith:** Core logic decoupled from transport and storage.
- **Real-Time Roster Sync:** Invisible system signals trigger automatic UI refreshes when participants are added.
- **WebSocket mTLS:** The WebSocket bridge propagates TLS identities into gRPC metadata.
- **Public Key Pinning:** Prevents impersonation by verifying the certificate's public key against the user's permanent database record.

## Monitoring

- **Metrics:** `https://localhost:8080/metrics`
- **Dashboards:** Grafana (port 3000) and Prometheus (port 9090) are included in the stack.
