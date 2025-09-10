package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type UnifiedClient struct {
	config        *ClientConfig
	conn          *websocket.Conn
	logger        *log.Logger
	clientID      string
	sessionID     string
	connected     bool
	agentList     []Agent
	currentAgent  string
	selectedAgent string
	portForwards  map[string]*UnifiedPortForward
	mutex         sync.RWMutex
	mode          string // "remote" or "port_forward"
}

type ClientConfig struct {
	ServerURL     string `json:"server_url"`
	ClientName    string `json:"client_name"`
	LogFile       string `json:"log_file"`
	Username      string `json:"username"`
	Password      string `json:"password"`
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

type UnifiedPortForward struct {
	LocalPort  int
	AgentID    string
	TargetHost string
	TargetPort int
	Listener   net.Listener
	Active     bool
	Client     *UnifiedClient
}

type DatabaseCommand struct {
	ID        int                    `json:"id"`
	SessionID string                 `json:"session_id"`
	AgentID   string                 `json:"agent_id"`
	Command   string                 `json:"command"`
	Protocol  string                 `json:"protocol"`
	ClientIP  string                 `json:"client_ip"`
	ProxyName string                 `json:"proxy_name"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	CreatedAt time.Time              `json:"created_at"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: unified-client.exe <config-file>")
	}

	client, err := NewUnifiedClient(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("🚀 GoTeleport Unified Client")
	fmt.Println("🔌 Connecting to server...")

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("✅ Connected to server successfully!")
	client.StartMainMenu()
}

func NewUnifiedClient(configFile string) (*UnifiedClient, error) {
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

	logger := log.New(logFile, "[UNIFIED_CLIENT] ", log.LstdFlags)
	clientID := fmt.Sprintf("client_%d", time.Now().Unix())

	return &UnifiedClient{
		config:       &config,
		logger:       logger,
		clientID:     clientID,
		portForwards: make(map[string]*UnifiedPortForward),
		mode:         "menu",
	}, nil
}

func (c *UnifiedClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	c.conn = conn
	c.connected = true

	// Register with server
	regMsg := Message{
		Type:     "register",
		ClientID: c.clientID,
		Data:     c.config.AuthToken,
		Metadata: map[string]interface{}{
			"name":     c.config.ClientName,
			"username": c.config.Username,
			"password": c.config.Password,
		},
		Timestamp: time.Now(),
	}

	if err := conn.WriteJSON(regMsg); err != nil {
		return fmt.Errorf("failed to send registration: %v", err)
	}

	// Start message handler
	go c.handleMessages()

	// Wait for registration response
	time.Sleep(1 * time.Second)
	return nil
}

func (c *UnifiedClient) handleMessages() {
	for c.connected {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			if c.connected {
				c.logger.Printf("Error reading message: %v", err)
			}
			return
		}

		switch msg.Type {
		case "registered":
			fmt.Printf("✅ Successfully registered as: %s\n", c.config.Username)
		case "auth_failed":
			fmt.Printf("❌ Authentication failed: %s\n", msg.Data)
			c.connected = false
		case "agent_list":
			c.handleAgentList(msg)
		case "session_created":
			c.handleSessionCreated(msg)
		case "access_denied":
			c.handleAccessDenied(msg)
		case "command_result":
			c.logger.Printf("Received command_result: %s", msg.Data)
			c.handleCommandResult(msg)
		case "agent_connected":
			fmt.Printf("🔗 Agent %s connected\n", msg.AgentID)
		case "agent_disconnected":
			fmt.Printf("💔 Agent %s disconnected\n", msg.AgentID)
		case "port_forward_started":
			fmt.Printf("✅ Port forward started successfully\n")
		case "port_forward_error":
			if metadata := msg.Metadata; metadata != nil {
				if errMsg, ok := metadata["error"].(string); ok {
					fmt.Printf("❌ Port forward error: %s\n", errMsg)
				}
			}
		case "authenticated":
			fmt.Printf("✅ Successfully authenticated as user: %s\n", c.config.Username)
		}
	}
}

