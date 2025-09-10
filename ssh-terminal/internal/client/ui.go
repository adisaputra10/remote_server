package client

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"ssh-terminal/internal/proto"
)

// UI represents the client terminal UI
type UI struct {
	client       *Client
	running      bool
	mu           sync.RWMutex
	agents       []proto.AgentInfo
	lastResponse interface{}
}

// NewUI creates a new UI instance
func NewUI(client *Client) (*UI, error) {
	return &UI{
		client:  client,
		running: false,
		agents:  make([]proto.AgentInfo, 0),
	}, nil
}

// Run starts the UI loop (keeping the same interface as original)
func (ui *UI) Run() {
	ui.mu.Lock()
	ui.running = true
	ui.mu.Unlock()

	defer func() {
		ui.mu.Lock()
		ui.running = false
		ui.mu.Unlock()
	}()

	fmt.Println("=== Database Tunnel Client ===")
	fmt.Println("Welcome to the enhanced tunnel client!")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for ui.isRunning() {
		ui.showMenu()
		fmt.Print("Enter your choice: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())
		if choice == "" {
			continue
		}

		if err := ui.handleChoice(choice, scanner); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}
}

// Stop stops the UI
func (ui *UI) Stop() {
	ui.mu.Lock()
	ui.running = false
	ui.mu.Unlock()
}

// isRunning checks if UI is running
func (ui *UI) isRunning() bool {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return ui.running
}

// showMenu displays the main menu (same as original)
func (ui *UI) showMenu() {
	fmt.Println("=== Main Menu ===")
	fmt.Println("1. List available agents")
	fmt.Println("2. Show agent details")
	fmt.Println("3. Create database tunnel")
	fmt.Println("4. List active tunnels")
	fmt.Println("5. Close tunnel")
	fmt.Println("6. Test database connection")
	fmt.Println("7. Show logs")
	fmt.Println("8. Exit")
	fmt.Println()
}

// handleChoice handles user menu selection
func (ui *UI) handleChoice(choice string, scanner *bufio.Scanner) error {
	switch choice {
	case "1":
		return ui.listAgents()
	case "2":
		return ui.showAgentDetails(scanner)
	case "3":
		return ui.createTunnel(scanner)
	case "4":
		return ui.listTunnels()
	case "5":
		return ui.closeTunnel(scanner)
	case "6":
		return ui.testConnection(scanner)
	case "7":
		return ui.showLogs()
	case "8":
		fmt.Println("Goodbye!")
		ui.Stop()
		return nil
	default:
		fmt.Println("Invalid choice. Please try again.")
		return nil
	}
}

// listAgents lists available agents
func (ui *UI) listAgents() error {
	fmt.Println("Requesting agent list from server...")
	
	if err := ui.client.ListAgents(); err != nil {
		return fmt.Errorf("failed to request agent list: %w", err)
	}

	// Wait a moment for response
	fmt.Println("Waiting for response...")
	// The response will be handled by NotifyAgentList
	
	return nil
}

// showAgentDetails shows details for a specific agent
func (ui *UI) showAgentDetails(scanner *bufio.Scanner) error {
	ui.mu.RLock()
	agentCount := len(ui.agents)
	ui.mu.RUnlock()

	if agentCount == 0 {
		fmt.Println("No agents available. Please list agents first.")
		return nil
	}

	fmt.Print("Enter agent number (1-" + strconv.Itoa(agentCount) + "): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	numStr := strings.TrimSpace(scanner.Text())
	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 || num > agentCount {
		return fmt.Errorf("invalid agent number")
	}

	ui.mu.RLock()
	agent := ui.agents[num-1]
	ui.mu.RUnlock()

	fmt.Printf("Requesting details for agent: %s\n", agent.ID)
	return ui.client.GetAgentInfo(agent.ID)
}

// createTunnel creates a new database tunnel
func (ui *UI) createTunnel(scanner *bufio.Scanner) error {
	ui.mu.RLock()
	agentCount := len(ui.agents)
	ui.mu.RUnlock()

	if agentCount == 0 {
		fmt.Println("No agents available. Please list agents first.")
		return nil
	}

	fmt.Println("=== Create Database Tunnel ===")

	// Select agent
	fmt.Print("Enter agent number (1-" + strconv.Itoa(agentCount) + "): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	numStr := strings.TrimSpace(scanner.Text())
	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 || num > agentCount {
		return fmt.Errorf("invalid agent number")
	}

	ui.mu.RLock()
	agent := ui.agents[num-1]
	ui.mu.RUnlock()

	// Select database type
	fmt.Println("Select database type:")
	fmt.Println("1. MySQL")
	fmt.Println("2. PostgreSQL")
	fmt.Print("Enter choice (1-2): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	dbChoice := strings.TrimSpace(scanner.Text())
	var dbType string
	switch dbChoice {
	case "1":
		dbType = "mysql"
	case "2":
		dbType = "postgresql"
	default:
		return fmt.Errorf("invalid database type")
	}

	// Get local port
	fmt.Print("Enter local port (e.g., 3306 for MySQL, 5432 for PostgreSQL): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	localPortStr := strings.TrimSpace(scanner.Text())
	localPort, err := strconv.Atoi(localPortStr)
	if err != nil || localPort < 1 || localPort > 65535 {
		return fmt.Errorf("invalid local port")
	}

	// Get remote host
	fmt.Print("Enter remote database host (default: localhost): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	remoteHost := strings.TrimSpace(scanner.Text())
	if remoteHost == "" {
		remoteHost = "localhost"
	}

	// Get remote port
	fmt.Print("Enter remote database port: ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	remotePortStr := strings.TrimSpace(scanner.Text())
	remotePort, err := strconv.Atoi(remotePortStr)
	if err != nil || remotePort < 1 || remotePort > 65535 {
		return fmt.Errorf("invalid remote port")
	}

	fmt.Printf("Creating %s tunnel: localhost:%d -> %s:%d (via agent %s)\n",
		dbType, localPort, remoteHost, remotePort, agent.Name)

	return ui.client.CreateTunnel(agent.ID, dbType, localPort, remoteHost, remotePort)
}

// listTunnels lists active tunnels
func (ui *UI) listTunnels() error {
	tunnels := ui.client.GetTunnels()

	fmt.Println("=== Active Tunnels ===")
	if len(tunnels) == 0 {
		fmt.Println("No active tunnels.")
		return nil
	}

	i := 1
	for _, tunnel := range tunnels {
		status := "Inactive"
		if tunnel.Active {
			status = "Active"
		}
		fmt.Printf("%d. %s tunnel: localhost:%d -> %s:%d [%s]\n",
			i, tunnel.Type, tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort, status)
		i++
	}

	return nil
}

// closeTunnel closes a tunnel
func (ui *UI) closeTunnel(scanner *bufio.Scanner) error {
	tunnels := ui.client.GetTunnels()

	if len(tunnels) == 0 {
		fmt.Println("No active tunnels to close.")
		return nil
	}

	fmt.Println("=== Close Tunnel ===")
	i := 1
	tunnelList := make([]*Tunnel, 0, len(tunnels))
	for _, tunnel := range tunnels {
		fmt.Printf("%d. %s tunnel: localhost:%d -> %s:%d\n",
			i, tunnel.Type, tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort)
		tunnelList = append(tunnelList, tunnel)
		i++
	}

	fmt.Print("Enter tunnel number to close (1-" + strconv.Itoa(len(tunnelList)) + "): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	numStr := strings.TrimSpace(scanner.Text())
	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 || num > len(tunnelList) {
		return fmt.Errorf("invalid tunnel number")
	}

	tunnel := tunnelList[num-1]
	fmt.Printf("Closing tunnel: %s\n", tunnel.ID)

	return ui.client.CloseTunnel(tunnel.ID)
}

// testConnection tests database connection
func (ui *UI) testConnection(scanner *bufio.Scanner) error {
	fmt.Println("=== Test Database Connection ===")
	fmt.Print("Enter database host (e.g., localhost:3306): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	host := strings.TrimSpace(scanner.Text())
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	fmt.Print("Enter database type (mysql/postgresql): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	dbType := strings.TrimSpace(scanner.Text())
	
	fmt.Print("Enter username: ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	username := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter password: ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	password := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter database name: ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	database := strings.TrimSpace(scanner.Text())

	fmt.Printf("Testing connection to %s database at %s...\n", dbType, host)
	fmt.Println("Note: This is a placeholder. Actual database testing would be implemented here.")
	fmt.Printf("Connection parameters: host=%s, user=%s, database=%s\n", host, username, database)

	return nil
}

// showLogs shows recent logs
func (ui *UI) showLogs() error {
	fmt.Println("=== Recent Logs ===")
	fmt.Println("Note: Log viewing would be implemented here.")
	fmt.Println("This would show recent client and tunnel activity.")
	return nil
}

// Notification methods for server responses

// NotifyAgentList handles agent list notification
func (ui *UI) NotifyAgentList(agentList *proto.AgentListData) {
	ui.mu.Lock()
	ui.agents = agentList.Agents
	ui.mu.Unlock()

	fmt.Printf("\n=== Available Agents ===\n")
	if len(agentList.Agents) == 0 {
		fmt.Println("No agents currently connected.")
		return
	}

	for i, agent := range agentList.Agents {
		fmt.Printf("%d. %s (%s) - Platform: %s\n", 
			i+1, agent.Name, agent.ID, agent.Platform)
	}
	fmt.Print("\nEnter your choice: ")
}

// NotifyAgentInfo handles agent info notification
func (ui *UI) NotifyAgentInfo(info *proto.AgentInfo) {
	fmt.Printf("\n=== Agent Details ===\n")
	fmt.Printf("ID: %s\n", info.ID)
	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Platform: %s\n", info.Platform)
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("Status: %s\n", info.Status)
	
	if len(info.Capabilities) > 0 {
		fmt.Printf("Capabilities: %s\n", strings.Join(info.Capabilities, ", "))
	}
	
	if info.Hostname != "" {
		fmt.Printf("Hostname: %s\n", info.Hostname)
	}
	
	if info.IPAddress != "" {
		fmt.Printf("IP Address: %s\n", info.IPAddress)
	}
	
	fmt.Print("\nEnter your choice: ")
}

// NotifyTunnelResponse handles tunnel response notification
func (ui *UI) NotifyTunnelResponse(resp *proto.TunnelResponseData) {
	if resp.Status == "success" {
		fmt.Printf("\n✅ Tunnel established successfully: %s\n", resp.TunnelID)
	} else {
		fmt.Printf("\n❌ Tunnel failed: %s - %s\n", resp.TunnelID, resp.Message)
	}
	fmt.Print("Enter your choice: ")
}

// NotifyTunnelClose handles tunnel close notification
func (ui *UI) NotifyTunnelClose(closeMsg *proto.TunnelCloseMessage) {
	fmt.Printf("\n🔌 Tunnel closed: %s - %s\n", closeMsg.TunnelID, closeMsg.Reason)
	fmt.Print("Enter your choice: ")
}
