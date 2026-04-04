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
  private _cursorRow = 0;
  private _cursorCol = 0;
  rows: number;
  cols: number;
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

  // cursorRow getter lazily resolves any pending scroll.
  // This means: after writeChar fills the last column and advances the row counter
  // past the last row, reading cursorRow triggers the scroll and returns the
  // clamped post-scroll position. setCursor bypasses the getter by writing
  // _cursorRow directly (clamped), so it cancels any pending scroll.
  get cursorRow(): number {
    if (this._cursorRow >= this.rows) {
      this.scrollUp();
    }
    return this._cursorRow;
  }

  get cursorCol(): number {
    return this._cursorCol;
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
    // Resolve any pending scroll before writing
    if (this._cursorRow >= this.rows) this.scrollUp();
    if (this._cursorCol >= this.cols) {
      this._cursorCol = 0;
      this._cursorRow++;
      if (this._cursorRow >= this.rows) this.scrollUp();
    }
    this.cells[this._cursorRow][this._cursorCol] = {
      char: ch,
      fg: style.fg,
      bg: style.bg,
      bold: style.bold,
    };
    this.dirty.add(this._cursorRow);
    this._cursorCol++;
    // Eagerly advance to next row when end of line reached.
    // If _cursorRow goes out of bounds, the scroll is deferred until cursorRow is read.
    if (this._cursorCol >= this.cols) {
      this._cursorCol = 0;
      this._cursorRow++;
    }
  }

  carriageReturn(): void {
    this._cursorCol = 0;
    if (this._cursorRow >= this.rows) this._cursorRow = this.rows - 1;
  }

  lineFeed(): void {
    this._cursorRow++;
    if (this._cursorRow >= this.rows) this.scrollUp();
  }

  private scrollUp(): void {
    this.cells.shift();
    this.cells.push(this.newRow());
    this._cursorRow = this.rows - 1;
    for (let r = 0; r < this.rows; r++) this.dirty.add(r);
  }

  setCursor(row: number, col: number): void {
    // Write directly to _cursorRow (not the getter) to cancel any pending scroll.
    this._cursorRow = Math.max(0, Math.min(row, this.rows - 1));
    this._cursorCol = Math.max(0, Math.min(col, this.cols - 1));
  }

  moveCursor(dRow: number, dCol: number): void {
    this.setCursor(this.cursorRow + dRow, this.cursorCol + dCol);
  }

  eraseInLine(mode: number): void {
    const row = this.cells[this._cursorRow];
    if (!row) return;
    let start: number, end: number;
    if (mode === 0) { start = this._cursorCol; end = this.cols; }
    else if (mode === 1) { start = 0; end = this._cursorCol + 1; }
    else { start = 0; end = this.cols; }
    for (let c = start; c < end; c++) row[c] = emptyCell();
    this.dirty.add(this._cursorRow);
  }

  eraseInDisplay(mode: number): void {
    if (mode === 0) {
      this.eraseInLine(0);
      for (let r = this._cursorRow + 1; r < this.rows; r++) {
        this.cells[r] = this.newRow();
        this.dirty.add(r);
      }
    } else if (mode === 1) {
      for (let r = 0; r < this._cursorRow; r++) {
        this.cells[r] = this.newRow();
        this.dirty.add(r);
      }
      this.eraseInLine(1);
    } else {
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
    this._cursorRow = Math.min(this._cursorRow, rows - 1);
    this._cursorCol = Math.min(this._cursorCol, cols - 1);
    this.markAllDirty();
  }
}
