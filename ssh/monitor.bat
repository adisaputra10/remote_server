# SSH Tunnel System Monitor & Debug Tool

echo "🔍 SSH Tunnel System Monitor"
echo "============================="
echo ""

function Show-SystemStatus {
    echo "📊 System Status Report"
    echo "======================="
    echo ""
    
    # Check processes
    echo "🔄 Running Processes:"
    $tunnelProcs = Get-Process | Where-Object { $_.ProcessName -match "relay|agent|client" } | 
                  Select-Object ProcessName, Id, CPU, WorkingSet, StartTime
    
    if ($tunnelProcs) {
        $tunnelProcs | Format-Table -AutoSize
    } else {
        echo "  No tunnel processes running"
    }
    echo ""
    
    # Check ports
    echo "🌐 Network Ports:"
    $ports = @(8080, 8081, 22)
    foreach ($port in $ports) {
        $listener = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
        if ($listener) {
            echo "  Port $port`: ✅ Listening (PID: $($listener.OwningProcess))"
        } else {
            echo "  Port $port`: ❌ Not listening"
        }
    }
    echo ""
    
    # Check log files
    echo "📝 Log Files:"
    $logDir = "logs"
    if (Test-Path $logDir) {
        $logFiles = Get-ChildItem $logDir -Filter "*.log" | Sort-Object LastWriteTime -Descending
        foreach ($log in $logFiles) {
            $size = [math]::Round($log.Length / 1KB, 2)
            $lastWrite = $log.LastWriteTime.ToString("yyyy-MM-dd HH:mm:ss")
            echo "  $($log.Name): $size KB (Last: $lastWrite)"
        }
    } else {
        echo "  No logs directory found"
    }
    echo ""
    
    # Check database connection
    echo "🗄️  Database Status:"
    & ".\load-env.bat"
    
    $dbHost = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
    $dbPort = if ($env:DB_PORT) { $env:DB_PORT } else { "3306" }
    
    try {
        $connection = New-Object System.Net.Sockets.TcpClient
        $connection.Connect($dbHost, $dbPort)
        $connection.Close()
        echo "  MySQL ${dbHost}:${dbPort}: ✅ Reachable"
    } catch {
        echo "  MySQL ${dbHost}:${dbPort}: ❌ Unreachable"
    }
    echo ""
}

function Show-RecentLogs {
    param($component, $lines = 20)
    
    $logFile = "logs\$component.log"
    if (Test-Path $logFile) {
        echo "📄 Recent $component logs (last $lines lines):"
        echo "==============================================" 
        Get-Content $logFile -Tail $lines
        echo ""
    } else {
        echo "❌ Log file not found: $logFile"
    }
}

function Test-DatabaseConnection {
    echo "🔧 Testing Database Connection..."
    echo ""
    
    & ".\load-env.bat"
    
    $dbUser = if ($env:DB_USER) { $env:DB_USER } else { "root" }
    $dbPassword = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "root" }
    $dbHost = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
    $dbPort = if ($env:DB_PORT) { $env:DB_PORT } else { "3306" }
    $dbName = if ($env:DB_NAME) { $env:DB_NAME } else { "logs" }
    
    echo "Configuration:"
    echo "  Host: $dbHost"
    echo "  Port: $dbPort"
    echo "  User: $dbUser"
    echo "  Database: $dbName"
    echo ""
    
    # Create test script
    $testScript = @"
package main

import (
    "database/sql"
    "fmt"
    "os"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", 
        os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5])
    
    fmt.Printf("Testing connection: %s:***@tcp(%s:%s)/%s\n", os.Args[1], os.Args[3], os.Args[4], os.Args[5])
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        fmt.Printf("❌ Failed to open connection: %v\n", err)
        os.Exit(1)
    }
    defer db.Close()
    
    err = db.Ping()
    if err != nil {
        fmt.Printf("❌ Failed to ping database: %v\n", err)
        os.Exit(1)
    }
    
    // Test tables
    rows, err := db.Query("SHOW TABLES")
    if err != nil {
        fmt.Printf("❌ Failed to show tables: %v\n", err)
        os.Exit(1)
    }
    defer rows.Close()
    
    fmt.Println("✅ Database connection successful!")
    fmt.Println("📋 Tables:")
    for rows.Next() {
        var tableName string
        rows.Scan(&tableName)
        fmt.Printf("  - %s\n", tableName)
    }
}
"@
    
    $testScript | Out-File -FilePath "test-db-debug.go" -Encoding UTF8
    go run test-db-debug.go $dbUser $dbPassword $dbHost $dbPort $dbName
    Remove-Item "test-db-debug.go" -ErrorAction SilentlyContinue
    echo ""
}

function Test-WebDashboard {
    echo "🌐 Testing Web Dashboard..."
    echo ""
    
    $dashboardUrl = "http://localhost:8080"
    
    try {
        $response = Invoke-WebRequest -Uri $dashboardUrl -TimeoutSec 5 -UseBasicParsing
        echo "✅ Dashboard accessible at $dashboardUrl"
        echo "   Status: $($response.StatusCode) $($response.StatusDescription)"
    } catch {
        echo "❌ Dashboard not accessible at $dashboardUrl"
        echo "   Error: $($_.Exception.Message)"
    }
    echo ""
}

function Watch-Logs {
    param($component)
    
    $logFile = "logs\$component.log"
    if (!(Test-Path $logFile)) {
        echo "❌ Log file not found: $logFile"
        echo "💡 Start the $component component first"
        return
    }
    
    echo "👁️  Watching $component logs (Ctrl+C to stop)..."
    echo "=============================================="
    echo ""
    
    # Show last 10 lines first
    Get-Content $logFile -Tail 10
    echo ""
    echo "--- Live updates ---"
    
    try {
        Get-Content $logFile -Wait -Tail 0
    } catch {
        echo "🛑 Log watching stopped"
    }
}

# Main menu
while ($true) {
    echo "🔍 SSH Tunnel Monitor & Debug"
    echo "=============================="
    echo ""
    echo "1. Show System Status"
    echo "2. View Recent Logs (Relay)"
    echo "3. View Recent Logs (Agent)"
    echo "4. View Recent Logs (Client)"
    echo "5. Test Database Connection"
    echo "6. Test Web Dashboard"
    echo "7. Watch Live Logs (Relay)"
    echo "8. Watch Live Logs (Agent)"
    echo "9. Watch Live Logs (Client)"
    echo "10. Kill All Tunnel Processes"
    echo "0. Exit"
    echo ""
    
    $choice = Read-Host "Select option"
    
    switch ($choice) {
        "1" { Show-SystemStatus }
        "2" { Show-RecentLogs "relay" }
        "3" { Show-RecentLogs "agent" }
        "4" { Show-RecentLogs "client" }
        "5" { Test-DatabaseConnection }
        "6" { Test-WebDashboard }
        "7" { Watch-Logs "relay" }
        "8" { Watch-Logs "agent" }
        "9" { Watch-Logs "client" }
        "10" {
            echo "🛑 Killing all tunnel processes..."
            Get-Process | Where-Object { $_.ProcessName -match "relay|agent|client" } | Stop-Process -Force
            echo "✅ All processes killed"
        }
        "0" { 
            echo "👋 Goodbye!"
            exit 0 
        }
        default { 
            echo "❌ Invalid option"
            Start-Sleep 1
        }
    }
    
    echo ""
    Read-Host "Press Enter to continue"
    Clear-Host
}