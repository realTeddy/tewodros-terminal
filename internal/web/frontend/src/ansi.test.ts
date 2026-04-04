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
    expect(sb.getCell(0, 2).char).toBe("A");
  });

  it("handles cursor horizontal absolute CSI n G", () => {
    const sb = parse("\x1b[10GA");
    expect(sb.getCell(0, 9).char).toBe("A");
  });

  it("handles erase to end of line CSI K", () => {
    const sb = parse("ABCDE\x1b[1;3H\x1b[K");
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
