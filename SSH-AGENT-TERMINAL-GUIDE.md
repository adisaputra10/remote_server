# SSH Agent Terminal - Interactive Go Terminal

## Overview
Terminal interaktif Go yang dapat mengakses SSH port 22 melalui agent tanpa PTY. Terminal ini menyediakan interface lengkap untuk mengelola koneksi SSH remote melalui agent forwarding.

## Features
- ✅ **Agent Management**: Start/stop SSH agent dan relay server
- ✅ **SSH Operations**: Interactive SSH, command execution, connectivity testing
- ✅ **System Monitoring**: Real-time status monitoring
- ✅ **No PTY Required**: Direct SSH connection tanpa PTY client
- ✅ **Built-in Commands**: Comprehensive command set untuk SSH management

## Quick Start

### 1. Start Terminal
```bash
start-ssh-agent-terminal.bat
```

### 2. Basic Commands
```bash
# Check system status
status

# Start services
start-relay
start-agent

# Test SSH
ssh-test

# Connect to SSH
ssh-connect
```

## Available Commands

### Connection Management
| Command | Description |
|---------|-------------|
| `start-relay` | Start relay server on port 8080 |
| `start-agent` | Start SSH agent with port 22 forwarding |
| `stop-relay` | Stop relay server |
| `stop-agent` | Stop SSH agent |
| `restart-all` | Restart all services |

### SSH Operations
| Command | Description |
|---------|-------------|
| `ssh-connect` | Interactive SSH session |
| `ssh-test` | Test SSH connectivity |
| `ssh-exec <cmd>` | Execute command via SSH |

### System Monitoring
| Command | Description |
|---------|-------------|
| `status` | Show complete system status |
| `check-ssh` | Check SSH service status |
| `check-port <port>` | Check port availability |

### General Commands
| Command | Description |
|---------|-------------|
| `help` | Show all available commands |
| `version` | Show terminal version |
| `clear` | Clear screen |
| `exit` / `quit` | Exit terminal |

## Architecture

```
[SSH Agent Terminal] --> [Relay:8080] --> [Agent] --> [SSH:22]
     Interactive CLI       WebSocket      Forward    john@localhost
```

## Configuration

### SSH Target
- **Host**: 127.0.0.1 (localhost)
- **Port**: 22
- **User**: john
- **Password**: john123

### Agent Settings
- **Relay URL**: wss://localhost:8080/ws/agent
- **Agent ID**: demo-agent
- **Token**: demo-token
- **Forwarding**: 127.0.0.1:22

## Usage Examples

### 1. Complete Setup
```bash
ssh-agent> status           # Check current status
ssh-agent> start-relay      # Start relay server
ssh-agent> start-agent      # Start SSH agent
ssh-agent> ssh-test         # Test connectivity
ssh-agent> ssh-connect      # Connect to SSH
```

### 2. Execute Remote Commands
```bash
ssh-agent> ssh-exec "whoami"
ssh-agent> ssh-exec "hostname"
ssh-agent> ssh-exec "dir C:"
```

### 3. System Monitoring
```bash
ssh-agent> status           # Full system status
ssh-agent> check-ssh        # SSH service status
ssh-agent> check-port 22    # Port 22 availability
```

## Status Indicators

### ✅ Running States
- **✓ Relay Server**: RUNNING (port 8080)
- **✓ SSH Agent**: RUNNING (forwarding port 22)
- **✓ SSH Service**: RUNNING
- **✓ SSH Connection**: WORKING

### ❌ Stopped States  
- **✗ Relay Server**: STOPPED
- **✗ SSH Agent**: STOPPED
- **✗ SSH Service**: STOPPED
- **✗ SSH Connection**: FAILED

## Files Structure

```
ssh-agent-terminal/
├── main.go                 # Terminal source code
├── go.mod                  # Go module file
└── ssh-agent-terminal.exe  # Built executable
```

## Build Instructions

### Manual Build
```bash
cd ssh-agent-terminal
go build -o ssh-agent-terminal.exe .
```

### Auto Build & Run
```bash
# Will build automatically if needed
start-ssh-agent-terminal.bat
```

## Troubleshooting

### SSH Connection Issues
```bash
# Check SSH service
ssh-agent> check-ssh

# Test connectivity
ssh-agent> ssh-test

# Check port
ssh-agent> check-port 22
```

### Agent Issues
```bash
# Check status
ssh-agent> status

# Restart services
ssh-agent> restart-all
```

### Common Problems

1. **SSH Service Not Running**
   ```bash
   net start sshd
   ```

2. **User Not Found**
   ```bash
   net user john john123 /add
   ```

3. **Port 22 Blocked**
   - Check Windows Firewall
   - Verify SSH server configuration

## Advantages Over PTY

### Traditional PTY Method
```bash
ssh-pty.exe -relay-url wss://localhost:8080/ws/client \
            -agent demo-agent -token demo-token -user john
```

### New Agent Terminal Method
```bash
# Agent forwards to SSH
start-agent    # Configures -allow 127.0.0.1:22

# Direct SSH connection
ssh-connect    # Uses ssh john@127.0.0.1
```

### Benefits
- ✅ **Simpler**: No PTY layer complexity
- ✅ **Faster**: Direct SSH protocol
- ✅ **Stable**: Better Windows compatibility
- ✅ **Interactive**: Full terminal interface
- ✅ **Manageable**: Built-in service management

## Integration with Existing Scripts

### Can be used with:
- `test-ssh-port22-access.bat` - Agent testing
- `quick-ssh-test-nopty.bat` - Quick verification
- `master-ssh-nopty.bat` - Master control script

### Replaces:
- `start-ssh-pty.bat` - No longer needed
- Manual SSH PTY commands
- Separate agent management scripts

## Success Indicators

When everything is working properly:

```bash
ssh-agent> status

✓ Relay Server: RUNNING (port 8080)
✓ SSH Agent: RUNNING (forwarding port 22)
✓ SSH Service: RUNNING
✓ SSH Connection: WORKING

Working Directory: D:\repo\remote
SSH Target: john@127.0.0.1:22
Relay URL: wss://localhost:8080/ws/agent
```

## Next Steps

1. **Run Terminal**: `start-ssh-agent-terminal.bat`
2. **Check Status**: `status` command
3. **Start Services**: `start-relay` then `start-agent`
4. **Test Connection**: `ssh-test`
5. **Connect**: `ssh-connect`

**SSH Agent Terminal ready for interactive SSH remote access!**
