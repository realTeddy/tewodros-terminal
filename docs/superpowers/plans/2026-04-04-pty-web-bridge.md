# PTY Web Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the line-based WebSocket bridge with a PTY-backed bridge so browser visitors get the identical Bubble Tea TUI as SSH users.

**Architecture:** Each WebSocket connection opens a PTY, runs Bubble Tea on the slave side, and shuttles raw bytes between PTY master and WebSocket. The frontend micro-terminal is rewritten as a screen-buffer renderer that interprets cursor positioning, erase sequences, and extended SGR codes.

**Tech Stack:** Go (creack/pty, Bubble Tea v2, Gorilla WebSocket), TypeScript (custom ANSI parser + screen buffer terminal), esbuild, Vitest

---

## File Structure

| File | Role |
|------|------|
| `internal/web/bridge.go` | **Rewrite** — PTY bridge: WebSocket ↔ PTY master, resize handling |
| `internal/web/bridge_windows.go` | **Create** — Build-tag stub for Windows (logs unsupported, closes connection) |
| `internal/web/frontend/src/screenbuffer.ts` | **Create** — Screen buffer: 2D cell grid, cursor state, cell manipulation |
| `internal/web/frontend/src/ansi.ts` | **Rewrite** — ANSI parser that operates on a screen buffer (cursor movement, erase, SGR) |
| `internal/web/frontend/src/terminal.ts` | **Rewrite** — DOM renderer: reads screen buffer, diffs dirty rows, renders to `<pre>` |
| `internal/web/frontend/src/websocket.ts` | **Modify** — Binary messages for terminal I/O, JSON for resize |
| `internal/web/frontend/src/main.ts` | **Modify** — Wire up ResizeObserver, pass dimensions to WS |
| `internal/web/frontend/src/ansi.test.ts` | **Rewrite** — Tests for screen-buffer-based parser |
| `internal/web/frontend/src/screenbuffer.test.ts` | **Create** — Tests for screen buffer operations |
| `internal/web/static/style.css` | **Modify** — Grid-based terminal styles |

---

### Task 1: Screen Buffer Module

**Files:**
- Create: `internal/web/frontend/src/screenbuffer.ts`
- Create: `internal/web/frontend/src/screenbuffer.test.ts`

- [ ] **Step 1: Write failing tests for screen buffer**

Create `internal/web/frontend/src/screenbuffer.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { ScreenBuffer } from "./screenbuffer";

describe("ScreenBuffer", () => {
  it("initializes with empty cells", () => {
    const sb = new ScreenBuffer(80, 24);
    expect(sb.rows).toBe(24);
    expect(sb.cols).toBe(80);
    expect(sb.getCell(0, 0)).toEqual({ char: " ", fg: "", bg: "", bold: false });
  });

  it("writes a character at cursor position and advances cursor", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.writeChar("A", { fg: "green", bg: "", bold: false });
    expect(sb.getCell(0, 0)).toEqual({ char: "A", fg: "green", bg: "", bold: false });
    expect(sb.cursorCol).toBe(1);
    expect(sb.cursorRow).toBe(0);
  });

  it("wraps cursor to next line at end of row", () => {
    const sb = new ScreenBuffer(3, 24);
    sb.writeChar("A", { fg: "", bg: "", bold: false });
    sb.writeChar("B", { fg: "", bg: "", bold: false });
    sb.writeChar("C", { fg: "", bg: "", bold: false });
    expect(sb.cursorRow).toBe(1);
    expect(sb.cursorCol).toBe(0);
  });

  it("scrolls up when cursor moves past last row", () => {
    const sb = new ScreenBuffer(3, 2);
    sb.writeChar("A", { fg: "", bg: "", bold: false });
    sb.writeChar("B", { fg: "", bg: "", bold: false });
    sb.writeChar("C", { fg: "", bg: "", bold: false }); // wraps to row 1
    sb.writeChar("D", { fg: "", bg: "", bold: false });
    sb.writeChar("E", { fg: "", bg: "", bold: false });
    sb.writeChar("F", { fg: "", bg: "", bold: false }); // wraps past row 1 → scroll
    // Row 0 should now contain D E F (old row 1), row 1 should be blank
    // Actually: after scroll, old row 1 (D E F) moves to row 0, new row 1 is empty
    // Cursor is at row 1, col 0
    expect(sb.cursorRow).toBe(1);
    expect(sb.cursorCol).toBe(0);
    expect(sb.getCell(0, 0).char).toBe("D");
  });

  it("handles carriage return", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.writeChar("A", { fg: "", bg: "", bold: false });
    sb.writeChar("B", { fg: "", bg: "", bold: false });
    sb.carriageReturn();
    expect(sb.cursorCol).toBe(0);
    expect(sb.cursorRow).toBe(0);
  });

  it("handles line feed", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.lineFeed();
    expect(sb.cursorRow).toBe(1);
    expect(sb.cursorCol).toBe(0);
  });

  it("handles setCursor", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.setCursor(5, 10);
    expect(sb.cursorRow).toBe(5);
    expect(sb.cursorCol).toBe(10);
  });

  it("clamps setCursor to bounds", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.setCursor(100, 200);
    expect(sb.cursorRow).toBe(23);
    expect(sb.cursorCol).toBe(79);
  });

  it("moves cursor up", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.setCursor(5, 10);
    sb.moveCursor(-3, 0);
    expect(sb.cursorRow).toBe(2);
    expect(sb.cursorCol).toBe(10);
  });

  it("clamps cursor up at row 0", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.setCursor(2, 0);
    sb.moveCursor(-10, 0);
    expect(sb.cursorRow).toBe(0);
  });

  it("moves cursor right", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.moveCursor(0, 5);
    expect(sb.cursorCol).toBe(5);
  });

  it("erases to end of line", () => {
    const sb = new ScreenBuffer(5, 1);
    const style = { fg: "", bg: "", bold: false };
    sb.writeChar("A", style);
    sb.writeChar("B", style);
    sb.writeChar("C", style);
    sb.writeChar("D", style);
    sb.writeChar("E", style);
    sb.setCursor(0, 2);
    sb.eraseInLine(0); // erase from cursor to end
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe("B");
    expect(sb.getCell(0, 2).char).toBe(" ");
    expect(sb.getCell(0, 3).char).toBe(" ");
    expect(sb.getCell(0, 4).char).toBe(" ");
  });

  it("erases entire line", () => {
    const sb = new ScreenBuffer(5, 1);
    const style = { fg: "", bg: "", bold: false };
    sb.writeChar("A", style);
    sb.writeChar("B", style);
    sb.eraseInLine(2);
    expect(sb.getCell(0, 0).char).toBe(" ");
    expect(sb.getCell(0, 1).char).toBe(" ");
  });

  it("erases to end of display", () => {
    const sb = new ScreenBuffer(3, 2);
    const style = { fg: "", bg: "", bold: false };
    sb.writeChar("A", style);
    sb.writeChar("B", style);
    sb.writeChar("C", style);
    sb.writeChar("D", style);
    sb.writeChar("E", style);
    sb.writeChar("F", style);
    sb.setCursor(0, 1);
    sb.eraseInDisplay(0); // erase from cursor to end of display
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe(" ");
    expect(sb.getCell(0, 2).char).toBe(" ");
    expect(sb.getCell(1, 0).char).toBe(" ");
  });

  it("erases entire display", () => {
    const sb = new ScreenBuffer(3, 2);
    const style = { fg: "", bg: "", bold: false };
    sb.writeChar("A", style);
    sb.writeChar("B", style);
    sb.setCursor(0, 0);
    sb.eraseInDisplay(2);
    expect(sb.getCell(0, 0).char).toBe(" ");
    expect(sb.getCell(0, 1).char).toBe(" ");
  });

  it("tracks dirty rows", () => {
    const sb = new ScreenBuffer(80, 24);
    sb.clearDirty();
    sb.writeChar("A", { fg: "", bg: "", bold: false });
    expect(sb.isDirty(0)).toBe(true);
    expect(sb.isDirty(1)).toBe(false);
  });

  it("resizes and preserves content", () => {
    const sb = new ScreenBuffer(5, 2);
    const style = { fg: "", bg: "", bold: false };
    sb.writeChar("A", style);
    sb.writeChar("B", style);
    sb.resize(10, 3);
    expect(sb.cols).toBe(10);
    expect(sb.rows).toBe(3);
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe("B");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/web/frontend && npx vitest run src/screenbuffer.test.ts`
Expected: FAIL — module `./screenbuffer` not found

