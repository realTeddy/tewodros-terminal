# tewodros-terminal

A real interactive terminal portfolio served over SSH and HTTPS. Not a fake terminal CSS theme -- visitors connect to a live [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI application and navigate a virtual filesystem containing portfolio content.

```
$ ssh tewodros.me

  tewodros.me — welcome to my terminal portfolio.

  Try about, contact, or help to see all commands.

visitor@tewodros.me:~$ help

  Available commands:

    about            About me
    contact          Send me a message
    github           View source code
    ls               List directory contents
    cd <dir>         Change directory (cd .., cd ~)
    cat <file>       Display file contents
    tree             Show full directory tree
    help             Show this help message
    clear            Clear the screen
    whoami           Who are you?
    neofetch         System info
    guestbook        Leave a message
    guestbook --read View recent messages
    exit / quit      Close the session

visitor@tewodros.me:~$
```

## How It Works

```
              ┌────────────────────────┐
              │    Let's Encrypt TLS   │
              └────────────┬───────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         SSH (22)    HTTPS (443)        │
              │            │            │
              ▼            ▼            │
        ┌──────────┐ ┌──────────┐      │
        │   Wish   │ │   HTTP   │      │
        │  Server  │ │  Server  │      │
        └────┬─────┘ └────┬─────┘      │
             │             │            │
             ▼             ▼            │
        ┌──────────┐ ┌──────────┐      │
        │ Bubble   │ │PTY Bridge│      │
        │   Tea    │ │→Bubble   │      │
        │          │ │  Tea     │      │
        └────┬─────┘ └────┬─────┘      │
             │             │            │
             └──────┬──────┘            │
                    ▼                   │
             ┌─────────────┐            │
             │  Virtual FS │            │
             │  + Commands │            │
             └──────┬──────┘            │
                    │                   │
             ┌──────┴──────┐            │
             │   SQLite    │            │
             │  Guestbook  │            │
             └─────────────┘            │
```

**SSH path:** Visitors `ssh` directly into a Bubble Tea TUI via [Wish](https://github.com/charmbracelet/wish) -- full terminal support with colors, key bindings, and tab completion.

**Web path:** The HTTPS server upgrades to a WebSocket connection, spawns a PTY running the same Bubble Tea app, and pipes raw bytes to a custom screen-buffer terminal in the browser (ANSI parser + renderer, no xterm.js dependency). Browser visitors see the identical TUI as SSH users.

Each connection gets its own isolated PTY, virtual filesystem, and session state.

## Features

- **Virtual filesystem** -- `ls`, `cd`, `cat`, `tree` through portfolio content
- **Guestbook** -- SQLite-backed with rate limiting (5 entries/IP/5min) and input sanitization (strips ANSI escapes, control chars)
- **Contact form** -- Interactive email form via [Resend](https://resend.com) API (optional, graceful fallback)
- **Tab completion** -- Commands and filesystem paths
- **Dual access** -- SSH and WebSocket, both running the same Bubble Tea TUI via PTY
- **Zero CGO** -- Pure Go SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), cross-compiles anywhere
- **TLS support** -- Optional TLS via Let's Encrypt for direct HTTPS

## Quick Start

### Prerequisites

- Go 1.26+ (uses `go.mod` version)
- Node.js 22+ (for frontend build)

### Run locally

```bash
git clone https://github.com/realTeddy/tewodros-terminal.git
cd tewodros-terminal
go run ./cmd/server
```

The server starts with defaults:
- SSH on `0.0.0.0:22`
- HTTP on `0.0.0.0:8080`
- SQLite guestbook at `./guestbook.db`
- SSH host keys generated in `./.ssh/`

Connect via:
```bash
ssh localhost
# or open http://localhost:8080 for the WebSocket interface
```

### Build

```bash
make build          # current platform
make build-linux    # linux/amd64
make build-arm      # linux/arm64
```

## Configuration

All configuration is via environment variables with sensible defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_HOST` | `0.0.0.0` | SSH listen address |
| `SSH_PORT` | `22` | SSH listen port |
| `HTTP_HOST` | `0.0.0.0` | HTTP listen address |
| `HTTP_PORT` | `8080` | HTTP listen port |
| `TLS_CERT` | *(none)* | Path to TLS certificate (enables HTTPS) |
| `TLS_KEY` | *(none)* | Path to TLS private key |
| `HOST_KEY_DIR` | `.ssh` | Directory for SSH host keys (auto-generated) |
| `DB_PATH` | `guestbook.db` | SQLite database path |
| `RESEND_API_KEY` | *(none)* | [Resend](https://resend.com) API key for contact form |
| `CONTACT_EMAIL` | `assefa@tewodros.me` | Email recipient for contact form |

## Deployment

The project includes a systemd service unit and GitHub Actions CI/CD pipeline.

### Systemd

```bash
# Copy and edit the service file
sudo cp deploy/tewodros-terminal.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now tewodros-terminal
```

See [deploy/tewodros-terminal.service](deploy/tewodros-terminal.service) for the full unit file with security hardening (NoNewPrivileges, ProtectSystem, PrivateTmp, etc.).

### CI/CD

The [GitHub Actions workflow](.github/workflows/deploy.yml) runs tests on every push and deploys to the server on pushes to `main`. It uses these repository secrets:

| Secret | Description |
|--------|-------------|
| `DEPLOY_SSH_KEY` | SSH private key for deployment |
| `SERVER_HOST` | Server IP or hostname |
| `SERVER_USER` | SSH username on the server |
| `RESEND_API_KEY` | Resend API key for contact form |
| `CONTACT_EMAIL` | Email recipient for contact form |

## Project Structure

```
cmd/server/          Entry point -- wires SSH, HTTP, guestbook, email
internal/
  content/           Portfolio content as a virtual filesystem tree
  email/             Resend API email client
  guestbook/         SQLite guestbook with rate limiting
  ssh/               Wish SSH server setup
  tui/               Bubble Tea app, commands, filesystem, views
  web/               HTTP server, PTY-WebSocket bridge, static assets
    frontend/        Screen-buffer terminal (TypeScript, ANSI parser)
deploy/              Systemd service unit
scripts/             Infrastructure provisioning helpers
```

## Customizing

To use this as a template for your own terminal portfolio:

1. **Edit content** -- Modify [internal/content/content.go](internal/content/content.go) with your own about, skills, projects, and contact info
2. **Update branding** -- Change the hostname and prompt in [internal/tui/views.go](internal/tui/views.go)
3. **Configure email** -- Set `RESEND_API_KEY` and `CONTACT_EMAIL` environment variables, or remove the email feature
4. **Deploy** -- Update the Makefile `SERVER` variable and systemd service paths for your infrastructure

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) -- TUI framework
- [Wish](https://github.com/charmbracelet/wish) -- SSH server for TUIs
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) -- Terminal styling
- [Gorilla WebSocket](https://github.com/gorilla/websocket) -- WebSocket server
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) -- Pure Go SQLite

## License

[MIT](LICENSE)
