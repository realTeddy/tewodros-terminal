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
setTimeout(() => ws.sendResize(cols, rows), 0);

const ro = new ResizeObserver(() => {
  const size = measureTermSize(el);
  term.resize(size.cols, size.rows);
  ws.sendResize(size.cols, size.rows);
});
ro.observe(el);
