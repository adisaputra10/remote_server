package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Printf("🔍 Simple Tunnel Test - Raw TCP Connection\n")
	
	// Test basic TCP connection to port 3308
	fmt.Printf("📡 Connecting to localhost:3308...\n")
	
	conn, err := net.DialTimeout("tcp", "localhost:3308", 5*time.Second)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()
	
	fmt.Printf("✅ Connected to tunnel successfully!\n")
	
	// Send simple test data
	testData := []byte("HELLO TUNNEL")
	fmt.Printf("📤 Sending test data: %s\n", string(testData))
	
	n, err := conn.Write(testData)
	if err != nil {
		fmt.Printf("❌ Failed to write: %v\n", err)
		return
	}
	fmt.Printf("✅ Sent %d bytes\n", n)
	
	// Try to read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("⚠️  Read error (expected for non-HTTP data): %v\n", err)
		fmt.Printf("💡 This is normal - we're sending raw data to MySQL proxy\n")
		return
	}
	
	fmt.Printf("✅ Received %d bytes: %s\n", n, string(buffer[:n]))
}
