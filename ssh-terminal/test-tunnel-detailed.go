package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	port := "3308"
	
	fmt.Printf("🧪 Testing MySQL Tunnel Connection...\n")
	
	// Test 1: Check if port is listening
	fmt.Printf("🔍 Step 1: Checking if port %s is listening...\n", port)
	conn, err := net.DialTimeout("tcp", "localhost:"+port, 2*time.Second)
	if err != nil {
		fmt.Printf("❌ Port %s is not listening: %v\n", port, err)
		return
	}
	conn.Close()
	fmt.Printf("✅ Port %s is listening\n", port)
	
	// Test 2: Try raw TCP connection with timeout
	fmt.Printf("🔍 Step 2: Testing raw TCP connection...\n")
	conn, err = net.DialTimeout("tcp", "localhost:"+port, 5*time.Second)
	if err != nil {
		fmt.Printf("❌ Raw TCP connection failed: %v\n", err)
		return
	}
	
	// Send MySQL handshake initial packet (basic)
	fmt.Printf("📤 Sending test data to tunnel...\n")
	testData := []byte{0x0a, 0x00, 0x00, 0x00} // Simple test packet
	n, err := conn.Write(testData)
	if err != nil {
		fmt.Printf("❌ Failed to write test data: %v\n", err)
		conn.Close()
		return
	}
	fmt.Printf("✅ Sent %d bytes to tunnel\n", n)
	
	// Try to read response with timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		conn.Close()
		return
	}
	fmt.Printf("✅ Received %d bytes from tunnel: %x\n", n, buffer[:n])
	conn.Close()
	
	// Test 3: Try MySQL driver connection
	fmt.Printf("🔍 Step 3: Testing MySQL driver connection...\n")
	dsn := fmt.Sprintf("root:rootpassword@tcp(localhost:%s)/mysql?timeout=10s", port)
	fmt.Printf("🔌 DSN: %s\n", dsn)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to create DB connection: %v\n", err)
		return
	}
	defer db.Close()
	
	// Test connection with ping
	fmt.Printf("📡 Pinging database...\n")
	err = db.Ping()
	if err != nil {
		fmt.Printf("❌ Database ping failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ MySQL connection successful!\n")
	
	// Test simple query
	fmt.Printf("🔍 Testing simple query...\n")
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ MySQL version: %s\n", version)
	fmt.Printf("🎉 All tests passed!\n")
}
