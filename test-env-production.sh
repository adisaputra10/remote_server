#!/bin/bash
# test-env-production.sh
# Script untuk test konfigurasi .env.production

echo "==============================================="
echo "Testing .env.production Configuration"
echo "==============================================="

# Check if .env.production exists
if [ ! -f ".env.production" ]; then
    echo "❌ Error: .env.production file not found!"
    echo "Please create .env.production file first"
    exit 1
fi

echo "✅ .env.production file found"
echo

# Load environment variables
echo "Loading environment variables..."
set -a
source <(grep -v '^#\|^$' .env.production)
set +a

echo "✅ Environment variables loaded"
echo

# Display configuration
echo "Production Configuration Summary:"
echo "================================="
echo

echo "🔐 Security:"
echo "- Token: ${TUNNEL_TOKEN:0:10}...${TUNNEL_TOKEN: -4} (${#TUNNEL_TOKEN} chars)"

echo
echo "🖥️  Relay Server:"
echo "- Host: ${RELAY_HOST:-not set}"
echo "- Port: ${RELAY_PORT:-not set}"
echo "- Address: ${RELAY_ADDR:-not set}"
echo "- Certificate: ${RELAY_CERT_FILE:-not set}"
echo "- Private Key: ${RELAY_KEY_FILE:-not set}"
echo "- TLS Enabled: ${TLS_ENABLED:-not set}"

echo
echo "🤖 Agent:"
echo "- ID: ${AGENT_ID:-not set}"
echo "- Relay URL: ${AGENT_RELAY_URL:-not set}"

echo
echo "🔌 Allowed Services:"
echo "- SSH: ${AGENT_ALLOW_SSH:-not configured}"
echo "- HTTP: ${AGENT_ALLOW_HTTP:-not configured}"
echo "- HTTPS: ${AGENT_ALLOW_HTTPS:-not configured}"
echo "- Web Dev: ${AGENT_ALLOW_WEB:-not configured}"
echo "- Dev Server: ${AGENT_ALLOW_DEV:-not configured}"
echo "- PostgreSQL: ${AGENT_ALLOW_POSTGRES:-not configured}"
echo "- MySQL: ${AGENT_ALLOW_MYSQL:-not configured}"
echo "- Redis: ${AGENT_ALLOW_REDIS:-not configured}"

echo
echo "📱 Client:"
echo "- Relay URL: ${CLIENT_RELAY_URL:-not set}"
echo "- SSH Port: ${CLIENT_SSH_PORT:-not set}"
echo "- Web Port: ${CLIENT_WEB_PORT:-not set}"
echo "- DB Port: ${CLIENT_DB_PORT:-not set}"

echo
echo "🔍 Validation:"
echo "=============="

# Validate required variables
ERRORS=0

if [ -z "$TUNNEL_TOKEN" ]; then
    echo "❌ TUNNEL_TOKEN is required"
    ERRORS=$((ERRORS + 1))
elif [ ${#TUNNEL_TOKEN} -lt 20 ]; then
    echo "⚠️  TUNNEL_TOKEN should be at least 20 characters for security"
fi

if [ -z "$RELAY_HOST" ]; then
    echo "❌ RELAY_HOST is required"
    ERRORS=$((ERRORS + 1))
fi

if [ -z "$AGENT_ID" ]; then
    echo "❌ AGENT_ID is required"
    ERRORS=$((ERRORS + 1))
fi

if [ -z "$AGENT_RELAY_URL" ]; then
    echo "❌ AGENT_RELAY_URL is required"
    ERRORS=$((ERRORS + 1))
fi

# Check if at least one service is configured
SERVICES_COUNT=0
[ -n "$AGENT_ALLOW_SSH" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_HTTP" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_HTTPS" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_WEB" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_DEV" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_POSTGRES" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_MYSQL" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))
[ -n "$AGENT_ALLOW_REDIS" ] && SERVICES_COUNT=$((SERVICES_COUNT + 1))

if [ $SERVICES_COUNT -eq 0 ]; then
    echo "⚠️  No services configured in AGENT_ALLOW_* variables"
    echo "   Services will be prompted during agent startup"
else
    echo "✅ $SERVICES_COUNT service(s) pre-configured"
fi

echo
if [ $ERRORS -eq 0 ]; then
    echo "✅ Configuration validation passed!"
    echo
    echo "🚀 Ready to start:"
    echo "1. Relay server: ./start-relay.sh"
    echo "2. Agent: ./start-agent.sh"
    echo "3. SSH Agent Terminal: ./start-ssh-agent-terminal.bat (Windows)"
else
    echo "❌ Configuration has $ERRORS error(s)"
    echo "Please fix the errors before starting services"
    exit 1
fi

echo
echo "💡 Example URLs after startup:"
echo "- Agent connection: wss://${RELAY_HOST}:${RELAY_PORT:-8443}/ws/agent"
echo "- Client connection: wss://${RELAY_HOST}:${RELAY_PORT:-8443}/ws/client"
echo "- Health check: https://${RELAY_HOST}:${RELAY_PORT:-8443}/health"

echo
echo "==============================================="
echo "Configuration test completed"
echo "==============================================="
