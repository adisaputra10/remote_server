package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Println("🧪 Simple TCP Test to Port 3308...")

	// Connect to port 3308
	conn, err := net.DialTimeout("tcp", "localhost:3308", 10*time.Second)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()
	
	fmt.Println("✅ Connected to port 3308")

	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// Send some test data
	testData := []byte("Hello, this is a test message")
	n, err := conn.Write(testData)
	if err != nil {
		fmt.Printf("❌ Failed to write: %v\n", err)
		return
	}
	fmt.Printf("✅ Sent %d bytes\n", n)

	// Try to read response
	buffer := make([]byte, 1024)
	n, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("❌ Failed to read: %v\n", err)
		fmt.Printf("   This indicates tunnel is not forwarding data properly\n")
		return
	}
	
	fmt.Printf("✅ Received %d bytes: %s\n", n, string(buffer[:n]))
}
