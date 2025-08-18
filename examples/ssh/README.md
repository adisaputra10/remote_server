# SSH Tunneling Examples

This directory contains examples of using the SSH client with the remote tunnel system.

## Basic SSH Access

### Example 1: Password Authentication
```bash
# Start relay server
./relay -addr :8443 -token demo-token

# Start agent on target server
./agent -relay-url wss://relay.example.com/ws/agent \
        -id server1 \
        -token demo-token \
        -services ssh:22

# Connect with SSH client
ssh-client -relay-url wss://relay.example.com/ws/client \
           -agent server1 \
           -token demo-token \
           -user admin
# Will prompt for password
```

### Example 2: Private Key Authentication
```bash
# Generate SSH key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/tunnel_key

# Copy public key to target server
ssh-copy-id -i ~/.ssh/tunnel_key.pub admin@target-server

# Connect using private key
ssh-client -relay-url wss://relay.example.com/ws/client \
           -agent server1 \
           -token demo-token \
           -user admin \
           -key ~/.ssh/tunnel_key
```

## Advanced Configuration

### Example 3: Production Setup with Compression
```bash
# Create environment file
cat > .env.production << EOF
RELAY_URL=wss://tunnel.production.com/ws/client
TUNNEL_TOKEN=prod-secret-token-123
SSH_USER=deploy
SSH_KEY=/home/admin/.ssh/production_key
LOG_DIR=/var/log/ssh-tunnel
EOF

# Start SSH client with compression
start-ssh-client.sh -agent production-server -compress
```

### Example 4: Multi-Server Access Script
```bash
#!/bin/bash
# multi-ssh.sh - Connect to multiple servers

SERVERS=("web1" "web2" "db1" "cache1")
TOKEN="production-token"
USER="ops"
KEY="~/.ssh/ops_key"

echo "Available servers:"
for i in "${!SERVERS[@]}"; do
    echo "$((i+1)). ${SERVERS[$i]}"
done

read -p "Select server (1-${#SERVERS[@]}): " choice
server=${SERVERS[$((choice-1))]}

if [[ -n "$server" ]]; then
    echo "Connecting to $server..."
    ssh-client -agent "$server" \
               -token "$TOKEN" \
               -user "$USER" \
               -key "$KEY" \
               -log-dir "logs/$server" \
               -compress
else
    echo "Invalid selection"
fi
```

## Command Logging Examples

### Example 5: Audit Mode with Enhanced Logging
```bash
# SSH client with detailed logging
ssh-client -agent secure-server \
           -token audit-token \
           -user auditor \
           -key ~/.ssh/audit_key \
           -log-dir /var/log/audit/ssh \
           -log=true

# View logs in real-time
tail -f /var/log/audit/ssh/session-*.log
tail -f /var/log/audit/ssh/commands-*.log
```

### Example 6: Log Analysis Script
```bash
#!/bin/bash
# analyze-ssh-logs.sh - Analyze SSH session logs

LOG_DIR="ssh-logs"

echo "=== SSH Session Summary ==="
echo "Total sessions: $(ls $LOG_DIR/session-*.log 2>/dev/null | wc -l)"
echo "Total commands: $(cat $LOG_DIR/commands-*.log 2>/dev/null | wc -l)"

echo ""
echo "=== Most used commands ==="
cat $LOG_DIR/commands-*.log 2>/dev/null | \
    grep '\[CMD\]' | \
    awk '{print $3}' | \
    sort | uniq -c | sort -nr | head -10

echo ""
echo "=== Recent sessions ==="
ls -lt $LOG_DIR/session-*.log 2>/dev/null | head -5
```

## Integration Examples

### Example 7: SSH through Jump Host
```bash
# Setup: Client → Relay → Jump Host → Target Server

# On jump host, run agent
./agent -relay-url wss://relay.example.com/ws/agent \
        -id jump-host \
        -token jump-token \
        -services ssh:22

# From client, connect to jump host
ssh-client -agent jump-host -token jump-token -user admin

# Inside SSH session on jump host, start another tunnel to target
./agent -relay-url wss://relay.example.com/ws/agent \
        -id target-server \
        -token target-token \
        -services ssh:22

# From another client terminal, connect to target via jump
ssh-client -agent target-server -token target-token -user root
```

### Example 8: SSH with Port Forwarding
```bash
# Connect with SSH and setup local port forwarding
ssh-client -agent database-server \
           -token db-token \
           -user dbadmin \
           -key ~/.ssh/db_key

# Inside SSH session, create port forward
ssh -L 3306:localhost:3306 localhost

# Now you can connect to MySQL via localhost:3306
mysql -h 127.0.0.1 -P 3306 -u root -p
```

## Scripted Automation

