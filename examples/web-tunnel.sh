#!/bin/bash
# Example: Web Server Tunnel Setup

# This example shows how to expose a local web server
# through the tunnel for remote access

# Configuration
RELAY_URL="wss://sh.adisaputra.online:8443/ws"
TUNNEL_TOKEN="your-secure-token"
AGENT_ID="web-server"
LOCAL_WEB_PORT="8080"
TARGET_WEB_PORT="80"

echo "Setting up web server tunnel..."
echo "Relay: $RELAY_URL"
echo "Agent ID: $AGENT_ID"
echo "Local port: $LOCAL_WEB_PORT -> Remote port: $TARGET_WEB_PORT"

# On the remote server (where web server is running)
echo "1. On remote server, run:"
echo "   ./agent -id $AGENT_ID -relay-url $RELAY_URL/agent -allow 127.0.0.1:$TARGET_WEB_PORT -token $TUNNEL_TOKEN -insecure"

# On your local machine
echo "2. On local machine, run:"
echo "   ./client -L :$LOCAL_WEB_PORT -relay-url $RELAY_URL/client -agent $AGENT_ID -target 127.0.0.1:$TARGET_WEB_PORT -token $TUNNEL_TOKEN -insecure"echo "3. Access web server:"
echo "   curl http://localhost:$LOCAL_WEB_PORT"
echo "   # Or open http://localhost:$LOCAL_WEB_PORT in browser"

echo
echo "Multiple ports example:"
echo "Agent with multiple services:"
echo "  ./agent -id multi-server -relay-url $RELAY_URL/agent \\"
echo "    -allow 127.0.0.1:80 -allow 127.0.0.1:443 -allow 127.0.0.1:3000 \\"
echo "    -token $TUNNEL_TOKEN"
echo
echo "Clients for each service:"
echo "  ./client -L :8080 -agent multi-server -target 127.0.0.1:80 -token $TUNNEL_TOKEN"
echo "  ./client -L :8443 -agent multi-server -target 127.0.0.1:443 -token $TUNNEL_TOKEN"
echo "  ./client -L :3000 -agent multi-server -target 127.0.0.1:3000 -token $TUNNEL_TOKEN"
