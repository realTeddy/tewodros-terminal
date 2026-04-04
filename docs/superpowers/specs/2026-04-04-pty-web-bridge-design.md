# PTY Web Bridge — Unified Rendering for SSH and Browser

**Date:** 2026-04-04
**Status:** Proposed

## Problem

The web terminal and SSH terminal run different rendering paths. SSH connections get the full Bubble Tea TUI via Wish. Browser connections get a line-based REPL in `bridge.go` that reimplements welcome messages, guestbook flows, contact flows, and tab completion separately. This means:

- Every feature must be implemented twice
- The SSH and web experiences differ (SSH gets cursor positioning, real-time screen updates; web gets a simpler line REPL)
- bridge.go duplicates ~250 lines of logic already in the Bubble Tea app

## Solution

Replace the line-based WebSocket bridge with a PTY-backed bridge. Each browser connection spawns a PTY running Bubble Tea — the exact same program SSH users get. The browser receives raw ANSI bytes, identical to SSH. The custom micro-terminal is extended to interpret the additional escape sequences Bubble Tea emits.

## Architecture

```
Browser                          Server (Linux)
┌─────────────┐         ┌──────────────────────────┐
│ micro-term  │◄──WS──► │  WebSocket handler        │
│ (extended)  │  bytes   │       │                   │
└─────────────┘         │  ┌────▼────┐  ┌─────────┐ │
                        │  │ PTY     │  │ SSH     │ │
                        │  │ master  │  │ (Wish)  │ │
                        │  └────┬────┘  └────┬────┘ │
                        │  ┌────▼────┐  ┌────▼────┐ │
                        │  │ Bubble  │  │ Bubble  │ │
                        │  │ Tea     │  │ Tea     │ │
                        │  └─────────┘  └─────────┘ │
                        └──────────────────────────┘
```

Both paths produce identical byte streams. The browser sees exactly the same ANSI output as an SSH terminal.

## Server: PTY Bridge (bridge.go rewrite)

### Connection lifecycle

1. WebSocket upgrade (unchanged from current server.go)
2. Create PTY pair via `creack/pty` (already in dependency tree via Wish). Set `TERM=xterm-256color` on the slave so Bubble Tea detects color support.
3. Build `tui.App` with content tree, guestbook, email — same as SSH handler
4. Start a goroutine that runs Bubble Tea on the PTY slave. Use `pty.Start()` with a small shim command, or use `pty.Open()` and run `tea.NewProgram(app, tea.WithInput(ptySlave), tea.WithOutput(ptySlave))` directly — the PTY slave is a real `*os.File`, so Bubble Tea treats it as a terminal.
5. Two goroutines shuttle bytes:
   - **PTY master → WebSocket**: read from PTY master, write as binary WebSocket message
   - **WebSocket → PTY master**: read WebSocket messages, write to PTY master
6. On disconnect (WebSocket close or PTY EOF): close PTY, Bubble Tea exits

### Message protocol

The current WebSocket protocol sends JSON for all messages. The new protocol:

- **Browser → Server (input)**: Raw bytes (binary WebSocket message) for terminal input. This matches what the PTY expects.
- **Browser → Server (resize)**: JSON `{"type":"resize","cols":N,"rows":N}` — the only structured message.
- **Server → Browser**: Raw bytes (binary WebSocket message) — ANSI stream straight from the PTY.

The server reads each WebSocket message. If it's a text message that parses as JSON with `type:"resize"`, it calls `pty.Setsize()`. Otherwise it writes the bytes to the PTY master.

### Resize handling

- Browser calculates cols/rows from container dimensions and font metrics
- Sends resize JSON on connect and on window resize
- Server calls `pty.Setsize()` — the PTY delivers SIGWINCH to Bubble Tea
- Bubble Tea picks up the new size via `tea.WindowSizeMsg` automatically
- Default: 80×24 until browser sends actual dimensions

### Client IP extraction

Preserved from current implementation: check `CF-Connecting-IP` header first, fall back to `RemoteAddr`. Passed to `app.SetClientIP()` before starting the program.

### Estimated size

~60-80 lines replacing the current ~300 lines in bridge.go.

## Frontend: Screen Buffer Terminal

### Current state

The micro-terminal is append-only: ANSI parser emits actions (`text`, `newline`, `backspace`, `clear`), and the terminal inserts DOM nodes before a cursor element. This works for the line-based REPL but cannot handle cursor positioning or screen overwrites.

### New architecture

The terminal becomes a **screen buffer renderer** — a 2D grid of cells that gets updated by ANSI sequences and rendered to the DOM.

```typescript
interface Cell {
  char: string;    // single character or empty
  classes: string; // CSS classes for SGR styling
}

class Terminal {
  private cells: Cell[][];        // rows × cols grid
  private cursorRow: number;
  private cursorCol: number;
  private cursorVisible: boolean;
  private rows: number;
  private cols: number;

  write(data: Uint8Array): void;  // parse ANSI, update cells at cursor
  private render(): void;         // diff cells → DOM update
}
```

### ANSI sequences to support

The parser must handle the following (in addition to existing SGR color support):

