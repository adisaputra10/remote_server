#!/bin/bash
# Generate secure token for tunnel authentication

echo "========================================"
echo "Remote Tunnel - Token Generator"
echo "========================================"

echo "Generating secure authentication token..."
echo

# Try different methods to generate random token
if command -v openssl >/dev/null 2>&1; then
    NEW_TOKEN=$(openssl rand -hex 32)
    METHOD="openssl"
elif [ -f /dev/urandom ]; then
    NEW_TOKEN=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
    METHOD="urandom"
else
    # Fallback method
    NEW_TOKEN=$(date +%s | sha256sum | base64 | head -c 32)
    METHOD="fallback"
fi

echo "Generated Token (using $METHOD): $NEW_TOKEN"
echo
echo "========================================"
echo "IMPORTANT: Save this token securely!"
echo "========================================"
echo
echo "1. Copy this token to your .env.production file:"
echo "   TUNNEL_TOKEN=$NEW_TOKEN"
echo
echo "2. Use the SAME token on:"
echo "   - Relay server (103.195.169.32)"
echo "   - Agent (your laptop)"
echo "   - Client (remote connections)"
echo
echo "3. Keep this token secret and secure!"
echo

read -p "Do you want to automatically update .env.production? (y/n): " update_env

if [[ $update_env =~ ^[Yy]$ ]]; then
    if [ -f ".env.production" ]; then
        # Backup existing file
        cp .env.production .env.production.backup
        echo "Backed up existing .env.production to .env.production.backup"
    fi
    
    # Update or create .env.production
    if [ -f ".env.production" ] && grep -q "TUNNEL_TOKEN=" .env.production; then
        # Replace existing token
        sed -i.bak "s/TUNNEL_TOKEN=.*/TUNNEL_TOKEN=$NEW_TOKEN/" .env.production
    else
        # Add new token line
        echo "TUNNEL_TOKEN=$NEW_TOKEN" >> .env.production
    fi
    
    echo
    echo "✅ Token updated in .env.production"
fi

echo
echo "Remember to:"
echo "- Copy this token to your relay server"
echo "- Keep it secure and private"
echo "- Change it regularly for security"
echo

# Optionally save to clipboard if available
if command -v xclip >/dev/null 2>&1; then
    echo -n "$NEW_TOKEN" | xclip -selection clipboard
    echo "🔗 Token copied to clipboard (xclip)"
elif command -v pbcopy >/dev/null 2>&1; then
    echo -n "$NEW_TOKEN" | pbcopy
    echo "🔗 Token copied to clipboard (pbcopy)"
fi
