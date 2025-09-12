package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"remote-tunnel/internal/logger"
	"remote-tunnel/internal/tunnel"
)

type TunnelClient struct {
	relayURL     string
	logger       *logger.Logger
	insecure     bool
	transport    *tunnel.Transport
	tunnels      map[string]*tunnel.ClientTunnel
	agents       []*tunnel.AgentInfo
}

func main() {
	var (
		relayURL     = flag.String("relay-url", "wss://localhost:8443/ws/client", "Relay server WebSocket URL")
		insecure     = flag.Bool("insecure", false, "Skip TLS certificate verification")
		localAddr    = flag.String("L", "", "Local address to bind (e.g., :2222)")
		agentID      = flag.String("agent", "", "Target agent ID")
		targetAddr   = flag.String("target", "", "Target address (e.g., 127.0.0.1:22)")
		interactive  = flag.Bool("i", false, "Interactive mode")
	)
	flag.Parse()

	log := logger.New("CLIENT")
	
	if *insecure {
		log.Warn("🔓 INSECURE mode enabled - TLS certificate verification disabled!")
	}

	client := &TunnelClient{
		relayURL: *relayURL,
		logger:   log,
		insecure: *insecure,
		tunnels:  make(map[string]*tunnel.ClientTunnel),
	}

	log.Info("🚀 Starting tunnel client")
	log.Info("📋 Relay URL: %s", *relayURL)

	// Connect to relay
	if err := client.connect(); err != nil {
		log.Error("❌ Failed to connect to relay: %v", err)
		os.Exit(1)
	}

	// Get agent list
	if err := client.getAgents(); err != nil {
		log.Error("❌ Failed to get agents: %v", err)
		os.Exit(1)
	}

	if *interactive {
		// Interactive mode
		client.interactiveMode()
	} else if *localAddr != "" && *agentID != "" && *targetAddr != "" {
		// Direct tunnel mode
		if err := client.createTunnel(*localAddr, *agentID, *targetAddr); err != nil {
			log.Error("❌ Failed to create tunnel: %v", err)
			os.Exit(1)
		}

		// Wait for interrupt
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
	} else {
		// Show usage
		fmt.Println("Usage:")
		fmt.Println("  Interactive mode: -i")
		fmt.Println("  Direct tunnel:    -L :2222 -agent agent_id -target 127.0.0.1:22")
		flag.Usage()
		os.Exit(1)
	}

	log.Info("🛑 Shutting down client...")
	client.disconnect()
	log.Info("👋 Client stopped")
}

func (c *TunnelClient) connect() error {
	c.logger.Info("🔗 Connecting to relay server...")

	// Setup WebSocket dialer
	dialer := websocket.DefaultDialer
	if c.insecure {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		c.logger.Warn("⚠️ TLS certificate verification disabled")
	}

	// Connect to WebSocket
	conn, _, err := dialer.Dial(c.relayURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket dial failed: %v", err)
	}

	c.logger.Info("✅ Connected to relay server")

	// Create transport
	transport, err := tunnel.NewTransport(conn, false, c.logger)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create transport: %v", err)
	}

	c.transport = transport

	// Send client connect message
	msg := tunnel.NewMessage(tunnel.MsgClientConnect)
	if err := c.transport.SendMessage(msg); err != nil {
		c.transport.Close()
		return fmt.Errorf("failed to send connect message: %v", err)
	}

	c.logger.Info("✅ Client connected to relay")
	return nil
}

func (c *TunnelClient) getAgents() error {
	c.logger.Command("SEND", "GET_AGENTS", "")

	// Receive agent list
	msg, err := c.transport.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("failed to receive agent list: %v", err)
	}

	c.logger.Command("RECV", msg.Type, "")

	if msg.Type != tunnel.MsgAgentList {
		return fmt.Errorf("unexpected response: %s", msg.Type)
	}

	// Parse agent list
	if data, ok := msg.Data.([]interface{}); ok {
		c.agents = make([]*tunnel.AgentInfo, 0, len(data))
		for _, item := range data {
			if agentData, ok := item.(map[string]interface{}); ok {
				var agent tunnel.AgentInfo
				tunnel.MapToStruct(agentData, &agent)
				c.agents = append(c.agents, &agent)
			}
		}
	}

	c.logger.Info("📋 Found %d agents", len(c.agents))
	for _, agent := range c.agents {
		c.logger.Info("  - %s (%s) - %s", agent.Name, agent.ID, agent.Status)
	}

	return nil
}

