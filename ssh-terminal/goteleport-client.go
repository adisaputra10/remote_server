package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type GoTeleportClient struct {
	config     *ClientConfig
	conn       *websocket.Conn
	logger     *log.Logger
	clientID   string
	sessionID  string
	connected  bool
}

type ClientConfig struct {
	ServerURL   string `json:"server_url"`
	ClientName  string `json:"client_name"`
	LogFile     string `json:"log_file"`
	AuthToken   string `json:"auth_token"`
	AutoReconnect bool `json:"auto_reconnect"`
}

type Message struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Command   string                 `json:"command,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type Agent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Platform    string                 `json:"platform"`
	Status      string                 `json:"status"`
	LastSeen    time.Time              `json:"last_seen"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: goteleport-client.exe <config-file>")
	}

	client, err := NewGoTeleportClient(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.Start()
}

func NewGoTeleportClient(configFile string) (*GoTeleportClient, error) {
	// Read config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Setup logger
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)

	client := &GoTeleportClient{
		config: &config,
		logger: logger,
	}

	return client, nil
}

func (c *GoTeleportClient) Start() {
	c.logEvent("CLIENT_START", "GoTeleport Client starting", c.config.ClientName)

	for {
		if err := c.connect(); err != nil {
			c.logEvent("ERROR", "Connection failed", err.Error())
			fmt.Printf("❌ Connection failed: %v\n", err)
			
			if c.config.AutoReconnect {
				fmt.Println("🔄 Retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			} else {
				break
			}
		}

		if c.config.AutoReconnect {
			fmt.Println("🔄 Connection lost, retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
}

func (c *GoTeleportClient) connect() error {
	fmt.Printf("🔗 Connecting to server: %s\n", c.config.ServerURL)

	// Connect to server
	conn, _, err := websocket.DefaultDialer.Dial(c.config.ServerURL+"/ws/client", nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	c.conn = conn
	c.connected = true

	// Register with server
	if err := c.register(); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	fmt.Printf("✅ Connected as: %s\n", c.config.ClientName)
	c.logEvent("CLIENT_CONNECT", "Connected to server", c.config.ServerURL)

	// Start message handler
	go c.handleMessages()

	// Start interactive shell
	c.interactiveShell()

	return nil
}

func (c *GoTeleportClient) register() error {
	regMsg := Message{
		Type: "register",
		Metadata: map[string]interface{}{
			"name":       c.config.ClientName,
			"auth_token": c.config.AuthToken,
		},
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(regMsg); err != nil {
		return err
	}

	// Wait for registration response
	var response Message
	if err := c.conn.ReadJSON(&response); err != nil {
		return err
	}

	if response.Type != "registered" {
		return fmt.Errorf("registration failed: %s", response.Type)
	}

	c.clientID = response.ClientID
	return nil
}

func (c *GoTeleportClient) handleMessages() {
	for c.connected {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			c.logEvent("CLIENT_DISCONNECT", "Disconnected from server", err.Error())
			c.connected = false
			return
		}

		switch msg.Type {
		case "agent_list":
			c.handleAgentList(&msg)
		case "session_created":
			c.handleSessionCreated(&msg)
		case "command_result":
			c.handleCommandResult(&msg)
		default:
			c.logEvent("UNKNOWN_MSG", "Unknown message type", msg.Type)
		}
	}
}

func (c *GoTeleportClient) handleAgentList(msg *Message) {
	if agents, ok := msg.Metadata["agents"].([]interface{}); ok {
		fmt.Println("\n📋 Available Agents:")
		fmt.Println("┌─────┬─────────────────────┬─────────────┬──────────┬─────────────────────┐")
		fmt.Printf("│ %-3s │ %-19s │ %-11s │ %-8s │ %-19s │\n", "No", "Name", "Platform", "Status", "Last Seen")
		fmt.Println("├─────┼─────────────────────┼─────────────┼──────────┼─────────────────────┤")

		for i, agentData := range agents {
			if agentMap, ok := agentData.(map[string]interface{}); ok {
				name := c.getString(agentMap, "name")
				platform := c.getString(agentMap, "platform")
				status := c.getString(agentMap, "status")
				lastSeen := c.getString(agentMap, "last_seen")

				// Parse and format time
				if t, err := time.Parse(time.RFC3339, lastSeen); err == nil {
					lastSeen = t.Format("15:04:05")
				}

				statusIcon := "🔴"
				if status == "online" {
					statusIcon = "🟢"
				}

				fmt.Printf("│ %-3d │ %-19s │ %-11s │ %s %-6s │ %-19s │\n", 
					i+1, c.truncate(name, 19), c.truncate(platform, 11), 
					statusIcon, status, lastSeen)
			}
		}
		fmt.Println("└─────┴─────────────────────┴─────────────┴──────────┴─────────────────────┘")
	}
}

func (c *GoTeleportClient) handleSessionCreated(msg *Message) {
	c.sessionID = msg.SessionID
	fmt.Printf("\n✅ Session created: %s\n", msg.SessionID)
	fmt.Printf("🎯 Connected to agent: %s\n", msg.AgentID)
	fmt.Println("💡 You can now execute commands. Type 'disconnect' to end session.")
	fmt.Println()
}

func (c *GoTeleportClient) handleCommandResult(msg *Message) {
	fmt.Printf("\n📤 Command Output:\n")
	fmt.Println("─────────────────────────────────────────")
	fmt.Print(msg.Data)
	fmt.Println("─────────────────────────────────────────")
}

func (c *GoTeleportClient) interactiveShell() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║                 GoTeleport Client                        ║")
	fmt.Println("║              Remote Command Interface                    ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")
	fmt.Println("║ Commands:                                                ║")
	fmt.Println("║   agents      - List available agents                   ║")
	fmt.Println("║   connect <n> - Connect to agent number                 ║")
	fmt.Println("║   disconnect  - Disconnect from agent                   ║")
	fmt.Println("║   help        - Show this help                          ║")
	fmt.Println("║   exit        - Exit client                             ║")
	fmt.Println("║                                                          ║")
	fmt.Println("║ When connected to agent, type any command to execute    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	reader := bufio.NewReader(os.Stdin)

	for c.connected {
		if c.sessionID != "" {
			fmt.Printf("agent@%s$ ", c.sessionID[:8])
		} else {
			fmt.Printf("teleport> ")
		}

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		c.logEvent("CLIENT_COMMAND", "Command entered", command)

		if c.sessionID == "" {
			// Not connected to agent, handle client commands
			c.handleClientCommand(command)
		} else {
			// Connected to agent, send command
			if command == "disconnect" {
				c.disconnect()
			} else {
				c.sendCommand(command)
			}
		}
	}
}

func (c *GoTeleportClient) handleClientCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "agents":
		c.listAgents()
	case "connect":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: connect <agent_number>")
			return
		}
		if num, err := strconv.Atoi(parts[1]); err == nil {
			c.connectToAgent(num)
		} else {
			fmt.Println("❌ Invalid agent number")
		}
	case "help":
		c.showHelp()
	case "exit":
		fmt.Println("👋 Goodbye!")
		c.connected = false
	default:
		fmt.Printf("❌ Unknown command: %s. Type 'help' for available commands.\n", parts[0])
	}
}

func (c *GoTeleportClient) listAgents() {
	msg := Message{
		Type:      "list_agents",
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to send message: %v\n", err)
	}
}

func (c *GoTeleportClient) connectToAgent(agentNum int) {
	// This is a simplified version - in real implementation,
	// we would need to store the agent list and map numbers to IDs
	fmt.Printf("🔗 Attempting to connect to agent #%d...\n", agentNum)
	
	// For now, we'll use a placeholder agent ID
	agentID := fmt.Sprintf("agent_%d", agentNum)
	
	msg := Message{
		Type:      "connect_agent",
		ClientID:  c.clientID,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to connect to agent: %v\n", err)
	}
}

func (c *GoTeleportClient) sendCommand(command string) {
	msg := Message{
		Type:      "command",
		SessionID: c.sessionID,
		ClientID:  c.clientID,
		Command:   command,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to send command: %v\n", err)
	}

	c.logEvent("COMMAND_SENT", "Command sent to agent", command)
}

func (c *GoTeleportClient) disconnect() {
	if c.sessionID == "" {
		return
	}

	msg := Message{
		Type:      "disconnect",
		SessionID: c.sessionID,
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}

	c.conn.WriteJSON(msg)
	c.sessionID = ""
	
	fmt.Println("🔌 Disconnected from agent")
	c.logEvent("SESSION_DISCONNECT", "Disconnected from agent", "")
}

func (c *GoTeleportClient) showHelp() {
	fmt.Println("\n📖 GoTeleport Client Help")
	fmt.Println("═══════════════════════════")
	fmt.Println("Available commands:")
	fmt.Println("  agents      - List all available agents")
	fmt.Println("  connect <n> - Connect to agent number <n>")
	fmt.Println("  disconnect  - Disconnect from current agent")
	fmt.Println("  help        - Show this help message")
	fmt.Println("  exit        - Exit the client")
	fmt.Println()
	fmt.Println("When connected to an agent:")
	fmt.Println("  - Type any shell command to execute on the remote agent")
	fmt.Println("  - Commands are logged and results are displayed")
	fmt.Println("  - Type 'disconnect' to end the session")
	fmt.Println()
}

func (c *GoTeleportClient) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func (c *GoTeleportClient) truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func (c *GoTeleportClient) logEvent(eventType, description, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "client"
	}

	logEntry := fmt.Sprintf("[%s] [%s] User: %s | Client: %s | Event: %s | Details: %s",
		timestamp, eventType, user, c.config.ClientName, description, details)

	if c.logger != nil {
		c.logger.Println(logEntry)
	}

	// Print important events to stdout
	if eventType == "CLIENT_START" || eventType == "CLIENT_CONNECT" || eventType == "ERROR" {
		fmt.Printf("📝 %s\n", logEntry)
	}
}
