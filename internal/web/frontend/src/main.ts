import { Terminal } from "./terminal";
import { WS } from "./websocket";

const el = document.getElementById("terminal") as HTMLDivElement;
const term = new Terminal(el);
const ws = new WS();

ws.onMessage = (data) => term.write(data);
ws.onDisconnect = (msg) => term.write(msg);
term.onInput((data) => ws.sendInput(data));

ws.connect();
