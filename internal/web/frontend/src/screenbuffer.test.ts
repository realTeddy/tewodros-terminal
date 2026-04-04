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
    sb.writeChar("C", { fg: "", bg: "", bold: false });
    sb.writeChar("D", { fg: "", bg: "", bold: false });
    sb.writeChar("E", { fg: "", bg: "", bold: false });
    sb.writeChar("F", { fg: "", bg: "", bold: false });
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
    sb.eraseInLine(0);
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
    sb.eraseInDisplay(0);
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
