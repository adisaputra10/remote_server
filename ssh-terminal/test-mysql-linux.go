package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-mysql-linux.go <linux-server-ip>")
		fmt.Println("Example: go run test-mysql-linux.go 192.168.1.100")
		os.Exit(1)
	}

	serverIP := os.Args[1]
	
	fmt.Println("🧪 Testing MySQL Database Proxy on Linux Agent...")
	fmt.Printf("🔌 Server IP: %s\n", serverIP)
	fmt.Println("=" * 50)

	// Test configurations
	configs := []struct {
		name   string
		port   int
		dsn    string
	}{
		{
			name: "MySQL Proxy Port 3307",
			port: 3307,
			dsn:  fmt.Sprintf("root:rootpassword@tcp(%s:3307)/mysql?timeout=5s", serverIP),
		},
		{
			name: "MySQL Proxy Port 3308", 
			port: 3308,
			dsn:  fmt.Sprintf("root:rootpassword@tcp(%s:3308)/mysql?timeout=5s", serverIP),
		},
		{
			name: "Direct MySQL Port 3306",
			port: 3306,
			dsn:  fmt.Sprintf("root:rootpassword@tcp(%s:3306)/mysql?timeout=5s", serverIP),
		},
	}

	for _, config := range configs {
		fmt.Printf("\n🔌 Testing %s...\n", config.name)
		fmt.Printf("   DSN: %s\n", config.dsn)
		
		db, err := sql.Open("mysql", config.dsn)
		if err != nil {
			fmt.Printf("❌ Failed to open connection: %v\n", err)
			continue
		}
		defer db.Close()

		// Set connection timeout
		db.SetConnMaxLifetime(5 * time.Second)
		
		err = db.Ping()
		if err != nil {
			fmt.Printf("❌ Failed to ping database: %v\n", err)
		} else {
			fmt.Printf("✅ Successfully connected!\n")
			
			// Test a simple query
			var version string
			err = db.QueryRow("SELECT VERSION()").Scan(&version)
			if err != nil {
				fmt.Printf("⚠️  Connected but query failed: %v\n", err)
			} else {
				fmt.Printf("📊 MySQL Version: %s\n", version)
			}
		}
	}

	fmt.Println("\n💡 Troubleshooting Tips:")
	fmt.Println("1. Verify goteleport-agent-db is running on Linux server")
	fmt.Println("2. Check agent-config-db.json proxy configuration")
	fmt.Println("3. Ensure MySQL server is running and accessible")
	fmt.Println("4. Check firewall rules on Linux server")
	fmt.Println("5. Verify credentials: root/rootpassword")
}