| Sequence | Name | Action |
|----------|------|--------|
| `\x1b[row;colH` | Cursor Position (CUP) | Move cursor to row,col |
| `\x1b[nA` | Cursor Up | Move cursor up n rows |
| `\x1b[nB` | Cursor Down | Move cursor down n rows |
| `\x1b[nC` | Cursor Forward | Move cursor right n cols |
| `\x1b[nD` | Cursor Backward | Move cursor left n cols |
| `\x1b[K` / `\x1b[0K` | Erase to End of Line | Clear from cursor to line end |
| `\x1b[1K` | Erase to Start of Line | Clear from line start to cursor |
| `\x1b[2K` | Erase Whole Line | Clear entire line |
| `\x1b[J` / `\x1b[0J` | Erase to End of Display | Clear from cursor to screen end |
| `\x1b[1J` | Erase to Start of Display | Clear from screen start to cursor |
| `\x1b[2J` | Erase Display | Clear entire screen (already supported) |
| `\x1b[?25h` | Show Cursor (DECTCEM) | Make cursor visible |
| `\x1b[?25l` | Hide Cursor (DECTCEM) | Make cursor invisible |
| `\r` | Carriage Return | Move cursor to column 0 |
| `\n` | Line Feed | Move cursor down one row, scroll if at bottom |
| `\x1b[nG` | Cursor Horizontal Absolute | Move cursor to column n |

Unknown sequences are silently ignored (current behavior, preserved).

### SGR extensions

The existing parser handles 3/4-bit foreground colors (30-37, 90-97) and bold. Extend to support:

- Background colors: 40-47, 100-107
- 256-color mode: `38;5;n` (fg), `48;5;n` (bg)
- Reset codes: 39 (default fg), 49 (default bg), 22 (no bold)

These are needed because Lipgloss may emit them depending on the terminal's color profile. The PTY will negotiate a 256-color profile.

### DOM rendering strategy

Instead of inserting individual `<span>` elements, the terminal renders the screen buffer to a `<pre>` element:

1. After each `write()`, build DOM from the cell grid
2. Each row is a `<div>` (or direct text node)
3. Consecutive cells with the same style are merged into a single `<span>`
4. Cursor is rendered as a styled `<span>` at `cursorRow, cursorCol` (or via CSS if cursor is visible)
5. Only re-render rows that changed (track dirty rows)

### Resize detection

```typescript
function measureTermSize(container: HTMLElement): { cols: number; rows: number } {
  const probe = document.createElement("span");
  probe.style.font = getComputedStyle(container).font;
  probe.style.visibility = "hidden";
  probe.style.position = "absolute";
  probe.textContent = "X";
  container.appendChild(probe);
  const charW = probe.offsetWidth;
  const charH = probe.offsetHeight;
  container.removeChild(probe);
  const cols = Math.floor(container.clientWidth / charW);
  const rows = Math.floor(container.clientHeight / charH);
  return { cols: Math.max(cols, 1), rows: Math.max(rows, 1) };
}
```

- Measure on init and on `ResizeObserver` callback
- Send `{"type":"resize", "cols":N, "rows":N}` over WebSocket
- Resize the cell grid (grow/shrink rows and cols, preserve content where possible)

### WebSocket protocol change

Currently the frontend sends JSON for all input:
```json
{"type":"input","data":"about"}
```

New protocol: send raw bytes for input, JSON only for resize. This means the `WS` class changes to:

- `sendInput(data: string)` → sends as binary WebSocket message (TextEncoder to Uint8Array)
- `sendResize(cols, rows)` → sends as JSON text message (unchanged)
- `onMessage` receives binary data (Uint8Array) instead of string

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/web/bridge.go` | Rewrite | PTY bridge replaces line-based REPL (~60-80 lines, down from ~300) |
| `internal/web/server.go` | No change | Static serving and WebSocket upgrade unchanged |
| `internal/ssh/server.go` | No change | SSH path untouched |
| `internal/tui/*` | No change | Bubble Tea app, commands, views unchanged |
| `internal/content/*` | No change | Content tree unchanged |
| `frontend/src/ansi.ts` | Rewrite | Parser returns screen-buffer operations instead of append-only actions |
| `frontend/src/terminal.ts` | Rewrite | Screen buffer renderer replaces append-only renderer |
| `frontend/src/ansi.test.ts` | Update | Tests for cursor positioning, erase, visibility sequences |
| `frontend/src/websocket.ts` | Small change | Binary messages for input, JSON for resize |
| `frontend/src/main.ts` | Small change | Wire up ResizeObserver for dynamic sizing |
| `static/style.css` | Update | Grid-based terminal styles |

## Dependencies

- **No new Go dependencies** — `creack/pty` is already in the module graph via Wish
- **No new JS dependencies** — all frontend work is in the existing TypeScript codebase

## What Gets Deleted

- `webSession` struct and all its methods (~250 lines of duplicate logic)
- Duplicate welcome message, guestbook flow, contact flow, tab completion in bridge.go

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| PTY is Unix-only — won't work on Windows dev | Cross-compile for Linux (already the workflow). Local dev on Windows: use `go run ./cmd/server` which still serves SSH (testable via `ssh localhost`). The PTY bridge can be guarded with a build tag (`//go:build !windows`) with a fallback that logs "PTY not supported on this platform" for the web path. Or dev via WSL. |
| Bubble Tea queries terminal capabilities via PTY | The PTY provides a real terminal interface; Bubble Tea's capability detection works normally |
| Screen buffer adds complexity to frontend | Bounded complexity — only ~6 new sequence types, all well-documented. Tests cover each one. |
| Binary WebSocket messages vs current text | Clean protocol: binary = terminal data, text = JSON control messages (resize). Easy to distinguish. |

## Testing

- **Go**: Existing tests remain unchanged (tui, content, guestbook packages untouched)
- **Frontend**: New ANSI parser tests for cursor positioning, erase sequences, cursor visibility. Screen buffer unit tests for grid operations.
- **Integration**: Manual test — open browser and SSH side by side, run same commands, verify identical output

## Success Criteria

1. Browser visitors see the exact same TUI as SSH users
2. bridge.go has no duplicate command/flow logic
3. No new dependencies added
4. Frontend bundle stays under 10KB minified
5. All existing tests pass
