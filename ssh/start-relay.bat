# Start SSH Tunnel Relay Server with Environment Configuration

echo "🚀 Starting SSH Tunnel Relay Server..."
echo ""

# Load environment variables
& ".\load-env.bat"

# Check if MySQL is accessible
echo "🔍 Checking MySQL connection..."
$mysqlCheck = $false

try {
    # Test MySQL connection using environment variables
    $mysqlHost = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
    $mysqlPort = if ($env:DB_PORT) { $env:DB_PORT } else { "3306" }
    
    $connection = New-Object System.Net.Sockets.TcpClient
    $connection.Connect($mysqlHost, $mysqlPort)
    $connection.Close()
    $mysqlCheck = $true
    echo "  ✅ MySQL server is accessible at ${mysqlHost}:${mysqlPort}"
} catch {
    echo "  ❌ MySQL server not accessible at ${mysqlHost}:${mysqlPort}"
    echo "  💡 Make sure MySQL is running and accessible"
}

if (-not $mysqlCheck) {
    echo ""
    echo "🐳 Quick MySQL setup with Docker:"
    echo "  docker run -d --name mysql-tunnel \"
    echo "    -e MYSQL_ROOT_PASSWORD=root \"
    echo "    -e MYSQL_DATABASE=logs \"
    echo "    -p 3306:3306 \"
    echo "    mysql:8.0"
    echo ""
    Read-Host "Press Enter to continue anyway, or Ctrl+C to exit"
}

# Build relay server if needed
if (-not (Test-Path "bin\tunnel-relay.exe")) {
    echo ""
    echo "🔨 Building relay server..."
    go build -o bin\tunnel-relay.exe cmd\relay\main.go
    if ($LASTEXITCODE -ne 0) {
        echo "❌ Failed to build relay server"
        exit 1
    }
    echo "✅ Relay server built successfully"
}

# Get port from environment or use default
$port = if ($env:RELAY_PORT) { $env:RELAY_PORT } else { "8080" }

echo ""
echo "🌐 Starting relay server on port $port..."
echo "📊 Dashboard URL: http://localhost:$port"
echo "🔐 Login: $env:ADMIN_USERNAME / $env:ADMIN_PASSWORD"
echo "🔌 WebSocket: ws://localhost:$port/ws/agent | ws://localhost:$port/ws/client"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start the relay server
.\bin\tunnel-relay.exe -p $port