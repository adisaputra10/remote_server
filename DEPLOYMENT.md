# Production Deployment Guide
## Scenario: Public Relay Server + Local Agent

### Overview
- **Relay Server**: sh.adisaputra.online (Public server)
- **Agent**: Your laptop/local machine
- **Client**: Any remote machine that needs access

### Architecture
```
[Remote Client] ---> [Relay Server] <--- [Your Laptop Agent]
                  sh.adisaputra.online    (Local services)
```

## Step 1: Prepare Relay Server (sh.adisaputra.online)

### 1.1 Install Dependencies
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go git

# CentOS/RHEL
sudo yum install golang git
```

### 1.2 Deploy Code
```bash
# Clone repository
git clone <your-repo-url>
cd remote-tunnel

# Build binaries
make build-linux

# Install system-wide (optional)
sudo make install
```

### 1.3 Configure Environment
```bash
# Edit production config
nano .env.production

# Set secure token (IMPORTANT!)
TUNNEL_TOKEN=your-very-secure-random-token-here
RELAY_HOST=sh.adisaputra.online
RELAY_PORT=443
```

### 1.4 Setup TLS Certificates

#### Option A: Let's Encrypt (Recommended for production)
```bash
# Install certbot
sudo apt install certbot

# Get certificate (replace with your domain)
sudo certbot certonly --standalone -d sh.adisaputra.online

# Update config
RELAY_CERT_FILE=/etc/letsencrypt/live/sh.adisaputra.online/fullchain.pem
RELAY_KEY_FILE=/etc/letsencrypt/live/sh.adisaputra.online/privkey.pem
```

#### Option B: Self-signed (Development/Testing)
```bash
# Will auto-generate when relay starts
./start-relay.sh
```

### 1.5 Start Relay Server
```bash
# Make executable
chmod +x start-relay.sh

# Start server (will listen on port 443)
sudo ./start-relay.sh
```

### 1.6 Setup as System Service (Production)
```bash
# Install systemd service
sudo ./deploy/install.sh

# Start and enable
sudo systemctl enable relay
sudo systemctl start relay

# Check status
sudo systemctl status relay
```

## Step 2: Configure Your Laptop (Agent)

### 2.1 Build Binaries
```bash
# Windows
build.bat

# Linux/Mac
make build
```

### 2.2 Configure Environment
Edit `.env.production`:
```ini
TUNNEL_TOKEN=your-very-secure-random-token-here
AGENT_ID=laptop-agent
AGENT_RELAY_URL=wss://sh.adisaputra.online/ws/agent
```

### 2.3 Test Connection
```bash
# Windows
test-connection.bat

# Linux/Mac
./test-connection.sh
```

### 2.4 Start Agent
```bash
# Windows (Interactive)
start-agent.bat

# Linux/Mac
./start-agent.sh

# Or manual command
./bin/agent -id laptop-agent -relay-url wss://sh.adisaputra.online/ws/agent -allow 127.0.0.1:22 -token your-token
```

## Step 3: Connect from Remote Client

### 3.1 Setup Client Machine
```bash
# Build or download client binary
go build -o client ./cmd/client

# Or download from releases
```

### 3.2 Create Tunnel
```bash
# SSH tunnel (access laptop's SSH via port 2222)
./client -L :2222 -relay-url wss://sh.adisaputra.online/ws/client -agent laptop-agent -target 127.0.0.1:22 -token your-token

# Web server tunnel (access laptop's web server via port 8080)
./client -L :8080 -relay-url wss://sh.adisaputra.online/ws/client -agent laptop-agent -target 127.0.0.1:8080 -token your-token
```

### 3.3 Use the Tunnel
```bash
# SSH to your laptop
ssh -p 2222 user@localhost

# Access web service
curl http://localhost:8080
```

## Security Considerations

### 1. Firewall Configuration
```bash
# On relay server - allow only necessary ports
sudo ufw allow 443/tcp
sudo ufw enable
```

### 2. Token Security
- Use a strong, random token (32+ characters)
- Rotate tokens regularly
- Store tokens securely (environment variables, not in code)

### 3. TLS Certificates
- Use Let's Encrypt or proper CA certificates in production
- Avoid self-signed certificates for production use

### 4. Agent Access Control
Configure agent to only allow specific services:
```bash
# Only SSH
./agent -allow 127.0.0.1:22

# Multiple specific services
./agent -allow 127.0.0.1:22 -allow 127.0.0.1:8080 -allow 127.0.0.1:5432
```

## Troubleshooting

### Connection Issues
1. **Check relay server logs**: `sudo journalctl -u relay -f`
2. **Verify firewall**: Ensure port 443 is open
3. **Test connectivity**: Use `test-connection.bat` script
4. **Check certificates**: Verify TLS setup

### Agent Issues
1. **Verify token**: Ensure same token on relay and agent
2. **Check network**: Agent needs outbound HTTPS access
3. **Service availability**: Ensure target services are running

### Client Issues
1. **WebSocket connection**: Check if relay is accessible
2. **Agent availability**: Verify agent is connected to relay
3. **Local ports**: Ensure local ports are not in use

## Monitoring

### Health Checks
```bash
# Relay server health
curl https://sh.adisaputra.online/health

# Check logs
sudo journalctl -u relay -f
sudo journalctl -u agent@laptop-agent -f
```

### Status Commands
```bash
# Systemd status
sudo systemctl status relay
sudo systemctl status agent@laptop-agent

# Process monitoring
ps aux | grep -E "(relay|agent|client)"
```
