.PHONY: build run test clean deploy deploy-setup

BINARY=tewodros-terminal
GOFLAGS=-ldflags="-s -w"
SERVER=deploy@150.136.59.125
SSH_OPTS=-i .ssh/oracle_vm

build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

build-linux:
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BINARY)-linux-amd64 ./cmd/server

build-arm:
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY)-linux-arm64 ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64

deploy: build-linux
	scp -i .ssh/oracle_vm $(BINARY)-linux-amd64 $(SERVER):/opt/tewodros-terminal/tewodros-terminal-new
	ssh $(SSH_OPTS) $(SERVER) 'cd /opt/tewodros-terminal && mv tewodros-terminal-new tewodros-terminal && chmod +x tewodros-terminal && sudo systemctl restart tewodros-terminal'

deploy-setup:
	scp -i .ssh/oracle_vm deploy/tewodros-terminal.service $(SERVER):/tmp/
	ssh $(SSH_OPTS) $(SERVER) 'sudo mv /tmp/tewodros-terminal.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable tewodros-terminal'