- [ ] **Step 3: Implement the screen buffer**

Create `internal/web/frontend/src/screenbuffer.ts`:

```typescript
export interface CellStyle {
  fg: string;
  bg: string;
  bold: boolean;
}

export interface Cell {
  char: string;
  fg: string;
  bg: string;
  bold: boolean;
}

function emptyCell(): Cell {
  return { char: " ", fg: "", bg: "", bold: false };
}

export class ScreenBuffer {
  private cells: Cell[][];
  private dirty: Set<number>;
  rows: number;
  cols: number;
  cursorRow = 0;
  cursorCol = 0;
  cursorVisible = true;

  constructor(cols: number, rows: number) {
    this.cols = cols;
    this.rows = rows;
    this.cells = [];
    this.dirty = new Set();
    for (let r = 0; r < rows; r++) {
      this.cells.push(this.newRow());
      this.dirty.add(r);
    }
  }

  private newRow(): Cell[] {
    const row: Cell[] = [];
    for (let c = 0; c < this.cols; c++) row.push(emptyCell());
    return row;
  }

  getCell(row: number, col: number): Cell {
    if (row < 0 || row >= this.rows || col < 0 || col >= this.cols) return emptyCell();
    return this.cells[row][col];
  }

  writeChar(ch: string, style: CellStyle): void {
    if (this.cursorCol >= this.cols) {
      this.cursorCol = 0;
      this.cursorRow++;
      if (this.cursorRow >= this.rows) this.scrollUp();
    }
    this.cells[this.cursorRow][this.cursorCol] = {
      char: ch,
      fg: style.fg,
      bg: style.bg,
      bold: style.bold,
    };
    this.dirty.add(this.cursorRow);
    this.cursorCol++;
  }

  carriageReturn(): void {
    this.cursorCol = 0;
  }

  lineFeed(): void {
    this.cursorRow++;
    if (this.cursorRow >= this.rows) this.scrollUp();
  }

  private scrollUp(): void {
    this.cells.shift();
    this.cells.push(this.newRow());
    this.cursorRow = this.rows - 1;
    // Mark all rows dirty after scroll
    for (let r = 0; r < this.rows; r++) this.dirty.add(r);
  }

  setCursor(row: number, col: number): void {
    this.cursorRow = Math.max(0, Math.min(row, this.rows - 1));
    this.cursorCol = Math.max(0, Math.min(col, this.cols - 1));
  }

  moveCursor(dRow: number, dCol: number): void {
    this.setCursor(this.cursorRow + dRow, this.cursorCol + dCol);
  }

  eraseInLine(mode: number): void {
    const row = this.cells[this.cursorRow];
    if (!row) return;
    let start: number, end: number;
    if (mode === 0) { start = this.cursorCol; end = this.cols; }
    else if (mode === 1) { start = 0; end = this.cursorCol + 1; }
    else { start = 0; end = this.cols; }
    for (let c = start; c < end; c++) row[c] = emptyCell();
    this.dirty.add(this.cursorRow);
  }

  eraseInDisplay(mode: number): void {
    if (mode === 0) {
      // Erase from cursor to end
      this.eraseInLine(0);
      for (let r = this.cursorRow + 1; r < this.rows; r++) {
        this.cells[r] = this.newRow();
        this.dirty.add(r);
      }
    } else if (mode === 1) {
      // Erase from start to cursor
      for (let r = 0; r < this.cursorRow; r++) {
        this.cells[r] = this.newRow();
        this.dirty.add(r);
      }
      this.eraseInLine(1);
    } else {
      // Erase entire display
      for (let r = 0; r < this.rows; r++) {
        this.cells[r] = this.newRow();
        this.dirty.add(r);
      }
    }
  }

  isDirty(row: number): boolean {
    return this.dirty.has(row);
  }

  clearDirty(): void {
    this.dirty.clear();
  }

  markAllDirty(): void {
    for (let r = 0; r < this.rows; r++) this.dirty.add(r);
  }

  getRow(row: number): Cell[] {
    return this.cells[row] || [];
  }

  resize(cols: number, rows: number): void {
    const newCells: Cell[][] = [];
    for (let r = 0; r < rows; r++) {
      const newRow: Cell[] = [];
      for (let c = 0; c < cols; c++) {
        if (r < this.rows && c < this.cols) {
          newRow.push(this.cells[r][c]);
        } else {
          newRow.push(emptyCell());
        }
      }
      newCells.push(newRow);
    }
    this.cells = newCells;
    this.rows = rows;
    this.cols = cols;
    this.cursorRow = Math.min(this.cursorRow, rows - 1);
    this.cursorCol = Math.min(this.cursorCol, cols - 1);
    this.markAllDirty();
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd internal/web/frontend && npx vitest run src/screenbuffer.test.ts`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/web/frontend/src/screenbuffer.ts internal/web/frontend/src/screenbuffer.test.ts
git commit -m "feat(web): add screen buffer module for grid-based terminal rendering"
```

---

### Task 2: ANSI Parser Rewrite

**Files:**
- Rewrite: `internal/web/frontend/src/ansi.ts`
- Rewrite: `internal/web/frontend/src/ansi.test.ts`

The parser no longer returns `Action[]`. Instead, it takes a `ScreenBuffer` and mutates it directly — writing characters, moving the cursor, erasing, and changing SGR state.

- [ ] **Step 1: Write failing tests for the new parser**

Rewrite `internal/web/frontend/src/ansi.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { AnsiParser } from "./ansi";
import { ScreenBuffer } from "./screenbuffer";

