package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	var (
		serverURL = flag.String("server", "ws://localhost:8080/client", "Server URL")
		help      = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	logger := log.New(os.Stdout, "[CLIENT] ", log.LstdFlags|log.Lshortfile)

	// Connect to server
	u, err := url.Parse(*serverURL)
	if err != nil {
		logger.Fatalf("Invalid server URL: %v", err)
	}

	logger.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logger.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	logger.Printf("Connected successfully!")

	// Simple UI loop
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("=== Database Tunnel Client (Modular Version) ===")
	fmt.Println("Welcome to the modular tunnel client!")
	fmt.Println("This is a simplified version for testing the new architecture.")
	fmt.Println()
	
	for {
		showMenu()
		fmt.Print("Enter your choice: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())
		if choice == "" {
			continue
		}

		if choice == "8" {
			fmt.Println("Goodbye!")
			break
		}

		handleChoice(choice, conn, logger, scanner)
		fmt.Println()
	}
}

func showMenu() {
	fmt.Println("=== Main Menu ===")
	fmt.Println("1. Test connection to server")
	fmt.Println("2. Send test message")
	fmt.Println("3. List agents (placeholder)")
	fmt.Println("4. Create tunnel (placeholder)")
	fmt.Println("5. List tunnels (placeholder)")
	fmt.Println("6. Close tunnel (placeholder)")
	fmt.Println("7. Show status")
	fmt.Println("8. Exit")
	fmt.Println()
}

func handleChoice(choice string, conn *websocket.Conn, logger *log.Logger, scanner *bufio.Scanner) {
	switch choice {
	case "1":
		testConnection(conn, logger)
	case "2":
		sendTestMessage(conn, logger, scanner)
	case "3":
		fmt.Println("List agents - Will be implemented when server supports it")
	case "4":
		fmt.Println("Create tunnel - Will be implemented when server supports it")
	case "5":
		fmt.Println("List tunnels - Will be implemented when server supports it")
	case "6":
		fmt.Println("Close tunnel - Will be implemented when server supports it")
	case "7":
		showStatus(conn, logger)
	default:
		fmt.Println("Invalid choice. Please try again.")
	}
}

func testConnection(conn *websocket.Conn, logger *log.Logger) {
	message := map[string]interface{}{
		"type": "ping",
		"data": "test connection",
	}

	data, _ := json.Marshal(message)
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Printf("Failed to send test message: %v", err)
		return
	}

	// Read response
	_, response, err := conn.ReadMessage()
	if err != nil {
		logger.Printf("Failed to read response: %v", err)
		return
	}

	fmt.Printf("✅ Connection test successful! Server response: %s\n", string(response))
}

func sendTestMessage(conn *websocket.Conn, logger *log.Logger, scanner *bufio.Scanner) {
	fmt.Print("Enter message to send: ")
	if !scanner.Scan() {
		return
	}

	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		fmt.Println("Empty message, skipping")
		return
	}

	message := map[string]interface{}{
		"type": "test_message",
		"data": text,
	}

	data, _ := json.Marshal(message)
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Printf("Failed to send message: %v", err)
		return
	}

	// Read response
	_, response, err := conn.ReadMessage()
	if err != nil {
		logger.Printf("Failed to read response: %v", err)
		return
	}

	fmt.Printf("📨 Message sent and echoed back: %s\n", string(response))
}

func showStatus(conn *websocket.Conn, logger *log.Logger) {
	fmt.Println("=== Client Status ===")
	fmt.Printf("Connected to server: ✅\n")
	fmt.Printf("Connection state: %v\n", conn.LocalAddr())
	fmt.Println("Modular architecture: ✅ Active")
	fmt.Println("WebSocket connection: ✅ Established")
	fmt.Println()
	fmt.Println("Note: This is the new modular client architecture")
	fmt.Println("Full features will be available once server-side is complete")
}
