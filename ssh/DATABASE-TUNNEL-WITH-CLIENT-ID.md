# 🎯 Database Tunnel Commands with Client ID

## Problem
When running:
```bash
bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308"
```
**Result**: CLIENT ID column shows "-" in dashboard

## Solution
Add `-c` (client ID) and `-n` (client name) parameters:

### 🗄️ MySQL Database Tunnel
```bash
bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308" -c "mysql-tunnel" -n "MySQL Database Connection"
```

### 🐘 PostgreSQL Database Tunnel
```bash
bin\universal-client.exe -L ":5433" -a "agent-linux" -t "103.41.206.153:5432" -c "postgres-tunnel" -n "PostgreSQL Database Connection"
```

### 🌐 Web Application Tunnel
```bash
bin\universal-client.exe -L ":8080" -a "agent-linux" -t "103.41.206.153:80" -c "web-tunnel" -n "Web Application Access"
```

### 🔗 SSH Tunnel
```bash
bin\universal-client.exe -L ":2222" -a "agent-linux" -t "103.41.206.153:22" -c "ssh-tunnel" -n "SSH Remote Access"
```

## Parameters Explanation

| Parameter | Purpose | Example | Dashboard Column |
|-----------|---------|---------|------------------|
| `-c, --client-id` | Unique identifier | `mysql-tunnel` | CLIENT ID |
| `-n, --name` | Descriptive name | `MySQL Database Connection` | Internal reference |
| `-a, --agent` | Target agent | `agent-linux` | AGENT ID |
| `-L, --local` | Local port | `:3307` | Local binding |
| `-t, --target` | Remote target | `103.41.206.153:3308` | Destination |

## Result
✅ **CLIENT ID column** will show the specified client ID instead of "-"
✅ **Better tracking** and identification in dashboard
✅ **Audit trail** with meaningful client names

## Quick Test Script
```bash
fixed-mysql-tunnel.bat
```