package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type ClientConfig struct {
	ServerURL    string `json:"server_url"`
	ClientName   string `json:"client_name"`
	LogFile      string `json:"log_file"`
	AuthToken    string `json:"auth_token"`
	AutoReconnect bool   `json:"auto_reconnect"`
}

type Message struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type SimpleClient struct {
	config *ClientConfig
	conn   *websocket.Conn
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: client-demo.exe <config-file>")
	}

	client, err := NewSimpleClient(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.Run()
}

func NewSimpleClient(configFile string) (*SimpleClient, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	return &SimpleClient{config: &config}, nil
}

func (c *SimpleClient) Run() {
	fmt.Println("🚀 GoTeleport Client Demo")
	fmt.Printf("📡 Server: %s\n", c.config.ServerURL)
	fmt.Printf("👤 Client: %s\n", c.config.ClientName)
	fmt.Println()

	// Connect to server
	if err := c.connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Setup signal handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Start message handler
	go c.handleMessages()

	// Interactive shell
	go c.interactiveShell()

	// Wait for interrupt
	<-interrupt
	fmt.Println("\n👋 Disconnecting...")
	c.conn.Close()
}

func (c *SimpleClient) connect() error {
	fmt.Printf("🔗 Connecting to %s...\n", c.config.ServerURL)

	conn, _, err := websocket.DefaultDialer.Dial(c.config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}

	c.conn = conn

	// Register client
	registerMsg := Message{
		Type: "register",
		From: c.config.ClientName,
		Data: map[string]interface{}{
			"token":     c.config.AuthToken,
			"client_id": c.config.ClientName,
			"type":      "client",
		},
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(registerMsg); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	fmt.Println("✅ Connected to server!")
	return nil
}

func (c *SimpleClient) handleMessages() {
	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("❌ Read error: %v\n", err)
			return
		}

		switch msg.Type {
		case "command_result":
			c.handleCommandResult(msg)
		case "agent_list":
			c.handleAgentList(msg)
		case "error":
			fmt.Printf("❌ Error: %v\n", msg.Data["message"])
		default:
			fmt.Printf("📨 Message from %s: %s\n", msg.From, msg.Type)
		}
	}
}

func (c *SimpleClient) handleCommandResult(msg Message) {
	fmt.Println("📤 Command Result:")
	fmt.Println("─────────────────")
	if output, ok := msg.Data["output"].(string); ok {
		fmt.Print(output)
	}
	if stderr, ok := msg.Data["stderr"].(string); ok && stderr != "" {
		fmt.Printf("stderr: %s", stderr)
	}
	fmt.Println("─────────────────")
	fmt.Print("remote> ")
}

func (c *SimpleClient) handleAgentList(msg Message) {
	fmt.Println("\n📋 Available Agents:")
	if agents, ok := msg.Data["agents"].([]interface{}); ok {
		for i, agent := range agents {
			if agentMap, ok := agent.(map[string]interface{}); ok {
				name := agentMap["name"]
				id := agentMap["id"]
				fmt.Printf("  %d. %s (ID: %s)\n", i+1, name, id)
			}
		}
	}
	fmt.Println()
}

func (c *SimpleClient) interactiveShell() {
	time.Sleep(1 * time.Second) // Wait for connection
	
	fmt.Println("🖥️  GoTeleport Remote Shell")
	fmt.Println("Commands:")
	fmt.Println("  list agents    - Show available agents")
	fmt.Println("  connect <id>   - Connect to agent")
	fmt.Println("  <command>      - Execute command on connected agent")
	fmt.Println("  exit           - Exit client")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	currentAgent := ""

	for {
		if currentAgent != "" {
			fmt.Printf("remote@%s> ", currentAgent)
		} else {
			fmt.Print("goteleport> ")
		}

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Input error: %v\n", err)
			continue
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		if command == "exit" {
			return
		}

		if command == "list agents" {
			c.sendMessage(Message{
				Type: "list_agents",
				From: c.config.ClientName,
				Data: map[string]interface{}{},
				Timestamp: time.Now(),
			})
			continue
		}

		if strings.HasPrefix(command, "connect ") {
			agentId := strings.TrimPrefix(command, "connect ")
			currentAgent = agentId
			fmt.Printf("🔗 Connected to agent: %s\n", agentId)
			fmt.Println("You can now execute commands on the remote agent.")
			continue
		}

		if currentAgent == "" {
			fmt.Println("❌ Please connect to an agent first using 'connect <agent_id>'")
			continue
		}

		// Send command to agent
		c.sendMessage(Message{
			Type: "execute_command",
			From: c.config.ClientName,
			To:   currentAgent,
			Data: map[string]interface{}{
				"command": command,
			},
			Timestamp: time.Now(),
		})
	}
}

func (c *SimpleClient) sendMessage(msg Message) {
	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Send error: %v\n", err)
	}
}
