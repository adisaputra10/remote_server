package main

import (
	"crypto/tls"
	"encoding/json"
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
		
		// New port configuration options
		portRange    = flag.String("port-range", "", "Port range for auto allocation (e.g., 2222:2230)")
		autoPort     = flag.Bool("auto-port", false, "Automatically allocate available port")
		portMap      = flag.String("map", "", "Port mapping (e.g., 2222:22,3306:3306)")
		multiTunnels = flag.String("tunnels", "", "JSON file with multiple tunnel configurations")
		localHost    = flag.String("local-host", "127.0.0.1", "Local host to bind (default: 127.0.0.1)")
		startPort    = flag.Int("start-port", 2222, "Starting port for auto allocation")
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
	} else if *multiTunnels != "" {
		// Multiple tunnels from JSON file
		if err := client.setupMultipleTunnels(*multiTunnels); err != nil {
			log.Error("❌ Failed to setup multiple tunnels: %v", err)
			os.Exit(1)
		}
	} else if *portMap != "" {
		// Port mapping mode
		if err := client.setupPortMapping(*portMap, *agentID); err != nil {
			log.Error("❌ Failed to setup port mapping: %v", err)
			os.Exit(1)
		}
	} else if *localAddr != "" && *agentID != "" && *targetAddr != "" {
		// Direct tunnel mode with optional auto-port
		localAddress := *localAddr
		if *autoPort {
			localAddress = client.findAvailablePort(*localHost, *startPort)
		}
		
		if err := client.createTunnel(localAddress, *agentID, *targetAddr); err != nil {
			log.Error("❌ Failed to create tunnel: %v", err)
			os.Exit(1)
		}
	} else if *portRange != "" && *agentID != "" && *targetAddr != "" {
		// Port range mode
		if err := client.setupPortRange(*portRange, *agentID, *targetAddr); err != nil {
			log.Error("❌ Failed to setup port range: %v", err)
			os.Exit(1)
		}
	} else if *autoPort && *agentID != "" && *targetAddr != "" {
		// Auto-port mode
		localAddress := client.findAvailablePort(*localHost, *startPort)
		if err := client.createTunnel(localAddress, *agentID, *targetAddr); err != nil {
			log.Error("❌ Failed to create tunnel: %v", err)
			os.Exit(1)
		}
	} else {
		// Show enhanced usage
		client.showUsage()
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

// New enhanced port management functions

func (c *TunnelClient) findAvailablePort(host string, startPort int) string {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf("%s:%d", host, port)
		if c.isPortAvailable(addr) {
			c.logger.Info("🔍 Found available port: %s", addr)
			return addr
		}
	}
	// Fallback to :0 for OS allocation
	c.logger.Warn("⚠️ No port found in range, using OS allocation")
	return fmt.Sprintf("%s:0", host)
}

func (c *TunnelClient) isPortAvailable(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func (c *TunnelClient) setupPortMapping(portMap, agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID required for port mapping")
	}

	mappings := strings.Split(portMap, ",")
	for _, mapping := range mappings {
		parts := strings.Split(strings.TrimSpace(mapping), ":")
		if len(parts) != 2 {
			c.logger.Warn("⚠️ Invalid port mapping: %s (expected format: localport:remoteport)", mapping)
			continue
		}

		localPort := strings.TrimSpace(parts[0])
		remotePort := strings.TrimSpace(parts[1])
		
		localAddr := fmt.Sprintf(":%s", localPort)
		targetAddr := fmt.Sprintf("127.0.0.1:%s", remotePort)

		c.logger.Info("🔀 Setting up port mapping: %s -> %s", localAddr, targetAddr)
		
		if err := c.createTunnel(localAddr, agentID, targetAddr); err != nil {
			c.logger.Error("❌ Failed to create tunnel for mapping %s: %v", mapping, err)
			continue
		}
	}

	// Wait for interrupt
	c.waitForInterrupt()
	return nil
}

