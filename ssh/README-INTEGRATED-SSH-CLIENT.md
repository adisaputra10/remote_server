# Integrated SSH Client

## Overview
The Integrated SSH Client (`integrated-ssh-client.go`) is a comprehensive solution that combines SSH tunneling and SSH client functionality into a single executable. It provides seamless connection to remote servers through a relay tunnel with automatic authentication and dual logging capabilities.

## Features

### 🚀 Core Functionality
- **Automatic SSH Tunnel Creation**: Establishes secure tunnel through relay server
- **Seamless SSH Connection**: Auto-connects to remote server via tunnel
- **Interactive Password Prompt**: Secure password input if not provided via CLI
- **Dual Logging System**: Logs to both local file and remote database
- **Command Capture**: Records all SSH commands and responses

### 🔧 Technical Features
- **WebSocket Tunnel**: Uses WebSocket for relay communication
- **Concurrent Processing**: Non-blocking tunnel and SSH operations
- **Error Recovery**: Graceful handling of connection failures
- **Session Management**: Unique session IDs for audit trails
- **Batch Logging**: Optimized logging performance

## Installation

### Prerequisites
- Go 1.19 or higher
- Access to a relay server
- Network connectivity to target SSH servers

### Building
```bash
# Build the integrated SSH client
go build -o bin/integrated-ssh-client.exe integrated-ssh-client.go
```

## Usage

### Basic Command Structure
```bash
bin\integrated-ssh-client.exe [OPTIONS]
```

### Command Line Options
| Flag | Description | Required | Default |
|------|-------------|----------|---------|
| `-c, --client` | Client identifier | Yes | - |
| `-a, --agent` | Agent identifier | Yes | - |
| `-H, --host` | SSH target host | No | 127.0.0.1 |
| `-P, --port` | SSH target port | No | 22 |
| `-u, --user` | SSH username | Yes | - |
| `-p, --password` | SSH password (interactive if not provided) | No | - |
| `-r, --relay` | Relay server URL | No | ws://168.231.119.242:8080/ws/client |
| `-l, --local-port` | Local tunnel port | No | 2222 |

### Usage Examples

#### 1. Interactive Password (Recommended)
```bash
# Connect with interactive password prompt
bin\integrated-ssh-client.exe -c "my-client" -a "agent-linux" -u "root"
```

#### 2. With Password Parameter
```bash
# Connect with password in command line
bin\integrated-ssh-client.exe -c "my-client" -a "agent-linux" -u "root" -p "mypassword"
```

#### 3. Custom Target Server
```bash
# Connect to specific host and port
bin\integrated-ssh-client.exe -c "web-client" -a "web-agent" -H "192.168.1.100" -P "2222" -u "admin"
```

#### 4. Custom Relay Server
```bash
# Use different relay server
bin\integrated-ssh-client.exe -c "my-client" -a "agent-linux" -u "root" -r "ws://my-relay.com:8080/ws/client"
```

## Connection Flow

```
[Client] → [Local Tunnel:2222] → [Relay Server] → [Target SSH Server]
    ↓
[SSH Session] → [Command Logging] → [Local File + Remote Database]
```

### Step-by-Step Process
1. **Tunnel Establishment**: Creates WebSocket tunnel to relay server
2. **Local Listener**: Starts local SSH listener on specified port
3. **SSH Connection**: Connects to target server through tunnel
4. **Interactive Session**: Provides terminal access with command logging
5. **Cleanup**: Gracefully closes connections on exit

## Logging System

### Dual Logging Architecture
The client implements a robust dual logging system:

#### 1. Local File Logging
- **Location**: `logs/commands.log`
- **Format**: `[TIMESTAMP] [Client:CLIENT_ID] [Agent:AGENT_ID] - COMMAND`
- **Example**: `[2025-09-18 14:31:59] [Client:my-client] [Agent:agent-linux] - ls -la`

