export class WS {
  private ws: WebSocket | null = null;
  private url: string;
  private encoder = new TextEncoder();
  private decoder = new TextDecoder();

  onMessage: ((data: string) => void) | null = null;
  onDisconnect: ((msg: string) => void) | null = null;

  constructor() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    this.url = proto + "//" + location.host + "/ws";
  }

  connect(): void {
    this.ws = new WebSocket(this.url);
    this.ws.binaryType = "arraybuffer";

    this.ws.onopen = () => {};

    this.ws.onmessage = (e: MessageEvent) => {
      if (this.onMessage) {
        if (e.data instanceof ArrayBuffer) {
          this.onMessage(this.decoder.decode(e.data));
        } else {
          this.onMessage(e.data);
        }
      }
    };

    this.ws.onclose = () => {
      if (this.onDisconnect) {
        this.onDisconnect(
          "\r\n\x1b[31mConnection closed. Refresh to reconnect.\x1b[0m\r\n" +
            "\x1b[90mOr connect via SSH: ssh tewodros.me\x1b[0m\r\n"
        );
      }
    };

    this.ws.onerror = () => {};
  }

  sendInput(data: string): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(this.encoder.encode(data));
    }
  }

  sendResize(cols: number, rows: number): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "resize", cols, rows }));
    }
  }
}
