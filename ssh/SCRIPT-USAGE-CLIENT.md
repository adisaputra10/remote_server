# Script Usage Guide - bin/client.exe

## Overview
`bin/client.exe` adalah executable dari **Tunnel Client** (`cmd/client/main.go`) yang berfungsi sebagai pure tunneling service. Script ini membuat local TCP listeners dan meneruskan koneksi melalui relay server ke remote agents.

## Command Structure
```bash
bin\client.exe [OPTIONS]
```

## Available Flags

| Flag | Short | Description | Required | Default | Example |
|------|-------|-------------|----------|---------|---------|
| `--client-id` | `-c` | Client identifier | No | Auto-generated | `-c "web-client"` |
| `--name` | `-n` | Client name | No | Auto-generated | `-n "Web Server Client"` |
| `--relay-url` | `-r` | Relay server WebSocket URL | No | ws://localhost:8080/ws/client | `-r "ws://relay.com:8080/ws/client"` |
| `--local` | `-L` | Local address to listen on | Yes* | - | `-L ":2222"` |
| `--agent` | `-a` | Target agent ID | Yes* | - | `-a "server-agent"` |
| `--target` | `-t` | Target address | Yes* | - | `-t "localhost:22"` |
| `--interactive` | `-i` | Run in interactive mode | No | false | `-i` |
| `--help` | `-h` | Show help information | No | - | `-h` |

*Required for single tunnel mode, optional for interactive mode

## Usage Examples

### 1. SSH Tunnel (Single Mode)
```bash
# Create SSH tunnel to remote server
bin\client.exe -L ":2222" -a "web-server-agent" -t "localhost:22"

# With custom relay server
bin\client.exe -L ":2222" -a "prod-server" -t "localhost:22" -r "ws://relay.example.com:8080/ws/client"

# With custom client identification
bin\client.exe -c "ssh-client-01" -n "Production SSH Client" -L ":2222" -a "prod-server" -t "localhost:22"
```

**Then connect via SSH:**
```bash
ssh user@localhost -p 2222
```

### 2. Database Tunnels
```bash
# PostgreSQL tunnel
bin\client.exe -L ":5432" -a "db-agent" -t "localhost:5432"

# MySQL tunnel  
bin\client.exe -L ":3306" -a "mysql-agent" -t "localhost:3306"

# Custom database port
bin\client.exe -L ":5433" -a "db-agent" -t "localhost:5432"
```

**Then connect to database:**
```bash
# PostgreSQL
psql -h localhost -p 5432 -U username -d database

# MySQL
mysql -h localhost -P 3306 -u username -p
```

### 3. Web Application Tunnels
```bash
# HTTP tunnel
bin\client.exe -L ":8080" -a "web-agent" -t "localhost:80"

# HTTPS tunnel
bin\client.exe -L ":8443" -a "web-agent" -t "localhost:443"

# Custom web application
bin\client.exe -L ":3000" -a "app-agent" -t "localhost:3000"
```

**Then access web application:**
```
http://localhost:8080
https://localhost:8443
```

### 4. Interactive Mode (Multiple Tunnels)
```bash
# Start interactive tunnel manager
bin\client.exe -i

# OR with custom relay
bin\client.exe -i -r "ws://relay.example.com:8080/ws/client"
```

**Interactive commands:**
```
> help
Available commands: help, tunnel, list, quit

> tunnel
Local address (e.g., :2222): :2222
Agent ID: ssh-agent
Target (e.g., localhost:22): localhost:22
Tunnel created: :2222 -> ssh-agent:localhost:22

> tunnel
Local address (e.g., :2222): :5432
Agent ID: db-agent  
Target (e.g., localhost:22): localhost:5432
Tunnel created: :5432 -> db-agent:localhost:5432

> list
Active sessions: 2
  - session-abc123
  - session-def456

> quit
```

## Real-World Scenarios

