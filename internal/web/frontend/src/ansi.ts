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
      // Consume param bytes (0x30–0x3F: digits, semicolon, ?, etc.)
      // and intermediate bytes (0x20–0x2F) — stop at final byte (0x40–0x7E)
      while (i < input.length && (input.charCodeAt(i) < 0x40 || input.charCodeAt(i) > 0x7e)) {
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