func (c *TunnelClient) setupPortRange(portRange, agentID, targetAddr string) error {
	parts := strings.Split(portRange, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid port range format (expected start:end, got %s)", portRange)
	}

	startPort, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid start port: %v", err)
	}

	endPort, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid end port: %v", err)
	}

	if startPort >= endPort {
		return fmt.Errorf("start port must be less than end port")
	}

	c.logger.Info("🔢 Setting up port range: %d-%d for target %s", startPort, endPort, targetAddr)

	// Create tunnels for the port range
	for port := startPort; port <= endPort; port++ {
		localAddr := fmt.Sprintf(":%d", port)
		
		c.logger.Info("🚇 Creating tunnel: %s -> %s via %s", localAddr, targetAddr, agentID)
		
		if err := c.createTunnel(localAddr, agentID, targetAddr); err != nil {
			c.logger.Error("❌ Failed to create tunnel for port %d: %v", port, err)
			continue
		}
	}

	// Wait for interrupt
	c.waitForInterrupt()
	return nil
}

type TunnelConfig struct {
	Name       string `json:"name"`
	LocalAddr  string `json:"local_addr"`
	AgentID    string `json:"agent_id"`
	TargetAddr string `json:"target_addr"`
	AutoPort   bool   `json:"auto_port"`
}

type MultiTunnelConfig struct {
	Tunnels []TunnelConfig `json:"tunnels"`
}

func (c *TunnelClient) setupMultipleTunnels(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config MultiTunnelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	c.logger.Info("🔧 Setting up %d tunnels from config", len(config.Tunnels))

	for _, tunnelConfig := range config.Tunnels {
		localAddr := tunnelConfig.LocalAddr
		if tunnelConfig.AutoPort {
			host := "127.0.0.1"
			if strings.Contains(localAddr, ":") {
				parts := strings.Split(localAddr, ":")
				if len(parts) > 0 && parts[0] != "" {
					host = parts[0]
				}
			}
			localAddr = c.findAvailablePort(host, 2222)
		}

		c.logger.Info("🚇 Creating tunnel '%s': %s -> %s via %s", 
			tunnelConfig.Name, localAddr, tunnelConfig.TargetAddr, tunnelConfig.AgentID)

		if err := c.createTunnel(localAddr, tunnelConfig.AgentID, tunnelConfig.TargetAddr); err != nil {
			c.logger.Error("❌ Failed to create tunnel '%s': %v", tunnelConfig.Name, err)
			continue
		}
	}

	// Wait for interrupt
	c.waitForInterrupt()
	return nil
}

func (c *TunnelClient) waitForInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}

func (c *TunnelClient) showUsage() {
	fmt.Println("\n🚇 Tunnel Client - Enhanced Port Configuration")
	fmt.Println("Usage modes:")
	fmt.Println("")
	fmt.Println("1️⃣  Interactive mode:")
	fmt.Println("   tunnel-client -relay-url ws://server:8080/ws/client -i")
	fmt.Println("")
	fmt.Println("2️⃣  Direct tunnel:")
	fmt.Println("   tunnel-client -L :2222 -agent agent_id -target 127.0.0.1:22")
	fmt.Println("")
	fmt.Println("3️⃣  Auto-port allocation:")
	fmt.Println("   tunnel-client -auto-port -agent agent_id -target 127.0.0.1:22")
	fmt.Println("   tunnel-client -auto-port -start-port 3000 -agent agent_id -target 127.0.0.1:22")
	fmt.Println("")
	fmt.Println("4️⃣  Port mapping (multiple ports):")
	fmt.Println("   tunnel-client -map '2222:22,3306:3306,5432:5432' -agent agent_id")
	fmt.Println("")
	fmt.Println("5️⃣  Multiple tunnels from JSON config:")
	fmt.Println("   tunnel-client -tunnels config.json")
	fmt.Println("")
	fmt.Println("Config file example (config.json):")
	fmt.Println(`{
  "tunnels": [
    {
      "name": "SSH",
      "local_addr": ":2222",
      "agent_id": "prod-server",
      "target_addr": "127.0.0.1:22"
    },
    {
      "name": "MySQL",
      "local_addr": ":3306", 
      "agent_id": "db-server",
      "target_addr": "127.0.0.1:3306",
      "auto_port": true
    }
  ]
}`)
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
}
