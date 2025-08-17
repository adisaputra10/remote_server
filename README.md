# Remote Tunnel

A simple remote tunnel implementation in Go, similar to Teleport, that allows secure access to services behind firewalls through a relay server.

## Quick Start

1. **Install Go 1.22+** (if not already installed)
2. **Build the project**:
   ```bash
   # Windows
   build.bat
   
   # Linux/Mac  
   make build
   ```
3. **Run the demo**:
   ```bash
   # Windows
   demo.bat
   
   # Linux/Mac
   make run-relay  # Terminal 1
   make run-agent  # Terminal 2  
   make run-client # Terminal 3
   ```

## Architecture

- **Relay Server**: Public endpoint that accepts WebSocket connections from agents and clients
- **Agent**: Runs in private network, connects outbound to relay, exposes local services
- **Client**: Creates local port forwarding to access services through the tunnel

The system uses WebSocket over TLS (WSS) for transport with yamux multiplexing for multiple concurrent streams.

## Features

- Reverse tunnel topology (agents connect outbound)
- Multiple agent support with unique IDs
- WebSocket transport over TLS (port 443)
- Connection multiplexing with yamux
- Automatic reconnection with backoff
- Keep-alive and health checks
- Simple token-based authentication

## Building

```bash
go mod tidy
go build -o relay.exe ./cmd/relay
go build -o agent.exe ./cmd/agent  
go build -o client.exe ./cmd/client
```

## Usage

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