### Example 9: Automated Deployment Script
```bash
#!/bin/bash
# deploy.sh - Automated deployment via SSH tunnel

SERVERS=("web1" "web2" "api1" "api2")
DEPLOY_KEY="~/.ssh/deploy_key"
TOKEN="deploy-token"
USER="deploy"

for server in "${SERVERS[@]}"; do
    echo "Deploying to $server..."
    
    # Connect and run deployment commands
    ssh-client -agent "$server" \
               -token "$TOKEN" \
               -user "$USER" \
               -key "$DEPLOY_KEY" \
               -log-dir "logs/deploy/$server" \
               << 'EOF'
        cd /opt/app
        git pull origin main
        ./build.sh
        sudo systemctl restart app
        sudo systemctl status app
EOF
    
    if [[ $? -eq 0 ]]; then
        echo "✅ Deployment to $server successful"
    else
        echo "❌ Deployment to $server failed"
    fi
done
```

### Example 10: Health Check Script
```bash
#!/bin/bash
# health-check.sh - Check server health via SSH tunnel

check_server() {
    local server=$1
    echo "Checking $server..."
    
    timeout 30 ssh-client -agent "$server" \
                          -token "$TOKEN" \
                          -user "monitor" \
                          -key "~/.ssh/monitor_key" \
                          -log=false \
                          << 'EOF' > /tmp/health-$server.log 2>&1
        # Basic health checks
        echo "=== System Info ==="
        uptime
        df -h
        free -m
        
        echo "=== Service Status ==="
        systemctl status nginx
        systemctl status mysql
        
        echo "=== Network ==="
        netstat -tlpn | head -10
EOF
    
    if [[ $? -eq 0 ]]; then
        echo "✅ $server: OK"
    else
        echo "❌ $server: FAILED"
        cat /tmp/health-$server.log
    fi
}

# Check all servers
SERVERS=("web1" "web2" "db1")
TOKEN="monitor-token"

for server in "${SERVERS[@]}"; do
    check_server "$server" &
done

wait
echo "Health check complete"
```

## Security Examples

### Example 11: Secure Key Management
```bash
#!/bin/bash
# secure-ssh.sh - SSH with secure key handling

# Key stored in secure location
SSH_KEY="/etc/tunnel-keys/production.pem"
if [[ ! -f "$SSH_KEY" ]]; then
    echo "Error: SSH key not found at $SSH_KEY"
    exit 1
fi

# Check key permissions
KEY_PERMS=$(stat -c %a "$SSH_KEY")
if [[ "$KEY_PERMS" != "600" ]]; then
    echo "Warning: SSH key has incorrect permissions ($KEY_PERMS)"
    echo "Setting correct permissions..."
    chmod 600 "$SSH_KEY"
fi

# Connect with secure configuration
ssh-client -agent "$1" \
           -token "$(cat /etc/tunnel-token)" \
           -user "secure-user" \
           -key "$SSH_KEY" \
           -log-dir "/var/log/secure-ssh" \
           -compress
```

### Example 12: Session Recording
```bash
#!/bin/bash
# record-session.sh - SSH with session recording

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
SESSION_DIR="/var/log/ssh-sessions/$TIMESTAMP"
mkdir -p "$SESSION_DIR"

# Start SSH with recording
script -f "$SESSION_DIR/terminal.log" -c "
    ssh-client -agent '$1' \
               -token '$TOKEN' \
               -user '$USER' \
               -key '$SSH_KEY' \
               -log-dir '$SESSION_DIR'
"

# Create session metadata
cat > "$SESSION_DIR/metadata.json" << EOF
{
    "timestamp": "$TIMESTAMP",
    "agent": "$1",
    "user": "$USER",
    "client_ip": "$(curl -s ifconfig.me)",
    "session_dir": "$SESSION_DIR"
}
EOF

echo "Session recorded in: $SESSION_DIR"
```

## Troubleshooting Examples

### Example 13: Debug Connection Issues
```bash
#!/bin/bash
# debug-ssh.sh - Debug SSH tunnel issues

echo "=== Debugging SSH Tunnel Connection ==="

# Test relay connectivity
echo "1. Testing relay connectivity..."
curl -I "$RELAY_URL" 2>/dev/null
if [[ $? -eq 0 ]]; then
    echo "✅ Relay is reachable"
else
    echo "❌ Relay is not reachable"
fi

# Test with verbose logging
echo "2. Testing SSH connection with debug..."
ssh-client -agent "$AGENT_ID" \
           -token "$TOKEN" \
           -user "$USER" \
           -key "$SSH_KEY" \
           -debug \
           -insecure \
           2>&1 | tee debug-ssh.log

echo "Debug log saved to: debug-ssh.log"
```

These examples demonstrate various ways to use the SSH client in different scenarios, from basic connectivity to advanced automation and security configurations.
