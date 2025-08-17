# Remote Tunnel Makefile

.PHONY: all build clean test deps install run-demo

# Default target
all: build

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build all binaries for current platform
build: deps
	mkdir -p bin
	go build -o bin/relay ./cmd/relay
	go build -o bin/agent ./cmd/agent
	go build -o bin/client ./cmd/client
	@chmod +x bin/*

# Install binaries to system (Linux/Mac)
install: build
	sudo cp bin/relay /usr/local/bin/
	sudo cp bin/agent /usr/local/bin/
	sudo cp bin/client /usr/local/bin/

# Build for different platforms
build-all: build-linux build-windows build-mac

build-linux: deps
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/relay-linux ./cmd/relay
	GOOS=linux GOARCH=amd64 go build -o bin/agent-linux ./cmd/agent
	GOOS=linux GOARCH=amd64 go build -o bin/client-linux ./cmd/client
	@chmod +x bin/*-linux

build-windows: deps
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/relay.exe ./cmd/relay
	GOOS=windows GOARCH=amd64 go build -o bin/agent.exe ./cmd/agent
	GOOS=windows GOARCH=amd64 go build -o bin/client.exe ./cmd/client

build-mac: deps
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/relay-mac ./cmd/relay
	GOOS=darwin GOARCH=amd64 go build -o bin/agent-mac ./cmd/agent
	GOOS=darwin GOARCH=amd64 go build -o bin/client-mac ./cmd/client
	@chmod +x bin/*-mac

# Build for ARM64 (Apple Silicon, Raspberry Pi, etc.)
build-arm64: deps
	mkdir -p bin
	GOOS=linux GOARCH=arm64 go build -o bin/relay-arm64 ./cmd/relay
	GOOS=linux GOARCH=arm64 go build -o bin/agent-arm64 ./cmd/agent
	GOOS=linux GOARCH=arm64 go build -o bin/client-arm64 ./cmd/client
	@chmod +x bin/*-arm64

# Test the code
test:
	go test -v ./...

# Run end-to-end test
test-e2e: build
	@chmod +x test-e2e.sh
	./test-e2e.sh

# Run demo (Linux/Mac)
run-demo: build
	@chmod +x demo.sh
	./demo.sh

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f server.crt server.key
	rm -f /tmp/tunnel-*.log

# Development targets (with custom ports to avoid conflicts)
run-relay: build
	@export TUNNEL_TOKEN=dev-token; ./bin/relay -addr :8443 -token $$TUNNEL_TOKEN

run-agent: build
	@export TUNNEL_TOKEN=dev-token; ./bin/agent -id test-agent -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1:22 -token $$TUNNEL_TOKEN

run-client: build
	@export TUNNEL_TOKEN=dev-token; ./bin/client -L :2222 -relay-url wss://localhost:8443/ws/client -agent test-agent -target 127.0.0.1:22 -token $$TUNNEL_TOKEN

# Generate certificates manually (alternative to auto-generation)
certs:
	openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"

# Docker targets
docker-build:
	docker build -t remote-tunnel .

docker-run: docker-build
	docker run -p 8443:443 -e TUNNEL_TOKEN=demo-token remote-tunnel

docker-compose-up:
	docker-compose up --build

docker-compose-down:
	docker-compose down

# Help target
help:
	@echo "Available targets:"
	@echo "  build       - Build binaries for current platform"
	@echo "  build-all   - Build for all platforms"  
	@echo "  build-linux - Build for Linux"
	@echo "  build-windows - Build for Windows"
	@echo "  build-mac   - Build for macOS"
	@echo "  build-arm64 - Build for ARM64"
	@echo "  install     - Install to system (Linux/Mac)"
	@echo "  test        - Run unit tests"
	@echo "  test-e2e    - Run end-to-end tests"
	@echo "  run-demo    - Run full demo"
	@echo "  run-relay   - Run relay server"
	@echo "  run-agent   - Run agent"
	@echo "  run-client  - Run client"
	@echo "  clean       - Clean build artifacts"
	@echo "  certs       - Generate TLS certificates"
	@echo "  docker-*    - Docker related targets"
