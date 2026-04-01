# Micro-Terminal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace xterm.js (~500 KB CDN) with a custom TypeScript micro-terminal (~2-4 KB minified) that renders ANSI-colored text and captures keyboard input.

**Architecture:** Four TypeScript modules — ANSI parser, terminal renderer, WebSocket client, entry point — compiled to a single minified JS file via esbuild. The backend (`bridge.go`) is untouched. The wire protocol is identical.

**Tech Stack:** TypeScript, esbuild (build), vitest (tests)

---

## File Map

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/web/frontend/package.json` | Dev dependencies (esbuild, vitest) |
| Create | `internal/web/frontend/tsconfig.json` | TypeScript config |
| Create | `internal/web/frontend/src/ansi.ts` | ANSI SGR parser — escape codes to styled segments |
| Create | `internal/web/frontend/src/ansi.test.ts` | Unit tests for ANSI parser |
| Create | `internal/web/frontend/src/terminal.ts` | DOM renderer + keyboard input capture |
| Create | `internal/web/frontend/src/websocket.ts` | WebSocket client (same JSON protocol) |
| Create | `internal/web/frontend/src/main.ts` | Entry point — wires terminal + websocket |
| Create | `internal/web/static/terminal.min.js` | Build output (committed for go:embed) |
| Modify | `internal/web/static/style.css` | Add ANSI color classes + terminal output styles |
| Modify | `internal/web/static/index.html` | Remove 4 xterm CDN refs, add terminal.min.js |
| Modify | `Makefile` | Add frontend-build and frontend-test targets |
| Modify | `.gitignore` | Add node_modules/ |
| Delete | `internal/web/static/terminal.js` | Old xterm.js glue code (replaced) |

---

### Task 1: Build Infrastructure

**Files:**
- Create: `internal/web/frontend/package.json`
- Create: `internal/web/frontend/tsconfig.json`
- Modify: `Makefile`
- Modify: `.gitignore`

- [ ] **Step 1: Create package.json**

```json
{
  "private": true,
  "scripts": {
    "build": "esbuild src/main.ts --bundle --minify --target=es2020 --outfile=../static/terminal.min.js",
    "test": "vitest run"
  },
  "devDependencies": {
    "esbuild": "^0.25.0",
    "vitest": "^3.0.0"
  }
}
```

Write to `internal/web/frontend/package.json`.

- [ ] **Step 2: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "isolatedModules": true
  },
  "include": ["src"]
}
```

Write to `internal/web/frontend/tsconfig.json`.

- [ ] **Step 3: Add node_modules to .gitignore**

Append to `.gitignore`:

```
# Node (frontend build)
node_modules/
```

- [ ] **Step 4: Add frontend targets to Makefile**

Add `frontend-deps frontend-build frontend-test` to the `.PHONY` line. Then add these targets before the existing `build:` target. Also update `build:`, `build-linux:`, and `build-arm:` to depend on `frontend-build`:

```makefile
frontend-deps:
	cd internal/web/frontend && npm install

frontend-build:
	cd internal/web/frontend && npx esbuild src/main.ts --bundle --minify --target=es2020 --outfile=../static/terminal.min.js

frontend-test:
	cd internal/web/frontend && npx vitest run
```

Update existing targets:

```makefile
build: frontend-build
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

build-linux: frontend-build
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BINARY)-linux-amd64 ./cmd/server

build-arm: frontend-build
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY)-linux-arm64 ./cmd/server
```

- [ ] **Step 5: Install dependencies**

Run: `cd internal/web/frontend && npm install`

Expected: `node_modules/` created, `package-lock.json` generated.

- [ ] **Step 6: Commit**

```bash
git add internal/web/frontend/package.json internal/web/frontend/tsconfig.json internal/web/frontend/package-lock.json Makefile .gitignore
git commit -m "chore: add frontend build infrastructure (esbuild + vitest)"
```

---

### Task 2: ANSI Parser — Tests

**Files:**
- Create: `internal/web/frontend/src/ansi.ts` (stub)
- Create: `internal/web/frontend/src/ansi.test.ts`

The ANSI codes the server sends (from `internal/web/bridge.go`):
- `\x1b[0m` reset, `\x1b[1m` bold, `\x1b[1;36m` bold+cyan
- `\x1b[31m` red, `\x1b[32m` green, `\x1b[34m` blue, `\x1b[36m` cyan, `\x1b[90m` bright-black
- `\x1b[2J` clear screen, `\x1b[H` cursor home
- `\b` backspace, `\r\n` newline

