# Micro-Terminal: Replace xterm.js with Lightweight TypeScript Terminal

**Date:** 2026-03-31
**Goal:** Reduce browser download size from ~500 KB (xterm.js CDN) to ~2-4 KB by replacing xterm.js with a custom TypeScript micro-terminal.

## Problem

The current frontend loads xterm.js v5 + 2 addons from jsdelivr CDN (~500 KB total). The WebSocket protocol is a simple line-based REPL — the backend handles input character-by-character and sends ANSI-colored text. Only ~5% of xterm.js's capabilities are used.

## Solution

A custom micro-terminal written in TypeScript, compiled to a single minified JS file via esbuild. No external dependencies. Zero CDN requests.

## Architecture & File Layout

```
internal/web/
├── frontend/
│   ├── src/
│   │   ├── main.ts          # Entry point — wires terminal + websocket
│   │   ├── terminal.ts      # DOM renderer — appends styled text, captures input
│   │   ├── ansi.ts          # ANSI SGR parser — escape codes -> styled segments
│   │   └── websocket.ts     # WebSocket client — same JSON protocol as today
│   └── tsconfig.json
├── static/
│   ├── index.html           # Updated — removes xterm CDN refs, loads terminal.min.js
│   ├── terminal.min.js      # Build output (~2-4 KB minified)
│   └── style.css            # Updated — ANSI color theme via CSS custom properties
```

## Micro-Terminal Renderer

The terminal is a `<div id="terminal">` with `overflow-y: auto` containing a `<pre>` for output.

### Data Flow

1. Server sends raw bytes with ANSI escape sequences over WebSocket
2. ANSI parser (`ansi.ts`) converts escape codes into styled segments: `{ text: "hello", fg: "green", bold: true }`
3. Renderer (`terminal.ts`) wraps each segment in a `<span>` with CSS classes: `<span class="fg-green bold">hello</span>`
4. Appends to the `<pre>` and auto-scrolls to bottom
5. Handles `\r\n` as line breaks, `\b \b` as backspace-erase (server's character echo)

### Input Handling

- Listens on `keydown` on the terminal container
- Sends each keystroke to WebSocket as `{ type: "input", data: "x" }` — identical protocol to today
- No local echo — the server echoes characters back (unchanged behavior)

### ANSI Codes Supported

Only what the server actually sends:

| Code | Meaning |
|------|---------|
| `\x1b[0m` | Reset all styles |
| `\x1b[1m` | Bold |
| `\x1b[30m`-`\x1b[37m` | Foreground colors (standard) |
| `\x1b[90m`-`\x1b[97m` | Foreground colors (bright) |
| `\x1b[2J` | Erase display (clear screen — used by `clear` command) |
| `\x1b[H` | Cursor home (reset to top — accompanies clear) |

Any other unrecognized escape sequences are silently stripped.

### CSS Color Mapping

The 16 ANSI colors are mapped to CSS custom properties in `style.css`, replacing the JS theme object in the current `terminal.js`. Uses the existing Dracula-ish palette:

```css
:root {
    --term-bg: #0a0a0a;
    --term-fg: #e0e0e0;
    --term-cursor: #00ff88;
    --ansi-black: #0a0a0a;
    --ansi-red: #ff5555;
    --ansi-green: #50fa7b;
    /* ... etc */
}
```

## WebSocket Client

### Protocol (Unchanged)

```
Browser -> Server:  { "type": "input", "data": "..." }
Browser -> Server:  { "type": "resize", "cols": 80, "rows": 24 }
Server -> Browser:  raw ANSI text (binary/text frames)
```

No backend changes required. The wire protocol is identical.

### Connection Lifecycle

1. **Connect** — derive WebSocket URL from `window.location` (`ws://` or `wss://`), connect to `/ws`
2. **On open** — send initial resize message with container dimensions
3. **On message** — pipe data through ANSI parser, then renderer
4. **On close** — display "Connection closed. Refresh to reconnect." in terminal
5. **On error** — display error message, no auto-reconnect (matches current behavior)

## Build Pipeline

### Tooling

- **esbuild** as the sole build dependency (TypeScript compilation + bundling + minification)
- Invoked via Makefile targets

### Makefile Targets

```makefile
frontend-deps:
	npm install --save-dev esbuild

frontend-build:
	npx esbuild internal/web/frontend/src/main.ts \
		--bundle --minify --target=es2020 \
		--outfile=internal/web/static/terminal.min.js

build: frontend-build
	go build -ldflags="-s -w" -o tewodros-terminal ./cmd/server
```

### What Gets Committed

- TypeScript source files in `internal/web/frontend/src/`
- `tsconfig.json`
- `package.json` (minimal — esbuild as sole devDependency)
- `terminal.min.js` (built artifact, committed so `go build` works without Node)
- `.gitignore` updated for `node_modules/`

### What Gets Removed

- `internal/web/static/terminal.js` (replaced by `terminal.min.js`)
- 3 xterm.js CDN `<script>` tags from `index.html`
- 1 xterm.css CDN `<link>` tag from `index.html`

## Expected Results

| Metric | Before | After |
|--------|--------|-------|
| External CDN requests | 4 (xterm.js + CSS + 2 addons) | 0 |
| JS download size | ~500 KB | ~2-4 KB |
| CSS download (CDN) | ~50 KB (xterm.css) | 0 (styles in existing style.css) |
| Total frontend download | ~558 KB | ~8-12 KB (HTML + CSS + JS) |
| Build dependencies | None | esbuild (devDependency) |
| Backend changes | N/A | None required |

## Out of Scope

- Text selection/copy (not needed per requirements)
- Clickable link detection (not needed per requirements)
- Scrollback buffer beyond browser-native scroll (the `<pre>` with `overflow-y: auto` provides this natively)
- Auto-reconnect on disconnect
- Canvas-based rendering
