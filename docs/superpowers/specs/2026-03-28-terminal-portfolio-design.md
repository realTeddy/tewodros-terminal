# Terminal Portfolio — Design Spec

**Date:** 2026-03-28
**Project:** tewodros-terminal
**Domain:** tewodros.me

## Summary

A personal portfolio website delivered as a real terminal experience over SSH (`ssh tewodros.me`) and HTTPS (`https://tewodros.me`). Built in Go using the Charm ecosystem (Wish, Bubble Tea, Lipgloss). Hosted on an Oracle Cloud free-tier ARM VM with Cloudflare in front.

Replaces the current simulated terminal at tewodros.me with a genuine interactive TUI.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     tewodros.me                         │
│                   (Cloudflare DNS)                      │
└──────────┬──────────────────────┬───────────────────────┘
           │                      │
     Port 22 (direct)      Port 443 (Cloudflare proxy)
           │                      │
┌──────────▼──────────────────────▼───────────────────────┐
│              Oracle Cloud ARM VM (Free)                  │
│                                                          │
│  ┌─────────────────┐    ┌────────────────────────────┐  │
│  │   Wish SSH       │    │   HTTP/WebSocket Server    │  │
│  │   Server (:22)   │    │   (:8080)                 │  │
│  └────────┬─────────┘    └─────┬──────────┬──────────┘  │
│           │                    │          │              │
│           │              Serves static   WebSocket       │
│           │              xterm.js page   endpoint        │
│           │                              │              │
│  ┌────────▼──────────────────────────────▼──────────┐   │
│  │          Shared Bubble Tea TUI Application        │   │
│  │   (each connection gets its own program instance) │   │
│  └───────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

- Single Go binary runs two listeners: SSH on `:22`, HTTP on `:8080`
- Cloudflare terminates TLS on 443 and forwards to origin on 8080
- SSH traffic bypasses Cloudflare (direct to VM IP on port 22)
- Each connection (SSH or WebSocket) spawns an independent Bubble Tea program instance
- WebSocket bridge: xterm.js <-> WebSocket <-> virtual PTY <-> Bubble Tea

## Capacity

Oracle Cloud free ARM VM (1 OCPU, 6GB RAM):
- Each Bubble Tea instance: ~1-5MB
- Estimated concurrent capacity: hundreds to low thousands
- Network: ~480 Mbps (Oracle free tier)
- Massively overpowered for a portfolio site

## TUI Experience

### Virtual Filesystem

Visitors explore a curated content tree — not a real filesystem.

```
~tewodros/
├── about.txt
├── skills/
│   ├── languages.txt
│   ├── tools.txt
│   └── frameworks.txt
├── projects/
│   ├── project-1/
│   │   ├── README.txt
│   │   └── demo.link
│   ├── project-2/
│   │   └── ...
│   └── ...
├── contact.txt
├── resume.txt
└── guestbook/
```

### Commands

| Command | Behavior |
|---|---|
| `ls` | List current directory |
| `cd <dir>` | Navigate into directory |
| `cat <file>` | Display file contents with lipgloss styling |
| `tree` | Show full directory tree |
| `help` | List available commands |
| `clear` | Clear screen |
| `whoami` | Fun response — "a curious visitor" or SSH fingerprint |
| `guestbook` | Interactive prompt for name + message |
| `guestbook --read` | View recent guestbook entries |
| `neofetch` | ASCII art + personal stats (about you, not the server) |
| `exit` / `quit` | Close session with farewell message |

### Welcome Screen

```
 ╔════════════════════════════════════════╗
 ║   tewodros.me — terminal portfolio    ║
 ╠════════════════════════════════════════╣
 ║                                       ║
 ║   Welcome. Type 'help' to begin.      ║
 ║                                       ║
 ╚════════════════════════════════════════╝

 visitor@tewodros.me:~$ _
```

### Interaction Design

- Commands echo familiar Unix behavior but are fully sandboxed
- Tab completion for commands and paths
- Colorful output using lipgloss
- Unknown commands: `"command not found. Type 'help' for available commands."`
- Sessions are stateless — no login required

## Data & Persistence

### Portfolio Content

- All content embedded in Go source files (compiled into the binary)
- Update by editing source and redeploying
- Zero I/O for content reads

### Guestbook

- SQLite single file on disk
- Schema:

```sql
CREATE TABLE guestbook (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

- Display capped at 100 most recent entries
- Spam protection: rate limit by IP (in-memory map), max message length, strip control characters
- IP source: SSH connections use socket remote addr; WebSocket connections use `CF-Connecting-IP` header

## Infrastructure

### Oracle Cloud VM

- Ampere A1 ARM instance (1 OCPU, 6GB RAM) — free tier
- Ubuntu 22.04 or 24.04 ARM
- Security list: ports 22, 8080, 2222 open (Cloudflare handles 443 externally)

### Cloudflare

- A record for `tewodros.me` -> VM public IP (proxied / orange cloud)
- Free SSL between browser and Cloudflare
- Cloudflare origin cert for Cloudflare <-> VM
- WebSocket support enabled (free on all plans)
- SSH on port 22 goes direct (bypasses proxy automatically)

### Process Management

- Single Go binary managed by systemd
- Auto-restart on crash, start on boot

### Admin Access

- VM admin SSH on port 2222 (non-standard to avoid conflict with Wish on 22)
- UFW firewall: only 22, 8080, 2222 open
- Fail2ban on admin SSH

### Deployment

- Cross-compile: `GOOS=linux GOARCH=arm64 go build`
- Deploy via `scp` binary to VM + systemd restart
- Optional later: GitHub Actions CI/CD on push to main

## Project Structure

```
tewodros-terminal/
├── cmd/
│   └── server/
│       └── main.go              # entry point — starts SSH + HTTP servers
├── internal/
│   ├── tui/
│   │   ├── app.go               # root Bubble Tea model
│   │   ├── commands.go          # command parser + dispatch
│   │   ├── views.go             # rendering logic (lipgloss styles)
│   │   └── filesystem.go        # virtual filesystem tree + navigation
│   ├── ssh/
│   │   └── server.go            # Wish SSH server setup + middleware
│   ├── web/
│   │   ├── server.go            # HTTP server + WebSocket handler
│   │   └── bridge.go            # WebSocket-to-BubbleTea bridge (virtual PTY)
│   ├── guestbook/
│   │   └── guestbook.go         # SQLite operations for guestbook
│   └── content/
│       ├── about.go             # embedded portfolio content
│       ├── projects.go
│       ├── skills.go
│       └── resume.go
├── web/
│   └── static/
│       ├── index.html           # xterm.js terminal page
│       ├── terminal.js          # WebSocket connection + xterm setup
│       └── style.css            # minimal styling for page shell
├── go.mod
├── go.sum
└── Makefile                     # build, deploy, run targets
```

### Dependencies

- `github.com/charmbracelet/wish` — SSH server
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — terminal styling
- `github.com/gorilla/websocket` — WebSocket server
- `modernc.org/sqlite` — SQLite driver (pure Go, no CGO — enables easy cross-compilation)
- `github.com/creack/pty` — pseudo-terminal for WebSocket bridge

### WebSocket Bridge

1. Browser connects via WebSocket to `/ws`
2. Server spawns a virtual PTY (pseudo-terminal)
3. Bubble Tea program attaches to that PTY
4. xterm.js <-> WebSocket <-> PTY <-> Bubble Tea
5. Bidirectional I/O — browser experience identical to SSH

## Out of Scope

- User authentication / accounts
- CMS or admin panel
- Analytics (can add later via simple request logging)
- Mobile-native app
- Real shell access of any kind
