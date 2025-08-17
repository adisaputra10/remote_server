#!/bin/bash
# Deploy script khusus untuk sh.adisaputra.online

set -e

echo "================================================"
echo "Remote Tunnel Deployment to sh.adisaputra.online"
echo "================================================"

DOMAIN="sh.adisaputra.online"
RELAY_PORT=443

echo "This script will help deploy relay server to $DOMAIN"
echo

# Check if running on the target server
current_hostname=$(hostname -f 2>/dev/null || hostname)
echo "Current hostname: $current_hostname"

if [[ "$current_hostname" != *"$DOMAIN"* ]]; then
    echo
    echo "⚠️  Warning: This doesn't appear to be the target server."
    echo "   Expected hostname containing: $DOMAIN"
    echo "   Current hostname: $current_hostname"
    echo
    read -p "Continue anyway? (y/N): " confirm
    if [[ ! $confirm =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi
fi

echo
echo "🔍 Checking prerequisites..."

# Check if Go is installed
if ! command -v go >/dev/null 2>&1; then
    echo "❌ Go is not installed. Installing..."
    sudo apt update
    sudo apt install -y golang-go
    echo "✅ Go installed"
else
    echo "✅ Go is available: $(go version)"
fi

# Check if git is installed
if ! command -v git >/dev/null 2>&1; then
    echo "❌ Git is not installed. Installing..."
    sudo apt update
    sudo apt install -y git
    echo "✅ Git installed"
else
    echo "✅ Git is available"
fi

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "cmd/relay" ]; then
    echo "❌ Not in the correct project directory"
    echo "Please cd to the remote-tunnel project directory and try again"
    exit 1
fi

echo "✅ Project directory confirmed"

echo
echo "🔧 Building relay server..."
make build-linux
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build successful"

echo
echo "🔐 Setting up TLS certificates..."

# Check if certbot is installed
if ! command -v certbot >/dev/null 2>&1; then
    echo "Installing certbot..."
    sudo apt update
    sudo apt install -y certbot
fi

# Check if certificates already exist
if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
    echo "✅ Let's Encrypt certificates already exist for $DOMAIN"
    CERT_PATH="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
    KEY_PATH="/etc/letsencrypt/live/$DOMAIN/privkey.pem"
else
    echo "Obtaining Let's Encrypt certificate for $DOMAIN..."
    echo "Note: This requires the domain to point to this server"
    
    read -p "Proceed with Let's Encrypt certificate? (y/n): " get_cert
    if [[ $get_cert =~ ^[Yy]$ ]]; then
        sudo certbot certonly --standalone -d "$DOMAIN"
        if [ $? -eq 0 ]; then
            echo "✅ Certificate obtained successfully"
            CERT_PATH="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
            KEY_PATH="/etc/letsencrypt/live/$DOMAIN/privkey.pem"
        else
            echo "❌ Failed to obtain certificate. Using self-signed instead."
            CERT_PATH=""
            KEY_PATH=""
        fi
    else
        echo "Using self-signed certificates (development only)"
        CERT_PATH=""
        KEY_PATH=""
    fi
fi

echo
echo "⚙️  Creating production configuration..."

# Create production environment file
cat > .env.production << EOF
# Production Configuration for sh.adisaputra.online
TUNNEL_TOKEN=$(openssl rand -hex 32)
RELAY_HOST=$DOMAIN
RELAY_PORT=$RELAY_PORT
RELAY_ADDR=:$RELAY_PORT
RELAY_CERT_FILE=$CERT_PATH
RELAY_KEY_FILE=$KEY_PATH
EOF

echo "✅ Configuration created with secure random token"

echo
echo "🚀 Installing as system service..."

# Install systemd service
if [ -f "deploy/install.sh" ]; then
    sudo ./deploy/install.sh
    echo "✅ System service installed"
else
    echo "⚠️  System service installation script not found"
fi

echo
echo "🔥 Starting relay server..."

# Update systemd service configuration if needed
if [ -f "/etc/systemd/system/relay.service" ] && [ -n "$CERT_PATH" ]; then
    sudo sed -i "s|ExecStart=.*|ExecStart=/usr/local/bin/relay -addr :$RELAY_PORT -cert $CERT_PATH -key $KEY_PATH|" /etc/systemd/system/relay.service
    sudo systemctl daemon-reload
fi

# Start and enable service
sudo systemctl enable relay
sudo systemctl start relay

echo
echo "🔍 Checking service status..."
sleep 2
sudo systemctl status relay --no-pager

echo
echo "🌐 Testing endpoints..."
sleep 3

# Test health endpoint
if curl -k -s "https://$DOMAIN/health" | grep -q "OK"; then
    echo "✅ Health endpoint responding"
else
    echo "⚠️  Health endpoint not responding (may take a moment to start)"
fi

echo
echo "================================================"
echo "🎉 Deployment Summary"
echo "================================================"
echo "Domain: $DOMAIN"
echo "Port: $RELAY_PORT"
echo "Certificates: $([ -n "$CERT_PATH" ] && echo "Let's Encrypt" || echo "Self-signed")"
echo
echo "Endpoints:"
echo "- Agent: wss://$DOMAIN/ws/agent"
echo "- Client: wss://$DOMAIN/ws/client"  
echo "- Health: https://$DOMAIN/health"
echo
echo "Authentication Token (SAVE THIS!):"
echo "$(grep TUNNEL_TOKEN .env.production | cut -d= -f2)"
echo
echo "Next steps:"
echo "1. Save the token securely"
echo "2. Configure your laptop agent with this token"
echo "3. Test connection from your laptop"
echo
echo "Monitoring commands:"
echo "- sudo systemctl status relay"
echo "- sudo journalctl -u relay -f"
echo "- curl https://$DOMAIN/health"
echo "================================================"
