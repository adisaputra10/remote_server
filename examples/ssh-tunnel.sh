#!/bin/bash
# Example: SSH Tunnel Setup

# This example shows how to set up a secure SSH tunnel
# from your local machine to a remote server through the relay

# Configuration
RELAY_URL="wss://sh.adisaputra.online:8443/ws"
TUNNEL_TOKEN="your-secure-token"
AGENT_ID="ssh-server"
LOCAL_SSH_PORT="2222"
TARGET_SSH_PORT="22"

echo "Setting up SSH tunnel..."
echo "Relay: $RELAY_URL"
echo "Agent ID: $AGENT_ID"
echo "Local port: $LOCAL_SSH_PORT -> Remote port: $TARGET_SSH_PORT"

# On the remote server (where SSH server is running)
echo "1. On remote server, run:"
echo "   ./agent -id $AGENT_ID -relay-url $RELAY_URL/agent -allow 127.0.0.1:$TARGET_SSH_PORT -token $TUNNEL_TOKEN -insecure"

# On your local machine
echo "2. On local machine, run:"
echo "   ./client -L :$LOCAL_SSH_PORT -relay-url $RELAY_URL/client -agent $AGENT_ID -target 127.0.0.1:$TARGET_SSH_PORT -token $TUNNEL_TOKEN -insecure"

echo "3. Connect via SSH:"
echo "   ssh -p $LOCAL_SSH_PORT user@localhost"

echo
echo "Note: Replace 'sh.adisaputra.online' with your actual relay server address"
echo "      and 'your-secure-token' with your actual authentication token"