- [ ] **Step 1: Create the type stubs in ansi.ts**

```typescript
export type Action =
  | { type: "text"; text: string; classes: string }
  | { type: "newline" }
  | { type: "backspace" }
  | { type: "clear" };

export function parse(_input: string): Action[] {
  return [];
}
```

Write to `internal/web/frontend/src/ansi.ts`.

- [ ] **Step 2: Write tests for the ANSI parser**

```typescript
import { describe, it, expect } from "vitest";
import { parse } from "./ansi";

describe("parse", () => {
  it("returns plain text as a single segment", () => {
    expect(parse("hello")).toEqual([
      { type: "text", text: "hello", classes: "" },
    ]);
  });

  it("parses foreground color", () => {
    expect(parse("\x1b[32mhello\x1b[0m")).toEqual([
      { type: "text", text: "hello", classes: "fg-green" },
    ]);
  });

  it("parses bold attribute", () => {
    expect(parse("\x1b[1mhello\x1b[0m")).toEqual([
      { type: "text", text: "hello", classes: "bold" },
    ]);
  });

  it("parses combined bold + color (1;36m)", () => {
    expect(parse("\x1b[1;36mhello\x1b[0m")).toEqual([
      { type: "text", text: "hello", classes: "fg-cyan bold" },
    ]);
  });

  it("resets styles on \\x1b[0m", () => {
    expect(parse("\x1b[32mgreen\x1b[0m plain")).toEqual([
      { type: "text", text: "green", classes: "fg-green" },
      { type: "text", text: " plain", classes: "" },
    ]);
  });

  it("handles multiple color switches", () => {
    expect(parse("\x1b[32mA\x1b[36mB\x1b[0m")).toEqual([
      { type: "text", text: "A", classes: "fg-green" },
      { type: "text", text: "B", classes: "fg-cyan" },
    ]);
  });

  it("parses bright foreground colors", () => {
    expect(parse("\x1b[90mgray\x1b[0m")).toEqual([
      { type: "text", text: "gray", classes: "fg-bright-black" },
    ]);
  });

  it("emits newline for \\n and ignores \\r", () => {
    expect(parse("a\r\nb")).toEqual([
      { type: "text", text: "a", classes: "" },
      { type: "newline" },
      { type: "text", text: "b", classes: "" },
    ]);
  });

  it("emits backspace action for \\b", () => {
    expect(parse("\b \b")).toEqual([
      { type: "backspace" },
      { type: "text", text: " ", classes: "" },
      { type: "backspace" },
    ]);
  });

  it("emits clear for \\x1b[2J and ignores \\x1b[H", () => {
    expect(parse("\x1b[2J\x1b[H")).toEqual([{ type: "clear" }]);
  });

  it("strips unknown escape sequences", () => {
    expect(parse("a\x1b[?25hb")).toEqual([
      { type: "text", text: "a", classes: "" },
      { type: "text", text: "b", classes: "" },
    ]);
  });

  it("handles the server prompt pattern", () => {
    const prompt =
      "\x1b[32mvisitor\x1b[90m@\x1b[36mtewodros.me\x1b[90m:\x1b[34m~\x1b[90m$ \x1b[0m";
    const result = parse(prompt);
    expect(result).toEqual([
      { type: "text", text: "visitor", classes: "fg-green" },
      { type: "text", text: "@", classes: "fg-bright-black" },
      { type: "text", text: "tewodros.me", classes: "fg-cyan" },
      { type: "text", text: ":", classes: "fg-bright-black" },
      { type: "text", text: "~", classes: "fg-blue" },
      { type: "text", text: "$ ", classes: "fg-bright-black" },
    ]);
  });

  it("handles empty input", () => {
    expect(parse("")).toEqual([]);
  });

  it("handles red error text", () => {
    expect(parse("\x1b[31mError\x1b[0m")).toEqual([
      { type: "text", text: "Error", classes: "fg-red" },
    ]);
  });
});
```

