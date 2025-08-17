# Remote Tunnel Makefile

.PHONY: all build clean test deps

# Build all binaries
all: build

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build all binaries
build: deps
	go build -o bin/relay.exe ./cmd/relay
	go build -o bin/agent.exe ./cmd/agent
	go build -o bin/client.exe ./cmd/client

# Build for different platforms
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/relay-linux ./cmd/relay
	GOOS=linux GOARCH=amd64 go build -o bin/agent-linux ./cmd/agent
	GOOS=linux GOARCH=amd64 go build -o bin/client-linux ./cmd/client

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/relay.exe ./cmd/relay
	GOOS=windows GOARCH=amd64 go build -o bin/agent.exe ./cmd/agent
	GOOS=windows GOARCH=amd64 go build -o bin/client.exe ./cmd/client

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o bin/relay-mac ./cmd/relay
	GOOS=darwin GOARCH=amd64 go build -o bin/agent-mac ./cmd/agent
	GOOS=darwin GOARCH=amd64 go build -o bin/client-mac ./cmd/client

# Test the code
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f server.crt server.key

# Run relay server (development)
run-relay:
	go run ./cmd/relay -addr :8443 -token dev-token

# Run agent (development)
run-agent:
	go run ./cmd/agent -id test-agent -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1:22 -token dev-token

# Run client (development)
run-client:
	go run ./cmd/client -L :2222 -relay-url wss://localhost:8443/ws/client -agent test-agent -target 127.0.0.1:22 -token dev-token

# Generate certificates
certs:
	openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"
