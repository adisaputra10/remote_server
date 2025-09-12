package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"remote-tunnel/internal/logger"
	"remote-tunnel/internal/tunnel"
)

type TunnelClient struct {
	relayURL  string
	logger    *logger.Logger
	transport *tunnel.Transport
	agents    map[string]*tunnel.AgentInfo
	tunnels   map[string]*ClientTunnel
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

type ClientTunnel struct {
	ID        string
	AgentID   string
	LocalPort int
	Target    string
	Listener  net.Listener
	Info      *tunnel.TunnelInfo
	Active    bool
}

func main() {
	var (
		relayURL = flag.String("relay-url", "ws://localhost:8443/ws/client", "Relay server WebSocket URL")
		localPort = flag.Int("L", 0, "Local port to listen on (for direct tunnel)")
		agent     = flag.String("agent", "", "Target agent ID (for direct tunnel)")
		target    = flag.String("target", "", "Target address (for direct tunnel)")
		interactive = flag.Bool("i", false, "Interactive mode")
	)
	flag.Parse()

	log := logger.New("CLIENT")
	log.Info("🚀 Starting tunnel client")
	log.Info("📋 Relay URL: %s", *relayURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &TunnelClient{
		relayURL: *relayURL,
		logger:   log,
		agents:   make(map[string]*tunnel.AgentInfo),
		tunnels:  make(map[string]*ClientTunnel),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Connect to relay
	if err := client.connectToRelay(); err != nil {
		log.Error("Failed to connect to relay: %v", err)
		os.Exit(1)
	}

	// Handle direct tunnel mode or interactive mode
	if *localPort > 0 && *agent != "" && *target != "" {
		// Direct tunnel mode
		log.Command("DIRECT_TUNNEL", "local", *localPort, "agent", *agent, "target", *target)
		client.createDirectTunnel(*localPort, *agent, *target)
	} else {
		// Interactive mode
		log.Command("INTERACTIVE_MODE", "starting")
		client.interactiveMode()
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info("🛑 Shutting down client...")
	client.shutdown()
	log.Info("👋 Client stopped")
}

func (c *TunnelClient) connectToRelay() error {
	c.logger.Info("🔗 Connecting to relay: %s", c.relayURL)

	u, err := url.Parse(c.relayURL)
	if err != nil {
		return fmt.Errorf("invalid relay URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	transport, err := tunnel.NewTransport(conn, true, c.logger)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create transport: %w", err)
	}

	c.transport = transport

	// Start message handler
	go c.handleMessages()

	// Request agent list
	msg := tunnel.NewMessage(tunnel.MsgClientConnect)
	if err := c.transport.SendMessage(msg); err != nil {
		transport.Close()
		return fmt.Errorf("failed to send connect message: %w", err)
	}

	c.logger.Info("✅ Connected to relay successfully")
	return nil
}

func (c *TunnelClient) handleMessages() {
	defer c.transport.Close()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		msg, err := c.transport.ReceiveMessage()
		if err != nil {
			c.logger.Error("Message receive error: %v", err)
			return
		}

		c.logger.Command("RECEIVED", msg.Type, msg.TunnelID)

		switch msg.Type {
		case tunnel.MsgAgentList:
			c.handleAgentList(msg)

		case tunnel.MsgTunnelCreated:
			c.handleTunnelCreated(msg)

		case tunnel.MsgTunnelError:
			c.handleTunnelError(msg)

		default:
			c.logger.Warn("Unknown message type: %s", msg.Type)
		}
	}
}

func (c *TunnelClient) handleAgentList(msg *tunnel.Message) {
	var agents []*tunnel.AgentInfo
	
	if data, ok := msg.Data.([]interface{}); ok {
		for _, item := range data {
			if agentData, ok := item.(map[string]interface{}); ok {
				agentBytes, _ := json.Marshal(agentData)
				var agent tunnel.AgentInfo
				if json.Unmarshal(agentBytes, &agent) == nil {
					agents = append(agents, &agent)
				}
			}
		}
	}

	c.mu.Lock()
	c.agents = make(map[string]*tunnel.AgentInfo)
	for _, agent := range agents {
		c.agents[agent.ID] = agent
	}
	c.mu.Unlock()

	c.logger.Info("📋 Received agent list: %d agents", len(agents))
	for _, agent := range agents {
		c.logger.Info("  - %s (%s) - %s on %s", agent.ID, agent.Name, agent.Status, agent.Platform)
	}
}

func (c *TunnelClient) handleTunnelCreated(msg *tunnel.Message) {
	var tunnelInfo tunnel.TunnelInfo
	
	if data, ok := msg.Data.(map[string]interface{}); ok {
		infoBytes, _ := json.Marshal(data)
		json.Unmarshal(infoBytes, &tunnelInfo)
	}

	c.mu.Lock()
	if clientTunnel, exists := c.tunnels[tunnelInfo.ID]; exists {
		clientTunnel.Info = &tunnelInfo
		clientTunnel.Active = true
	}
	c.mu.Unlock()

	c.logger.Tunnel("CREATED", tunnelInfo.ID, fmt.Sprintf("remote endpoint: %s", tunnelInfo.RemoteAddr))
}

func (c *TunnelClient) handleTunnelError(msg *tunnel.Message) {
	c.logger.Error("Tunnel error [%s]: %s", msg.TunnelID, msg.Error)
	
	c.mu.Lock()
	if clientTunnel, exists := c.tunnels[msg.TunnelID]; exists {
		clientTunnel.Active = false
		if clientTunnel.Listener != nil {
			clientTunnel.Listener.Close()
		}
		delete(c.tunnels, msg.TunnelID)
	}
	c.mu.Unlock()
}

func (c *TunnelClient) createDirectTunnel(localPort int, agentID, target string) {
	parts := strings.Split(target, ":")
	if len(parts) != 2 {
		c.logger.Error("Invalid target format. Use host:port")
		return
	}

	remotePort, err := strconv.Atoi(parts[1])
	if err != nil {
		c.logger.Error("Invalid port in target: %v", err)
		return
	}

	tunnelID := fmt.Sprintf("tunnel_%d", time.Now().UnixNano())
	
	if err := c.createTunnel(tunnelID, agentID, localPort, parts[0], remotePort); err != nil {
		c.logger.Error("Failed to create tunnel: %v", err)
		return
	}

	c.logger.Info("✅ Direct tunnel created: localhost:%d -> %s via %s", localPort, target, agentID)
	
	// Keep running until interrupted
	select {
	case <-c.ctx.Done():
	}
}

func (c *TunnelClient) interactiveMode() {
	c.logger.Info("🎮 Starting interactive mode")
	c.printHelp()

	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("tunnel> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		c.logger.Command("USER_INPUT", line)
		c.handleCommand(line)
	}
}

func (c *TunnelClient) handleCommand(line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	
	switch command {
	case "help", "h":
		c.printHelp()

	case "agents", "list":
		c.listAgents()

	case "tunnels":
		c.listTunnels()

	case "create", "tunnel":
		if len(parts) < 5 {
			fmt.Println("Usage: create <local_port> <agent_id> <remote_host> <remote_port>")
			return
		}
		localPort, _ := strconv.Atoi(parts[1])
		agentID := parts[2]
		remoteHost := parts[3]
		remotePort, _ := strconv.Atoi(parts[4])
		
		tunnelID := fmt.Sprintf("tunnel_%d", time.Now().UnixNano())
		if err := c.createTunnel(tunnelID, agentID, localPort, remoteHost, remotePort); err != nil {
			fmt.Printf("Failed to create tunnel: %v\n", err)
		}

	case "close":
		if len(parts) < 2 {
			fmt.Println("Usage: close <tunnel_id>")
			return
		}
		c.closeTunnel(parts[1])

	case "status":
		c.printStatus()

	case "refresh":
		msg := tunnel.NewMessage(tunnel.MsgClientConnect)
		c.transport.SendMessage(msg)
		fmt.Println("Refreshing agent list...")

	case "quit", "exit":
		c.cancel()
		return

	default:
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
	}
}

func (c *TunnelClient) createTunnel(tunnelID, agentID string, localPort int, remoteHost string, remotePort int) error {
	// Check if agent exists
	c.mu.RLock()
	_, exists := c.agents[agentID]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Create local listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", localPort, err)
	}

	// Create tunnel request
	req := &tunnel.TunnelRequest{
		TunnelID:   tunnelID,
		AgentID:    agentID,
		LocalHost:  "0.0.0.0",
		LocalPort:  0, // Agent will assign
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Protocol:   "tcp",
	}

	clientTunnel := &ClientTunnel{
		ID:        tunnelID,
		AgentID:   agentID,
		LocalPort: localPort,
		Target:    fmt.Sprintf("%s:%d", remoteHost, remotePort),
		Listener:  listener,
		Active:    false,
	}

	c.mu.Lock()
	c.tunnels[tunnelID] = clientTunnel
	c.mu.Unlock()

	// Send tunnel request
	msg := tunnel.NewMessage(tunnel.MsgTunnelRequest).
		SetTunnelID(tunnelID).
		SetData(req)

	if err := c.transport.SendMessage(msg); err != nil {
		listener.Close()
		c.mu.Lock()
		delete(c.tunnels, tunnelID)
		c.mu.Unlock()
		return fmt.Errorf("failed to send tunnel request: %w", err)
	}

	// Start accepting connections
	go c.handleLocalConnections(clientTunnel)

	c.logger.Tunnel("REQUESTED", tunnelID, fmt.Sprintf("local:%d -> %s via %s", localPort, clientTunnel.Target, agentID))
	return nil
}

func (c *TunnelClient) handleLocalConnections(clientTunnel *ClientTunnel) {
	defer func() {
		clientTunnel.Listener.Close()
		c.mu.Lock()
		delete(c.tunnels, clientTunnel.ID)
		c.mu.Unlock()
		c.logger.Tunnel("CLOSED", clientTunnel.ID, "local listener stopped")
	}()

	for {
		conn, err := clientTunnel.Listener.Accept()
		if err != nil {
			c.logger.Debug("Accept error for tunnel %s: %v", clientTunnel.ID, err)
			return
		}

		if !clientTunnel.Active {
			c.logger.Warn("Connection rejected - tunnel not active: %s", clientTunnel.ID)
			conn.Close()
			continue
		}

		connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
		c.logger.Debug("New local connection [%s] for tunnel %s", connID, clientTunnel.ID)

		go c.handleLocalConnection(clientTunnel, connID, conn)
	}
}

func (c *TunnelClient) handleLocalConnection(clientTunnel *ClientTunnel, connID string, localConn net.Conn) {
	defer func() {
		localConn.Close()
		c.logger.Debug("Local connection [%s] closed for tunnel %s", connID, clientTunnel.ID)
	}()

	// Open stream to agent via relay
	stream, err := c.transport.OpenStream()
	if err != nil {
		c.logger.Error("Failed to open stream for tunnel %s: %v", clientTunnel.ID, err)
		return
	}
	defer stream.Close()

	c.logger.Debug("Stream opened [%s] for tunnel %s", connID, clientTunnel.ID)

	// Bridge connections
	done := make(chan bool, 2)

	// Local -> Stream
	go func() {
		defer func() { done <- true }()
		written, err := io.Copy(stream, localConn)
		if err != nil {
			c.logger.Debug("Local->Stream copy error [%s]: %v", connID, err)
		}
		c.logger.Debug("Local->Stream [%s]: %d bytes", connID, written)
	}()

	// Stream -> Local
	go func() {
		defer func() { done <- true }()
		written, err := io.Copy(localConn, stream)
		if err != nil {
			c.logger.Debug("Stream->Local copy error [%s]: %v", connID, err)
		}
		c.logger.Debug("Stream->Local [%s]: %d bytes", connID, written)
	}()

	// Wait for one direction to close
	<-done
}

func (c *TunnelClient) closeTunnel(tunnelID string) {
	c.mu.Lock()
	clientTunnel, exists := c.tunnels[tunnelID]
	if exists {
		delete(c.tunnels, tunnelID)
	}
	c.mu.Unlock()

	if !exists {
		fmt.Printf("Tunnel not found: %s\n", tunnelID)
		return
	}

	// Close local listener
	if clientTunnel.Listener != nil {
		clientTunnel.Listener.Close()
	}

	// Send close request to relay
	msg := tunnel.NewMessage(tunnel.MsgTunnelClose).SetTunnelID(tunnelID)
	c.transport.SendMessage(msg)

	c.logger.Tunnel("CLOSED", tunnelID, "by user request")
	fmt.Printf("Tunnel closed: %s\n", tunnelID)
}

func (c *TunnelClient) listAgents() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.agents) == 0 {
		fmt.Println("No agents available")
		return
	}

	fmt.Println("\n📋 Available Agents:")
	fmt.Println("┌─────────────────────┬─────────────────────┬─────────────┬──────────┬─────────────────┐")
	fmt.Println("│ Agent ID            │ Name                │ Platform    │ Status   │ Last Seen       │")
	fmt.Println("├─────────────────────┼─────────────────────┼─────────────┼──────────┼─────────────────┤")

	for _, agent := range c.agents {
		lastSeen := time.Since(agent.LastSeen).Truncate(time.Second).String()
		fmt.Printf("│ %-19s │ %-19s │ %-11s │ %-8s │ %-15s │\n",
			agent.ID, agent.Name, agent.Platform, agent.Status, lastSeen)
	}

	fmt.Println("└─────────────────────┴─────────────────────┴─────────────┴──────────┴─────────────────┘")
}