#### 2. Remote Database Logging
- **Target**: Relay server database
- **Method**: HTTP API calls
- **Fallback**: Silent failure if relay unavailable

### Log Files Created
- `logs/commands.log` - SSH commands executed
- `logs/INTEGRATED-SSH-{client}.log` - Client session logs

## Configuration

### Environment Variables
```bash
# Optional: Override default relay server
export RELAY_URL="ws://your-relay.com:8080/ws/client"

# Optional: Override default local port
export LOCAL_PORT="3333"
```

### Relay Server Requirements
- WebSocket endpoint: `/ws/client`
- HTTP API endpoint: `/api/log-ssh`
- Support for SSH tunnel agent connections

## Security Features

### Password Security
- Interactive password prompt (no echo)
- Passwords not stored in process memory longer than necessary
- Command line passwords supported but not recommended

### Connection Security
- WebSocket secure connections (WSS supported)
- SSH key-based authentication (future enhancement)
- Session isolation with unique identifiers

### Audit Trail
- Complete command logging with timestamps
- Client and agent identification
- Session tracking for compliance

## Error Handling

### Connection Failures
- Automatic retry for tunnel connections
- Graceful fallback for logging failures
- Clear error messages for user guidance

### Logging Failures
- Local logging always prioritized
- Remote logging failures are silent (no user disruption)
- Debug logging available for troubleshooting

## Troubleshooting

### Common Issues

#### 1. Tunnel Connection Failed
```
Error: Failed to create tunnel
```
**Solution**: Check relay server connectivity and URL

#### 2. SSH Authentication Failed
```
Error: Failed to connect SSH
```
**Solution**: Verify username/password and target server accessibility

#### 3. Port Already in Use
```
Error: Failed to start local listener
```
**Solution**: Use different local port with `-l` flag

#### 4. Permission Denied
```
Error: Failed to open commands.log
```
**Solution**: Check write permissions in logs directory

### Debug Mode
Enable debug logging by modifying the logger level in the source code.

## Performance Considerations

### Optimization Features
- **Batch Logging**: Commands are logged in batches for performance
- **Async Operations**: Non-blocking logging and tunnel operations
- **Connection Pooling**: Reused HTTP connections for database logging
- **Memory Management**: Efficient buffer management for large outputs

### Resource Usage
- **Memory**: ~10-20MB typical usage
- **CPU**: Low usage except during active SSH sessions
- **Network**: Minimal overhead for tunnel maintenance

## Integration

### With Relay Server
Ensure your relay server supports:
- WebSocket client connections
- SSH tunnel agent management
- HTTP API for log storage

### With Monitoring Systems
Log files can be integrated with:
- Log aggregation systems (ELK, Splunk)
- SIEM platforms for security monitoring
- Custom monitoring dashboards

## Development

### Source Code Structure
```
integrated-ssh-client.go
├── CombinedSSHClient struct
├── Tunnel management
├── SSH connection handling
├── Logging system
└── Command line interface
```

### Building from Source
```bash
# Install dependencies
go mod tidy

# Build with debug info
go build -ldflags "-X main.version=dev" -o bin/integrated-ssh-client.exe integrated-ssh-client.go

# Build optimized
go build -ldflags "-s -w" -o bin/integrated-ssh-client.exe integrated-ssh-client.go
```

### Testing
```bash
# Test tunnel creation
go test -v ./...

# Manual testing
./bin/integrated-ssh-client.exe -c "test-client" -a "test-agent" -u "testuser"
```

## Changelog

### Version 2.1.0 (Current)
- Added interactive password prompt
- Implemented dual logging system
- Added client name in log entries
- Improved error handling for relay failures

### Version 2.0.0
- Combined tunnel and SSH client functionality
- Added WebSocket tunnel support
- Implemented command logging

### Version 1.0.0
- Initial release with basic SSH client functionality

## License
This software is part of the SSH Relay System project.

## Support
For issues and questions, please check the main project documentation or contact the development team.