func (c *UnifiedClient) handleAgentList(msg Message) {
	if data, ok := msg.Metadata["agents"].([]interface{}); ok {
		c.agentList = nil
		for _, item := range data {
			if agentData, ok := item.(map[string]interface{}); ok {
				agent := Agent{
					ID:       getString(agentData, "id"),
					Name:     getString(agentData, "name"),
					Platform: getString(agentData, "platform"),
					Status:   getString(agentData, "status"),
				}
				c.agentList = append(c.agentList, agent)
			}
		}
	}
}

func (c *UnifiedClient) handleCommandResult(msg Message) {
	// Show command results when in remote session or remote mode
	c.logger.Printf("handleCommandResult: sessionID='%s', currentAgent='%s', msgData='%s'", c.sessionID, c.currentAgent, msg.Data)
	if c.sessionID != "" && c.currentAgent != "" {
		if msg.Data != "" {
			fmt.Print(msg.Data)
		} else {
			c.logger.Printf("Empty command result data")
		}
	} else {
		c.logger.Printf("Not in remote session - sessionID='%s', currentAgent='%s'", c.sessionID, c.currentAgent)
	}
}

func (c *UnifiedClient) handleSessionCreated(msg Message) {
	if msg.SessionID != "" {
		c.sessionID = msg.SessionID
		c.logger.Printf("Session created: %s", msg.SessionID)
	}
}

func (c *UnifiedClient) handleAccessDenied(msg Message) {
	fmt.Printf("❌ Access denied: %s\n", msg.Data)
	c.currentAgent = ""
	c.sessionID = ""
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func (c *UnifiedClient) StartMainMenu() {
	scanner := bufio.NewScanner(os.Stdin)

	// Langsung masuk ke mode port forward (simplified mode)
	c.startSimplePortForwardMode(scanner)
}

func (c *UnifiedClient) startSimplePortForwardMode(scanner *bufio.Scanner) {
	c.mode = "port_forward"
	fmt.Println("\n🚀 GoTeleport Simple Client - Port Forward Manager")

	// Show menu only once at startup
	c.showSimpleMenu()

	for c.connected && c.mode == "port_forward" {
		fmt.Print("command> ")

		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			c.processSimpleCommand(input)
		}
	}
}

func (c *UnifiedClient) showSimpleMenu() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║             🔄 GoTeleport Unified Client                 ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")
	fmt.Println("║ Commands:                                                ║")
	fmt.Println("║   agents                 - List available agents        ║")
	fmt.Println("║   connect <agent_id>     - Connect to specific agent    ║")
	fmt.Println("║   remote                 - Enter remote shell mode      ║")
	fmt.Println("║   forward <local> <target> <port> - Create port forward ║")
	fmt.Println("║   list                   - List active port forwards    ║")
	fmt.Println("║   stop <local_port>      - Stop port forward            ║")
	fmt.Println("║   logs                   - Show database query logs     ║")
	fmt.Println("║   shell                  - Info about shell commands    ║")
	fmt.Println("║   help                   - Show this help               ║")
	fmt.Println("║   exit                   - Exit application             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	fmt.Println("\n💡 Port Forward + Remote Shell Support!")
	fmt.Println("💡 Connect to agent, then use 'remote' for shell commands")

	fmt.Println("\n💡 Examples:")
	fmt.Println("   agents                               # List all agents")
	fmt.Println("   connect 1862343a04e880f4             # Connect to agent")
	fmt.Println("   remote                               # Enter shell mode")
	fmt.Println("   forward 3308 localhost 3306          # Create MySQL/PostgreSQL proxy")
	fmt.Println("   list                                 # Show active forwards")
	fmt.Println("   stop 3308                            # Stop port forward")
}