func (c *TunnelClient) listTunnels() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.tunnels) == 0 {
		fmt.Println("No active tunnels")
		return
	}

	fmt.Println("\n🚇 Active Tunnels:")
	fmt.Println("┌─────────────────────┬─────────────────────┬─────────────┬─────────────────────┬──────────┐")
	fmt.Println("│ Tunnel ID           │ Agent ID            │ Local Port  │ Target              │ Status   │")
	fmt.Println("├─────────────────────┼─────────────────────┼─────────────┼─────────────────────┼──────────┤")

	for _, tunnel := range c.tunnels {
		status := "inactive"
		if tunnel.Active {
			status = "active"
		}
		
		fmt.Printf("│ %-19s │ %-19s │ %-11d │ %-19s │ %-8s │\n",
			tunnel.ID, tunnel.AgentID, tunnel.LocalPort, tunnel.Target, status)
	}

	fmt.Println("└─────────────────────┴─────────────────────┴─────────────┴─────────────────────┴──────────┘")
}

func (c *TunnelClient) printStatus() {
	c.mu.RLock()
	agentCount := len(c.agents)
	tunnelCount := len(c.tunnels)
	activeTunnels := 0
	for _, tunnel := range c.tunnels {
		if tunnel.Active {
			activeTunnels++
		}
	}
	c.mu.RUnlock()

	fmt.Printf("\n📊 Status:\n")
	fmt.Printf("  Connected agents: %d\n", agentCount)
	fmt.Printf("  Total tunnels: %d\n", tunnelCount)
	fmt.Printf("  Active tunnels: %d\n", activeTunnels)
}

func (c *TunnelClient) printHelp() {
	fmt.Println("\n🎮 Tunnel Client Commands:")
	fmt.Println("  agents              - List available agents")
	fmt.Println("  tunnels             - List active tunnels")
	fmt.Println("  create <local_port> <agent_id> <remote_host> <remote_port>")
	fmt.Println("                      - Create new tunnel")
	fmt.Println("  close <tunnel_id>   - Close specific tunnel")
	fmt.Println("  status              - Show connection status")
	fmt.Println("  refresh             - Refresh agent list")
	fmt.Println("  help                - Show this help")
	fmt.Println("  quit                - Exit client")
	fmt.Println()
}

func (c *TunnelClient) shutdown() {
	c.cancel()

	// Close all tunnels
	c.mu.Lock()
	for _, tunnel := range c.tunnels {
		if tunnel.Listener != nil {
			tunnel.Listener.Close()
		}
	}
	c.mu.Unlock()

	// Send disconnect message
	if c.transport != nil {
		msg := tunnel.NewMessage(tunnel.MsgClientDisconnect)
		c.transport.SendMessage(msg)
		time.Sleep(100 * time.Millisecond) // Give time for message to send
		c.transport.Close()
	}

	c.logger.Info("All tunnels closed")
}
