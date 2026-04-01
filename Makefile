.PHONY: build run test clean deploy deploy-setup frontend-deps frontend-build frontend-test

BINARY=tewodros-terminal
GOFLAGS=-ldflags="-s -w"
SERVER=deploy@150.136.59.125
SSH_OPTS=-p 2222 -i .ssh/oracle_vm

frontend-deps:
	cd internal/web/frontend && npm install

frontend-build:
	cd internal/web/frontend && npx esbuild src/main.ts --bundle --minify --target=es2020 --outfile=../static/terminal.min.js
	cd internal/web/frontend && npx esbuild ../static/style.css --minify --outfile=../static/style.min.css
	cd internal/web/frontend && npx html-minifier-terser --collapse-whitespace --remove-comments --remove-redundant-attributes --minify-js '{"mangle":false}' --minify-urls -o ../static/index.html ../static/index.dev.html

frontend-test:
	cd internal/web/frontend && npx vitest run

build: frontend-build
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

build-linux: frontend-build
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BINARY)-linux-amd64 ./cmd/server

build-arm: frontend-build
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY)-linux-arm64 ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64

deploy: build-linux
	scp -P 2222 -i .ssh/oracle_vm $(BINARY)-linux-amd64 $(SERVER):/opt/tewodros-terminal/tewodros-terminal-new
	ssh $(SSH_OPTS) $(SERVER) 'cd /opt/tewodros-terminal && mv tewodros-terminal-new tewodros-terminal && chmod +x tewodros-terminal && sudo systemctl restart tewodros-terminal'

deploy-setup:
	scp -P 2222 -i .ssh/oracle_vm deploy/tewodros-terminal.service $(SERVER):/tmp/
	ssh $(SSH_OPTS) $(SERVER) 'sudo mv /tmp/tewodros-terminal.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable tewodros-terminal'
