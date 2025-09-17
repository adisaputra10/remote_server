# SSH Tunnel System - Client Components

## Overview
This project contains two main client components for SSH tunneling and remote access. Each component serves different use cases and can be used independently or together.

## Components Comparison

| Feature | **Integrated SSH Client** | **Tunnel Client** |
|---------|---------------------------|-------------------|
| **File** | `integrated-ssh-client.go` | `cmd/client/main.go` |
| **Purpose** | All-in-one SSH solution | Pure tunneling service |
| **SSH Client** | ✅ Built-in | ❌ External required |
| **Interactive SSH** | ✅ Terminal access | ❌ No terminal |
| **Multiple Tunnels** | ❌ Single tunnel | ✅ Multiple tunnels |
| **Command Logging** | ✅ File + Database | ✅ Database only |
| **Password Prompt** | ✅ Interactive | ❌ N/A |
| **Auto-Connect** | ✅ Seamless | ❌ Manual SSH |
| **Best For** | Quick SSH access | General tunneling |

## When to Use Each Component

### 🚀 Use **Integrated SSH Client** When:
- You need **immediate SSH access** to remote servers
- You want **seamless connection** (tunnel + SSH in one command)
- You need **command logging** for audit/compliance
- You prefer **interactive password prompts**
- You're doing **one-time or occasional** SSH sessions
- You want **simplified workflow** without multiple tools

### 🔗 Use **Tunnel Client** When:
- You need **multiple simultaneous tunnels**
- You're forwarding **non-SSH services** (databases, web apps)
- You want **persistent tunnels** that stay active
- You prefer using **your own SSH client** or other tools
- You need **flexible tunnel management**
- You're building **automated systems** that need tunneling

## Quick Start Examples

### Integrated SSH Client (Recommended for SSH)
```bash
# Build
go build -o bin/integrated-ssh-client.exe integrated-ssh-client.go

# Use with interactive password
bin\integrated-ssh-client.exe -c "my-client" -a "server-agent" -u "root"

# Use with password parameter (not recommended)
bin\integrated-ssh-client.exe -c "my-client" -a "server-agent" -u "root" -p "password"
```

### Tunnel Client (Recommended for General Tunneling)
```bash
# Build
go build -o bin/tunnel-client.exe cmd/client/main.go

# Single tunnel mode
bin\tunnel-client.exe -L ":2222" -a "server-agent" -t "localhost:22"

# Interactive mode for multiple tunnels
bin\tunnel-client.exe -i
```

## Common Usage Scenarios

### Scenario 1: Quick SSH Access
**Need**: Connect to remote server for administration
**Solution**: Use **Integrated SSH Client**
```bash
bin\integrated-ssh-client.exe -c "admin-client" -a "prod-server" -u "admin"
```

### Scenario 2: Database Access
**Need**: Connect to remote database through tunnel
**Solution**: Use **Tunnel Client** + database client
```bash
# Step 1: Create tunnel
bin\tunnel-client.exe -L ":5432" -a "db-agent" -t "localhost:5432"

# Step 2: Connect with your preferred DB client
psql -h localhost -p 5432 -U username -d database
```

### Scenario 3: Web Development
**Need**: Access remote web application for testing
**Solution**: Use **Tunnel Client**
```bash
bin\tunnel-client.exe -L ":8080" -a "web-agent" -t "localhost:80"
# Access via http://localhost:8080
```

### Scenario 4: Multiple Services
**Need**: Access SSH + database + web app simultaneously
**Solution**: Use **Tunnel Client** in interactive mode
```bash
bin\tunnel-client.exe -i
> tunnel
Local address: :2222
Agent ID: server-agent
Target: localhost:22

> tunnel  
Local address: :5432
Agent ID: db-agent
Target: localhost:5432

> tunnel
Local address: :8080
Agent ID: web-agent  
Target: localhost:80
```