func (c *TunnelClient) interactiveMode() {
	fmt.Println("\n🚀 Tunnel Client Interactive Mode")
	fmt.Println("Available commands:")
	fmt.Println("  agents                           - List available agents")
	fmt.Println("  tunnel <agent_id> <local> <target> - Create tunnel")
	fmt.Println("  tunnels                          - List active tunnels")
	fmt.Println("  close <tunnel_id>                - Close tunnel")
	fmt.Println("  quit                             - Exit")
	fmt.Println()

	for {
		fmt.Print("tunnel> ")
		
		var input string
		fmt.Scanln(&input)
		
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		c.logger.Command("USER", cmd, strings.Join(parts[1:], " "))

		switch cmd {
		case "agents":
			c.showAgents()

		case "tunnel":
			if len(parts) != 4 {
				fmt.Println("Usage: tunnel <agent_id> <local_addr> <target_addr>")
				fmt.Println("Example: tunnel agent_123 :2222 127.0.0.1:22")
				continue
			}
			agentID := parts[1]
			localAddr := parts[2]
			targetAddr := parts[3]
			
			if err := c.createTunnel(localAddr, agentID, targetAddr); err != nil {
				fmt.Printf("❌ Failed to create tunnel: %v\n", err)
			}

		case "tunnels":
			c.showTunnels()

		case "close":
			if len(parts) != 2 {
				fmt.Println("Usage: close <tunnel_id>")
				continue
			}
			tunnelID := parts[1]
			c.closeTunnel(tunnelID)

		case "quit", "exit":
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}

func (c *TunnelClient) showAgents() {
	if len(c.agents) == 0 {
		fmt.Println("No agents available")
		return
	}

	fmt.Println("\n📋 Available Agents:")
	fmt.Println("┌────────────────────┬────────────────────┬──────────┬────────────────────┐")
	fmt.Println("│ Agent ID           │ Name               │ Status   │ Last Seen          │")
	fmt.Println("├────────────────────┼────────────────────┼──────────┼────────────────────┤")
	
	for _, agent := range c.agents {
		lastSeen := agent.LastSeen.Format("15:04:05")
		fmt.Printf("│ %-18s │ %-18s │ %-8s │ %-18s │\n", 
			agent.ID, agent.Name, agent.Status, lastSeen)
	}
	
	fmt.Println("└────────────────────┴────────────────────┴──────────┴────────────────────┘")
}

func (c *TunnelClient) showTunnels() {
	if len(c.tunnels) == 0 {
		fmt.Println("No active tunnels")
		return
	}

	fmt.Println("\n🚇 Active Tunnels:")
	fmt.Println("┌────────────────────┬────────────────────┬────────────────────┬──────────┐")
	fmt.Println("│ Tunnel ID          │ Local Address      │ Target Address     │ Status   │")
	fmt.Println("├────────────────────┼────────────────────┼────────────────────┼──────────┤")
	
	for _, tunnel := range c.tunnels {
		status := "Active"
		if !tunnel.IsActive() {
			status = "Closed"
		}
		fmt.Printf("│ %-18s │ %-18s │ %-18s │ %-8s │\n", 
			tunnel.ID, tunnel.LocalAddr, tunnel.TargetAddr, status)
	}
	
	fmt.Println("└────────────────────┴────────────────────┴────────────────────┴──────────┘")
}

func (c *TunnelClient) createTunnel(localAddr, agentID, targetAddr string) error {
	// Parse target address
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return fmt.Errorf("invalid target address: %v", err)
	}
	
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %v", err)
	}

	// Generate tunnel ID
	tunnelID := fmt.Sprintf("tunnel_%d", time.Now().UnixNano())

	c.logger.Tunnel("CREATE", tunnelID, fmt.Sprintf("%s -> %s:%d via %s", localAddr, host, port, agentID))

	// Create tunnel request
	req := &tunnel.TunnelRequest{
		TunnelID:   tunnelID,
		AgentID:    agentID,
		RemoteHost: host,
		RemotePort: port,
	}

	// Send tunnel request
	msg := tunnel.NewMessage(tunnel.MsgTunnelRequest).
		SetTunnelID(tunnelID).
		SetData(req)

	if err := c.transport.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send tunnel request: %v", err)
	}

	// Wait for response
	response, err := c.transport.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("failed to receive tunnel response: %v", err)
	}

	c.logger.Command("RECV", response.Type, response.TunnelID)

	if response.Type == tunnel.MsgTunnelError {
		return fmt.Errorf("tunnel error: %v", response.Error)
	}

	if response.Type != tunnel.MsgTunnelSuccess {
		return fmt.Errorf("unexpected response: %s", response.Type)
	}

	// Create local listener
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", localAddr, err)
	}

	actualAddr := listener.Addr().String()
	c.logger.Info("✅ Tunnel created: %s -> %s via %s", actualAddr, targetAddr, agentID)

	// Create client tunnel
	clientTunnel := tunnel.NewClientTunnel(tunnelID, actualAddr, targetAddr, listener, c.transport, c.logger)
	c.tunnels[tunnelID] = clientTunnel

	// Start tunnel
	go clientTunnel.Start()

	fmt.Printf("✅ Tunnel created: %s -> %s via %s\n", actualAddr, targetAddr, agentID)
	return nil
}

func (c *TunnelClient) closeTunnel(tunnelID string) {
	clientTunnel, exists := c.tunnels[tunnelID]
	if !exists {
		fmt.Printf("❌ Tunnel not found: %s\n", tunnelID)
		return
	}

	c.logger.Tunnel("CLOSE", tunnelID, "user request")

	// Send close message to relay
	msg := tunnel.NewMessage(tunnel.MsgTunnelClose).SetTunnelID(tunnelID)
	c.transport.SendMessage(msg)

	// Close tunnel
	clientTunnel.Close()
	delete(c.tunnels, tunnelID)

	fmt.Printf("✅ Tunnel closed: %s\n", tunnelID)
}

func (c *TunnelClient) disconnect() {
	// Close all tunnels
	for tunnelID, clientTunnel := range c.tunnels {
		clientTunnel.Close()
		delete(c.tunnels, tunnelID)
	}

	// Close transport
	if c.transport != nil {
		msg := tunnel.NewMessage(tunnel.MsgClientDisconnect)
		c.transport.SendMessage(msg)
		c.transport.Close()
	}
}
