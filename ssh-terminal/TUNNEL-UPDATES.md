# Updates Required for Tunnel Support

## 🔄 Summary of Changes

Server sudah diupdate untuk mendukung tunnel, tetapi agent dan client juga perlu disesuaikan untuk NAT architecture.

### 1. **Server Updates** ✅ DONE
- **NEW**: `/ws/tunnel` endpoint untuk handle tunnel requests
- **NEW**: `handleTunnelConnection()` function
- **NEW**: `TunnelSessions` field di Agent struct  
- **NEW**: `tunnel_data` message handling di `handleAgentMessage()`

### 2. **Agent Updates** ✅ DONE
- **NEW**: `handleTunnelStart()` function untuk start database tunnel
- **NEW**: `handleTunnelData()` function untuk forward data ke database proxy
- **NEW**: Message handling untuk `tunnel_start` dan `tunnel_data`
- **MODIFIED**: Session structure untuk support tunnel metadata

### 3. **Client Updates** ✅ DONE
- **NEW**: `createTunnelThroughServer()` function untuk tunnel via server
- **MODIFIED**: `handleConnection()` untuk use tunnel instead of direct connection
- **MODIFIED**: Port forward logic untuk work dengan agents behind NAT

## 🚀 Deployment Process

### Step 1: Build and Test Locally
```powershell
# Test builds locally first
go build -o test-server.exe goteleport-server-db.go
go build -o test-agent.exe goteleport-agent-db.go
go build -o unified-client.exe unified-client.go
```

### Step 2: Deploy to Linux Server  
```powershell
.\deploy-to-linux.bat
```

### Step 3: Start Services on Linux
```bash
ssh root@168.231.119.242
cd /opt/goteleport
./start-server.sh
./start-agent.sh
```

### Step 4: Test from Windows Client
```powershell
# Update client config if needed
.\unified-client.exe client-config-clean.json
```

## 🔧 Network Flow (After Updates)

```
Windows Client (unified-client.exe)
         ↓ WebSocket to /ws/client
         ↓
Linux Server (168.231.119.242:8081)
         ↓ /ws/tunnel endpoint  
         ↓ Forward tunnel request
         ↓
Linux Agent (behind NAT)
         ↓ Database proxy (3307, 5435)
         ↓
MySQL/PostgreSQL Database
```

## 🐛 Key Fixes

### Issue: Client Direct Connection Failed
- **Before**: Client tried direct connection to agent ports
- **After**: Client creates tunnel through server WebSocket

### Issue: Agent Behind NAT
- **Before**: Agent ports not accessible from Windows client
- **After**: Server acts as proxy between client and agent

### Issue: No Database Logging for NAT Agents  
- **Before**: Database commands not logged for NAT setups
- **After**: Tunnel preserves all logging functionality

## 📋 Testing Checklist

- [ ] Server builds without errors
- [ ] Agent builds without errors  
- [ ] Client builds without errors
- [ ] Server starts and listens on port 8081
- [ ] Agent connects to server successfully
- [ ] Client can see agents in list
- [ ] Client can create port forwards via tunnel
- [ ] Database connections work through tunnel
- [ ] Database commands are logged properly

## 🔍 Verification Commands

```bash
# Check server logs
tail -f /opt/goteleport/server.log | grep -E "TUNNEL|CLIENT|AGENT"

# Check agent logs  
tail -f /opt/goteleport/agent-db.log | grep -E "TUNNEL|DB_COMMAND"

# Test client connection
.\test-connection.bat

# Test unified client
.\unified-client.exe client-config-clean.json
```

Semua perubahan ini memungkinkan Windows client untuk mengakses database melalui agent yang ada di belakang NAT tanpa perlu port forwarding kompleks di firewall.
