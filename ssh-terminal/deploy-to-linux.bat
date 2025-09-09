param(
    [string]$ServerIP = "168.231.119.242",
    [string]$ServerUser = "root",
    [string]$ProjectDir = "/opt/goteleport"
)

Write-Host "🚀 Building and deploying GoTeleport with Tunnel Support to Linux server..." -ForegroundColor Green

# Test local build first to catch errors
Write-Host "🔍 Testing local build for errors..." -ForegroundColor Yellow

Write-Host "Testing server build..."
go build -o test-server.exe goteleport-server-db.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Server has compile errors! Fix before deploying." -ForegroundColor Red
    exit 1
}
Remove-Item -Path "test-server.exe" -ErrorAction SilentlyContinue

Write-Host "Testing agent build..."
go build -o test-agent.exe goteleport-agent-db.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Agent has compile errors! Fix before deploying." -ForegroundColor Red
    exit 1
}
Remove-Item -Path "test-agent.exe" -ErrorAction SilentlyContinue

Write-Host "✅ Local builds test passed!" -ForegroundColor Green

# Build binaries for Linux
Write-Host "📦 Building binaries for Linux..." -ForegroundColor Yellow

Write-Host "Building agent for Linux..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o goteleport-agent-db-linux goteleport-agent-db.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to build agent for Linux" -ForegroundColor Red
    exit 1
}

Write-Host "Building server for Linux..."
go build -o goteleport-server-db-linux goteleport-server-db.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to build server for Linux" -ForegroundColor Red
    exit 1
}

# Reset environment
Remove-Item Env:GOOS
Remove-Item Env:GOARCH

Write-Host "✅ Local builds completed successfully" -ForegroundColor Green

Write-Host "📤 Transferring files to Linux server..." -ForegroundColor Yellow

# Create project directory on server
ssh ${ServerUser}@${ServerIP} "mkdir -p ${ProjectDir}"

# Transfer binaries
scp goteleport-agent-db-linux ${ServerUser}@${ServerIP}:${ProjectDir}/goteleport-agent-db
scp goteleport-server-db-linux ${ServerUser}@${ServerIP}:${ProjectDir}/goteleport-server-db

# Transfer config files
scp agent-config-db.json ${ServerUser}@${ServerIP}:${ProjectDir}/
scp server-config-db.json ${ServerUser}@${ServerIP}:${ProjectDir}/

# Transfer startup scripts if they exist
if (Test-Path "start-agent.sh") {
    scp start-agent.sh ${ServerUser}@${ServerIP}:${ProjectDir}/
}
if (Test-Path "start-agent-verbose.sh") {
    scp start-agent-verbose.sh ${ServerUser}@${ServerIP}:${ProjectDir}/
}
if (Test-Path "start-server.sh") {
    scp start-server.sh ${ServerUser}@${ServerIP}:${ProjectDir}/
}
if (Test-Path "stop-services.sh") {
    scp stop-services.sh ${ServerUser}@${ServerIP}:${ProjectDir}/
}

Write-Host "🔧 Setting up permissions on server..." -ForegroundColor Yellow

# Set executable permissions
ssh ${ServerUser}@${ServerIP} "cd ${ProjectDir} && chmod +x goteleport-* *.sh"

Write-Host "✅ Transfer completed successfully" -ForegroundColor Green

Write-Host ""
Write-Host "🎯 Ready to start services on Linux server:" -ForegroundColor Cyan
Write-Host "   ssh ${ServerUser}@${ServerIP}" -ForegroundColor White
Write-Host "   cd ${ProjectDir}" -ForegroundColor White
Write-Host "   ./start-server.sh" -ForegroundColor White
Write-Host "   ./start-agent.sh" -ForegroundColor White

Write-Host ""
Write-Host "📋 Service management commands:" -ForegroundColor Cyan
Write-Host "   Start services: ./start-agent.sh && ./start-server.sh" -ForegroundColor White
Write-Host "   Stop services:  ./stop-services.sh" -ForegroundColor White
Write-Host "   Check logs:     tail -f server.log agent-db.log" -ForegroundColor White

Write-Host ""
Write-Host "🔗 Access points after startup:" -ForegroundColor Cyan
Write-Host "   Server WebSocket: ws://${ServerIP}:8081/ws/client" -ForegroundColor White
Write-Host "   Agent WebSocket:  ws://${ServerIP}:8080/ws/agent" -ForegroundColor White
Write-Host "   🆕 Tunnel Endpoint: ws://${ServerIP}:8081/ws/tunnel (NEW!)" -ForegroundColor Yellow
Write-Host "   MySQL Proxy: ${ServerIP}:3307" -ForegroundColor White
Write-Host "   PostgreSQL Proxy: ${ServerIP}:5435" -ForegroundColor White

Write-Host ""
Write-Host "🔧 To connect from Windows client:" -ForegroundColor Cyan
Write-Host "   Update client-config-clean.json with server_url: ws://${ServerIP}:8081/ws/client" -ForegroundColor White
Write-Host "   Run: .\unified-client.exe client-config-clean.json" -ForegroundColor White
Write-Host "   🆕 Tunnel support: Agent behind NAT now supported!" -ForegroundColor Yellow

Write-Host ""
Write-Host "🔍 For debugging agent issues:" -ForegroundColor Cyan
Write-Host "   ./start-agent-verbose.sh (shows stdout + file logs)" -ForegroundColor White
Write-Host "   tail -f agent-db.log (file logs only)" -ForegroundColor White

# Cleanup local Linux binaries
Write-Host "🧹 Cleaning up local Linux binaries..." -ForegroundColor Yellow
Remove-Item -Path "goteleport-agent-db-linux" -ErrorAction SilentlyContinue
Remove-Item -Path "goteleport-server-db-linux" -ErrorAction SilentlyContinue

Write-Host "🎉 Deployment completed!" -ForegroundColor Green
