#!/bin/bash
# Installation script for Linux

set -e

echo "Installing Remote Tunnel on Linux..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)" 
   exit 1
fi

# Create user and directories
echo "Creating tunnel user and directories..."
useradd -r -s /bin/false tunnel 2>/dev/null || echo "User tunnel already exists"
mkdir -p /opt/remote-tunnel
mkdir -p /etc/ssl/tunnel
mkdir -p /var/log/remote-tunnel
chown tunnel:tunnel /var/log/remote-tunnel

# Copy binaries
echo "Installing binaries..."
if [ -f "bin/relay" ]; then
    cp bin/relay /usr/local/bin/
    chmod +x /usr/local/bin/relay
fi

if [ -f "bin/agent" ]; then
    cp bin/agent /usr/local/bin/
    chmod +x /usr/local/bin/agent
fi

if [ -f "bin/client" ]; then
    cp bin/client /usr/local/bin/
    chmod +x /usr/local/bin/client
fi

# Install systemd services
echo "Installing systemd services..."
cp deploy/relay.service /etc/systemd/system/
cp deploy/agent@.service /etc/systemd/system/

# Create default configuration
echo "Creating default configuration..."
cat > /etc/default/remote-tunnel << EOF
# Remote Tunnel Configuration
TUNNEL_TOKEN=change-this-token-in-production
RELAY_ADDR=:443
RELAY_URL=wss://your-relay-server.com
LOG_LEVEL=info
EOF

# Generate self-signed certificate if not exists
if [ ! -f "/etc/ssl/tunnel/server.crt" ]; then
    echo "Generating self-signed certificate..."
    openssl req -x509 -newkey rsa:4096 -keyout /etc/ssl/tunnel/server.key -out /etc/ssl/tunnel/server.crt -days 365 -nodes -subj "/CN=localhost"
    chown tunnel:tunnel /etc/ssl/tunnel/server.*
    chmod 600 /etc/ssl/tunnel/server.key
    chmod 644 /etc/ssl/tunnel/server.crt
fi

# Reload systemd
systemctl daemon-reload

echo "Installation complete!"
echo
echo "Next steps:"
echo "1. Edit /etc/default/remote-tunnel with your configuration"
echo "2. For relay server:"
echo "   sudo systemctl enable relay"
echo "   sudo systemctl start relay"
echo "3. For agent (replace 'myagent' with your agent ID):"
echo "   sudo systemctl enable agent@myagent"
echo "   sudo systemctl start agent@myagent"
echo
echo "Check status with:"
echo "   sudo systemctl status relay"
echo "   sudo systemctl status agent@myagent"
echo
echo "View logs with:"
echo "   sudo journalctl -u relay -f"
echo "   sudo journalctl -u agent@myagent -f"
