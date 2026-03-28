(function () {
    "use strict";

    var term = new Terminal({
        cursorBlink: true,
        fontSize: 15,
        fontFamily: "'Fira Code', 'Cascadia Code', 'Consolas', monospace",
        theme: {
            background: "#0a0a0a",
            foreground: "#e0e0e0",
            cursor: "#00ff88",
            selectionBackground: "#264f78",
            black: "#0a0a0a",
            red: "#ff5555",
            green: "#50fa7b",
            yellow: "#f1fa8c",
            blue: "#6272a4",
            magenta: "#ff79c6",
            cyan: "#8be9fd",
            white: "#e0e0e0",
            brightBlack: "#6272a4",
            brightRed: "#ff6e6e",
            brightGreen: "#69ff94",
            brightYellow: "#ffffa5",
            brightBlue: "#d6acff",
            brightMagenta: "#ff92df",
            brightCyan: "#a4ffff",
            brightWhite: "#ffffff"
        }
    });

    var fitAddon = new FitAddon.FitAddon();
    var webLinksAddon = new WebLinksAddon.WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(document.getElementById("terminal"));
    fitAddon.fit();

    var protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    var wsUrl = protocol + "//" + window.location.host + "/ws";
    var ws = null;
    var reconnectAttempts = 0;
    var maxReconnectAttempts = 5;

    function connect() {
        ws = new WebSocket(wsUrl);

        ws.onopen = function () {
            reconnectAttempts = 0;
            sendResize();
        };

        ws.onmessage = function (event) {
            term.write(event.data);
        };

        ws.onclose = function () {
            if (reconnectAttempts < maxReconnectAttempts) {
                reconnectAttempts++;
                term.write("\r\n\x1b[33mConnection lost. Reconnecting...\x1b[0m\r\n");
                setTimeout(connect, 1000 * reconnectAttempts);
            } else {
                term.write("\r\n\x1b[31mConnection lost. Refresh the page to reconnect.\x1b[0m\r\n");
                term.write("\x1b[90mOr connect via SSH: ssh tewodros.me\x1b[0m\r\n");
            }
        };

        ws.onerror = function () {};
    }

    function sendInput(data) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "input", data: data }));
        }
    }

    function sendResize() {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: "resize",
                cols: term.cols,
                rows: term.rows
            }));
        }
    }

    term.onData(function (data) {
        sendInput(data);
    });

    window.addEventListener("resize", function () {
        fitAddon.fit();
        sendResize();
    });

    term.onResize(function () {
        sendResize();
    });

    connect();
})();