describe("AnsiParser", () => {
  function parse(input: string, cols = 80, rows = 24): ScreenBuffer {
    const sb = new ScreenBuffer(cols, rows);
    const parser = new AnsiParser(sb);
    parser.feed(input);
    return sb;
  }

  it("writes plain text at cursor position", () => {
    const sb = parse("hello");
    expect(sb.getCell(0, 0).char).toBe("h");
    expect(sb.getCell(0, 4).char).toBe("o");
    expect(sb.cursorCol).toBe(5);
  });

  it("applies foreground color", () => {
    const sb = parse("\x1b[32mhi\x1b[0m");
    expect(sb.getCell(0, 0)).toMatchObject({ char: "h", fg: "green" });
    expect(sb.getCell(0, 1)).toMatchObject({ char: "i", fg: "green" });
  });

  it("applies bold", () => {
    const sb = parse("\x1b[1mhi\x1b[0m");
    expect(sb.getCell(0, 0).bold).toBe(true);
  });

  it("applies combined bold + color", () => {
    const sb = parse("\x1b[1;36mhi\x1b[0m");
    expect(sb.getCell(0, 0)).toMatchObject({ char: "h", fg: "cyan", bold: true });
  });

  it("resets style on \\x1b[0m", () => {
    const sb = parse("\x1b[32mA\x1b[0mB");
    expect(sb.getCell(0, 0).fg).toBe("green");
    expect(sb.getCell(0, 1).fg).toBe("");
  });

  it("applies bright foreground colors", () => {
    const sb = parse("\x1b[90mA\x1b[0m");
    expect(sb.getCell(0, 0).fg).toBe("bright-black");
  });

  it("applies background color", () => {
    const sb = parse("\x1b[41mA\x1b[0m");
    expect(sb.getCell(0, 0)).toMatchObject({ char: "A", bg: "red" });
  });

  it("applies bright background color", () => {
    const sb = parse("\x1b[101mA\x1b[0m");
    expect(sb.getCell(0, 0)).toMatchObject({ char: "A", bg: "bright-red" });
  });

  it("applies 256-color foreground", () => {
    const sb = parse("\x1b[38;5;196mA\x1b[0m");
    expect(sb.getCell(0, 0)).toMatchObject({ char: "A", fg: "color-196" });
  });

  it("applies 256-color background", () => {
    const sb = parse("\x1b[48;5;21mA\x1b[0m");
    expect(sb.getCell(0, 0)).toMatchObject({ char: "A", bg: "color-21" });
  });

  it("resets fg with code 39", () => {
    const sb = parse("\x1b[32mA\x1b[39mB");
    expect(sb.getCell(0, 0).fg).toBe("green");
    expect(sb.getCell(0, 1).fg).toBe("");
  });

  it("resets bg with code 49", () => {
    const sb = parse("\x1b[41mA\x1b[49mB");
    expect(sb.getCell(0, 0).bg).toBe("red");
    expect(sb.getCell(0, 1).bg).toBe("");
  });

  it("resets bold with code 22", () => {
    const sb = parse("\x1b[1mA\x1b[22mB");
    expect(sb.getCell(0, 0).bold).toBe(true);
    expect(sb.getCell(0, 1).bold).toBe(false);
  });

  it("handles \\r\\n as carriage return + line feed", () => {
    const sb = parse("AB\r\nCD");
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe("B");
    expect(sb.getCell(1, 0).char).toBe("C");
    expect(sb.getCell(1, 1).char).toBe("D");
  });

  it("handles \\r alone as carriage return (overwrite)", () => {
    const sb = parse("ABC\rXY");
    expect(sb.getCell(0, 0).char).toBe("X");
    expect(sb.getCell(0, 1).char).toBe("Y");
    expect(sb.getCell(0, 2).char).toBe("C");
  });

  it("handles cursor position CSI row;col H", () => {
    const sb = parse("\x1b[3;5HA");
    // CSI params are 1-based
    expect(sb.getCell(2, 4).char).toBe("A");
  });

  it("handles cursor position CSI H (home)", () => {
    const sb = parse("ZZZ\x1b[HA");
    expect(sb.getCell(0, 0).char).toBe("A");
  });

  it("handles cursor up CSI n A", () => {
    const sb = parse("\x1b[5;1H\x1b[2AA");
    expect(sb.getCell(2, 0).char).toBe("A");
  });

  it("handles cursor down CSI n B", () => {
    const sb = parse("\x1b[2BA");
    expect(sb.getCell(2, 0).char).toBe("A");
  });

  it("handles cursor forward CSI n C", () => {
    const sb = parse("\x1b[5CA");
    expect(sb.getCell(0, 5).char).toBe("A");
  });

  it("handles cursor backward CSI n D", () => {
    const sb = parse("ABCDE\x1b[3DA");
    // cursor at col 5 after ABCDE, move back 3 → col 2
    expect(sb.getCell(0, 2).char).toBe("A");
  });

  it("handles cursor horizontal absolute CSI n G", () => {
    const sb = parse("\x1b[10GA");
    // 1-based, so column 10 → index 9
    expect(sb.getCell(0, 9).char).toBe("A");
  });

  it("handles erase to end of line CSI K", () => {
    const sb = parse("ABCDE\x1b[1;3H\x1b[K");
    // cursor at row 0, col 2; erase from col 2 to end
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe("B");
    expect(sb.getCell(0, 2).char).toBe(" ");
    expect(sb.getCell(0, 3).char).toBe(" ");
  });

  it("handles erase entire line CSI 2 K", () => {
    const sb = parse("ABCDE\x1b[1;3H\x1b[2K");
    expect(sb.getCell(0, 0).char).toBe(" ");
    expect(sb.getCell(0, 4).char).toBe(" ");
  });

  it("handles erase display CSI 2 J", () => {
    const sb = parse("ABC\r\nDEF\x1b[2J");
    expect(sb.getCell(0, 0).char).toBe(" ");
    expect(sb.getCell(1, 0).char).toBe(" ");
  });

  it("handles erase to end of display CSI J", () => {
    const sb = parse("ABC\r\nDEF\x1b[1;2H\x1b[J");
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe(" ");
    expect(sb.getCell(1, 0).char).toBe(" ");
  });

  it("handles show cursor CSI ?25h", () => {
    const sb = new ScreenBuffer(80, 24);
    const parser = new AnsiParser(sb);
    parser.feed("\x1b[?25l");
    expect(sb.cursorVisible).toBe(false);
    parser.feed("\x1b[?25h");
    expect(sb.cursorVisible).toBe(true);
  });

  it("handles backspace", () => {
    const sb = parse("AB\bC");
    // backspace moves cursor left, then C overwrites B
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe("C");
  });

  it("silently ignores unknown sequences", () => {
    const sb = parse("A\x1b[?1049hB");
    expect(sb.getCell(0, 0).char).toBe("A");
    expect(sb.getCell(0, 1).char).toBe("B");
  });

  it("handles the server prompt pattern", () => {
    const prompt =
      "\x1b[32mvisitor\x1b[90m@\x1b[36mtewodros.me\x1b[90m:\x1b[34m~\x1b[90m$ \x1b[0m";
    const sb = parse(prompt);
    expect(sb.getCell(0, 0)).toMatchObject({ char: "v", fg: "green" });
    expect(sb.getCell(0, 7)).toMatchObject({ char: "@", fg: "bright-black" });
    expect(sb.getCell(0, 8)).toMatchObject({ char: "t", fg: "cyan" });
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/web/frontend && npx vitest run src/ansi.test.ts`
Expected: FAIL — `AnsiParser` not exported from `./ansi`

- [ ] **Step 3: Implement the ANSI parser**

Rewrite `internal/web/frontend/src/ansi.ts`:

```typescript
import { ScreenBuffer, CellStyle } from "./screenbuffer";

const FG_NAMES: Record<number, string> = {
  30: "black", 31: "red", 32: "green", 33: "yellow",
  34: "blue", 35: "magenta", 36: "cyan", 37: "white",
  90: "bright-black", 91: "bright-red", 92: "bright-green", 93: "bright-yellow",
  94: "bright-blue", 95: "bright-magenta", 96: "bright-cyan", 97: "bright-white",
};

const BG_NAMES: Record<number, string> = {
  40: "black", 41: "red", 42: "green", 43: "yellow",
  44: "blue", 45: "magenta", 46: "cyan", 47: "white",
  100: "bright-black", 101: "bright-red", 102: "bright-green", 103: "bright-yellow",
  104: "bright-blue", 105: "bright-magenta", 106: "bright-cyan", 107: "bright-white",
};

export class AnsiParser {
  private sb: ScreenBuffer;
  private fg = "";
  private bg = "";
  private bold = false;

  constructor(sb: ScreenBuffer) {
    this.sb = sb;
  }

  private style(): CellStyle {
    return { fg: this.fg, bg: this.bg, bold: this.bold };
  }

  feed(input: string): void {
    let i = 0;
    while (i < input.length) {
      const ch = input[i];

      if (ch === "\x1b" && input[i + 1] === "[") {
        i += 2;
        let params = "";
        while (i < input.length && (input.charCodeAt(i) < 0x40 || input.charCodeAt(i) > 0x7e)) {
          params += input[i++];
        }
        const cmd = input[i++];
        this.handleCSI(params, cmd);
      } else if (ch === "\x1b") {
        // Skip other ESC sequences (e.g. ESC ] for OSC)
        i++;
        if (i < input.length) i++; // skip next char
      } else if (ch === "\r") {
        this.sb.carriageReturn();
        i++;
      } else if (ch === "\n") {
        this.sb.lineFeed();
        i++;
      } else if (ch === "\b") {
        this.sb.moveCursor(0, -1);
        i++;
      } else if (ch === "\t") {
        // Tab: move to next multiple of 8
        const nextTab = (Math.floor(this.sb.cursorCol / 8) + 1) * 8;
        this.sb.setCursor(this.sb.cursorRow, Math.min(nextTab, this.sb.cols - 1));
        i++;
      } else if (ch.charCodeAt(0) >= 32) {
        this.sb.writeChar(ch, this.style());
        i++;
      } else {
        i++; // skip other control chars
      }
    }
  }

  private handleCSI(params: string, cmd: string): void {
    const parts = params ? params.split(";").map(s => parseInt(s, 10) || 0) : [];
    const n = parts[0] || 0;

    switch (cmd) {
      case "m":
        this.handleSGR(params ? params.split(";").map(Number) : [0]);
        break;
      case "A": // Cursor Up
        this.sb.moveCursor(-(n || 1), 0);
        break;
      case "B": // Cursor Down
        this.sb.moveCursor(n || 1, 0);
        break;
      case "C": // Cursor Forward
        this.sb.moveCursor(0, n || 1);
        break;
      case "D": // Cursor Backward
        this.sb.moveCursor(0, -(n || 1));
        break;
      case "G": // Cursor Horizontal Absolute (1-based)
        this.sb.setCursor(this.sb.cursorRow, (n || 1) - 1);
        break;
      case "H": // Cursor Position (1-based)
      case "f":
        this.sb.setCursor((parts[0] || 1) - 1, (parts[1] || 1) - 1);
        break;
      case "J": // Erase in Display
        this.sb.eraseInDisplay(n);
        break;
      case "K": // Erase in Line
        this.sb.eraseInLine(n);
        break;
      case "h": // Set Mode
        if (params === "?25") this.sb.cursorVisible = true;
        break;
      case "l": // Reset Mode
        if (params === "?25") this.sb.cursorVisible = false;
        break;
      // Silently ignore everything else
    }
  }

  private handleSGR(codes: number[]): void {
    let i = 0;
    while (i < codes.length) {
      const code = codes[i];
      if (code === 0) {
        this.bold = false;
        this.fg = "";
        this.bg = "";
      } else if (code === 1) {
        this.bold = true;
      } else if (code === 22) {
        this.bold = false;
      } else if (code === 39) {
        this.fg = "";
      } else if (code === 49) {
        this.bg = "";
      } else if (FG_NAMES[code]) {
        this.fg = FG_NAMES[code];
      } else if (BG_NAMES[code]) {
        this.bg = BG_NAMES[code];
      } else if (code === 38 && codes[i + 1] === 5) {
        // 256-color foreground: 38;5;n
        this.fg = "color-" + (codes[i + 2] ?? 0);
        i += 2;
      } else if (code === 48 && codes[i + 1] === 5) {
        // 256-color background: 48;5;n
        this.bg = "color-" + (codes[i + 2] ?? 0);
        i += 2;
      }
      i++;
    }
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd internal/web/frontend && npx vitest run src/ansi.test.ts`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/web/frontend/src/ansi.ts internal/web/frontend/src/ansi.test.ts
git commit -m "feat(web): rewrite ANSI parser to operate on screen buffer"
```

---

### Task 3: Terminal DOM Renderer Rewrite

**Files:**
- Rewrite: `internal/web/frontend/src/terminal.ts`
- Modify: `internal/web/static/style.css`

- [ ] **Step 1: Rewrite terminal.ts with screen buffer rendering**

Rewrite `internal/web/frontend/src/terminal.ts`:

```typescript
import { ScreenBuffer } from "./screenbuffer";
import { AnsiParser } from "./ansi";

export class Terminal {
  private container: HTMLDivElement;
  private output: HTMLPreElement;
  private hiddenInput: HTMLTextAreaElement;
  private sb: ScreenBuffer;
  private parser: AnsiParser;
  private rowEls: HTMLDivElement[] = [];
  private cursorEl: HTMLSpanElement;
  private inputHandler: ((data: string) => void) | null = null;

  constructor(element: HTMLDivElement, cols: number, rows: number) {
    this.container = element;
    this.sb = new ScreenBuffer(cols, rows);
    this.parser = new AnsiParser(this.sb);

    this.output = document.createElement("pre");
    this.output.className = "term-output";
    this.container.appendChild(this.output);

    // Create row elements
    for (let r = 0; r < rows; r++) {
      const row = document.createElement("div");
      row.className = "term-row";
      this.output.appendChild(row);
      this.rowEls.push(row);
    }

    // Cursor overlay
    this.cursorEl = document.createElement("span");
    this.cursorEl.className = "term-cursor";
    this.container.appendChild(this.cursorEl);

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
    this.render();
  }

  onInput(handler: (data: string) => void): void {
    this.inputHandler = handler;
  }

  write(data: string): void {
    this.parser.feed(data);
    this.render();
  }

  resize(cols: number, rows: number): void {
    this.sb.resize(cols, rows);
    // Adjust row elements
    while (this.rowEls.length < rows) {
      const row = document.createElement("div");
      row.className = "term-row";
      this.output.appendChild(row);
      this.rowEls.push(row);
    }
    while (this.rowEls.length > rows) {
      const row = this.rowEls.pop()!;
      this.output.removeChild(row);
    }
    this.render();
  }

  private render(): void {
    for (let r = 0; r < this.sb.rows; r++) {
      if (!this.sb.isDirty(r)) continue;
      this.renderRow(r);
    }
    this.sb.clearDirty();
    this.updateCursor();
  }

  private renderRow(r: number): void {
    const rowEl = this.rowEls[r];
    if (!rowEl) return;
    rowEl.innerHTML = "";

    const cells = this.sb.getRow(r);
    let i = 0;
    while (i < cells.length) {
      const cell = cells[i];
      const cls = this.cellClasses(cell);

      // Merge consecutive cells with the same style
      let text = cell.char;
      let j = i + 1;
      while (j < cells.length && this.cellClasses(cells[j]) === cls) {
        text += cells[j].char;
        j++;
      }

      // Trim trailing spaces on the last span of the row
      if (j === cells.length) text = text.replace(/ +$/, "");

      if (text) {
        if (cls) {
          const span = document.createElement("span");
          span.className = cls;
          span.textContent = text;
          rowEl.appendChild(span);
        } else {
          rowEl.appendChild(document.createTextNode(text));
        }
      }

      i = j;
    }

    // Auto-linkify URLs in the rendered text
    this.linkifyRow(rowEl);
  }

  private linkifyRow(rowEl: HTMLDivElement): void {
    const walker = document.createTreeWalker(rowEl, NodeFilter.SHOW_TEXT);
    const urlRe = /(https?:\/\/[^\s)]+)/g;
    const textNodes: Text[] = [];
    let node: Text | null;
    while ((node = walker.nextNode() as Text | null)) textNodes.push(node);

    for (const textNode of textNodes) {
      const text = textNode.textContent || "";
      if (!urlRe.test(text)) continue;
      urlRe.lastIndex = 0;

      const frag = document.createDocumentFragment();
      let last = 0;
      let match: RegExpExecArray | null;
      while ((match = urlRe.exec(text)) !== null) {
        if (match.index > last) {
          frag.appendChild(document.createTextNode(text.slice(last, match.index)));
        }
        const a = document.createElement("a");
        a.href = match[0];
        a.target = "_blank";
        a.rel = "noopener noreferrer";
        a.textContent = match[0];
        frag.appendChild(a);
        last = match.index + match[0].length;
      }
      if (last < text.length) {
        frag.appendChild(document.createTextNode(text.slice(last)));
      }
      textNode.parentNode!.replaceChild(frag, textNode);
    }
  }

  private cellClasses(cell: { fg: string; bg: string; bold: boolean }): string {
    const parts: string[] = [];
    if (cell.fg) parts.push("fg-" + cell.fg);
    if (cell.bg) parts.push("bg-" + cell.bg);
    if (cell.bold) parts.push("bold");
    return parts.join(" ");
  }

  private updateCursor(): void {
    if (!this.sb.cursorVisible) {
      this.cursorEl.style.display = "none";
      return;
    }
    this.cursorEl.style.display = "";
    // Position cursor relative to the output container
    const rowEl = this.rowEls[this.sb.cursorRow];
    if (!rowEl) return;

    // Measure character size from the output font
    const style = getComputedStyle(this.output);
    const probe = document.createElement("span");
    probe.style.font = style.font;
    probe.style.visibility = "hidden";
    probe.style.position = "absolute";
    probe.textContent = "X";
    this.output.appendChild(probe);
    const charW = probe.getBoundingClientRect().width;
    const charH = parseFloat(style.lineHeight) || probe.getBoundingClientRect().height;
    this.output.removeChild(probe);

    const outputRect = this.output.getBoundingClientRect();
    const x = this.sb.cursorCol * charW;
    const y = this.sb.cursorRow * charH;

    this.cursorEl.style.position = "absolute";
    this.cursorEl.style.left = (outputRect.left - this.container.getBoundingClientRect().left + x) + "px";
    this.cursorEl.style.top = (outputRect.top - this.container.getBoundingClientRect().top + y) + "px";
    this.cursorEl.style.width = charW + "px";
    this.cursorEl.style.height = charH + "px";
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
      } else if (e.key === "Escape") {
        e.preventDefault();
        this.emit("\x1b");
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        this.emit("\x1b[A");
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        this.emit("\x1b[B");
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        this.emit("\x1b[C");
      } else if (e.key === "ArrowLeft") {
        e.preventDefault();
        this.emit("\x1b[D");
      }
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

export function measureTermSize(container: HTMLElement): { cols: number; rows: number } {
  const style = getComputedStyle(container);
  const probe = document.createElement("span");
  probe.style.font = style.font || "15px 'Fira Code', 'Cascadia Code', 'Consolas', monospace";
  probe.style.visibility = "hidden";
  probe.style.position = "absolute";
  probe.textContent = "X";
  container.appendChild(probe);
  const charW = probe.getBoundingClientRect().width;
  const charH = probe.getBoundingClientRect().height;
  container.removeChild(probe);
  const padding = parseFloat(style.paddingLeft || "0") + parseFloat(style.paddingRight || "0");
  const vPadding = parseFloat(style.paddingTop || "0") + parseFloat(style.paddingBottom || "0");
  const cols = Math.floor((container.clientWidth - padding) / charW);
  const rows = Math.floor((container.clientHeight - vPadding) / charH);
  return { cols: Math.max(cols, 1), rows: Math.max(rows, 1) };
}
```

- [ ] **Step 2: Update style.css for grid-based rendering**

Replace the `.term-output` and `.term-cursor` sections in `internal/web/static/style.css`. Add background color classes. The changes:

In `internal/web/static/style.css`, replace the `.term-output` block:

```css
.term-output {
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    font-size: 15px;
    color: #e0e0e0;
    line-height: 1.4;
    white-space: pre;
    margin: 0;
}
```

Note: changed from `pre-wrap` to `pre` — the screen buffer handles wrapping.

Add `.term-row` after `.term-output`:

```css
.term-row {
    height: 1.4em;
}
```

Add background color classes after the existing foreground classes:

```css
/* ANSI background colors */
.bg-black { background-color: #0a0a0a; }
.bg-red { background-color: #ff5555; }
.bg-green { background-color: #50fa7b; }
.bg-yellow { background-color: #f1fa8c; }
.bg-blue { background-color: #6272a4; }
.bg-magenta { background-color: #ff79c6; }
.bg-cyan { background-color: #8be9fd; }
.bg-white { background-color: #e0e0e0; }

/* ANSI bright background colors */
.bg-bright-black { background-color: #6272a4; }
.bg-bright-red { background-color: #ff6e6e; }
.bg-bright-green { background-color: #69ff94; }
.bg-bright-yellow { background-color: #ffffa5; }
.bg-bright-blue { background-color: #d6acff; }
.bg-bright-magenta { background-color: #ff92df; }
.bg-bright-cyan { background-color: #a4ffff; }
.bg-bright-white { background-color: #ffffff; }
```

Replace the `.term-cursor` block:

```css
.term-cursor {
    position: absolute;
    width: 0.6em;
    height: 1.4em;
    background: rgba(0, 255, 136, 0.7);
    animation: blink 1s step-end infinite;
    pointer-events: none;
}
```

Remove the old `word-wrap: break-word;` property since screen buffer handles wrapping.

- [ ] **Step 3: Run all frontend tests**

Run: `cd internal/web/frontend && npx vitest run`
Expected: all tests in `ansi.test.ts` and `screenbuffer.test.ts` pass

- [ ] **Step 4: Commit**

```bash
git add internal/web/frontend/src/terminal.ts internal/web/static/style.css
git commit -m "feat(web): rewrite terminal as screen-buffer DOM renderer"
```

---

### Task 4: WebSocket Protocol Update

**Files:**
- Rewrite: `internal/web/frontend/src/websocket.ts`
- Modify: `internal/web/frontend/src/main.ts`

- [ ] **Step 1: Rewrite websocket.ts for binary protocol**

Rewrite `internal/web/frontend/src/websocket.ts`:

```typescript
export class WS {
  private ws: WebSocket | null = null;
  private url: string;
  private encoder = new TextEncoder();
  private decoder = new TextDecoder();

  onMessage: ((data: string) => void) | null = null;
  onDisconnect: ((msg: string) => void) | null = null;

  constructor() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    this.url = proto + "//" + location.host + "/ws";
  }

  connect(): void {
    this.ws = new WebSocket(this.url);
    this.ws.binaryType = "arraybuffer";

    this.ws.onopen = () => {};

    this.ws.onmessage = (e: MessageEvent) => {
      if (this.onMessage) {
        if (e.data instanceof ArrayBuffer) {
          this.onMessage(this.decoder.decode(e.data));
        } else {
          this.onMessage(e.data);
        }
      }
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
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(this.encoder.encode(data));
    }
  }

  sendResize(cols: number, rows: number): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "resize", cols, rows }));
    }
  }
}
```

- [ ] **Step 2: Rewrite main.ts with resize wiring**

Rewrite `internal/web/frontend/src/main.ts`:

```typescript
import { Terminal, measureTermSize } from "./terminal";
import { WS } from "./websocket";

const el = document.getElementById("terminal") as HTMLDivElement;
const { cols, rows } = measureTermSize(el);
const term = new Terminal(el, cols, rows);
const ws = new WS();

ws.onMessage = (data) => term.write(data);
ws.onDisconnect = (msg) => term.write(msg);
term.onInput((data) => ws.sendInput(data));

ws.connect();
// Send initial size after connect
setTimeout(() => ws.sendResize(cols, rows), 0);

// Resize on window change
const ro = new ResizeObserver(() => {
  const size = measureTermSize(el);
  term.resize(size.cols, size.rows);
  ws.sendResize(size.cols, size.rows);
});
ro.observe(el);
```

- [ ] **Step 3: Run frontend tests**

Run: `cd internal/web/frontend && npx vitest run`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/web/frontend/src/websocket.ts internal/web/frontend/src/main.ts
git commit -m "feat(web): update WebSocket to binary protocol with resize support"
```

---

### Task 5: PTY Bridge (Go Server)

**Files:**
- Rewrite: `internal/web/bridge.go`
- Create: `internal/web/bridge_windows.go`

- [ ] **Step 1: Create shared upgrader in ws.go**

The `upgrader` variable is currently in `bridge.go`, but both `bridge.go` (Unix) and `bridge_windows.go` need it. Extract it to a shared file.

Create `internal/web/ws.go`:

```go
package web

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
```

- [ ] **Step 2: Create Windows stub**

Create `internal/web/bridge_windows.go`:

```go
//go:build windows

package web

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"tewodros-terminal/internal/email"
	gb "tewodros-terminal/internal/guestbook"
)

func HandleWebSocket(_ *gb.SQLiteGuestbook, _ *email.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		conn.WriteMessage(websocket.BinaryMessage, []byte("\x1b[31mWeb terminal requires a Unix server. Connect via SSH instead.\x1b[0m\r\n"))
	}
}
```

- [ ] **Step 3: Rewrite bridge.go with PTY**

Rewrite `internal/web/bridge.go`:

```go
//go:build !windows

package web

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"tewodros-terminal/internal/content"
	"tewodros-terminal/internal/email"
	gb "tewodros-terminal/internal/guestbook"
	"tewodros-terminal/internal/tui"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type resizeMsg struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func HandleWebSocket(guestbook *gb.SQLiteGuestbook, emailSender *email.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Create PTY pair
		ptmx, ttyFile, err := pty.Open()
		if err != nil {
			log.Printf("pty open failed: %v", err)
			return
		}
		defer ptmx.Close()
		defer ttyFile.Close()

		// Set initial size
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

		// Set TERM for color support
		environ := []string{"TERM=xterm-256color"}

		// Extract client IP
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
		}

		// Build Bubble Tea app
		root := content.BuildTree()
		app := tui.NewApp(root, guestbook, emailSender)
		app.SetClientIP(clientIP)

		// Run Bubble Tea on the PTY slave
		p := tea.NewProgram(app,
			tea.WithInput(ttyFile),
			tea.WithOutput(ttyFile),
			tea.WithEnvironment(environ),
			tea.WithWindowSize(80, 24),
		)

		var wg sync.WaitGroup

		// Start Bubble Tea
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.Run(); err != nil {
				log.Printf("bubbletea program exited: %v", err)
			}
			// Close PTY master to signal EOF to the read goroutine
			ptmx.Close()
		}()

		// PTY master → WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
						break
					}
				}
				if err != nil {
					break
				}
			}
			conn.Close()
		}()

		// WebSocket → PTY master
		for {
			msgType, raw, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if msgType == websocket.TextMessage {
				var msg resizeMsg
				if json.Unmarshal(raw, &msg) == nil && msg.Type == "resize" && msg.Cols > 0 && msg.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(msg.Rows),
						Cols: uint16(msg.Cols),
					})
					p.Send(tea.WindowSizeMsg{Width: msg.Cols, Height: msg.Rows})
					continue
				}
			}

			// Write terminal input to PTY
			if _, err := ptmx.Write(raw); err != nil {
				break
			}
		}

		// Clean up: close PTY slave to signal EOF to Bubble Tea
		ttyFile.Close()
		// Send interrupt to kill Bubble Tea if still running
		ptmx.Write([]byte{3}) // Ctrl+C
		io.Copy(io.Discard, ptmx)

		wg.Wait()
	}
}
```

- [ ] **Step 4: Verify Go builds**

Run: `go build ./...`
Expected: clean build with no errors

- [ ] **Step 5: Run Go tests**

Run: `go test ./...`
Expected: all tests pass (tui, content, guestbook packages unchanged)

- [ ] **Step 6: Commit**

```bash
git add internal/web/bridge.go internal/web/bridge_windows.go internal/web/ws.go
git commit -m "feat(web): replace line-based REPL with PTY bridge to Bubble Tea"
```

---

### Task 6: Build Frontend Assets and Verify

**Files:**
- Rebuild: `internal/web/static/terminal.min.js`
- Rebuild: `internal/web/static/style.min.css`
- Rebuild: `internal/web/static/index.html`

- [ ] **Step 1: Build minified frontend assets**

Run:
```bash
cd internal/web/frontend
npx esbuild src/main.ts --bundle --minify --target=es2020 --outfile=../static/terminal.min.js
npx esbuild ../static/style.css --minify --outfile=../static/style.min.css
npx html-minifier-terser --collapse-whitespace --remove-comments --remove-redundant-attributes --minify-js '{"mangle":false}' --minify-urls -o ../static/index.html ../static/index.dev.html
```

Expected: all three commands succeed

- [ ] **Step 2: Verify bundle size**

Run: `ls -la internal/web/static/terminal.min.js`
Expected: under 10KB (success criterion from spec)

- [ ] **Step 3: Run all tests one final time**

Run:
```bash
cd internal/web/frontend && npx vitest run
go test ./...
```

Expected: all tests pass

- [ ] **Step 4: Commit built assets**

```bash
git add internal/web/static/terminal.min.js internal/web/static/style.min.css internal/web/static/index.html
git commit -m "build: rebuild frontend assets for PTY bridge"
```

---

### Task 7: Cleanup and Final Verification

- [ ] **Step 1: Verify no unused imports or dead code**

Run: `go vet ./...`
Expected: no warnings

- [ ] **Step 2: Remove the old `ansi.test.ts` tests that relied on the old Action type**

Already handled in Task 2 — verify by running: `cd internal/web/frontend && npx vitest run`
Expected: all tests pass, no references to old `Action` type

- [ ] **Step 3: Final commit with any cleanup**

If any cleanup was needed:
```bash
git add -A
git commit -m "chore: cleanup after PTY bridge migration"
```

- [ ] **Step 4: Verify success criteria**

Check against the spec:
1. Browser visitors see the exact same TUI as SSH users — ✅ same Bubble Tea, same bytes
2. bridge.go has no duplicate command/flow logic — ✅ old webSession deleted
3. No new dependencies added — ✅ creack/pty was already in go.sum
4. Frontend bundle stays under 10KB minified — ✅ verified in Task 6
5. All existing tests pass — ✅ verified in Task 6
