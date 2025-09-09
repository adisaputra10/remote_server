# NAT Architecture Setup Guide

## 🏗️ Network Architecture

```
Windows Client (unified-client.exe)
         ↓
         ↓ WebSocket Tunnel
         ↓
Linux Server (168.231.119.242:8081)
         ↓
         ↓ Forward to Agent
         ↓  
Linux Agent (behind NAT) - Database Proxy (3307, 5435)
         ↓
         ↓ Database Connection
         ↓
MySQL/PostgreSQL Databases
```

## 🔧 Key Changes untuk NAT Support

### 1. **Unified Client (Windows)**
- **Before**: Langsung connect ke agent ports (3307, 5435)  
- **After**: Membuat tunnel melalui server WebSocket

```go
// Old approach (gagal untuk NAT)
proxyAddr = fmt.Sprintf("%s:3307", agentIP)
targetConn, err := net.Dial("tcp", proxyAddr)

// New approach (works dengan NAT)  
err := pf.createTunnelThroughServer(clientConn, agentID, targetPort, dbType)
```

### 2. **Server (Linux)**  
Server perlu endpoint `/ws/tunnel` untuk:
- Menerima tunnel requests dari client
- Forward data ke agent yang tepat
- Handle bidirectional data transfer

### 3. **Agent (Linux behind NAT)**
- Tetap running database proxy (3307, 5435)
- Connect ke server via WebSocket
- Tidak perlu port forwarding di NAT/firewall

## 🚀 Testing Connection

### 1. Test Server Accessibility
```bash
test-connection.bat
```

### 2. Check Yang Perlu Diverifikasi
```bash
# Server harus accessible dari Windows
telnet 168.231.119.242 8081

# Agent harus connect ke server (check server logs)
# Database proxy harus running di agent (check agent logs)
```

## 🐛 Troubleshooting

### Issue: "No connection could be made" 
**Cause**: Client masih mencoba direct connection ke agent

**Solution**: 
- Update unified-client.exe dengan tunnel support
- Pastikan server mendukung `/ws/tunnel` endpoint

### Issue: Agent tidak terdeteksi
**Cause**: Agent tidak connect ke server atau server tidak forward request

**Solution**:
- Check agent connection ke server
- Verify server tunnel endpoint implementation
- Check logs di server dan agent

### Issue: Database connection gagal
**Cause**: Agent database proxy tidak running atau database tidak accessible

**Solution**:
- Verify agent database proxy running (port 3307, 5435)
- Check database credentials dan connectivity
- Review agent-config-db.json

## 📝 Next Steps

1. **Server Update**: Implement `/ws/tunnel` endpoint di server
2. **Agent Logging**: Tambah logging untuk tunnel requests  
3. **Client Testing**: Test dengan real database connections

## 🔍 Monitoring

```bash
# Windows Client
.\unified-client.exe client-config-clean.json

# Linux Server (monitor tunnel requests)
tail -f server.log | grep -i tunnel

# Linux Agent (monitor database proxy)  
tail -f agent-db.log | grep -i "DB_COMMAND\|proxy"
```

Dengan arsitektur ini, client Windows bisa mengakses database melalui agent yang ada di belakang NAT tanpa perlu port forwarding kompleks.
