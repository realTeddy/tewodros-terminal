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
