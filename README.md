# Crossbeam

**Crossbeam** is an open-source, self-hostable tool for sending text and files instantly between your devices over the internet — an unofficial successor to Pushbullet.

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Screenshots](#screenshots)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
  - [Running Locally](#running-locally)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [API Documentation](#api-documentation)
- [Authentication](#authentication)
- [File Storage](#file-storage)
- [Deployment](#deployment)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

Crossbeam lets you instantly push text snippets, links, and files from one of your devices to another. Open a file on your phone and have it appear on your desktop in seconds, or copy a URL on your laptop and paste it on your tablet — no cables, no cloud accounts required beyond your own server.

The project was born out of frustration with Pushbullet's decline and the lack of a privacy-respecting, self-hostable alternative. With Crossbeam, you own your data and control your server.

---

## Features

### Text & Links
- **Clipboard push** — Send any text or URL to one or all of your devices instantly
- **Push history** — Browse a chronological log of everything you've sent
- **Copy on arrival** — Optionally auto-copy received text to the clipboard on the target device

### File Transfer
- **File push** — Send any file from one device to another over the internet
- **Inline previews** — Images and supported media render inline in the push history
- **Download on demand** — Files are stored server-side and downloaded when needed

### Devices
- **Multi-device support** — Register as many devices as you like under one account
- **Targeted pushes** — Send to a specific device or broadcast to all at once
- **Device management** — Name, view, and revoke devices from any client

### Real-time Delivery
- **WebSocket push** — Pushes are delivered in real time via a persistent WebSocket connection
- **Offline queue** — Pushes sent while a device is offline are delivered when it reconnects

### Privacy & Self-hosting
- **Fully self-hostable** — Run your own instance with Docker or bare metal
- **No telemetry** — Zero analytics or tracking
- **Open source** — MIT licensed

---

## Screenshots

> Screenshots will be added as the UI matures.

---

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Desktop App   │     │   Mobile App    │     │ Browser Ext.    │
│ (Electron)      │     │ (React Native)  │     │ (Web Extension) │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │ HTTPS / WSS           │ HTTPS / WSS            │ HTTPS / WSS
         │           ┌───────────┴────────────┐           │
         └──────────►│        API Server      │◄──────────┘
                     │  (Bun + Hono)          │
                     │  REST + WebSocket      │
                     └──────┬─────────────────┘
                            │
               ┌────────────┼────────────┐
               │            │            │
        ┌──────▼──┐  ┌──────▼──┐  ┌─────▼──────┐
        │  Auth   │  │  Push   │  │  Presence  │
        │ Service │  │ Service │  │  (Redis)   │
        └──────┬──┘  └──────┬──┘  └────────────┘
               │            │
        ┌──────▼────────────▼──────┐
        │        PostgreSQL        │
        └──────────────────────────┘
                     │
              ┌──────▼──────┐
              │  S3 / MinIO │
              │ (file store)│
              └─────────────┘
```

- **API Server** — Bun/Hono REST API and WebSocket gateway for real-time push delivery
- **Auth Service** — JWT-based authentication and device registration
- **Push Service** — Stores, fans out, and queues pushes for offline devices
- **Presence (Redis)** — Tracks connected devices; pub/sub for real-time delivery across server instances
- **PostgreSQL** — Primary store for users, devices, and push history
- **MinIO** — Self-hosted S3-compatible object storage for uploaded files

---

## Getting Started

### Prerequisites

| Tool | Version |
|------|---------|
| Bun | latest |
| Docker | `24.x+` |
| Docker Compose | `2.x+` |

### Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/low-stack-technologies/crossbeam.git
   cd crossbeam
   ```

2. **Install dependencies**

   ```bash
   bun install
   ```

3. **Copy the environment template**

   ```bash
   cp .env.example .env
   ```

### Configuration

Edit `.env` and fill in the required values:

```env
# Application
NODE_ENV=development
APP_URL=http://localhost:3000
PORT=3000

# Database
DATABASE_URL=postgresql://crossbeam:password@localhost:5432/crossbeam

# Redis
REDIS_URL=redis://localhost:6379

# Authentication
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRES_IN=30d

# File Storage (S3-compatible)
# In development, MinIO runs via Docker Compose at localhost:9000
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=crossbeam-uploads
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_REGION=us-east-1
# In production, point S3_ENDPOINT at your own MinIO instance or any S3-compatible provider
```

### Running Locally

**Option 1 — Docker Compose (recommended)**

Starts all infrastructure (Postgres, Redis, MinIO) alongside the application:

```bash
docker compose up
```

The app will be available at [http://localhost:3000](http://localhost:3000). MinIO's management console is available at [http://localhost:9001](http://localhost:9001) (default credentials: `minioadmin` / `minioadmin`).

**Option 2 — Manual**

Start the infrastructure services first:

```bash
docker compose up postgres redis minio -d
```

Then run database migrations:

```bash
bun db:migrate
```

Start the development server:

```bash
bun dev
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop Client | TypeScript, Electron, React, TailwindCSS |
| Mobile | React Native (planned) |
| Browser Extension | TypeScript, Web Extensions API (planned) |
| Backend | TypeScript, Bun, Hono |
| Real-time | WebSockets |
| Database | PostgreSQL (via Drizzle ORM) |
| Cache / Pub-Sub | Redis |
| Auth | JWT |
| File Storage | S3-compatible via MinIO (self-hosted) |
| Testing | Bun test, Playwright |
| CI/CD | GitHub Actions |
| Containerization | Docker, Docker Compose |

---

## Project Structure

```
crossbeam/
├── apps/
│   ├── desktop/              # Electron desktop client
│   │   ├── src/
│   │   │   ├── main/         # Electron main process
│   │   │   └── renderer/     # React UI (renderer process)
│   │   │       ├── components/
│   │   │       ├── pages/
│   │   │       ├── hooks/
│   │   │       └── stores/
│   │   └── package.json
│   └── server/               # Bun backend
│       ├── src/
│       │   ├── routes/       # API route handlers
│       │   ├── services/     # Business logic
│       │   ├── db/           # PostgreSQL schema & migrations
│       │   ├── ws/           # WebSocket gateway
│       │   └── types/        # Shared TypeScript types
│       └── package.json
├── packages/
│   ├── shared/               # Types and utilities shared across apps
│   └── config/               # Shared ESLint / TS config
├── docker/                   # Dockerfiles and compose configs
├── .github/
│   ├── ISSUE_TEMPLATE/
│   ├── workflows/
│   └── PULL_REQUEST_TEMPLATE.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── LICENSE
└── README.md
```

---

## API Documentation

The REST API follows standard HTTP conventions. All endpoints are prefixed with `/api/v1`.

Interactive API docs (Swagger UI) are available at `/api/docs` when running in development mode.

### Key Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/auth/register` | Create a new account |
| `POST` | `/api/v1/auth/login` | Authenticate and receive a JWT |
| `GET` | `/api/v1/users/@me` | Get the authenticated user's profile |
| `GET` | `/api/v1/devices` | List all registered devices |
| `POST` | `/api/v1/devices` | Register a new device |
| `DELETE` | `/api/v1/devices/:id` | Remove a device |
| `GET` | `/api/v1/pushes` | List push history |
| `POST` | `/api/v1/pushes` | Send a push (text, link, or file) |
| `DELETE` | `/api/v1/pushes/:id` | Delete a push |

### WebSocket Gateway

Clients connect to the gateway at `wss://your-instance/gateway`. After authenticating with a JWT, clients receive real-time push events:

| Event | Direction | Description |
|-------|-----------|-------------|
| `READY` | Server → Client | Initial state after auth, including pending pushes |
| `PUSH_CREATE` | Server → Client | A new push was sent to this device |
| `PUSH_DELETE` | Server → Client | A push was deleted |
| `DEVICE_UPDATE` | Server → Client | A device was added or removed from the account |

---

## Authentication

Crossbeam uses **JSON Web Tokens (JWT)** for stateless authentication.

- Tokens are issued on login and expire after `JWT_EXPIRES_IN` (default: 30 days)
- Each device registers independently and receives its own token
- Passwords are hashed with **Argon2id**

---

## File Storage

Files pushed between devices are stored server-side in an **S3-compatible object store**. Crossbeam ships with [MinIO](https://min.io) as the default storage backend — MIT licensed, fully self-hosted, and API-compatible with Amazon S3.

### How it works

- When a client pushes a file, it is uploaded to the API server
- The server streams it to the configured S3 bucket and records the object key in PostgreSQL
- Download URLs are pre-signed on demand, keeping the bucket private
- Files are retained until the push is deleted or a configurable retention period expires

### Development

MinIO starts automatically with `docker compose up`. The `.env.example` defaults point to the local instance.

The MinIO management console is available at [http://localhost:9001](http://localhost:9001):
- Username: `minioadmin`
- Password: `minioadmin`

### Production

1. **Self-hosted MinIO** (recommended) — run MinIO on your own infrastructure and update `S3_ENDPOINT`, `S3_ACCESS_KEY`, and `S3_SECRET_KEY`.
2. **Any S3-compatible provider** — Backblaze B2, Cloudflare R2, Wasabi, etc.

---

## Deployment

### Docker

```bash
docker build -t crossbeam .
docker run -p 3000:3000 --env-file .env crossbeam
```

### Docker Compose (Production)

```bash
docker compose -f docker-compose.prod.yml up -d
```

### Environment Variables

All required environment variables are listed in `.env.example`. Never commit your `.env` file.

### Reverse Proxy

It is strongly recommended to put Crossbeam behind a reverse proxy (e.g. **nginx** or **Caddy**) to handle TLS termination. Example Caddyfile:

```
your-domain.com {
    reverse_proxy localhost:3000
}
```

WebSocket connections (`/gateway`) require long-lived connections — ensure your proxy does not time them out prematurely.

---

## Roadmap

- [ ] Desktop app (Electron)
- [ ] Mobile app (React Native — iOS & Android)
- [ ] Browser extension (Chrome / Firefox)
- [ ] End-to-end encryption for pushes
- [ ] Configurable file retention period
- [ ] Shared channels (push to other users, not just your own devices)
- [ ] Notification mirroring (mirror mobile notifications to desktop)
- [ ] Clipboard sync (continuous background sync mode)
- [ ] CLI client

---

## Contributing

Contributions of all kinds are welcome — bug reports, feature requests, documentation improvements, and code. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

---

## License

Crossbeam is licensed under the **MIT License**. See [LICENSE](LICENSE) for the full text.