## System Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Local Client  │    │  Relay Server   │    │  Remote Agent   │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Integrated   │ │────┤ │             │ ├────┤ │             │ │
│ │SSH Client   │ │    │ │  WebSocket  │ │    │ │   Target    │ │
│ └─────────────┘ │    │ │   Router    │ │    │ │  Services   │ │
│                 │    │ │             │ │    │ │ (SSH, DB,   │ │
│ ┌─────────────┐ │    │ │             │ │    │ │  Web, etc.) │ │
│ │Tunnel       │ │────┤ │             │ ├────┤ │             │ │
│ │Client       │ │    │ │             │ │    │ │             │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Installation and Setup

### Prerequisites
- Go 1.19 or higher
- Network access to relay server  
- Remote agents running on target systems

### Building Both Components
```bash
# Build integrated SSH client
go build -o bin/integrated-ssh-client.exe integrated-ssh-client.go

# Build tunnel client
go build -o bin/tunnel-client.exe cmd/client/main.go

# Or build both with make/batch script
build.bat  # Windows
./build.sh # Linux/Mac
```

### Configuration
Both components use similar configuration:
- **Relay URL**: WebSocket endpoint to relay server
- **Client ID**: Unique identifier for the client
- **Agent ID**: Target agent identifier on remote system

## Logging and Monitoring

### Integrated SSH Client Logging
- **File**: `logs/commands.log` - SSH commands with client/agent info
- **Format**: `[TIMESTAMP] [Client:ID] [Agent:ID] - COMMAND`
- **Remote**: Sends logs to relay server database

### Tunnel Client Logging  
- **File**: `logs/CLIENT-{name}.log` - Session and connection logs
- **Database**: Query logging for database tunnels
- **Monitoring**: Session status and connection health

## Security Best Practices

### For Both Components
1. **Use WSS** (WebSocket Secure) for relay connections
2. **Restrict local listeners** to localhost when possible  
3. **Monitor active sessions** regularly
4. **Implement proper firewall rules**
5. **Use strong authentication** for SSH connections

### Integrated SSH Client Specific
1. **Use interactive password prompts** instead of command-line passwords
2. **Enable command logging** for audit trails
3. **Rotate SSH credentials** regularly

### Tunnel Client Specific  
1. **Limit concurrent tunnels** based on resource capacity
2. **Monitor database query logs** for suspicious activity
3. **Implement session timeouts** for idle connections

## Troubleshooting

### Common Issues
1. **Relay Connection Failed**: Check relay server URL and connectivity
2. **Agent Not Found**: Verify agent ID and ensure agent is connected
3. **Port Already in Use**: Use different local port or stop conflicting services
4. **Authentication Failed**: Check SSH credentials and target server access

### Debug Mode
Enable debug logging in both components for detailed troubleshooting information.

## Performance Considerations

### Resource Usage Comparison
| Component | Memory | CPU | Connections |
|-----------|---------|-----|-------------|
| Integrated Client | ~10-20MB | Low | 1 tunnel + SSH |
| Tunnel Client | ~5-15MB | Low | Multiple tunnels |

### Optimization Tips
1. **Reuse connections** when possible
2. **Monitor memory usage** with multiple sessions  
3. **Implement connection pooling** for high-traffic scenarios
4. **Use appropriate buffer sizes** for data transfer

## Development and Contributing

### Project Structure
```
├── integrated-ssh-client.go      # All-in-one SSH solution
├── cmd/client/main.go            # Pure tunneling client
├── internal/common/              # Shared utilities
├── logs/                         # Log files
├── bin/                          # Compiled binaries
└── README files                  # Documentation
```

### Building from Source
```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build with debug info
go build -ldflags "-X main.version=dev" -o bin/integrated-ssh-client.exe integrated-ssh-client.go
```

## License
This software is part of the SSH Relay System project.

## Documentation
- [Integrated SSH Client README](README-INTEGRATED-SSH-CLIENT.md)
- [Tunnel Client README](cmd/client/README.md)
- [Docker Setup](README-Docker.md)

## Support
For issues and questions, please check the component-specific README files or contact the development team.