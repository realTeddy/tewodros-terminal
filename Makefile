.PHONY: build run test clean deploy

BINARY=tewodros-terminal
GOFLAGS=-ldflags="-s -w"

build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

build-arm:
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY)-linux-arm64 ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v

clean:
	rm -f $(BINARY) $(BINARY)-linux-arm64

deploy: build-arm
	scp $(BINARY)-linux-arm64 deploy@your-vm:/opt/tewodros-terminal/tewodros-terminal
	ssh deploy@your-vm 'sudo systemctl restart tewodros-terminal'
