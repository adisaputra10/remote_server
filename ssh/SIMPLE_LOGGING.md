# 📁 Simplified Log File System

Sistem logging yang disederhanakan dengan 3 file log konsisten untuk semua koneksi dan operasi.

## 📋 Log Files

### 🔸 **server-relay.log**
- **File**: `logs/server-relay.log`
- **Content**: Semua aktivitas relay server
- **Includes**: WebSocket connections, client/agent registrations, message routing
- **Usage**: Monitor relay server performance dan connectivity

### 🔸 **agent.log**
- **File**: `logs/agent.log`
- **Content**: Semua aktivitas agent + **DATABASE QUERY LOGGING**
- **Includes**: 
  - Agent connection ke relay
  - Database connections ke target servers
  - **Comprehensive database query logging (MySQL, PostgreSQL, Redis)**
  - Session management
- **Usage**: **Primary file untuk monitoring database operations**

### 🔸 **client.log**
- **File**: `logs/client.log`
- **Content**: Semua aktivitas client connections
- **Includes**: Tunnel setups, port forwarding, connection status
- **Usage**: Monitor client-side tunnel operations

## 🔄 Benefits

### ✅ **Simplified Management**
- **3 consistent files** instead of per-session files
- **Easy to locate** - no timestamp suffixes
- **Predictable names** - always the same

### ✅ **Centralized Logging**
- **All agent activities** in one file (agent.log)
- **All database queries** in one place
- **Multiple client connections** logged to same file

### ✅ **Operational Efficiency**
- **Easy monitoring** - fixed file names
- **Log rotation** can be setup easily
- **Backup/archival** simplified

## 🚀 Usage Examples

### Quick Log Monitoring
```bash
# Monitor database queries (most important)
Get-Content logs\agent.log -Wait -Tail 50 | Select-String "MySQL|PostgreSQL|Redis"

# Monitor relay server
Get-Content logs\server-relay.log -Wait -Tail 20

# Monitor client connections
Get-Content logs\client.log -Wait -Tail 20
```

### All Logs Simultaneously
```bash
# Monitor all logs with color coding
.\monitor-logs.bat
```

### Specific Operation Monitoring
```bash
# Database operations only
Get-Content logs\agent.log -Wait | Select-String "CREATE_TABLE|INSERT|UPDATE|DELETE"

# Connection events
Get-Content logs\server-relay.log -Wait | Select-String "connection|WebSocket"

# Error monitoring across all logs
Get-Content logs\*.log -Wait | Select-String "ERROR|FAILED"
```

## 📊 Log Content Examples

### server-relay.log
```
2025/09/14 18:30:15 [RELAY] INFO: Starting relay server on port 8080
2025/09/14 18:30:20 [RELAY] INFO: Agent registered: my-agent
2025/09/14 18:30:25 [RELAY] INFO: Client connected: session-abc123
2025/09/14 18:30:26 [RELAY] INFO: Forwarding data: client session-abc123 -> agent my-agent
```

### agent.log (Database Queries!)
```
2025/09/14 18:31:15 [AGENT-my-agent] INFO: Connected to relay server
2025/09/14 18:31:20 [AGENT-my-agent] INFO: Accepting connection for session: abc123
2025/09/14 18:31:25 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL CREATE_TABLE - Session: abc123 - Table: users - SQL: CREATE TABLE users...
2025/09/14 18:31:26 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL INSERT - Session: abc123 - Table: users - SQL: INSERT INTO users VALUES...
2025/09/14 18:31:27 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL SELECT - Session: abc123 - Table: users - SQL: SELECT * FROM users WHERE...
2025/09/14 18:31:28 [AGENT-my-agent] INFO: [CLIENT->TARGET] Redis STRING_OP - Session: def456 - CMD: SET
```

### client.log
```
2025/09/14 18:32:15 [CLIENT] INFO: Starting tunnel: localhost:3307 -> my-agent -> 127.0.0.1:3306
2025/09/14 18:32:16 [CLIENT] INFO: Connected to relay server
2025/09/14 18:32:17 [CLIENT] INFO: Local server listening on :3307
2025/09/14 18:32:20 [CLIENT] INFO: New connection accepted on :3307
```

## 🧪 Testing

### Test Simple Logging
```bash
# Test new log file system
.\test-simple-logs.bat
```

### Test Database Logging
```bash
# Generate database operations and check agent.log
go run test-db-golang.go

# Check database logs
Get-Content logs\agent.log -Tail 20 | Select-String "MySQL"
```

## 📈 Log Analysis

### Database Query Statistics
```bash
# Count database operations by type
Get-Content logs\agent.log | Select-String "MySQL" | Group-Object {($_ -split " ")[4]} | Sort-Object Count -Descending

# Find most accessed tables
Get-Content logs\agent.log | Select-String "Table:" | ForEach-Object { ($_ -split "Table: ")[1] -split " " | Select-Object -First 1 } | Group-Object | Sort-Object Count -Descending

# Error analysis
Get-Content logs\*.log | Select-String "ERROR" | Group-Object {($_ -split "]")[1].Trim().Split(" ")[0]} | Sort-Object Count -Descending
```

### Performance Monitoring
```bash
# Connection events timeline
Get-Content logs\server-relay.log | Select-String "connected|registered" | Select-Object -Last 20

# Database query patterns
Get-Content logs\agent.log | Select-String "MySQL|PostgreSQL|Redis" | Select-Object -Last 50
```

## 🔧 File Management

### Log Rotation (Recommended for Production)
```bash
# Backup current logs
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
Copy-Item "logs\server-relay.log" "logs\server-relay_$timestamp.log"
Copy-Item "logs\agent.log" "logs\agent_$timestamp.log" 
Copy-Item "logs\client.log" "logs\client_$timestamp.log"

# Clear current logs
Clear-Content "logs\server-relay.log"
Clear-Content "logs\agent.log"
Clear-Content "logs\client.log"
```

### Archive Old Logs
```bash
# Move old logs to archive directory
New-Item -ItemType Directory -Path "logs\archive" -Force
Move-Item "logs\*_*.log" "logs\archive\"
```

## 💡 Best Practices

### Development
- **Monitor agent.log** for database query debugging
- **Use monitor-logs.bat** for real-time monitoring
- **Enable DEBUG=1** for detailed information

### Production
- **Setup log rotation** to prevent large files
- **Monitor disk space** for log directory
- **Archive logs** periodically
- **Consider log filtering** for sensitive data

### Troubleshooting
- **Check server-relay.log** for connectivity issues
- **Check agent.log** for database and forwarding issues
- **Check client.log** for local tunnel problems
- **All logs together** for complete picture

## 🎯 Key Benefits Summary

1. **Consistent Naming** - Always the same 3 files
2. **Centralized Database Logging** - All in agent.log
3. **Easy Monitoring** - Fixed paths for scripts
4. **Multiple Connections** - All logged to same files
5. **Operational Simplicity** - No timestamp management
6. **Better Analysis** - Consolidated data per component type

**Perfect for production environments with multiple agents, clients, and database connections!** 🎉