func (c *UnifiedClient) processSimpleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	switch cmd {
	case "agents":
		c.listAgents()
	case "connect":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: connect <agent_id>")
			fmt.Println("💡 Example: connect 1862343a04e880f4")
			return
		}
		agentID := parts[1]
		c.selectedAgent = agentID
		fmt.Printf("✅ Selected agent: %s\n", agentID)
	case "forward":
		if len(parts) < 4 {
			fmt.Println("❌ Usage: forward <local_port> <target_host> <target_port>")
			fmt.Println("💡 Example: forward 3308 localhost 3306")
			return
		}
		if c.selectedAgent == "" {
			fmt.Println("❌ No agent selected. Use 'connect <agent_id>' first")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])
		targetHost := parts[2]
		targetPort, _ := strconv.Atoi(parts[3])
		c.createPortForward(localPort, c.selectedAgent, targetHost, targetPort)
	case "mysql":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: mysql <local_port>")
			fmt.Println("💡 Example: mysql 3308")
			fmt.Println("💡 This will create port forward: localhost:<local_port> -> agent:3307")
			return
		}
		if c.selectedAgent == "" {
			fmt.Println("❌ No agent selected. Use 'connect <agent_id>' first")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])
		fmt.Printf("🐬 Creating MySQL port forward: localhost:%d -> agent:3307\n", localPort)
		c.createPortForward(localPort, c.selectedAgent, "127.0.0.1", 3307)
	case "postgresql":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: postgresql <local_port>")
			fmt.Println("💡 Example: postgresql 5433")
			fmt.Println("💡 This will create port forward: localhost:<local_port> -> agent:<postgres_port>")
			return
		}
		if c.selectedAgent == "" {
			fmt.Println("❌ No agent selected. Use 'connect <agent_id>' first")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])

		// Read PostgreSQL port dynamically from agent config
		postgresPort := c.getPostgreSQLPort()
		fmt.Printf("🐘 Creating PostgreSQL port forward: localhost:%d -> agent:%d\n", localPort, postgresPort)
		c.createPortForward(localPort, c.selectedAgent, "127.0.0.1", postgresPort)
	case "list":
		c.listPortForwards()
	case "stop":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: stop <local_port>")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])
		c.stopPortForward(localPort)
	case "logs":
		c.getDatabaseLogs()
	case "help":
		c.showSimpleHelp()
	case "remote":
		if c.selectedAgent == "" {
			fmt.Println("❌ No agent selected. Use 'connect <agent_id>' first")
			return
		}
		c.startRemoteWithAgent(c.selectedAgent)
	case "shell":
		if c.selectedAgent == "" {
			fmt.Println("❌ No agent selected. Use 'connect <agent_id>' first")
			return
		}
		fmt.Printf("💡 To execute shell commands on agent %s:\n", c.selectedAgent)
		fmt.Println("💡 Type 'remote' to enter remote shell mode")
		fmt.Println("💡 Or use: interactive-client-clean.exe for standalone shell")
	case "exit", "quit":
		fmt.Println("👋 Goodbye!")
		c.connected = false
	default:
		fmt.Printf("❌ Unknown command: %s\n", cmd)
		fmt.Println("💡 This is a PORT FORWARD MANAGER, not a shell")
		fmt.Println("💡 Available commands: agents, connect, forward, list, stop, logs, help, remote, exit")
		fmt.Println("💡 For shell commands, use 'remote' after connecting to an agent")
	}
}

