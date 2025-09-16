package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	// Test direct connection to SSH server
	fmt.Println("Testing direct connection to 168.231.119.242:22...")

	conn, err := net.DialTimeout("tcp", "168.231.119.242:22", 5*time.Second)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("✅ Successfully connected to SSH server")

	// Read SSH banner
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Failed to read SSH banner: %v\n", err)
		return
	}

	fmt.Printf("SSH Banner: %s", string(buffer[:n]))

	// Send SSH client version
	clientVersion := "SSH-2.0-TestClient\r\n"
	_, err = conn.Write([]byte(clientVersion))
	if err != nil {
		fmt.Printf("Failed to send client version: %v\n", err)
		return
	}

	fmt.Println("✅ SSH handshake initiated successfully")
}
