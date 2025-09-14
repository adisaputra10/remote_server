# Start SSH Tunnel Relay Server with Environment Configuration

Write-Host "Starting SSH Tunnel Relay Server..." -ForegroundColor Green
Write-Host ""

# Load environment variables
Write-Host "Loading environment variables from .env..." -ForegroundColor Cyan
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match '^([^#].*)=(.*)$') {
            $name = $matches[1]
            $value = $matches[2]
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
            Write-Host "  OK: $name=$value" -ForegroundColor Green
        }
    }
    Write-Host "Environment variables loaded successfully" -ForegroundColor Green
    Write-Host ""
} else {
    Write-Host "Error: .env file not found" -ForegroundColor Red
    Write-Host "Please create .env file from env.template" -ForegroundColor Yellow
    exit 1
}

# Check if MySQL is accessible
Write-Host "Checking MySQL connection..." -ForegroundColor Cyan
$mysqlCheck = $false

try {
    # Test MySQL connection using environment variables
    $mysqlHost = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
    $mysqlPort = if ($env:DB_PORT) { [int]$env:DB_PORT } else { 3306 }
    
    $connection = New-Object System.Net.Sockets.TcpClient
    $connection.Connect($mysqlHost, $mysqlPort)
    $connection.Close()
    $mysqlCheck = $true
    Write-Host "  OK: MySQL server is accessible at ${mysqlHost}:${mysqlPort}" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: MySQL server not accessible at ${mysqlHost}:${mysqlPort}" -ForegroundColor Red
    Write-Host "  Make sure MySQL is running and accessible" -ForegroundColor Yellow
}

if (-not $mysqlCheck) {
    Write-Host ""
    Write-Host "Quick MySQL setup with Docker:" -ForegroundColor Yellow
    Write-Host "docker run --name mysql8 -e MYSQL_ROOT_PASSWORD=rootpassword -e MYSQL_DATABASE=tunnel -p 3306:3306 -d mysql:8" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Press Enter to continue anyway or Ctrl+C to exit..."
    Read-Host
}

# Build the relay if needed
Write-Host "Building relay server..." -ForegroundColor Cyan
$buildResult = & go build -o bin/relay.exe cmd/relay/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}
Write-Host "Build successful" -ForegroundColor Green
Write-Host ""

# Start the relay server
Write-Host "Starting relay server..." -ForegroundColor Cyan
Write-Host "Dashboard will be available at: http://localhost:8080" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

& ./bin/relay.exe