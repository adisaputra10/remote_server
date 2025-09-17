# SSH Tunnel Client (`cmd/client`)

## Overview
The SSH Tunnel Client (`cmd/client/main.go`) is a dedicated tunneling client that creates local TCP listeners and forwards connections through a relay server to remote agents. This component is part of the SSH Relay System and focuses purely on tunnel management without SSH client functionality.

## Features

### 🔗 Core Tunneling Features
- **Local TCP Listeners**: Creates local ports that forward to remote agents
- **WebSocket Relay Communication**: Connects to relay server via WebSocket
- **Multi-Session Support**: Handles multiple concurrent tunnel sessions
- **Interactive Mode**: Command-line interface for dynamic tunnel management
- **Single Tunnel Mode**: Direct tunnel creation via command line arguments

### 🛠️ Technical Features
- **Concurrent Processing**: Handles multiple connections simultaneously
- **Session Management**: Tracks active sessions with unique identifiers
- **Database Query Logging**: Logs database operations for audit trails
- **Heartbeat Monitoring**: Maintains connection health with relay server
- **Graceful Shutdown**: Clean connection termination on exit signals

## Architecture

### Connection Flow
```
[Local App] → [Local Port] → [Tunnel Client] → [Relay Server] → [Remote Agent] → [Target Service]
```

### Component Interaction
- **Client**: Creates local listeners and manages WebSocket connection
- **Relay Server**: Routes traffic between clients and agents
- **Remote Agent**: Forwards traffic to target services (SSH, databases, etc.)

## Installation

### Prerequisites
- Go 1.19 or higher
- Network access to relay server
- Target agents running on remote systems

### Building
```bash
# Build the tunnel client
cd cmd/client
go build -o ../../bin/tunnel-client.exe main.go

# Or from project root
go build -o bin/tunnel-client.exe cmd/client/main.go
```

## Usage

### Command Structure
```bash
bin/tunnel-client.exe [OPTIONS]
```

### Command Line Options
| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
| `--client-id` | `-c` | Client identifier | No | Auto-generated |
| `--name` | `-n` | Client name | No | Auto-generated |
| `--relay-url` | `-r` | Relay server WebSocket URL | No | ws://localhost:8080/ws/client |
| `--local` | `-L` | Local address to listen on | Yes* | - |
| `--agent` | `-a` | Target agent ID | Yes* | - |
| `--target` | `-t` | Target address | Yes* | - |
| `--interactive` | `-i` | Run in interactive mode | No | false |

*Required for single tunnel mode, optional for interactive mode

### Usage Examples

#### 1. Single Tunnel Mode
```bash
# Create SSH tunnel to remote server
bin/tunnel-client.exe -L ":2222" -a "web-server-agent" -t "localhost:22" -r "ws://relay.example.com:8080/ws/client"

# Create database tunnel
bin/tunnel-client.exe -L ":5432" -a "db-agent" -t "localhost:5432"

# Create HTTP tunnel
bin/tunnel-client.exe -L ":8080" -a "app-agent" -t "localhost:80"
```

#### 2. Interactive Mode
```bash
# Start interactive tunnel manager
bin/tunnel-client.exe -i -r "ws://relay.example.com:8080/ws/client"

# Interactive commands:
> tunnel
Local address (e.g., :2222): :2222
Agent ID: web-agent
Target (e.g., localhost:22): localhost:22

> list
Active sessions: 1
  - session-abc123

> quit
```

#### 3. Custom Client Configuration
```bash
# Use specific client ID and name
bin/tunnel-client.exe -c "web-client-01" -n "Web Server Client" -L ":2222" -a "web-agent" -t "localhost:22"
```

## Connection Examples

### SSH Access Through Tunnel
```bash
# Step 1: Start tunnel client
bin/tunnel-client.exe -L ":2222" -a "server-agent" -t "localhost:22"

# Step 2: Connect via SSH (in another terminal)
ssh user@localhost -p 2222
```

### Database Access Through Tunnel
```bash
# Step 1: Start database tunnel
bin/tunnel-client.exe -L ":5432" -a "db-agent" -t "localhost:5432"

# Step 2: Connect to database (in another terminal)
psql -h localhost -p 5432 -U username -d database
```

### Web Application Access
```bash
# Step 1: Start HTTP tunnel
bin/tunnel-client.exe -L ":8080" -a "app-agent" -t "localhost:80"

# Step 2: Access web application
# Open browser to http://localhost:8080
```

## Interactive Mode

### Available Commands
- `help` - Display available commands
- `tunnel` - Create a new tunnel interactively
- `list` - Show active tunnel sessions
- `quit` - Exit the client

### Interactive Session Example
```
> tunnel
Local address (e.g., :2222): :3306
Agent ID: mysql-agent
Target (e.g., localhost:22): localhost:3306
Tunnel created: :3306 -> mysql-agent:localhost:3306

> list
Active sessions: 1
  - session-def456

> help
Available commands: help, tunnel, list, quit

> quit
```

## Configuration

### Relay Server Configuration
```bash
# Default relay server
RELAY_URL="ws://localhost:8080/ws/client"

# Production relay server
RELAY_URL="ws://relay.company.com:8080/ws/client"

# Secure relay server
RELAY_URL="wss://relay.company.com:443/ws/client"
```

