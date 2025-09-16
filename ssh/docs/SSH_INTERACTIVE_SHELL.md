# SSH Interactive Shell Documentation

## Overview

The SSH Interactive Shell provides a full remote terminal experience similar to PuTTY, allowing you to connect to remote agents through the relay server and execute commands with real-time feedback.

## Features

### ✅ Core Features
- **Dynamic Prompt**: Automatically updates to reflect remote user, hostname, and current working directory
- **Working Directory Tracking**: Maintains and displays current remote directory state
- **Real-time Command Execution**: Commands are executed on remote agent with live output
- **Command Logging**: All SSH commands are logged to the database and visible in the dashboard
- **Session Management**: Maintains persistent connection through relay server
- **Base64 Message Handling**: Properly decodes agent responses for clean output

### 🎯 Remote Shell Experience
- Behaves like a native SSH terminal
- No local commands (help, status) - pure remote experience
- Exit commands (exit, quit) handled locally for clean termination
- All other commands forwarded to remote agent
- Response synchronization for proper command flow

## Usage

### Basic Command
```bash
bin/interactive-shell.exe -relay ws://localhost:8080/ws -agent agent-01 -remote-user root -remote-host 192.168.1.100
```

### Parameters
- `-relay`: WebSocket URL of the relay server (e.g., `ws://localhost:8080/ws`)
- `-agent`: Target agent ID to connect to
- `-remote-user`: Username for SSH connection display (e.g., `root`, `ubuntu`)
- `-remote-host`: Hostname/IP for SSH connection display (e.g., `192.168.1.100`)

### Example Session
```
🚀 Starting SSH Interactive Shell...
🔗 Connecting to relay server: ws://localhost:8080/ws
📡 Target agent: agent-01
👤 Remote: root@192.168.1.100
🔑 Session ID: abc12345

✅ Connected to agent: agent-01

root@server:/home/ubuntu$ ls -la
total 24
drwxr-xr-x 3 ubuntu ubuntu 4096 Dec 20 10:30 .
drwxr-xr-x 3 root   root   4096 Dec 20 10:25 ..
-rw-r--r-- 1 ubuntu ubuntu  220 Dec 20 10:25 .bash_logout
-rw-r--r-- 1 ubuntu ubuntu 3771 Dec 20 10:25 .bashrc
-rw-r--r-- 1 ubuntu ubuntu  807 Dec 20 10:25 .profile

root@server:/home/ubuntu$ cd /var/log

root@server:/var/log$ pwd
/var/log

root@server:/var/log$ exit
👋 Connection closed
```

## Technical Implementation

### Message Flow
1. **Client Input**: User types command in interactive shell
2. **Command Send**: Shell sends command to relay server via WebSocket
3. **Agent Forward**: Relay forwards command to target agent
4. **Agent Execute**: Agent executes command on remote system
5. **Response Send**: Agent sends output back through relay
6. **Client Display**: Shell displays response and updates prompt

### Dynamic Prompt System
- **Initial Setup**: Shell queries `pwd` and `hostname` during initialization
- **Directory Tracking**: `cd` commands automatically trigger prompt updates
- **Prompt Format**: `user@hostname:directory$` format maintained
- **Real-time Updates**: Prompt reflects current remote state

### Command Logging
- **Database Storage**: All commands logged to `ssh_logs` table
- **Dashboard Integration**: Logs visible in web dashboard SSH tab
- **Session Tracking**: Commands associated with session and agent IDs
- **Direction Tracking**: Input/output direction recorded

### Connection Management
- **Auto-reconnect**: Handles agent disconnections gracefully
- **Session Persistence**: Maintains session across reconnections
- **Error Handling**: Displays connection errors with retry logic
- **Clean Termination**: Proper cleanup on exit commands

## Security Considerations

- **Relay-only Communication**: No direct SSH connection to agents
- **Command Logging**: All commands are logged for audit purposes
- **Session Isolation**: Each session is isolated with unique ID
- **Agent Authentication**: Only authenticated agents can receive commands

## Troubleshooting

### Common Issues

**Connection Failed**
- Verify relay server is running on specified URL
- Check agent ID exists and is connected
- Ensure WebSocket endpoint is accessible

**Commands Not Executing**
- Verify agent is still connected
- Check relay server logs for forwarding issues
- Ensure agent can execute commands on remote system

**Prompt Not Updating**
- Agent may not support directory tracking
- Check if `pwd` command works manually
- Verify agent response format

### Debug Mode
Use environment variable for debug logging:
```bash
DEBUG=1 bin/interactive-shell.exe -relay ws://localhost:8080/ws -agent agent-01 -remote-user root -remote-host 192.168.1.100
```

## Integration with Dashboard

The interactive shell integrates seamlessly with the web dashboard:

1. **SSH Logs Tab**: View all executed commands in real-time
2. **Session Tracking**: Filter logs by session ID
3. **Agent Monitoring**: See which agents are being accessed
4. **Command History**: Full audit trail of SSH activities

## Best Practices

1. **Use Descriptive Agent IDs**: Make agent identification easy
2. **Set Appropriate Remote User**: Use actual SSH username for clarity
3. **Monitor Dashboard**: Regular check of SSH logs for security
4. **Clean Termination**: Always use `exit` or `quit` to close sessions
5. **Session Management**: Don't leave idle sessions open