Write to `internal/web/frontend/src/ansi.test.ts`.

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd internal/web/frontend && npx vitest run`

Expected: All 14 tests FAIL (the stub returns `[]` for everything).

- [ ] **Step 4: Commit**

```bash
git add internal/web/frontend/src/ansi.ts internal/web/frontend/src/ansi.test.ts
git commit -m "test: add ANSI parser tests (red phase)"
```

---

### Task 3: ANSI Parser — Implementation

**Files:**
- Modify: `internal/web/frontend/src/ansi.ts`

- [ ] **Step 1: Implement the parser**

Replace the contents of `internal/web/frontend/src/ansi.ts` with:

```typescript
export type Action =
  | { type: "text"; text: string; classes: string }
  | { type: "newline" }
  | { type: "backspace" }
  | { type: "clear" };

const FG_NAMES: Record<number, string> = {
  30: "black",
  31: "red",
  32: "green",
  33: "yellow",
  34: "blue",
  35: "magenta",
  36: "cyan",
  37: "white",
  90: "bright-black",
  91: "bright-red",
  92: "bright-green",
  93: "bright-yellow",
  94: "bright-blue",
  95: "bright-magenta",
  96: "bright-cyan",
  97: "bright-white",
};

export function parse(input: string): Action[] {
  const actions: Action[] = [];
  let bold = false;
  let fg = "";
  let buf = "";
  let i = 0;

  function flush(): void {
    if (buf) {
      const parts: string[] = [];
      if (fg) parts.push("fg-" + fg);
      if (bold) parts.push("bold");
      actions.push({ type: "text", text: buf, classes: parts.join(" ") });
      buf = "";
    }
  }

  while (i < input.length) {
    const ch = input[i];

    if (ch === "\x1b" && input[i + 1] === "[") {
      flush();
      i += 2;
      let params = "";
      while (
        i < input.length &&
        ((input[i] >= "0" && input[i] <= "9") || input[i] === ";")
      ) {
        params += input[i++];
      }
      const cmd = input[i++];

      if (cmd === "m") {
        const codes = params ? params.split(";").map(Number) : [0];
        for (const code of codes) {
          if (code === 0) {
            bold = false;
            fg = "";
          } else if (code === 1) {
            bold = true;
          } else if (FG_NAMES[code]) {
            fg = FG_NAMES[code];
          }
        }
      } else if (cmd === "J" && params === "2") {
        actions.push({ type: "clear" });
      }
      // \x1b[H (cursor home) and other unknown sequences are silently ignored
    } else if (ch === "\n") {
      flush();
      actions.push({ type: "newline" });
      i++;
    } else if (ch === "\r") {
      i++;
    } else if (ch === "\b") {
      flush();
      actions.push({ type: "backspace" });
      i++;
    } else {
      buf += ch;
      i++;
    }
  }

  flush();
  return actions;
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `cd internal/web/frontend && npx vitest run`

Expected: All 14 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/web/frontend/src/ansi.ts
git commit -m "feat: implement ANSI SGR parser"
```

---

### Task 4: Terminal Renderer

**Files:**
- Create: `internal/web/frontend/src/terminal.ts`

This module creates the DOM structure, renders ANSI-parsed output, and captures keyboard input.

- [ ] **Step 1: Write terminal.ts**

```typescript
import { parse } from "./ansi";

export class Terminal {
  private output: HTMLPreElement;
  private container: HTMLDivElement;
  private hiddenInput: HTMLTextAreaElement;
  private inputHandler: ((data: string) => void) | null = null;

  constructor(element: HTMLDivElement) {
    this.container = element;

    this.output = document.createElement("pre");
    this.output.className = "term-output";
    this.container.appendChild(this.output);

    // Hidden textarea captures keyboard on both desktop and mobile
    this.hiddenInput = document.createElement("textarea");
    this.hiddenInput.className = "term-hidden-input";
    this.hiddenInput.autocapitalize = "off";
    this.hiddenInput.setAttribute("autocomplete", "off");
    this.hiddenInput.setAttribute("autocorrect", "off");
    this.hiddenInput.setAttribute("spellcheck", "false");
    this.container.appendChild(this.hiddenInput);

    this.setupInput();
    this.hiddenInput.focus();
  }

  onInput(handler: (data: string) => void): void {
    this.inputHandler = handler;
  }

  write(data: string): void {
    const actions = parse(data);
    for (const action of actions) {
      switch (action.type) {
        case "text":
          this.appendText(action.text, action.classes);
          break;
        case "newline":
          this.output.appendChild(document.createElement("br"));
          break;
        case "backspace":
          this.removeLastChar();
          break;
        case "clear":
          this.output.innerHTML = "";
          break;
      }
    }
    this.container.scrollTop = this.container.scrollHeight;
  }

  private appendText(text: string, classes: string): void {
    const span = document.createElement("span");
    if (classes) span.className = classes;
    span.textContent = text;
    this.output.appendChild(span);
  }

  private removeLastChar(): void {
    const last = this.output.lastChild;
    if (last && last instanceof HTMLElement && last.textContent) {
      last.textContent = last.textContent.slice(0, -1);
      if (!last.textContent) this.output.removeChild(last);
    }
  }

  private emit(data: string): void {
    if (this.inputHandler) this.inputHandler(data);
  }

  private setupInput(): void {
    this.container.addEventListener("click", () => this.hiddenInput.focus());

    this.hiddenInput.addEventListener("keydown", (e: KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        this.emit("\r");
      } else if (e.key === "Backspace") {
        e.preventDefault();
        this.emit("\x7f");
      } else if (e.key === "Tab") {
        e.preventDefault();
        this.emit("\t");
      } else if (e.ctrlKey && e.key === "c") {
        e.preventDefault();
        this.emit("\x03");
      } else if (e.ctrlKey && e.key === "d") {
        e.preventDefault();
        this.emit("\x04");
      }
      // Printable chars are handled by the 'input' event below
    });

    this.hiddenInput.addEventListener("input", () => {
      const val = this.hiddenInput.value;
      if (val) {
        this.emit(val);
        this.hiddenInput.value = "";
      }
    });
  }
}
```

Write to `internal/web/frontend/src/terminal.ts`.

- [ ] **Step 2: Commit**

```bash
git add internal/web/frontend/src/terminal.ts
git commit -m "feat: add terminal DOM renderer with keyboard input"
```

---

### Task 5: WebSocket Client

**Files:**
- Create: `internal/web/frontend/src/websocket.ts`

Matches the protocol in `internal/web/bridge.go:25-30` — sends `{ type, data }` JSON, receives raw text.

- [ ] **Step 1: Write websocket.ts**

```typescript
export class WS {
  private ws: WebSocket | null = null;
  private url: string;

