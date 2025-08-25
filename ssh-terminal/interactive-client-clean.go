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

type InteractiveClient struct {
	config       *ClientConfig
	conn         *websocket.Conn
	logger       *log.Logger
	clientID     string
	sessionID    string
	connected    bool
	agentList    []Agent
	currentAgent string
}

type ClientConfig struct {
	ServerURL     string `json:"server_url"`
	ClientName    string `json:"client_name"`
	LogFile       string `json:"log_file"`
	AuthToken     string `json:"auth_token"`
	AutoReconnect bool   `json:"auto_reconnect"`
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
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Address  string                 `json:"address"`
	Platform string                 `json:"platform"`
	Status   string                 `json:"status"`
	LastSeen time.Time              `json:"last_seen"`
	Metadata map[string]interface{} `json:"metadata"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: interactive-client.exe <config-file>")
	}

	client, err := NewInteractiveClient(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("🚀 GoTeleport Interactive Client")
	fmt.Println("🔌 Connecting to server...")

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("✅ Connected to server successfully!")
	fmt.Println("📡 Starting interactive shell...")

	client.StartInteractiveShell()
}

func NewInteractiveClient(configFile string) (*InteractiveClient, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Setup logging
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "[CLIENT] ", log.LstdFlags)
	clientID := fmt.Sprintf("client_%d", time.Now().Unix())

	return &InteractiveClient{
		config:    &config,
		logger:    logger,
		clientID:  clientID,
		connected: false,
		agentList: make([]Agent, 0),
	}, nil
}

func (c *InteractiveClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %v", err)
	}

	c.conn = conn
	c.connected = true

	// Send registration
	authMsg := Message{
		Type:      "register",
		ClientID:  c.clientID,
		Data:      c.config.AuthToken,
		Metadata: map[string]interface{}{
			"name": c.config.ClientName,
		},
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(authMsg); err != nil {
		return fmt.Errorf("failed to send auth: %v", err)
	}

	c.logEvent("CLIENT_CONNECT", "Connected to server", c.config.ServerURL)

	// Start message handler
	go c.handleMessages()

	return nil
}

func (c *InteractiveClient) StartInteractiveShell() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║             🌐 GoTeleport Interactive Shell             ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")
	fmt.Println("║ Commands:                                                ║")
	fmt.Println("║   agents      - List available agents                   ║")
	fmt.Println("║   connect <n> - Connect to agent number                 ║")
	fmt.Println("║   disconnect  - Disconnect from current agent           ║")
	fmt.Println("║   status      - Show connection status                  ║")
	fmt.Println("║   help        - Show help                               ║")
	fmt.Println("║   exit        - Exit client                             ║")
	fmt.Println("║                                                          ║")
	fmt.Println("║ When connected: Type any Linux command to execute       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// Auto-refresh agent list
	c.refreshAgentList()

	reader := bufio.NewReader(os.Stdin)

	for c.connected {
		prompt := c.getPrompt()
		fmt.Print(prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			continue
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		c.processCommand(command)
	}
}

func (c *InteractiveClient) getPrompt() string {
	if c.currentAgent != "" {
		return fmt.Sprintf("\033[32m%s@agent\033[0m$ ", c.currentAgent)
	}
	return "\033[36mteleport\033[0m> "
}

func (c *InteractiveClient) processCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	c.logEvent("CLIENT_INPUT", "Command entered", command)

	if c.currentAgent == "" {
		// Client-level commands
		c.handleClientCommand(parts, command)
	} else {
		// Agent commands
		c.executeRemoteCommand(command)
	}
}

func (c *InteractiveClient) handleClientCommand(parts []string, fullCommand string) {
	cmd := parts[0]

	switch cmd {
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
	case "status":
		c.showStatus()
	case "help":
		c.showHelp()
	case "exit", "quit":
		fmt.Println("👋 Disconnecting from server...")
		c.connected = false
	default:
		fmt.Printf("❌ Unknown command: %s\n", cmd)
		fmt.Println("💡 Type 'help' for available commands or 'agents' to list agents")
	}
}

func (c *InteractiveClient) executeRemoteCommand(command string) {
	if command == "disconnect" {
		c.disconnectFromAgent()
		return
	}

	if command == "exit" {
		fmt.Println("💡 Use 'disconnect' to disconnect from agent, or 'exit' at main prompt to quit")
		return
	}

	msg := Message{
		Type:      "command",
		SessionID: c.sessionID,
		AgentID:   c.currentAgent,
		ClientID:  c.clientID,
		Command:   command,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to send command: %v\n", err)
	}
}

func (c *InteractiveClient) listAgents() {
	c.refreshAgentList()
	
	// Wait a moment for response
	time.Sleep(100 * time.Millisecond)
	
	if len(c.agentList) == 0 {
		fmt.Println("📭 No agents available")
		return
	}

	fmt.Println("\n📋 Available Agents:")
	fmt.Println("┌─────┬─────────────────────┬─────────────┬──────────┬──────────────────┐")
	fmt.Printf("│ %-3s │ %-19s │ %-11s │ %-8s │ %-16s │\n", "No", "Agent ID", "Platform", "Status", "Last Seen")
	fmt.Println("├─────┼─────────────────────┼─────────────┼──────────┼──────────────────┤")

	for i, agent := range c.agentList {
		lastSeen := agent.LastSeen.Format("15:04:05")
		fmt.Printf("│ %-3d │ %-19s │ %-11s │ %-8s │ %-16s │\n",
			i+1, agent.ID, agent.Platform, agent.Status, lastSeen)
	}
	fmt.Println("└─────┴─────────────────────┴─────────────┴──────────┴──────────────────┘")
	fmt.Printf("\n💡 Use 'connect <number>' to connect to an agent\n\n")
}

func (c *InteractiveClient) connectToAgent(agentNum int) {
	if agentNum < 1 || agentNum > len(c.agentList) {
		fmt.Printf("❌ Invalid agent number. Available: 1-%d\n", len(c.agentList))
		return
	}

	selectedAgent := c.agentList[agentNum-1]
	fmt.Printf("🔗 Connecting to agent: %s...\n", selectedAgent.ID)

	msg := Message{
		Type:      "connect_agent",
		AgentID:   selectedAgent.ID,
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to connect to agent: %v\n", err)
		return
	}

	// Wait for session creation
	fmt.Println("⏳ Waiting for session...")
}

func (c *InteractiveClient) disconnectFromAgent() {
	if c.currentAgent == "" {
		fmt.Println("❌ Not connected to any agent")
		return
	}

	msg := Message{
		Type:      "disconnect_agent",
		SessionID: c.sessionID,
		AgentID:   c.currentAgent,
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to disconnect: %v\n", err)
	}

	fmt.Printf("🔌 Disconnected from agent: %s\n", c.currentAgent)
	c.currentAgent = ""
	c.sessionID = ""
}

func (c *InteractiveClient) refreshAgentList() {
	msg := Message{
		Type:      "list_agents",
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		fmt.Printf("❌ Failed to request agent list: %v\n", err)
	}
}

func (c *InteractiveClient) showStatus() {
	fmt.Printf("\n📊 Connection Status:\n")
	fmt.Printf("   Server: %s\n", c.config.ServerURL)
	fmt.Printf("   Client ID: %s\n", c.clientID)
	fmt.Printf("   Connected: %v\n", c.connected)
	fmt.Printf("   Current Agent: %s\n", func() string {
		if c.currentAgent == "" {
			return "None"
		}
		return c.currentAgent
	}())
	fmt.Printf("   Session ID: %s\n", func() string {
		if c.sessionID == "" {
			return "None"
		}
		return c.sessionID
	}())
	fmt.Printf("   Available Agents: %d\n\n", len(c.agentList))
}

func (c *InteractiveClient) showHelp() {
	fmt.Println("\n📖 GoTeleport Client Help:")
	fmt.Println("   agents      - List all available agents")
	fmt.Println("   connect <n> - Connect to agent number n")
	fmt.Println("   disconnect  - Disconnect from current agent")
	fmt.Println("   status      - Show connection status")
	fmt.Println("   help        - Show this help")
	fmt.Println("   exit        - Exit client")
	fmt.Println("\n🔗 When connected to an agent:")
	fmt.Println("   Type any Linux command to execute remotely")
	fmt.Println("   disconnect  - Return to main client prompt")
	fmt.Println()
}

func (c *InteractiveClient) handleMessages() {
	for c.connected {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if c.connected {
				fmt.Printf("\n❌ Connection error: %v\n", err)
				c.connected = false
			}
			break
		}

		c.processMessage(&msg)
	}
}

func (c *InteractiveClient) processMessage(msg *Message) {
	switch msg.Type {
	case "agent_list":
		c.handleAgentList(msg)
	case "session_created":
		c.handleSessionCreated(msg)
	case "command_result":
		c.handleCommandOutput(msg)
	case "agent_disconnected":
		c.handleAgentDisconnected(msg)
	case "error":
		fmt.Printf("\n❌ Server error: %s\n", msg.Data)
		fmt.Print(c.getPrompt())
	default:
		c.logEvent("CLIENT_MESSAGE", "Unknown message type", msg.Type)
	}
}

func (c *InteractiveClient) handleAgentList(msg *Message) {
	// Server sends agents in metadata["agents"]
	if msg.Metadata != nil {
		if agentsData, exists := msg.Metadata["agents"]; exists {
			// Convert interface{} to JSON bytes then unmarshal to agents
			if jsonBytes, err := json.Marshal(agentsData); err == nil {
				var agents []Agent
				if err := json.Unmarshal(jsonBytes, &agents); err == nil {
					c.agentList = agents
					c.logEvent("CLIENT_AGENTS", "Received agent list", fmt.Sprintf("Count: %d", len(agents)))
					return
				}
			}
		}
	}
	// Fallback: try Data field as JSON string  
	if msg.Data != "" {
		var agents []Agent
		if err := json.Unmarshal([]byte(msg.Data), &agents); err == nil {
			c.agentList = agents
		}
	}
}

func (c *InteractiveClient) handleSessionCreated(msg *Message) {
	c.sessionID = msg.SessionID
	c.currentAgent = msg.AgentID
	fmt.Printf("\n✅ Connected to agent: %s\n", msg.AgentID)
	fmt.Printf("📋 Session ID: %s\n", msg.SessionID)
	fmt.Println("💡 You can now execute commands. Type 'disconnect' to return.")
	fmt.Print(c.getPrompt())
}

func (c *InteractiveClient) handleCommandOutput(msg *Message) {
	// Agent sends command output in Data field
	if msg.Data != "" {
		fmt.Printf("%s", msg.Data)
	}
	fmt.Print(c.getPrompt())
}

func (c *InteractiveClient) handleAgentDisconnected(msg *Message) {
	fmt.Printf("\n🔌 Agent disconnected: %s\n", msg.AgentID)
	c.currentAgent = ""
	c.sessionID = ""
	fmt.Print(c.getPrompt())
}

func (c *InteractiveClient) logEvent(eventType, description, data string) {
	if c.logger != nil {
		logEntry := fmt.Sprintf("[%s] %s: %s | Data: %s",
			time.Now().Format("2006-01-02 15:04:05"),
			eventType, description, data)
		c.logger.Println(logEntry)
	}
}

func (c *InteractiveClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.connected = false
}
