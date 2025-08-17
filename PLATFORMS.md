# Platform Support

## Supported Platforms

### ✅ **Windows**
- **Build**: `build.bat` or `make build-windows`
- **Demo**: `demo.bat`
- **Binaries**: `relay.exe`, `agent.exe`, `client.exe`
- **Requirements**: Go 1.22+, PowerShell

### ✅ **Linux** 
- **Build**: `./build.sh` or `make build-linux`
- **Demo**: `./demo.sh` or `make run-demo`
- **Binaries**: `relay-linux`, `agent-linux`, `client-linux`
- **Requirements**: Go 1.22+, bash, systemd (for services)
- **Production**: systemd services, Docker support

### ✅ **macOS**
- **Build**: `./build.sh` or `make build-mac`  
- **Demo**: `./demo.sh` or `make run-demo`
- **Binaries**: `relay-mac`, `agent-mac`, `client-mac`
- **Requirements**: Go 1.22+, bash

### ✅ **Docker (All Platforms)**
- **Build**: `docker build -t remote-tunnel .`
- **Demo**: `docker-compose up --build`
- **Production**: Multi-container deployment with docker-compose

### ✅ **ARM64 (Raspberry Pi, Apple Silicon)**
- **Build**: `make build-arm64`
- **Binaries**: `relay-arm64`, `agent-arm64`, `client-arm64`

## Quick Start by Platform

### Windows Users
```cmd
git clone <repository>
cd remote-tunnel
build.bat
demo.bat
```

### Linux/macOS Users  
```bash
git clone <repository>
cd remote-tunnel
chmod +x setup.sh && ./setup.sh
./build.sh
./demo.sh
```

### Docker Users (Any Platform)
```bash
git clone <repository>
cd remote-tunnel
docker-compose up --build
```

## Production Deployment

### Linux Systemd Services
```bash
sudo ./deploy/install.sh
sudo systemctl enable relay
sudo systemctl start relay
```

### Docker Production
```bash
cp .env.docker .env
# Edit .env with your settings
docker-compose -f examples/docker-compose.prod.yml up -d
```

## Testing

### Unit Tests
```bash
go test -v ./...
# Or: make test
```

### End-to-End Tests (Linux/macOS)
```bash
./test-e2e.sh
# Or: make test-e2e
```

### Manual Testing
1. Start relay: `./relay -addr :8443 -token test-token`
2. Start agent: `./agent -id test -relay-url wss://localhost:8443/ws/agent -allow 127.0.0.1:22 -token test-token`
3. Start client: `./client -L :2222 -relay-url wss://localhost:8443/ws/client -agent test -target 127.0.0.1:22 -token test-token`
4. Test: `ssh -p 2222 localhost` (if SSH server running)
