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

    for (let r = 0; r < rows; r++) {
      const row = document.createElement("div");
      row.className = "term-row";
      this.output.appendChild(row);
      this.rowEls.push(row);
    }

    this.cursorEl = document.createElement("span");
    this.cursorEl.className = "term-cursor";
    this.container.appendChild(this.cursorEl);

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

      let text = cell.char;
      let j = i + 1;
      while (j < cells.length && this.cellClasses(cells[j]) === cls) {
        text += cells[j].char;
        j++;
      }

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
    const rowEl = this.rowEls[this.sb.cursorRow];
    if (!rowEl) return;

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
    this.container.addEventListener("click", (e) => {
      if ((e.target as HTMLElement).tagName === "A") return;
      this.hiddenInput.focus();
    });

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
