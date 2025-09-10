#!/bin/bash

# Deploy tunnel system to Linux server
echo "🚀 Deploying Tunnel System to Linux Server"
echo "==========================================="

# Configuration
REMOTE_USER=${REMOTE_USER:-"root"}
REMOTE_HOST=${REMOTE_HOST:-"your-server-ip"}
REMOTE_PATH=${REMOTE_PATH:-"/opt/tunnel-system"}

# Check if we have the files to deploy
if [ ! -d "build/linux" ]; then
    echo "❌ Build directory not found!"
    echo "Please run: ./build-linux.sh first"
    exit 1
fi

echo "📋 Deployment Configuration:"
echo "  - Remote User: $REMOTE_USER"
echo "  - Remote Host: $REMOTE_HOST"
echo "  - Remote Path: $REMOTE_PATH"
echo ""

# Confirm deployment
read -p "Continue with deployment? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled"
    exit 1
fi

# Create deployment package
echo "📦 Creating deployment package..."
cd build/linux
tar -czf ../tunnel-system-linux.tar.gz *
cd ../..

# Upload to server
echo "📤 Uploading to server..."
scp build/tunnel-system-linux.tar.gz $REMOTE_USER@$REMOTE_HOST:/tmp/

# Upload scripts
echo "📤 Uploading deployment scripts..."
scp start-server-linux.sh start-agent-linux.sh stop-server-linux.sh stop-agent-linux.sh status-linux.sh $REMOTE_USER@$REMOTE_HOST:/tmp/

# Deploy on server
echo "🔧 Deploying on server..."
ssh $REMOTE_USER@$REMOTE_HOST << 'EOF'
# Create directory
sudo mkdir -p /opt/tunnel-system
cd /opt/tunnel-system

# Extract files
sudo tar -xzf /tmp/tunnel-system-linux.tar.gz

# Move scripts
sudo mv /tmp/*-linux.sh .

# Make everything executable
sudo chmod +x tunnel-* *.sh

# Create systemd service for server
sudo tee /etc/systemd/system/tunnel-server.service > /dev/null << 'EOL'
[Unit]
Description=Tunnel Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/tunnel-system
ExecStart=/opt/tunnel-system/tunnel-server -config=/opt/tunnel-system/server-config-db.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL

# Create systemd service for agent
sudo tee /etc/systemd/system/tunnel-agent.service > /dev/null << 'EOL'
[Unit]
Description=Tunnel Agent
After=network.target tunnel-server.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/tunnel-system
ExecStart=/opt/tunnel-system/tunnel-agent -config=/opt/tunnel-system/agent-config-db.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL

# Reload systemd
sudo systemctl daemon-reload

echo "✅ Deployment completed!"
echo ""
echo "To start services:"
echo "  sudo systemctl start tunnel-server"
echo "  sudo systemctl start tunnel-agent" 
echo ""
echo "To enable auto-start:"
echo "  sudo systemctl enable tunnel-server"
echo "  sudo systemctl enable tunnel-agent"
echo ""
echo "Manual start:"
echo "  cd /opt/tunnel-system"
echo "  ./start-server-linux.sh"
echo "  ./start-agent-linux.sh"
EOF

echo ""
echo "🎉 Deployment completed successfully!"
echo ""
echo "Next steps on your Linux server:"
echo "1. SSH to server: ssh $REMOTE_USER@$REMOTE_HOST"
echo "2. Go to directory: cd $REMOTE_PATH"
echo "3. Edit configurations: nano server-config-db.json && nano agent-config-db.json"
echo "4. Start server: ./start-server-linux.sh"
echo "5. Start agent: ./start-agent-linux.sh"
echo "6. Check status: ./status-linux.sh"