### Scenario 1: Development Environment Access
```bash
# Setup tunnels for full development stack
bin\client.exe -i -c "dev-client" -r "ws://dev-relay.company.com:8080/ws/client"

# In interactive mode:
> tunnel
Local address: :2222
Agent ID: dev-server
Target: localhost:22

> tunnel  
Local address: :5432
Agent ID: dev-db
Target: localhost:5432

> tunnel
Local address: :3000
Agent ID: dev-app
Target: localhost:3000
```

### Scenario 2: Production Database Access
```bash
# Secure database tunnel for admin tasks
bin\client.exe -c "admin-db-client" -L ":5432" -a "prod-db-agent" -t "localhost:5432" -r "wss://secure-relay.company.com:443/ws/client"
```

### Scenario 3: Multi-Server Management
```bash
# Server 1 SSH access
bin\client.exe -L ":2221" -a "server1-agent" -t "localhost:22" &

# Server 2 SSH access  
bin\client.exe -L ":2222" -a "server2-agent" -t "localhost:22" &

# Server 3 SSH access
bin\client.exe -L ":2223" -a "server3-agent" -t "localhost:22" &
```

**Connect to different servers:**
```bash
ssh admin@localhost -p 2221  # Server 1
ssh admin@localhost -p 2222  # Server 2  
ssh admin@localhost -p 2223  # Server 3
```

## Comparison with Other Components

| Component | Purpose | Best For |
|-----------|---------|----------|
| **bin/client.exe** | Pure tunneling | Multiple tunnels, non-SSH services |
| **bin/integrated-ssh-client.exe** | Tunnel + SSH client | Quick SSH access, command logging |
| **bin/agent.exe** | Remote agent | Running on target servers |
| **bin/relay.exe** | Relay server | Central coordination |

## Configuration Examples

### Basic Configuration
```bash
# Minimal SSH tunnel
bin\client.exe -L ":2222" -a "my-agent" -t "localhost:22"
```

### Production Configuration
```bash
# Production with full identification
bin\client.exe \
  -c "prod-client-web-01" \
  -n "Production Web Client" \
  -L ":8080" \
  -a "web-server-prod" \
  -t "localhost:80" \
  -r "wss://relay.company.com:443/ws/client"
```

### Development Configuration
```bash
# Development with local relay
bin\client.exe \
  -c "dev-client" \
  -L ":3000" \
  -a "dev-app" \
  -t "localhost:3000" \
  -r "ws://localhost:8080/ws/client"
```

## Automation Scripts

### Windows Batch Script
```batch
@echo off
echo Starting SSH tunnel...
bin\client.exe -L ":2222" -a "prod-server" -t "localhost:22" -r "ws://relay.company.com:8080/ws/client"
pause
```

### PowerShell Script  
```powershell
# start-tunnels.ps1
Write-Host "Starting development tunnels..."

# SSH tunnel
Start-Process -FilePath "bin\client.exe" -ArgumentList "-L", ":2222", "-a", "dev-server", "-t", "localhost:22"

# Database tunnel
Start-Process -FilePath "bin\client.exe" -ArgumentList "-L", ":5432", "-a", "dev-db", "-t", "localhost:5432"

# Web tunnel
Start-Process -FilePath "bin\client.exe" -ArgumentList "-L", ":8080", "-a", "dev-web", "-t", "localhost:80"

Write-Host "All tunnels started!"
```

### Linux/Mac Shell Script
```bash
#!/bin/bash
# start-tunnels.sh

echo "Starting production tunnels..."

# SSH tunnel
bin/client -L ":2222" -a "prod-server" -t "localhost:22" &

# Database tunnel  
bin/client -L ":5432" -a "prod-db" -t "localhost:5432" &

# Web tunnel
bin/client -L ":8080" -a "prod-web" -t "localhost:80" &

echo "All tunnels started in background!"
```

## Monitoring and Logging

### Check Running Processes
```bash
# Windows
tasklist | findstr client.exe

# Linux/Mac
ps aux | grep client
```

