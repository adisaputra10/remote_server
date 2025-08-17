# Remote Tunnel

A simple remote tunnel implementation in Go, similar to Teleport, that allows secure access to services behind firewalls through a relay server.

## Quick Start

### Prerequisites
- **Go 1.22+** (download from https://golang.org/dl/)
- **Linux/Windows/macOS** supported

### Build & Run

#### **Windows**
```bash
# Build
build.bat

# Run demo  
demo.bat
```

#### **Linux/macOS**
```bash
# Make scripts executable
chmod +x build.sh demo.sh

# Build
./build.sh

# Run demo
./demo.sh

# Or use Makefile
make build
make run-demo
```

#### **Using Docker (All Platforms)**
```bash
# Quick test with docker-compose
docker-compose up --build

# Or build and run manually
docker build -t remote-tunnel .
docker run -p 8443:443 -e TUNNEL_TOKEN=demo-token remote-tunnel
```

## Documentation

- 📚 **[Platform Support](PLATFORMS.md)** - Detailed platform-specific instructions
- 🔧 **[Examples](examples/)** - Common use cases and configurations  
- 🐳 **[Docker](docker-compose.yml)** - Container deployment
- ⚙️ **[Systemd](deploy/)** - Linux service configuration

## Architecture

- **Relay Server**: Public endpoint that accepts WebSocket connections from agents and clients
- **Agent**: Runs in private network, connects outbound to relay, exposes local services
- **Client**: Creates local port forwarding to access services through the tunnel

The system uses WebSocket over TLS (WSS) for transport with yamux multiplexing for multiple concurrent streams.

## Features

- ✅ **Cross-Platform**: Windows, Linux, macOS, Docker, ARM64
- ✅ **Reverse Tunnel**: Agents connect outbound, no firewall changes needed
- ✅ **Secure Transport**: WebSocket over TLS (WSS) on port 443  
- ✅ **Multiplexing**: Multiple concurrent connections via yamux
- ✅ **Multiple Agents**: Support many agents with unique IDs
- ✅ **Auto-Reconnect**: Automatic reconnection with exponential backoff
- ✅ **Keep-Alive**: Built-in ping/pong and health checks
- ✅ **Simple Auth**: Token-based authentication
- ✅ **Production Ready**: systemd services, Docker deployment
- ✅ **Self-Signed TLS**: Automatic certificate generation for development

## Building

```bash
go mod tidy
go build -o relay.exe ./cmd/relay
go build -o agent.exe ./cmd/agent  
go build -o client.exe ./cmd/client
```

## Usage

### Development Testing

#### **Quick Test (All Platforms)**
```bash
# Windows
demo.bat

# Linux/macOS  
./demo.sh
# Or: make run-demo

# Docker
docker-compose up --build
```

#### **End-to-End Testing (Linux/macOS)**
```bash
# Automated E2E test
make test-e2e
# Or: ./test-e2e.sh

# Unit tests
make test
```

### Cross-Platform Building

The project supports building for multiple platforms:

```bash
# Build for all platforms
make build-all

# Build specific platforms
make build-linux    # Linux AMD64
make build-windows  # Windows AMD64  
make build-mac      # macOS AMD64
make build-arm64    # Linux ARM64 (Raspberry Pi, etc.)
```

### Examples

See the `examples/` directory for common use cases:
- `ssh-tunnel.sh` - SSH tunneling setup (updated for sh.adisaputra.online)
- `web-tunnel.sh` - Web server tunneling (updated for sh.adisaputra.online)
- `docker-compose.prod.yml` - Production Docker deployment

#### **Production Setup with Domain (sh.adisaputra.online)**
```bash
# Quick setup for domain deployment (includes self-signed certificates)
./setup-domain.sh  # Linux/Mac
setup-domain.bat   # Windows

# Generate certificates manually
./generate-certs.sh  # Linux/Mac  
generate-certs.bat   # Windows

# Test domain connectivity
./test-domain.sh   # Linux/Mac
test-domain.bat    # Windows

# Monitor connection status
./monitor-connection.sh  # Linux/Mac
monitor-connection.bat   # Windows
```

#### **Build from Source**
```bash
# Clone repository
git clone <repository-url>
cd remote-tunnel

# Build binaries
make build

# Optional: Install system-wide
sudo make install
```

#### **Production Deployment (Linux with Domain)**
```bash
# Deploy to sh.adisaputra.online
./deploy/deploy-domain.sh

# Start agent on laptop
./start-agent.sh

# Connect from remote machine
./bin/client -L :2222 -relay-url wss://sh.adisaputra.online:8443/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN -insecure
```

#### **Traditional Production Deployment (Linux)**
```bash
# Install as systemd services
sudo ./deploy/install.sh

# Configure
sudo nano /etc/default/remote-tunnel

# Start relay server
sudo systemctl enable relay
sudo systemctl start relay

# Start agent (replace 'myagent' with your agent ID)
sudo systemctl enable agent@myagent
sudo systemctl start agent@myagent

# Check status
sudo systemctl status relay
sudo systemctl status agent@myagent
```

### 1. Start Relay Server

```bash
# Set auth token
set TUNNEL_TOKEN=your-secret-token

# Start relay (generates self-signed cert if needed)
relay.exe -addr :443 -token %TUNNEL_TOKEN%
```

The relay will generate a self-signed certificate (`server.crt` and `server.key`) if they don't exist.

### 2. Start Agent

```bash
# Set auth token
set TUNNEL_TOKEN=your-secret-token

# Start agent allowing SSH access
agent.exe -id agent-ssh -relay-url wss://localhost/ws/agent -allow 127.0.0.1:22 -token %TUNNEL_TOKEN%
```

Options:
- `-id`: Unique identifier for this agent
- `-relay-url`: WebSocket URL of the relay server
- `-allow`: Allowed target addresses (can specify multiple times)
- `-token`: Authentication token

### 3. Start Client

```bash
# Set auth token  
set TUNNEL_TOKEN=your-secret-token

# Start client to forward local port 2222 to agent's SSH
client.exe -L :2222 -relay-url wss://localhost/ws/client -agent agent-ssh -target 127.0.0.1:22 -token %TUNNEL_TOKEN%
```

Options:
- `-L`: Local listen address
- `-relay-url`: WebSocket URL of the relay server
- `-agent`: Target agent ID
- `-target`: Target address on agent side
- `-token`: Authentication token

### 4. Test SSH Connection

```bash
ssh -p 2222 user@127.0.0.1
```

## Configuration

### Environment Variables

- `TUNNEL_TOKEN`: Authentication token (alternative to `-token` flag)

### TLS Certificates

By default, the relay generates self-signed certificates. For production, provide your own:

```bash
relay.exe -cert /path/to/server.crt -key /path/to/server.key
```

## Protocol

### Control Messages

The system uses JSON control messages for coordination:

```go
type Control struct {
    Type       MsgType `json:"type"`
    AgentID    string  `json:"agent_id,omitempty"`
    StreamID   string  `json:"stream_id,omitempty"`
    TargetAddr string  `json:"target_addr,omitempty"`
    Token      string  `json:"token,omitempty"`
    Error      string  `json:"error,omitempty"`
}
```

Message types:
- `REGISTER`: Agent registration
- `DIAL`: Client tunnel request
- `ACCEPT`/`REFUSE`: Dial response
- `PING`/`PONG`: Keep-alive
- `ERROR`: Error notification

### Connection Flow

1. Agent connects to relay and sends `REGISTER`
2. Client connects to relay
3. When client receives local connection:
   - Client sends `DIAL` to relay
   - Relay forwards `DIAL` to agent
   - Agent dials target and opens stream
   - Relay sends `ACCEPT` to client
   - Client opens stream and data flows through relay

## Examples

### SSH Tunnel
```bash
# Relay
relay.exe

# Agent (on target machine)
agent.exe -id ssh-server -relay-url wss://relay.example.com/ws/agent -allow 127.0.0.1:22

# Client  
client.exe -L :2222 -relay-url wss://relay.example.com/ws/client -agent ssh-server -target 127.0.0.1:22

# Connect
ssh -p 2222 user@127.0.0.1
```

### HTTP Service Tunnel
```bash
# Agent
agent.exe -id web-server -relay-url wss://relay.example.com/ws/agent -allow 127.0.0.1:8080

# Client
client.exe -L :8080 -relay-url wss://relay.example.com/ws/client -agent web-server -target 127.0.0.1:8080

# Access
curl http://127.0.0.1:8080
```

### Multiple Services
```bash
# Agent with multiple allowed targets
agent.exe -id multi-server -relay-url wss://relay.example.com/ws/agent -allow 127.0.0.1:22 -allow 127.0.0.1:80 -allow 127.0.0.1:443

# SSH client
client.exe -L :2222 -agent multi-server -target 127.0.0.1:22

# HTTP client  
client.exe -L :8080 -agent multi-server -target 127.0.0.1:80
```

## Security

- All connections use TLS (WSS)
- Token-based authentication
- Agent allowlist for target addresses
- Optional mTLS support (can be added)

## Troubleshooting

### Connection Issues
- Verify firewall allows outbound HTTPS (443)
- Check relay server is reachable
- Validate authentication tokens match

### Certificate Issues
- For self-signed certs, ignore certificate errors during development
- For production, use proper CA-signed certificates

### Debug Mode
Add verbose logging by modifying the log level in the source code.
