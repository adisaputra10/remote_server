#!/bin/bash
# Monitor connection status for sh.adisaputra.online

echo "========================================"
echo "Remote Tunnel - Connection Monitor"
echo "========================================"

DOMAIN="sh.adisaputra.online"
RELAY_URL="wss://$DOMAIN:8443/ws"

echo "Monitoring connection to $DOMAIN..."
echo "Press Ctrl+C to stop monitoring"
echo

while true; do
    echo "[$(date)] Checking connection status..."
    
    # Test basic connectivity
    if ping -c 1 "$DOMAIN" >/dev/null 2>&1; then
        echo "✅ [$(date '+%H:%M:%S')] Domain $DOMAIN is reachable"
    else
        echo "❌ [$(date '+%H:%M:%S')] CRITICAL: Cannot reach $DOMAIN"
    fi
    
    # Test HTTPS port
    if timeout 5 bash -c "</dev/tcp/$DOMAIN/8443" 2>/dev/null; then
        echo "✅ [$(date '+%H:%M:%S')] Port 8443 is accessible"
    else
        echo "❌ [$(date '+%H:%M:%S')] Port 8443 is not accessible"
    fi
    
    # Test relay health endpoint
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "https://$DOMAIN:8443/health" --connect-timeout 5 2>/dev/null)
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ [$(date '+%H:%M:%S')] Relay server is healthy"
    else
        echo "❌ [$(date '+%H:%M:%S')] Relay server is not responding (HTTP $HTTP_CODE)"
    fi
    
    # Check if agent process is running
    if pgrep -f "bin/agent" >/dev/null; then
        echo "✅ [$(date '+%H:%M:%S')] Agent process is running"
    else
        echo "❌ [$(date '+%H:%M:%S')] Agent process is not running"
    fi
    
    # Check if client process is running
    if pgrep -f "bin/client" >/dev/null; then
        echo "✅ [$(date '+%H:%M:%S')] Client process is running"
    else
        echo "⚠️  [$(date '+%H:%M:%S')] No client process detected"
    fi
    
    echo
    sleep 30
done