  onMessage: ((data: string) => void) | null = null;
  onDisconnect: ((msg: string) => void) | null = null;

  constructor() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    this.url = proto + "//" + location.host + "/ws";
  }

  connect(): void {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      this.send({ type: "resize", cols: 80, rows: 24 });
    };

    this.ws.onmessage = (e: MessageEvent) => {
      if (this.onMessage) this.onMessage(e.data);
    };

    this.ws.onclose = () => {
      if (this.onDisconnect) {
        this.onDisconnect(
          "\r\n\x1b[31mConnection closed. Refresh to reconnect.\x1b[0m\r\n" +
            "\x1b[90mOr connect via SSH: ssh tewodros.me\x1b[0m\r\n"
        );
      }
    };

    this.ws.onerror = () => {};
  }

  sendInput(data: string): void {
    this.send({ type: "input", data });
  }

  private send(msg: Record<string, unknown>): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }
}
```

Write to `internal/web/frontend/src/websocket.ts`.

- [ ] **Step 2: Commit**

```bash
git add internal/web/frontend/src/websocket.ts
git commit -m "feat: add WebSocket client"
```

---

### Task 6: Entry Point

**Files:**
- Create: `internal/web/frontend/src/main.ts`

- [ ] **Step 1: Write main.ts**

```typescript
import { Terminal } from "./terminal";
import { WS } from "./websocket";

const el = document.getElementById("terminal") as HTMLDivElement;
const term = new Terminal(el);
const ws = new WS();

ws.onMessage = (data) => term.write(data);
ws.onDisconnect = (msg) => term.write(msg);
term.onInput((data) => ws.sendInput(data));

ws.connect();
```

Write to `internal/web/frontend/src/main.ts`.

- [ ] **Step 2: Build the bundle**

Run: `cd internal/web/frontend && npx esbuild src/main.ts --bundle --minify --target=es2020 --outfile=../static/terminal.min.js`

Expected: `internal/web/static/terminal.min.js` created. Check file size:

Run: `wc -c internal/web/static/terminal.min.js`

Expected: Under 4096 bytes (4 KB).

- [ ] **Step 3: Commit**

```bash
git add internal/web/frontend/src/main.ts internal/web/static/terminal.min.js
git commit -m "feat: add entry point and build initial bundle"
```

---

### Task 7: Update CSS

**Files:**
- Modify: `internal/web/static/style.css`

Add terminal output styles and ANSI color classes. The color values come from the existing theme in `internal/web/static/terminal.js:8-29`.

- [ ] **Step 1: Update #terminal and add new styles to style.css**

First, update the existing `#terminal` block (lines 14-18) to add `overflow-y: auto` and `cursor: text`:

```css
#terminal {
    width: 100%;
    height: 100%;
    padding: 8px;
    overflow-y: auto;
    cursor: text;
}
```

Then add these new rules between the `#terminal` block and `.no-js` (line 20):

```css
.term-output {
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    font-size: 15px;
    color: #e0e0e0;
    line-height: 1.4;
    white-space: pre-wrap;
    word-wrap: break-word;
    margin: 0;
}

.term-hidden-input {
    position: absolute;
    left: -9999px;
    opacity: 0;
    width: 1px;
    height: 1px;
}

/* ANSI foreground colors */
.fg-black { color: #0a0a0a; }
.fg-red { color: #ff5555; }
.fg-green { color: #50fa7b; }
.fg-yellow { color: #f1fa8c; }
.fg-blue { color: #6272a4; }
.fg-magenta { color: #ff79c6; }
.fg-cyan { color: #8be9fd; }
.fg-white { color: #e0e0e0; }

/* ANSI bright foreground colors */
.fg-bright-black { color: #6272a4; }
.fg-bright-red { color: #ff6e6e; }
.fg-bright-green { color: #69ff94; }
.fg-bright-yellow { color: #ffffa5; }
.fg-bright-blue { color: #d6acff; }
.fg-bright-magenta { color: #ff92df; }
.fg-bright-cyan { color: #a4ffff; }
.fg-bright-white { color: #ffffff; }

.bold { font-weight: bold; }
```

- [ ] **Step 2: Commit**

```bash
git add internal/web/static/style.css
git commit -m "feat: add ANSI color classes and terminal output styles"
```

---

### Task 8: Update HTML

**Files:**
- Modify: `internal/web/static/index.html`

Remove all xterm.js references, add the new bundle.

- [ ] **Step 1: Remove xterm.js CDN link in head**

Remove line 47:

```html
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5/css/xterm.css">
```

- [ ] **Step 2: Remove xterm.js CDN scripts and old terminal.js**

Remove lines 81-84:

```html
    <script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5/lib/xterm.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0/lib/addon-fit.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0/lib/addon-web-links.js"></script>
    <script src="terminal.js"></script>
```

Replace with:

```html
    <script src="terminal.min.js"></script>
```

- [ ] **Step 3: Commit**

```bash
git add internal/web/static/index.html
git commit -m "feat: replace xterm.js CDN with micro-terminal bundle"
```

---

### Task 9: Clean Up and Verify

**Files:**
- Delete: `internal/web/static/terminal.js`

- [ ] **Step 1: Delete the old terminal.js**

Run: `rm internal/web/static/terminal.js`

- [ ] **Step 2: Rebuild the full project**

Run: `make build`

Expected: `frontend-build` runs first (generates `terminal.min.js`), then `go build` succeeds (the embedded static files include the new bundle, not the old one).

- [ ] **Step 3: Run ANSI parser tests**

Run: `cd internal/web/frontend && npx vitest run`

Expected: All 14 tests PASS.

- [ ] **Step 4: Verify bundle size**

Run: `wc -c internal/web/static/terminal.min.js`

Expected: Under 4096 bytes.

- [ ] **Step 5: Manual smoke test**

Run: `make run` (or `go run ./cmd/server`)

Open `http://localhost:8080` in a browser. Verify:
- Terminal renders with dark background
- Welcome message appears with colored text (green, cyan, gray)
- Prompt shows `visitor@tewodros.me:~$` with correct colors
- Typing echoes characters
- Backspace erases characters
- Enter submits commands
- `help` shows plain text output
- `about` shows about text
- `clear` clears the screen
- `ls` shows directory listing
- `cd guestbook` changes directory, prompt updates
- Tab completion works
- Ctrl+C shows `^C` and new prompt

- [ ] **Step 6: Commit deletion and final state**

```bash
git rm internal/web/static/terminal.js
git commit -m "chore: remove old xterm.js terminal client"
```
