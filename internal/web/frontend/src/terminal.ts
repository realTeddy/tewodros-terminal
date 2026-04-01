import { parse } from "./ansi";

export class Terminal {
  private output: HTMLPreElement;
  private cursor: HTMLSpanElement;
  private container: HTMLDivElement;
  private hiddenInput: HTMLTextAreaElement;
  private inputHandler: ((data: string) => void) | null = null;

  constructor(element: HTMLDivElement) {
    this.container = element;

    this.output = document.createElement("pre");
    this.output.className = "term-output";
    this.container.appendChild(this.output);

    this.cursor = document.createElement("span");
    this.cursor.className = "term-cursor";
    this.output.appendChild(this.cursor);

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
          this.output.insertBefore(document.createElement("br"), this.cursor);
          break;
        case "backspace":
          this.removeLastChar();
          break;
        case "clear":
          this.output.innerHTML = "";
          this.output.appendChild(this.cursor);
          break;
      }
    }
    this.container.scrollTop = this.container.scrollHeight;
  }

  private appendText(text: string, classes: string): void {
    const span = document.createElement("span");
    if (classes) span.className = classes;
    span.textContent = text;
    this.output.insertBefore(span, this.cursor);
  }

  private removeLastChar(): void {
    const last = this.cursor.previousSibling;
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
