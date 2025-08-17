# Quick Start Guide: Production Scenario
## Relay Server (sh.adisaputra.online) + Local Agent

### 🚀 Quick Setup (5 minutes)

#### 1. **Prepare Your Laptop (Agent)**

```bash
# Windows
git clone <repo-url>
cd remote-tunnel
build.bat
copy .env.production.example .env.production

# Linux/macOS
git clone <repo-url>
cd remote-tunnel
chmod +x setup.sh && ./setup.sh
./build.sh
cp .env.production .env.production.local
```

#### 2. **Configure Security Token**

Edit `.env.production`:
```ini
TUNNEL_TOKEN=your-very-secure-random-token-here-min-32-chars
```

**Generate secure token:**
```bash
# Linux/macOS
openssl rand -hex 32

# Windows (PowerShell)
[System.Web.Security.Membership]::GeneratePassword(32, 4)

# Or use online generator (ensure it's secure)
```

#### 3. **Test Connection**

```bash
# Windows
test-connection.bat

# Linux/macOS
./test-connection.sh
```

#### 4. **Start Agent on Your Laptop**

```bash
# Windows (Interactive menu)
start-agent.bat

# Linux/macOS
./start-agent.sh

# Or manual command for SSH only
./bin/agent -id laptop-agent -relay-url wss://sh.adisaputra.online:8443/ws/agent -allow 127.0.0.1:22 -token YOUR_TOKEN -insecure
```

#### 5. **Setup Relay Server (sh.adisaputra.online)**

```bash
# SSH to your server
ssh user@sh.adisaputra.online

# Install Go and dependencies
sudo apt update && sudo apt install golang-go git

# Deploy code
git clone <repo-url>
cd remote-tunnel
make build-linux

# Copy your token configuration
nano .env.production
# Set same TUNNEL_TOKEN as laptop

# Start relay (as root for port 443)
sudo ./start-relay.sh
```

#### 6. **Test Tunnel from Remote Client**

```bash
# Download client binary or build locally
go build -o client ./cmd/client

# Create SSH tunnel (laptop SSH accessible via local port 2222)
./client -L :2222 -relay-url wss://sh.adisaputra.online:8443/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN -insecure

# In another terminal, test SSH connection
ssh -p 2222 your-username@localhost
```

---

### 🔧 **Common Use Cases**

#### **SSH Access to Laptop**
```bash
# Agent (laptop)
./agent -id laptop-ssh -relay-url wss://sh.adisaputra.online:8443/ws/agent -allow 127.0.0.1:22 -token TOKEN -insecure

# Client (remote)
./client -L :2222 -relay-url wss://sh.adisaputra.online:8443/ws/client -agent laptop-ssh -target 127.0.0.1:22 -token TOKEN -insecure

# Connect
ssh -p 2222 user@localhost
```

#### **Web Development Server Access**
```bash
# Agent (laptop running dev server on port 3000)
./agent -id laptop-dev -relay-url wss://sh.adisaputra.online:8443/ws/agent -allow 127.0.0.1:3000 -token TOKEN -insecure

# Client (remote)
./client -L :3000 -relay-url wss://sh.adisaputra.online:8443/ws/client -agent laptop-dev -target 127.0.0.1:3000 -token TOKEN -insecure

# Access
curl http://localhost:3000
```

#### **Database Access**
```bash
# Agent (laptop running PostgreSQL)
./agent -id laptop-db -relay-url wss://sh.adisaputra.online:8443/ws/agent -allow 127.0.0.1:5432 -token TOKEN -insecure

# Client (remote)
./client -L :5432 -relay-url wss://sh.adisaputra.online:8443/ws/client -agent laptop-db -target 127.0.0.1:5432 -token TOKEN -insecure

# Connect
psql -h localhost -p 5432 -U username dbname
```

---

### 🛠️ **Troubleshooting**

#### **Connection Issues**
1. **Firewall**: Ensure port 443 is open on relay server
2. **Token**: Verify same token on all components
3. **Network**: Check if laptop can reach relay server
4. **Certificates**: Self-signed certs may need `-k` flag with curl

#### **Quick Tests**
```bash
# Test relay server health
curl -k https://sh.adisaputra.online:8443/health

# Test if agent is connected (check relay logs)
sudo journalctl -u relay -f

# Test local services
netstat -tlnp | grep :22    # SSH
netstat -tlnp | grep :8080  # Web server
```

#### **Security Checklist**
- ✅ Use strong random token (32+ characters)
- ✅ Restrict agent `-allow` to only needed services
- ✅ Use proper TLS certificates in production
- ✅ Monitor relay server logs
- ✅ Update tokens regularly

---

### 📋 **Summary**

1. **Laptop Agent**: Connects outbound to relay, exposes local services
2. **Relay Server**: Public endpoint, routes connections
3. **Remote Client**: Creates local tunnels to access laptop services

**Data Flow**: `Remote Client ↔ Relay Server ↔ Laptop Agent ↔ Local Service`
