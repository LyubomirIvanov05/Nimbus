# Nimbus

A production-inspired TCP message broker built from scratch to learn Go engineering practices and systems programming.

## Overview

Nimbus is a lightweight publish/subscribe message broker, designed as a learning project to understand concurrent server architecture, connection lifecycle management, and message persistence. Clients communicate over raw TCP using a simple text protocol.

## Features

- **Pub/Sub Messaging** — Channels with multiple subscribers; messages broadcast to all active subscribers
- **Message Persistence** — All messages logged to per-channel `.log` files; replayed on server restart
- **Message Buffering** — Backlog queue for slow/disconnected clients; flushed automatically on reconnect
- **LastSeen Tracking** — Per-client, per-channel watermark; missed messages delivered on re-subscribe
- **Heartbeat Monitoring** — Clients timed out after inactivity; connections cleaned up automatically
- **Channel Management** — Create, list, delete channels; subscriber and message counts via INFO
- **Retention Control** — Trim channel history to N most recent messages; file rewritten atomically
- **Simple Text Protocol** — Interact directly with `nc` or any TCP client; no SDK needed

## Installation

```bash
git clone https://github.com/LyubomirIvanov05/nimbus
cd nimbus
go build ./cmd/server
./server
```

Or run directly:

```bash
go run ./cmd/server
```

Server starts on port **7070**.

## Quick Start

```bash
# Connect with netcat
nc 127.0.0.1 7070

# Subscribe to a channel
SUBSCRIBE chat

# In another terminal — publish a message
PUBLISH chat Hello!

# List all channels
LIST

# Fetch all messages in a channel
FETCH chat
```

## Protocol Reference

| Command | Arguments | Description |
|---------|-----------|-------------|
| `SUBSCRIBE` | `<channel>` | Subscribe to a channel; delivers missed messages |
| `UNSUBSCRIBE` | `<channel>` | Unsubscribe from a channel |
| `PUBLISH` | `<channel> <message>` | Publish a message to a channel |
| `LIST` | — | List all active channels |
| `FETCH` | `<channel>` | Fetch full message history for a channel |
| `GET` | `<channel> <id>` | Fetch all messages after a given message ID |
| `INFO` | `<channel>` | Show subscriber and message count for a channel |
| `DELETE` | `<channel>` | Delete a channel and its log file |
| `RETENTION` | `<channel> <n>` | Keep only the last N messages |
| `PING` | — | Heartbeat; returns `PONG` |
| `WHOAMI` | — | Returns your remote address |

## Project Structure

```
nimbus/
├── cmd/
│   └── server/          # Entry point
├── internals/
│   ├── broker/          # Server, command routing, handlers
│   ├── channel/         # Channel state, subscribers, broadcast, file I/O
│   ├── client/          # Client struct (conn, backlog, heartbeat, LastSeen)
│   └── message/         # Message struct and constructor
├── logs/                # Per-channel log files
│   └── <channel>.log
└── go.mod
```

## Example Workflow

```bash
# Terminal 1 — subscribe
nc 127.0.0.1 7070
SUBSCRIBE news

# Terminal 2 — publish
nc 127.0.0.1 7070
PUBLISH news "First message"
PUBLISH news "Second message"

# Terminal 1 receives:
# MSG news First message
# MSG news Second message

# Check channel info
INFO news
# SUBSCRIBERS 1
# MESSAGES 2

# Trim to last 1 message
RETENTION news 1

# Clean up
DELETE news
```

## Learning Objectives

This project demonstrates:
- TCP server architecture in Go
- Goroutines and concurrent connection handling
- Mutex and RWMutex for shared state
- Go interfaces (`net.Conn`)
- Struct methods and pointer receivers
- File I/O and log persistence
- Pub/sub design pattern
- Client lifecycle management (connect, heartbeat, disconnect)
- Message buffering and delivery guarantees

## Future Extensions

- Authentication and per-channel ACLs
- Message TTL and auto-expiry
- Persistent client IDs across reconnects
- Binary protocol for efficiency
- Metrics endpoint (message rates, subscriber counts)
- Web UI dashboard
- Docker deployment
