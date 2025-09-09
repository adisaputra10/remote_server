package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run check-agent-connection.go <linux-server-ip>")
		fmt.Println("Example: go run check-agent-connection.go 192.168.1.100")
		os.Exit(1)
	}

	serverIP := os.Args[1]
	
	fmt.Printf("🔍 Checking connection to GoTeleport Agent on Linux server: %s\n", serverIP)
	fmt.Println("=" * 60)

	// Check different ports
	ports := []int{8080, 3307, 3308, 5435}
	
	for _, port := range ports {
		fmt.Printf("\n🔌 Testing connection to %s:%d\n", serverIP, port)
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", serverIP, port), 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Port %d: %v\n", port, err)
		} else {
			fmt.Printf("✅ Port %d: Connected successfully\n", port)
			conn.Close()
		}
	}

	fmt.Println("\n📋 Diagnostic Information:")
	fmt.Println("Port 8080: GoTeleport Agent main port")
	fmt.Println("Port 3307: MySQL proxy (from agent-config-db.json)")
	fmt.Println("Port 3308: Alternative MySQL proxy port")
	fmt.Println("Port 5435: PostgreSQL proxy")

	fmt.Println("\n🛠️ Troubleshooting Steps:")
	fmt.Println("1. Make sure goteleport-agent-db is running on Linux server")
	fmt.Println("2. Check firewall rules on Linux server")
	fmt.Println("3. Verify agent-config-db.json configuration")
	fmt.Println("4. Check MySQL server is running on Linux server")
	fmt.Println("5. Update unified-client.go with correct server IP")
}
