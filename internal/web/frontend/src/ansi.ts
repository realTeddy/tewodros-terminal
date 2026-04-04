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
        i++;
        if (i < input.length) i++;
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
        const nextTab = (Math.floor(this.sb.cursorCol / 8) + 1) * 8;
        this.sb.setCursor(this.sb.cursorRow, Math.min(nextTab, this.sb.cols - 1));
        i++;
      } else if (ch.charCodeAt(0) >= 32) {
        this.sb.writeChar(ch, this.style());
        i++;
      } else {
        i++;
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
      case "A":
        this.sb.moveCursor(-(n || 1), 0);
        break;
      case "B":
        this.sb.moveCursor(n || 1, 0);
        break;
      case "C":
        this.sb.moveCursor(0, n || 1);
        break;
      case "D":
        this.sb.moveCursor(0, -(n || 1));
        break;
      case "G":
        this.sb.setCursor(this.sb.cursorRow, (n || 1) - 1);
        break;
      case "H":
      case "f":
        this.sb.setCursor((parts[0] || 1) - 1, (parts[1] || 1) - 1);
        break;
      case "J":
        this.sb.eraseInDisplay(n);
        break;
      case "K":
        this.sb.eraseInLine(n);
        break;
      case "h":
        if (params === "?25") this.sb.cursorVisible = true;
        break;
      case "l":
        if (params === "?25") this.sb.cursorVisible = false;
        break;
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
        this.fg = "color-" + (codes[i + 2] ?? 0);
        i += 2;
      } else if (code === 48 && codes[i + 1] === 5) {
        this.bg = "color-" + (codes[i + 2] ?? 0);
        i += 2;
      }
      i++;
    }
  }
}