### Environment Variables
```bash
# Override default relay URL
export RELAY_URL="ws://your-relay.com:8080/ws/client"

# Set client identifier
export CLIENT_ID="production-client-01"
```

## Logging and Monitoring

### Log Files
- Session logs are written to `logs/CLIENT-{clientName}.log`
- Database query logs for audit trails
- Connection status and error logging

### Database Query Logging
The client automatically logs database operations when forwarding database connections:
- Query text and parameters
- Client and agent identification
- Timestamps and session tracking
- Direction (inbound/outbound)

### Monitoring Commands
```bash
# Monitor active sessions
bin/tunnel-client.exe -i
> list

# Check log files
tail -f logs/CLIENT-*.log
```

## Error Handling

### Common Issues

#### 1. Relay Connection Failed
```
Error: Failed to connect to relay server
```
**Solutions**:
- Verify relay server URL and port
- Check network connectivity
- Ensure relay server is running

#### 2. Local Port Already in Use
```
Error: Failed to start local listener
```
**Solutions**:
- Use a different local port
- Stop conflicting services
- Check for existing tunnel clients

#### 3. Agent Not Found
```
Error: Target agent not connected
```
**Solutions**:
- Verify agent ID is correct
- Ensure target agent is running and connected
- Check agent connectivity to relay server

#### 4. Target Service Unreachable
```
Error: Connection refused to target
```
**Solutions**:
- Verify target address and port
- Ensure target service is running
- Check agent's network access to target

### Debug Mode
Enable debug logging by setting the log level in the client configuration.

## Security Considerations

### Network Security
- Use WSS (WebSocket Secure) for encrypted relay communication
- Implement client authentication for relay connections
- Restrict local listeners to localhost when possible

### Access Control
- Agent-based access control through relay server
- Session isolation between different clients
- Audit logging for compliance requirements

### Best Practices
- Use specific client IDs for identification
- Monitor active sessions regularly
- Implement proper firewall rules for local listeners

## Performance Optimization

### Connection Management
- Reuse WebSocket connections to relay server
- Implement connection pooling for high-traffic scenarios
- Monitor memory usage with multiple concurrent sessions

### Resource Usage
- **Memory**: ~5-15MB per client instance
- **CPU**: Low usage, scales with connection count
- **Network**: Direct passthrough, minimal overhead

## Integration

### With SSH Clients
```bash
# Create tunnel
bin/tunnel-client.exe -L ":2222" -a "server-agent" -t "localhost:22"

# Use with any SSH client
ssh -p 2222 user@localhost
scp -P 2222 file.txt user@localhost:/tmp/
```

### With Database Clients
```bash
# MySQL tunnel
bin/tunnel-client.exe -L ":3306" -a "mysql-agent" -t "localhost:3306"

# PostgreSQL tunnel  
bin/tunnel-client.exe -L ":5432" -a "postgres-agent" -t "localhost:5432"

# Connect with standard clients
mysql -h localhost -P 3306 -u user -p
psql -h localhost -p 5432 -U user -d database
```

### With Application Servers
```bash
# HTTP/HTTPS tunnels
bin/tunnel-client.exe -L ":8080" -a "web-agent" -t "localhost:80"
bin/tunnel-client.exe -L ":8443" -a "web-agent" -t "localhost:443"
```

## Automation Scripts

### Batch File Example (Windows)
```batch
@echo off
echo Starting SSH tunnel...
bin\tunnel-client.exe -L ":2222" -a "prod-server" -t "localhost:22"
```

### Shell Script Example (Linux/Mac)
```bash
#!/bin/bash
echo "Starting database tunnel..."
bin/tunnel-client -L ":5432" -a "db-server" -t "localhost:5432"
```

## Development

### Source Code Structure
```
cmd/client/main.go
├── Client struct and methods
├── WebSocket message handling
├── Local listener management
├── Session tracking
├── Database query logging
└── Command line interface
```

### Key Components
- **Client**: Main client structure with connection management
- **WebSocket Handler**: Processes relay server messages
- **Local Listener**: Creates and manages local TCP listeners
- **Session Manager**: Tracks active tunnel sessions
- **Database Logger**: Logs database operations for audit

### Building with Debug Info
```bash
go build -ldflags "-X main.version=dev" -o bin/tunnel-client.exe cmd/client/main.go
```

### Testing
```bash
# Unit tests
go test ./cmd/client/...

# Integration testing with relay server
bin/tunnel-client.exe -L ":2222" -a "test-agent" -t "localhost:22"
```

## Comparison with Integrated Client

| Feature | Tunnel Client | Integrated Client |
|---------|---------------|-------------------|
| **Purpose** | Pure tunneling | Tunnel + SSH client |
| **SSH Client** | No | Yes |
| **Interactive SSH** | No | Yes |
| **Multiple Tunnels** | Yes | Single tunnel |
| **Command Logging** | Database only | File + Database |
| **Use Case** | General tunneling | SSH-specific |

## Troubleshooting

### Debugging Connection Issues
1. Check relay server connectivity: `telnet relay-server 8080`
2. Verify agent is connected to relay server
3. Test target service accessibility from agent
4. Monitor client logs for specific error messages

### Performance Issues
1. Monitor concurrent session count
2. Check network latency to relay server
3. Verify target service performance
4. Consider connection pooling for high traffic

## License
This software is part of the SSH Relay System project.

## Support
For issues and questions, please refer to the main project documentation or contact the development team.