### Log Files
- Session logs: `logs/CLIENT-{clientName}.log`
- Database query logs (automatic for DB tunnels)
- Connection status and errors

### Monitor Active Sessions
```bash
# Use interactive mode to check sessions
bin\client.exe -i
> list
Active sessions: 3
  - session-abc123 (SSH tunnel)
  - session-def456 (DB tunnel)  
  - session-ghi789 (Web tunnel)
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use
```
Error: Failed to start local listener
```
**Solution**: Use different port
```bash
bin\client.exe -L ":2223" -a "my-agent" -t "localhost:22"
```

#### 2. Agent Not Found
```
Error: Target agent not connected
```
**Solutions**:
- Check agent ID spelling
- Ensure agent is running and connected to relay
- Verify relay server connectivity

#### 3. Relay Connection Failed
```
Error: Failed to connect to relay server
```
**Solutions**:
- Check relay URL and port
- Verify network connectivity
- Try different relay endpoint

### Debug Commands
```bash
# Test relay connectivity
telnet relay.company.com 8080

# Check local port availability
netstat -an | findstr :2222

# Verify agent connectivity (check relay server logs)
```

## Security Best Practices

### 1. Use Secure Relay Connections
```bash
# Use WSS instead of WS for production
bin\client.exe -L ":2222" -a "agent" -t "localhost:22" -r "wss://relay.company.com:443/ws/client"
```

### 2. Limit Local Listeners
```bash
# Bind to localhost only
bin\client.exe -L "127.0.0.1:2222" -a "agent" -t "localhost:22"
```

### 3. Use Specific Client IDs
```bash
# Use meaningful client identifiers for audit
bin\client.exe -c "john-dev-client" -L ":2222" -a "dev-server" -t "localhost:22"
```

## Performance Tips

### 1. Resource Management
- Limit concurrent tunnels based on system resources
- Monitor memory usage with multiple sessions
- Use appropriate buffer sizes

### 2. Connection Optimization
- Reuse client connections when possible
- Implement connection pooling for high traffic
- Monitor network latency

## Integration Examples

### With CI/CD Pipelines
```yaml
# GitHub Actions example
- name: Setup Database Tunnel
  run: |
    bin/client.exe -L ":5432" -a "test-db" -t "localhost:5432" &
    sleep 5
    
- name: Run Database Tests
  run: |
    psql -h localhost -p 5432 -c "SELECT 1"
```

### With Docker Compose
```yaml
version: '3'
services:
  tunnel-client:
    build: .
    command: bin/client -L ":2222" -a "app-agent" -t "localhost:22"
    ports:
      - "2222:2222"
```

## Advanced Usage

### Environment Variables
```bash
# Set default relay URL
set RELAY_URL=ws://relay.company.com:8080/ws/client

# Use in script
bin\client.exe -L ":2222" -a "agent" -t "localhost:22" -r %RELAY_URL%
```

### Configuration Files (Future Enhancement)
```json
{
  "client_id": "prod-client-01",
  "name": "Production Client",
  "relay_url": "wss://relay.company.com:443/ws/client",
  "tunnels": [
    {
      "local": ":2222",
      "agent": "ssh-agent",
      "target": "localhost:22"
    },
    {
      "local": ":5432", 
      "agent": "db-agent",
      "target": "localhost:5432"
    }
  ]
}
```

## Support and Documentation

### Related Documentation
- [Complete Client Components Guide](README-CLIENT-COMPONENTS.md)
- [Integrated SSH Client](README-INTEGRATED-SSH-CLIENT.md)
- [Docker Setup](README-Docker.md)

### Getting Help
```bash
# Show help
bin\client.exe -h

# Interactive help
bin\client.exe -i
> help
```

---
**Note**: `bin/client.exe` is compiled from `cmd/client/main.go` and focuses purely on tunneling functionality. For integrated SSH access with command logging, use `bin/integrated-ssh-client.exe` instead.