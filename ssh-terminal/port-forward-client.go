package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type PortForwardClient struct {
	config        *ClientConfig
	conn          *websocket.Conn
	logger        *log.Logger
	clientID      string
	connected     bool
	agentList     []Agent
	currentAgent  string
	portForwards  map[string]*PortForward
	mutex         sync.RWMutex
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

type PortForward struct {
	LocalPort  int
	AgentID    string
	TargetHost string
	TargetPort int
	Listener   net.Listener
	Active     bool
	Client     *PortForwardClient
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
		log.Fatal("Usage: port-forward-client.exe <config-file>")
	}

	client, err := NewPortForwardClient(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("🚀 GoTeleport Port Forward Client")
	fmt.Println("🔌 Connecting to server...")

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("✅ Connected to server successfully!")
	client.StartPortForwardShell()
}

func NewPortForwardClient(configFile string) (*PortForwardClient, error) {
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

	logger := log.New(logFile, "[PORT_FORWARD] ", log.LstdFlags)
	clientID := fmt.Sprintf("client_%d", time.Now().Unix())

	return &PortForwardClient{
		config:       &config,
		logger:       logger,
		clientID:     clientID,
		portForwards: make(map[string]*PortForward),
	}, nil
}

func (c *PortForwardClient) Connect() error {
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

	// Wait for auth response
	time.Sleep(1 * time.Second)
	return nil
}

func (c *PortForwardClient) handleMessages() {
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
		case "authenticated":
			fmt.Printf("✅ Successfully authenticated as user: %s\n", c.config.Username)
		}
	}
}

func (c *PortForwardClient) handleAgentList(msg Message) {
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

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func (c *PortForwardClient) StartPortForwardShell() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║             🔄 Port Forward Manager                     ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")
	fmt.Println("║ Commands:                                                ║")
	fmt.Println("║   agents                 - List available agents        ║")
	fmt.Println("║   forward <local> <agent> <target> <port>               ║")
	fmt.Println("║   list                   - List active forwards         ║")
	fmt.Println("║   stop <local_port>      - Stop port forward            ║")
	fmt.Println("║   status                 - Show connection status       ║")
	fmt.Println("║   help                   - Show help                    ║")
	fmt.Println("║   exit                   - Exit client                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println("\n💡 Example: forward 3308 agent-1 localhost 3306")

	// Auto-refresh agent list
	c.refreshAgentList()

	scanner := bufio.NewScanner(os.Stdin)
	for c.connected {
		fmt.Print("\n🔄 port-forward> ")
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			c.processCommand(input)
		}
	}
}

func (c *PortForwardClient) processCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	switch cmd {
	case "agents":
		c.listAgents()
	case "forward":
		if len(parts) < 5 {
			fmt.Println("❌ Usage: forward <local_port> <agent_id> <target_host> <target_port>")
			fmt.Println("💡 Example: forward 3308 agent-1 localhost 3306")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])
		agentID := parts[2]
		targetHost := parts[3]
		targetPort, _ := strconv.Atoi(parts[4])
		c.createPortForward(localPort, agentID, targetHost, targetPort)
	case "list":
		c.listPortForwards()
	case "stop":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: stop <local_port>")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])
		c.stopPortForward(localPort)
	case "status":
		c.showStatus()
	case "help":
		c.showHelp()
	case "exit", "quit":
		fmt.Println("👋 Exiting port forward client...")
		c.connected = false
	default:
		fmt.Printf("❌ Unknown command: %s\n", cmd)
		fmt.Println("💡 Type 'help' for available commands")
	}
}

func (c *PortForwardClient) refreshAgentList() {
	msg := Message{
		Type:      "get_agents",
		ClientID:  c.clientID,
		Timestamp: time.Now(),
	}
	c.conn.WriteJSON(msg)
}

func (c *PortForwardClient) listAgents() {
	c.refreshAgentList()
	time.Sleep(500 * time.Millisecond) // Wait for response

	fmt.Println("\n📋 Available Agents:")
	fmt.Println("┌─────────────────────┬─────────────┬──────────┬──────────────────┐")
	fmt.Println("│ Agent ID            │ Platform    │ Status   │ Last Seen        │")
	fmt.Println("├─────────────────────┼─────────────┼──────────┼──────────────────┤")

	if len(c.agentList) == 0 {
		fmt.Println("│ (No agents available)                                      │")
	} else {
		for _, agent := range c.agentList {
			fmt.Printf("│ %-19s │ %-11s │ %-8s │ %-16s │\n",
				agent.ID, agent.Platform, agent.Status,
				agent.LastSeen.Format("15:04:05"))
		}
	}
	fmt.Println("└─────────────────────┴─────────────┴──────────┴──────────────────┘")
}

func (c *PortForwardClient) createPortForward(localPort int, agentID, targetHost string, targetPort int) {
	// Check if port already in use
	c.mutex.Lock()
	if _, exists := c.portForwards[fmt.Sprintf("%d", localPort)]; exists {
		c.mutex.Unlock()
		fmt.Printf("❌ Port %d already forwarded\n", localPort)
		return
	}
	c.mutex.Unlock()

	// Create port forward
	pf := &PortForward{
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

func (c *PortForwardClient) listPortForwards() {
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

func (c *PortForwardClient) stopPortForward(localPort int) {
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

func (c *PortForwardClient) showStatus() {
	fmt.Printf("\n📊 Port Forward Client Status:\n")
	fmt.Printf("🔌 Connected: %t\n", c.connected)
	fmt.Printf("🆔 Client ID: %s\n", c.clientID)
	fmt.Printf("👤 Username: %s\n", c.config.Username)
	fmt.Printf("🖥️  Agents: %d available\n", len(c.agentList))
	fmt.Printf("🔄 Port Forwards: %d active\n", len(c.portForwards))
}

func (c *PortForwardClient) showHelp() {
	fmt.Println("\n💡 Port Forward Commands:")
	fmt.Println("   agents                           - List available agents")
	fmt.Println("   forward <local> <agent> <host> <port> - Create port forward")
	fmt.Println("   list                             - List active port forwards")
	fmt.Println("   stop <local_port>                - Stop specific port forward")
	fmt.Println("   status                           - Show client status")
	fmt.Println("   exit                             - Exit client")
	fmt.Println("\n📝 Examples:")
	fmt.Println("   forward 3308 agent-1 localhost 3306    # MySQL proxy")
	fmt.Println("   forward 5433 agent-2 192.168.1.10 5432 # PostgreSQL proxy")
	fmt.Println("   stop 3308                               # Stop MySQL proxy")
}

func (c *PortForwardClient) Close() {
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
func (pf *PortForward) Start() error {
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

func (pf *PortForward) acceptConnections() {
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

func (pf *PortForward) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// In a real implementation, this would tunnel through the agent
	// For now, we'll simulate direct connection for demonstration
	targetAddr := fmt.Sprintf("%s:%d", pf.TargetHost, pf.TargetPort)
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		pf.Client.logger.Printf("Failed to connect to target %s: %v", targetAddr, err)
		return
	}
	defer targetConn.Close()

	pf.Client.logger.Printf("Port forward connection: localhost:%d -> %s (via %s)",
		pf.LocalPort, targetAddr, pf.AgentID)

	// Bidirectional copy
	done := make(chan bool, 2)
	go func() {
		io.Copy(clientConn, targetConn)
		done <- true
	}()
	go func() {
		io.Copy(targetConn, clientConn)
		done <- true
	}()

	<-done
}

func (pf *PortForward) Stop() {
	pf.Active = false
	if pf.Listener != nil {
		pf.Listener.Close()
	}
}