// getPostgreSQLPort reads agent config and returns the enabled PostgreSQL proxy port
func (c *UnifiedClient) getPostgreSQLPort() int {
	configFile := "agent-config-db.json"
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("⚠️  Could not read agent config %s, using default port 5435\n", configFile)
		return 5435 // fallback to default
	}

	// Parse JSON manually to get PostgreSQL proxy port
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("⚠️  Could not parse agent config, using default port 5435\n")
		return 5435 // fallback to default
	}

	// Find enabled PostgreSQL proxy
	if proxies, ok := config["database_proxies"].([]interface{}); ok {
		for _, p := range proxies {
			if proxy, ok := p.(map[string]interface{}); ok {
				if protocol, ok := proxy["protocol"].(string); ok && protocol == "postgres" {
					if enabled, ok := proxy["enabled"].(bool); ok && enabled {
						if localPort, ok := proxy["local_port"].(float64); ok {
							port := int(localPort)
							fmt.Printf("📖 Found PostgreSQL proxy on port %d\n", port)
							return port
						}
					}
				}
			}
		}
	}

	fmt.Printf("⚠️  No enabled PostgreSQL proxy found in config, using default port 5435\n")
	return 5435 // fallback to default
}

func (c *UnifiedClient) startRemoteWithAgent(agentID string) {
	// Find agent info
	var selectedAgent *Agent
	for i := range c.agentList {
		if c.agentList[i].ID == agentID {
			selectedAgent = &c.agentList[i]
			break
		}
	}

	if selectedAgent == nil {
		fmt.Printf("❌ Agent not found: %s\n", agentID)
		return
	}

	// Connect to agent for remote session
	connectMsg := Message{
		Type:      "connect_agent",
		ClientID:  c.clientID,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	if err := c.conn.WriteJSON(connectMsg); err != nil {
		fmt.Printf("❌ Failed to connect to agent: %v\n", err)
		return
	}

	c.currentAgent = agentID

	// Wait for session creation
	fmt.Printf("🔗 Connecting to agent: %s (%s)...\n", selectedAgent.Name, agentID)

	maxWait := 10
	for i := 0; i < maxWait && c.sessionID == ""; i++ {
		time.Sleep(100 * time.Millisecond)
	}

	if c.sessionID == "" {
		fmt.Printf("❌ Failed to create session with agent\n")
		c.currentAgent = ""
		return
	}

	fmt.Printf("✅ Connected! Session ID: %s\n", c.sessionID)
	fmt.Println("💡 You are now in REMOTE SHELL mode")
	fmt.Println("💡 Type 'exit' to return to port forward manager")
	fmt.Println("💡 Database commands: 'database logs', 'database stats'")

	// Remote shell loop
	scanner := bufio.NewScanner(os.Stdin)
	for c.connected && c.currentAgent == agentID && c.sessionID != "" {
		fmt.Printf("%s> ", selectedAgent.Name)
		if scanner.Scan() {
			command := strings.TrimSpace(scanner.Text())
			if command == "" {
				continue
			}

			if command == "exit" {
				c.disconnectFromAgent()
				fmt.Println("📋 Returned to Port Forward Manager")
				return
			}

			if strings.HasPrefix(command, "database") {
				c.handleDatabaseCommand(command)
				continue
			}

			// Send command to agent
			cmdMsg := Message{
				Type:      "command",
				SessionID: c.sessionID,
				AgentID:   agentID,
				ClientID:  c.clientID,
				Command:   command,
				Timestamp: time.Now(),
			}

			c.logger.Printf("Sending command: %s to agent: %s, session: %s", command, agentID, c.sessionID)
			if err := c.conn.WriteJSON(cmdMsg); err != nil {
				fmt.Printf("❌ Failed to send command: %v\n", err)
				continue
			}

			// Wait a bit for response
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (c *UnifiedClient) handleDatabaseCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) < 2 {
		fmt.Println("❌ Usage: database <logs|stats|help>")
		return
	}

	subCmd := parts[1]
	switch subCmd {
	case "logs":
		c.getDatabaseLogs()
	case "stats":
		c.getDatabaseStats()
	case "help":
		c.showDatabaseHelp()
	default:
		fmt.Printf("❌ Unknown database command: %s\n", subCmd)
		fmt.Println("💡 Available: logs, stats, help")
	}
}

func (c *UnifiedClient) getDatabaseStats() {
	serverURL := strings.Replace(c.config.ServerURL, "ws://", "http://", 1)
	serverURL = strings.Replace(serverURL, "/ws/client", "", 1)

	resp, err := http.Get(serverURL + "/api/database-commands/stats")
	if err != nil {
		fmt.Printf("❌ Failed to get stats: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		fmt.Printf("❌ Failed to parse stats: %v\n", err)
		return
	}

	fmt.Println("\n📊 Database Command Statistics:")
	if total, ok := stats["total_commands"].(float64); ok {
		fmt.Printf("📈 Total Commands: %.0f\n", total)
	}
	if success, ok := stats["successful_commands"].(float64); ok {
		fmt.Printf("✅ Successful: %.0f\n", success)
	}
	if failed, ok := stats["failed_commands"].(float64); ok {
		fmt.Printf("❌ Failed: %.0f\n", failed)
	}
	if avgDuration, ok := stats["avg_duration"].(float64); ok {
		fmt.Printf("⏱️  Average Duration: %.2f ms\n", avgDuration)
	}
}

func (c *UnifiedClient) showDatabaseHelp() {
	fmt.Println("\n💡 Database Commands:")
	fmt.Println("   database logs    - Show recent SQL command logs")
	fmt.Println("   database stats   - Show database statistics")
	fmt.Println("   database help    - Show this help")
}

func (c *UnifiedClient) disconnectFromAgent() {
	if c.sessionID != "" {
		disconnectMsg := Message{
			Type:      "disconnect",
			SessionID: c.sessionID,
			ClientID:  c.clientID,
			Timestamp: time.Now(),
		}
		c.conn.WriteJSON(disconnectMsg)
		c.sessionID = ""
		c.currentAgent = ""
	}
}

func (c *UnifiedClient) showSimpleHelp() {
	fmt.Println("\n💡 Available Commands:")
	fmt.Println("   agents                           - List all available agents")
	fmt.Println("   connect <agent_id>               - Select agent for connections")
	fmt.Println("   remote                           - Enter remote shell mode")
	fmt.Println("   forward <local> <host> <port>    - Create port forward through selected agent")
	fmt.Println("   mysql <local_port>               - Quick MySQL forward (3307)")
	fmt.Println("   postgresql <local_port>          - Quick PostgreSQL forward (auto-detect)")
	fmt.Println("   list                             - List all active port forwards")
	fmt.Println("   stop <local_port>                - Stop specific port forward")
	fmt.Println("   logs                             - Show database command logs")
	fmt.Println("   shell                            - Info about shell commands")
	fmt.Println("   help                             - Show this help message")
	fmt.Println("   exit                             - Exit the application")
	fmt.Println("")
	fmt.Println("🎯 This is a UNIFIED CLIENT - supports both port forwarding and remote shell!")
	fmt.Println("💡 Connect to agent first, then use 'remote' for shell commands")
	fmt.Println("")
	fmt.Println("📋 Usage Flow:")
	fmt.Println("   1. agents                        # See available agents")
	fmt.Println("   2. connect <agent_id>            # Select an agent")
	fmt.Println("   3. remote                        # Enter remote shell mode")
	fmt.Println("   4. mysql 3308                    # Create MySQL proxy")
	fmt.Println("   5. postgresql 5433               # Create PostgreSQL proxy")
	fmt.Println("   6. logs                          # View SQL command logs")
}

func (c *UnifiedClient) refreshAgentList() {
	msg := Message{
		Type:      "get_agents",
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}
	c.conn.WriteJSON(msg)
}

func (c *UnifiedClient) listAgents() {
	c.refreshAgentList()
	time.Sleep(500 * time.Millisecond) // Wait for response

	fmt.Println("┌─────────────────────┬─────────────────────┬─────────────┬──────────┬──────────────────┐")
	fmt.Println("│ Agent ID            │ Agent Name          │ Platform    │ Status   │ Last Seen        │")
	fmt.Println("├─────────────────────┼─────────────────────┼─────────────┼──────────┼──────────────────┤")

	if len(c.agentList) == 0 {
		fmt.Println("│ (No agents available)                                                                │")
	} else {
		for _, agent := range c.agentList {
			fmt.Printf("│ %-19s │ %-19s │ %-11s │ %-8s │ %-16s │\n",
				agent.ID, agent.Name, agent.Platform, agent.Status,
				agent.LastSeen.Format("15:04:05"))
		}
	}
	fmt.Println("└─────────────────────┴─────────────────────┴─────────────┴──────────┴──────────────────┘")
}

func (c *UnifiedClient) createPortForward(localPort int, agentID, targetHost string, targetPort int) {
	// Check if port already in use
	c.mutex.Lock()
	if _, exists := c.portForwards[fmt.Sprintf("%d", localPort)]; exists {
		c.mutex.Unlock()
		fmt.Printf("❌ Port %d already forwarded\n", localPort)
		return
	}
	c.mutex.Unlock()

	// Create port forward
	pf := &UnifiedPortForward{
		LocalPort:  localPort,
		AgentID:    agentID,
		TargetHost: targetHost,
		TargetPort: targetPort,
		Client:     c,
	}

	if err := pf.Start(); err != nil {
		fmt.Printf("❌ Failed to start port forward: %v\n", err)
		return
	}

	c.mutex.Lock()
	c.portForwards[fmt.Sprintf("%d", localPort)] = pf
	c.mutex.Unlock()

	fmt.Printf("✅ Port forward started: localhost:%d -> %s:%s:%d\n",
		localPort, agentID, targetHost, targetPort)
}

func (c *UnifiedClient) listPortForwards() {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	fmt.Println("\n🔄 Active Port Forwards:")
	fmt.Println("┌────────────┬─────────────────────┬─────────────────────┬────────────┐")
	fmt.Println("│ Local Port │ Agent ID            │ Target              │ Status     │")
	fmt.Println("├────────────┼─────────────────────┼─────────────────────┼────────────┤")

	if len(c.portForwards) == 0 {
		fmt.Println("│ (No active port forwards)                                           │")
	} else {
		for _, pf := range c.portForwards {
			status := "Active"
			if !pf.Active {
				status = "Stopped"
			}
			fmt.Printf("│ %-10d │ %-19s │ %s:%-8d │ %-10s │\n",
				pf.LocalPort, pf.AgentID, pf.TargetHost, pf.TargetPort, status)
		}
	}
	fmt.Println("└────────────┴─────────────────────┴─────────────────────┴────────────┘")
}

func (c *UnifiedClient) stopPortForward(localPort int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := fmt.Sprintf("%d", localPort)
	if pf, exists := c.portForwards[key]; exists {
		pf.Stop()
		delete(c.portForwards, key)
		fmt.Printf("✅ Port forward stopped: localhost:%d\n", localPort)
	} else {
		fmt.Printf("❌ No port forward on port %d\n", localPort)
	}
}

func (c *UnifiedClient) getDatabaseLogs() {
	serverURL := strings.Replace(c.config.ServerURL, "ws://", "http://", 1)
	serverURL = strings.Replace(serverURL, "/ws/client", "", 1)

	resp, err := http.Get(serverURL + "/api/database-commands")
	if err != nil {
		fmt.Printf("❌ Failed to get logs: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Parse response as object with commands array
	var response struct {
		Commands []DatabaseCommand `json:"commands"`
		Total    int               `json:"total"`
		Limit    int               `json:"limit"`
		Offset   int               `json:"offset"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("❌ Failed to parse logs: %v\n", err)
		return
	}

	commands := response.Commands
	fmt.Printf("\n📋 Database Command Logs (%d entries):\n", response.Total)

	if len(commands) == 0 {
		fmt.Println("📝 No database commands logged yet")
		fmt.Println("💡 Try executing some SQL commands through the database proxy first")
		return
	}

	fmt.Println("┌──────────────────────┬─────────────────────────────────────────────┬──────────┬──────────┐")
	fmt.Println("│ Timestamp            │ Command                                     │ Protocol │ Proxy    │")
	fmt.Println("├──────────────────────┼─────────────────────────────────────────────┼──────────┼──────────┤")

	count := 0
	for i := len(commands) - 1; i >= 0 && count < 10; i-- {
		cmd := commands[i]
		cmdStr := cmd.Command
		if len(cmdStr) > 40 {
			cmdStr = cmdStr[:37] + "..."
		}
		protocol := cmd.Protocol
		if protocol == "" {
			protocol = "unknown"
		}
		proxyName := cmd.ProxyName
		if len(proxyName) > 8 {
			proxyName = proxyName[:5] + "..."
		}
		fmt.Printf("│ %-20s │ %-43s │ %-8s │ %-8s │\n",
			cmd.Timestamp.Format("2006-01-02 15:04:05"),
			cmdStr, protocol, proxyName)
		count++
	}
	fmt.Println("└──────────────────────┴─────────────────────────────────────────────┴──────────┴──────────┘")
}

func (c *UnifiedClient) Close() {
	c.connected = false

	// Stop all port forwards
	c.mutex.Lock()
	for _, pf := range c.portForwards {
		pf.Stop()
	}
	c.mutex.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
}

// Port Forward Implementation
func (pf *UnifiedPortForward) Start() error {
	addr := fmt.Sprintf(":%d", pf.LocalPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	pf.Listener = listener
	pf.Active = true

	go pf.acceptConnections()
	return nil
}

// getServerHost extracts the host from server URL (e.g., "ws://168.231.119.242:8080/ws/client" -> "168.231.119.242")
func (c *UnifiedClient) getServerHost() string {
	serverURL := c.config.ServerURL
	// Remove protocol (ws:// or wss://)
	if strings.HasPrefix(serverURL, "ws://") {
		serverURL = serverURL[5:]
	} else if strings.HasPrefix(serverURL, "wss://") {
		serverURL = serverURL[6:]
	}

	// Extract host part (before first colon or slash)
	if colonIndex := strings.Index(serverURL, ":"); colonIndex != -1 {
		return serverURL[:colonIndex]
	}
	if slashIndex := strings.Index(serverURL, "/"); slashIndex != -1 {
		return serverURL[:slashIndex]
	}

	return serverURL
}

func (pf *UnifiedPortForward) acceptConnections() {
	for pf.Active {
		conn, err := pf.Listener.Accept()
		if err != nil {
			if pf.Active {
				pf.Client.logger.Printf("Error accepting connection: %v", err)
			}
			return
		}

		go pf.handleConnection(conn)
	}
}

func (pf *UnifiedPortForward) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// For agents behind NAT, we need to create a tunnel through the server
	// Instead of connecting directly to agent, we send a port forward request to server

	var targetPort int
	var dbType string

	if pf.TargetPort == 5433 || (pf.TargetPort == 5432 && pf.TargetHost == "localhost") {
		// PostgreSQL connection
		if pf.TargetPort == 5432 {
			targetPort = 5435 // Use intercept proxy for logging
		} else {
			targetPort = 5433 // Direct PostgreSQL proxy
		}
		dbType = "postgresql"
	} else {
		// MySQL connection (default)
		targetPort = 3306  // MySQL server port (not agent port)
		dbType = "mysql"
	}

	// Create a tunnel request through the server to the agent
	err := pf.createTunnelThroughServer(clientConn, pf.AgentID, targetPort, dbType)
	if err != nil {
		pf.Client.logger.Printf("Failed to create tunnel through server for %s:%d: %v",
			pf.AgentID, targetPort, err)
		return
	}
}

func (pf *UnifiedPortForward) Stop() {
	pf.Active = false
	if pf.Listener != nil {
		pf.Listener.Close()
	}
}

// createTunnelThroughServer creates a database tunnel through the server to an agent behind NAT
func (pf *UnifiedPortForward) createTunnelThroughServer(clientConn net.Conn, agentID string, targetPort int, dbType string) error {
	// Generate tunnel ID
	tunnelID := fmt.Sprintf("tunnel_%d_%s", time.Now().UnixNano(), agentID)
	
	pf.Client.logger.Printf("Creating tunnel through server: %s -> agent:%s port:%d (%s)",
		tunnelID, agentID, targetPort, dbType)

	// Send tunnel request through existing WebSocket connection
	tunnelRequest := map[string]interface{}{
		"type":        "tunnel_request",
		"tunnel_id":   tunnelID,
		"agent_id":    agentID,
		"tunnel_type": dbType,
		"remote_host": "localhost",
		"remote_port": targetPort,
		"local_port":  0, // Will be auto-assigned by agent
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	pf.Client.logger.Printf("📤 Sending tunnel request to server...")
	pf.Client.logger.Printf("🔍 Tunnel request details: %+v", tunnelRequest)
	pf.Client.mutex.Lock()
	err := pf.Client.conn.WriteJSON(tunnelRequest)
	pf.Client.mutex.Unlock()
	
	if err != nil {
		return fmt.Errorf("failed to send tunnel request: %v", err)
	}
	
	pf.Client.logger.Printf("✅ Tunnel request sent successfully")

	// Wait for tunnel response from agent (through server)
	pf.Client.logger.Printf("⏳ Waiting for tunnel response...")
	
	// For now, let's assume tunnel is created successfully 
	// Agent should create tunnel and we connect to agent's tunnel port
	// For MySQL: client -> agent:3307 -> localhost:3306
	// For PostgreSQL: client -> agent:5433 -> localhost:5432
	
	// Try to connect to agent tunnel (try multiple ports)
	var agentConn net.Conn
	var connectedPort int
	
	// Try ports 3307, 3308, 3309, etc.
	if dbType == "mysql" {
		for port := 3307; port <= 3320; port++ {
			agentHost := pf.Client.getServerHost()
			agentTunnelAddr := fmt.Sprintf("%s:%d", agentHost, port)
			
			pf.Client.logger.Printf("🔗 Trying to connect to agent tunnel: %s", agentTunnelAddr)
			conn, dialErr := net.DialTimeout("tcp", agentTunnelAddr, 3*time.Second)
			if dialErr == nil {
				agentConn = conn
				connectedPort = port
				pf.Client.logger.Printf("✅ Connected to agent tunnel on port: %d", port)
				break
			} else {
				pf.Client.logger.Printf("⚠️  Port %d not available, trying next...", port)
			}
		}
	} else {
		// PostgreSQL
		agentTunnelPort := 5433
		agentHost := pf.Client.getServerHost()
		agentTunnelAddr := fmt.Sprintf("%s:%d", agentHost, agentTunnelPort)
		
		pf.Client.logger.Printf("🔗 Connecting to agent tunnel: %s", agentTunnelAddr)
		conn, dialErr := net.DialTimeout("tcp", agentTunnelAddr, 10*time.Second)
		if dialErr == nil {
			agentConn = conn
			connectedPort = agentTunnelPort
		}
	}
	
	if agentConn == nil {
		return fmt.Errorf("failed to connect to agent tunnel on any port")
	}
	
	pf.Client.logger.Printf("🎯 Using agent tunnel on port: %d", connectedPort)
	defer agentConn.Close()

	// Proxy data between client and agent tunnel
	done := make(chan bool, 2)

	// Copy client -> agent  
	go func() {
		defer func() { done <- true }()
		io.Copy(agentConn, clientConn)
	}()

	// Copy agent -> client
	go func() {
		defer func() { done <- true }()
		io.Copy(clientConn, agentConn)
	}()

	// Wait for either direction to finish
	<-done
	
	return nil
}
