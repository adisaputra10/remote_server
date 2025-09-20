package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"ssh-tunnel/internal/common"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// Configuration structure
type Config struct {
	RelayURL    string `json:"relay_url"`
	DefaultUser string `json:"default_user,omitempty"`
	LocalPort   string `json:"local_port,omitempty"`
}

// Universal client that can work in both tunnel mode and integrated SSH mode
type UniversalClient struct {
	// Common fields
	id       string
	name     string
	relayURL string
	agentID  string
	logger   *common.Logger
	running  bool

	// Tunnel mode fields
	sessions  map[string]net.Conn
	targets   map[string]string
	agentIDs  map[string]string
	conn      *websocket.Conn
	heartbeat *time.Ticker
	mutex     sync.RWMutex

	// SSH mode fields
	localPort   string
	sshHost     string
	sshUser     string
	sshPassword string
	sshClient   *ssh.Client
	tunnelConn  *websocket.Conn
	sessionID   string
	httpClient  *http.Client
}

// SSH log request structure
type UniversalSSHLogRequest struct {
	SessionID string `json:"session_id"`
	ClientID  string `json:"client_id"`
	AgentID   string `json:"agent_id"`
	Direction string `json:"direction"`
	User      string `json:"user"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	Command   string `json:"command"`
	Data      string `json:"data"`
}

// Load configuration from file, environment, or defaults
func loadConfig() Config {
	config := Config{
		RelayURL:    "ws://168.231.119.242:8080/ws/client", // Default
		DefaultUser: "root",
		LocalPort:   "2222",
	}

	// Try to load from config.json file
	if configFile, err := os.ReadFile("config.json"); err == nil {
		if err := json.Unmarshal(configFile, &config); err == nil {
			fmt.Printf("📋 Configuration loaded from config.json\n")
		}
	}

	// Override with environment variables if they exist
	if envRelayURL := os.Getenv("RELAY_URL"); envRelayURL != "" {
		config.RelayURL = envRelayURL
		fmt.Printf("🌍 Relay URL loaded from environment: %s\n", "ENV")
	}

	if envUser := os.Getenv("SSH_USER"); envUser != "" {
		config.DefaultUser = envUser
	}

	if envPort := os.Getenv("LOCAL_PORT"); envPort != "" {
		config.LocalPort = envPort
	}

	return config
}

// Get display name for relay URL (show ENV if from environment)
func getRelayDisplayName(relayURL string) string {
	if os.Getenv("RELAY_URL") != "" {
		return "ENV"
	}

	// Check if using config file
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}

	// Return the actual URL without ws:// prefix
	return strings.Replace(relayURL, "ws://", "", 1)
}

func main() {
	// Load configuration from file, environment, or defaults
	config := loadConfig()

	var (
		clientID   string
		clientName string
		relayURL   string
		agentID    string

		// Tunnel mode parameters
		localAddr   string
		target      string
		interactive bool

		// SSH mode parameters
		sshUser     string
		sshPassword string
		sshHost     string
		localPort   string
		tunnelOnly  bool
	)

	var rootCmd = &cobra.Command{
		Use:   "universal-ssh-client",
		Short: "Universal SSH Client - Tunnel or Integrated Mode",
		Long:  "A universal client that can work as tunnel client (with -L) or integrated SSH client (without -L)",
		Run: func(cmd *cobra.Command, args []string) {
			// Set defaults
			if clientID == "" {
				clientID = common.GenerateID()
			}
			if clientName == "" {
				clientName = fmt.Sprintf("client-%s", clientID[:8])
			}
			if sshHost == "" {
				sshHost = "127.0.0.1"
			}
			if localPort == "" {
				localPort = "2222"
			}

			client := &UniversalClient{
				id:          clientID,
				name:        clientName,
				relayURL:    relayURL,
				agentID:     agentID,
				logger:      common.NewLogger(fmt.Sprintf("UNIVERSAL-%s", clientID)),
				sessions:    make(map[string]net.Conn),
				targets:     make(map[string]string),
				agentIDs:    make(map[string]string),
				localPort:   localPort,
				sshHost:     sshHost,
				sshUser:     sshUser,
				sshPassword: sshPassword,
				httpClient:  &http.Client{Timeout: 5 * time.Second},
			}

			// Setup signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				client.logger.Info("Received shutdown signal, stopping client...")
				client.stop()
				os.Exit(0)
			}()

			// Check mode based on -L flag
			if localAddr != "" {
				// TUNNEL MODE
				fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
				fmt.Println("║                    🔗 Tunnel Client Mode                         ║")
				fmt.Println("╚════════════════════════════════════════════════════════════════════╝")

				if err := client.runTunnelMode(localAddr, target, interactive); err != nil {
					log.Fatalf("Tunnel mode failed: %v", err)
				}
			} else {
				// INTEGRATED SSH MODE
				fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
				fmt.Println("║              🚀 SSH Client Mode                       ║")
				fmt.Println("╚════════════════════════════════════════════════════════════════════╝")

				// If username is empty, prompt for it
				if client.sshUser == "" {
					client.sshUser = promptUsername()
				}

				// If password is empty, prompt for it
				if client.sshPassword == "" {
					client.sshPassword = promptPassword(client.sshUser, client.sshHost)
				}

				if err := client.runIntegratedMode(tunnelOnly); err != nil {
					log.Fatalf("Integrated mode failed: %v", err)
				}
			}
		},
	}

	// Define flags
	rootCmd.Flags().StringVarP(&clientID, "client-id", "c", "", "Client ID")
	rootCmd.Flags().StringVarP(&clientName, "name", "n", "", "Client name")
	rootCmd.Flags().StringVarP(&relayURL, "relay-url", "r", config.RelayURL, "Relay server WebSocket URL (default from config.json, RELAY_URL env, or built-in)")
	rootCmd.Flags().StringVarP(&agentID, "agent", "a", "", "Target agent ID")

	// Tunnel mode flags
	rootCmd.Flags().StringVarP(&localAddr, "local", "L", "", "Local address for TUNNEL MODE (e.g., :2222)")
	rootCmd.Flags().StringVarP(&target, "target", "t", "", "Target address for TUNNEL MODE (e.g., localhost:22)")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode for TUNNEL MODE")

	// SSH mode flags
	rootCmd.Flags().StringVarP(&sshUser, "ssh-user", "u", "", "SSH username for INTEGRATED MODE (interactive if empty)")
	rootCmd.Flags().StringVarP(&sshPassword, "ssh-password", "P", "", "SSH password for INTEGRATED MODE")
	rootCmd.Flags().StringVarP(&sshHost, "ssh-host", "H", "127.0.0.1", "SSH host for INTEGRATED MODE")
	rootCmd.Flags().StringVarP(&localPort, "local-port", "p", config.LocalPort, "Local tunnel port for INTEGRATED MODE")
	rootCmd.Flags().BoolVarP(&tunnelOnly, "tunnel-only", "x", false, "Only create tunnel for INTEGRATED MODE")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// ================ TUNNEL MODE IMPLEMENTATION ================

func (c *UniversalClient) runTunnelMode(localAddr, target string, interactive bool) error {
	if err := c.connectToRelay(); err != nil {
		return fmt.Errorf("failed to connect to relay: %v", err)
	}

	// Start heartbeat and message handling
	c.startTunnelServices()

	if interactive {
		return c.runTunnelInteractive()
	} else {
		if localAddr == "" || c.agentID == "" || target == "" {
			return fmt.Errorf("local address (-L), agent ID (-a), and target (-t) are required")
		}
		fmt.Printf("🚀 Starting tunnel: %s -> %s -> %s\n", localAddr, c.agentID, target)
		return c.startTunnelListener(localAddr, target)
	}
}

func (c *UniversalClient) connectToRelay() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	c.running = true

	// Register with relay
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.ClientID = c.id
	registerMsg.ClientName = c.name
	return c.sendMessage(registerMsg)
}

func (c *UniversalClient) startTunnelServices() {
	// Start heartbeat
	c.heartbeat = time.NewTicker(30 * time.Second)
	go func() {
		for c.running {
			select {
			case <-c.heartbeat.C:
				heartbeatMsg := common.NewMessage(common.MsgTypeHeartbeat)
				heartbeatMsg.ClientID = c.id
				c.sendMessage(heartbeatMsg)
			}
		}
	}()

	// Start message handler
	go c.tunnelMessageLoop()
}

func (c *UniversalClient) tunnelMessageLoop() {
	defer c.stop()
	for c.running {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if c.running {
				c.logger.Error("Failed to read message: %v", err)
			}
			break
		}

		message, err := common.FromJSON(messageData)
		if err != nil {
			c.logger.Error("Failed to parse message: %v", err)
			continue
		}

		c.handleTunnelMessage(message)
	}
}

func (c *UniversalClient) handleTunnelMessage(msg *common.Message) {
	switch msg.Type {
	case common.MsgTypeData:
		c.mutex.RLock()
		conn, exists := c.sessions[msg.SessionID]
		c.mutex.RUnlock()

		if exists {
			conn.Write(msg.Data)
		}
	case common.MsgTypeClose:
		c.closeTunnelSession(msg.SessionID)
	case common.MsgTypeError:
		c.logger.Error("Received error: %s", msg.Error)
		if msg.SessionID != "" {
			c.closeTunnelSession(msg.SessionID)
		}
	}
}

func (c *UniversalClient) startTunnelListener(localAddr, target string) error {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	c.logger.Info("Tunnel listening on %s -> %s -> %s", localAddr, c.agentID, target)
	fmt.Printf("✅ Tunnel established: %s -> %s -> %s\n", localAddr, c.agentID, target)

	// Send tunnel listening notification to relay server for logging
	go c.sendTunnelListeningLog(localAddr, target)

	for {
		conn, err := listener.Accept()
		if err != nil {
			c.logger.Error("Accept failed: %v", err)
			continue
		}
		go c.handleTunnelConnection(conn, target)
	}
}

func (c *UniversalClient) handleTunnelConnection(conn net.Conn, target string) {
	sessionID := common.GenerateID()

	c.mutex.Lock()
	c.sessions[sessionID] = conn
	c.targets[sessionID] = target
	c.agentIDs[sessionID] = c.agentID
	c.mutex.Unlock()

	// Send connect request
	connectMsg := common.NewMessage(common.MsgTypeConnect)
	connectMsg.SessionID = sessionID
	connectMsg.ClientID = c.id
	connectMsg.AgentID = c.agentID
	connectMsg.Target = target

	if err := c.sendMessage(connectMsg); err != nil {
		conn.Close()
		c.closeTunnelSession(sessionID)
		return
	}

	// Forward data
	go c.forwardTunnelData(sessionID, conn)
}

func (c *UniversalClient) sendTunnelListeningLog(localAddr, target string) {
	// Send log message to relay server for connection logging
	logMsg := common.NewMessage(common.MsgTypeData)
	logMsg.ClientID = c.id
	logMsg.AgentID = c.agentID
	logMsg.Target = target
	logMsg.Data = []byte(fmt.Sprintf("tunnel_listening:%s->%s->%s", localAddr, c.agentID, target))
	
	if err := c.sendMessage(logMsg); err != nil {
		c.logger.Error("Failed to send tunnel listening log: %v", err)
	}
}

func (c *UniversalClient) forwardTunnelData(sessionID string, conn net.Conn) {
	defer c.closeTunnelSession(sessionID)
	buffer := make([]byte, 32*1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			break
		}

		dataMsg := common.NewMessage(common.MsgTypeData)
		dataMsg.SessionID = sessionID
		dataMsg.ClientID = c.id
		dataMsg.Data = make([]byte, n)
		copy(dataMsg.Data, buffer[:n])

		if err := c.sendMessage(dataMsg); err != nil {
			break
		}
	}

	// Send close message
	closeMsg := common.NewMessage(common.MsgTypeClose)
	closeMsg.SessionID = sessionID
	closeMsg.ClientID = c.id
	c.sendMessage(closeMsg)
}

func (c *UniversalClient) runTunnelInteractive() error {
	fmt.Println("🔗 Interactive Tunnel Mode")
	fmt.Println("Commands: tunnel, list, quit")

	for {
		fmt.Print("> ")
		var command string
		fmt.Scanln(&command)

		switch command {
		case "tunnel":
			var localAddr, agentID, target string
			fmt.Print("Local address: ")
			fmt.Scanln(&localAddr)
			fmt.Print("Agent ID: ")
			fmt.Scanln(&agentID)
			fmt.Print("Target: ")
			fmt.Scanln(&target)

			c.agentID = agentID
			go c.startTunnelListener(localAddr, target)
			fmt.Printf("✅ Tunnel created: %s -> %s -> %s\n", localAddr, agentID, target)
		case "list":
			c.mutex.RLock()
			fmt.Printf("Active sessions: %d\n", len(c.sessions))
			c.mutex.RUnlock()
		case "quit":
			return nil
		default:
			fmt.Println("Unknown command")
		}
	}
}

func (c *UniversalClient) closeTunnelSession(sessionID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if conn, exists := c.sessions[sessionID]; exists {
		conn.Close()
		delete(c.sessions, sessionID)
	}
	delete(c.targets, sessionID)
	delete(c.agentIDs, sessionID)
}

func (c *UniversalClient) sendMessage(msg *common.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// ================ INTEGRATED SSH MODE IMPLEMENTATION ================

func (c *UniversalClient) runIntegratedMode(tunnelOnly bool) error {
	// Create tunnel
	if err := c.createSSHTunnel(); err != nil {
		return fmt.Errorf("failed to create tunnel: %v", err)
	}

	if tunnelOnly {
		fmt.Printf("🔧 Tunnel-only mode: ssh %s@%s -p %s\n", c.sshUser, c.sshHost, c.localPort)
		select {} // Keep alive
	}

	// Auto-connect SSH with retry
	time.Sleep(500 * time.Millisecond) // Brief wait for tunnel

	if err := c.connectSSHWithRetry(); err != nil {
		//fmt.Printf("💡 Manual: ssh %s@%s -p %s\n", c.sshUser, c.sshHost, c.localPort)
		select {} // Keep tunnel alive
	}

	return c.startSSHSession()
}

func (c *UniversalClient) createSSHTunnel() error {
	// Connect to relay
	conn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		return err
	}
	c.tunnelConn = conn

	// Register
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.ClientID = c.id
	registerMsg.ClientName = c.name
	if err := c.tunnelConn.WriteJSON(registerMsg); err != nil {
		return err
	}

	// Request tunnel
	c.sessionID = fmt.Sprintf("ssh_%d", time.Now().UnixNano())
	connectMsg := common.NewMessage(common.MsgTypeConnect)
	connectMsg.SessionID = c.sessionID
	connectMsg.ClientID = c.id
	connectMsg.AgentID = c.agentID
	connectMsg.Target = "127.0.0.1:22"

	if err := c.tunnelConn.WriteJSON(connectMsg); err != nil {
		return err
	}

	// Start tunnel listener
	go c.startSSHTunnelListener()
	return nil
}

func (c *UniversalClient) startSSHTunnelListener() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", c.localPort))
	if err != nil {
		c.logger.Error("SSH tunnel listener failed: %v", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go c.handleSSHTunnelConnection(conn)
	}
}

func (c *UniversalClient) handleSSHTunnelConnection(conn net.Conn) {
	defer conn.Close()

	// Forward data between local connection and tunnel
	done := make(chan bool, 2)

	// Local -> Tunnel
	go func() {
		defer func() { done <- true }()
		buffer := make([]byte, 4096)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				return
			}

			dataMsg := common.NewMessage(common.MsgTypeData)
			dataMsg.SessionID = c.sessionID
			dataMsg.ClientID = c.id
			dataMsg.Data = buffer[:n]
			if err := c.tunnelConn.WriteJSON(dataMsg); err != nil {
				return
			}
		}
	}()

	// Tunnel -> Local
	go func() {
		defer func() { done <- true }()
		for {
			var msg common.Message
			if err := c.tunnelConn.ReadJSON(&msg); err != nil {
				return
			}
			if msg.Type == common.MsgTypeData && msg.SessionID == c.sessionID {
				conn.Write(msg.Data)
			}
		}
	}()

	<-done
}

func (c *UniversalClient) connectSSHWithRetry() error {
	maxAttempts := 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := c.connectSSH()
		if err == nil {
			return nil // Success
		}

		// Check if it's an authentication error
		if strings.Contains(err.Error(), "unable to authenticate") ||
			strings.Contains(err.Error(), "handshake failed") ||
			strings.Contains(err.Error(), "authentication failed") {

			if attempt < maxAttempts {
				fmt.Printf("🔐 Authentication failed. Try again Enter Password (%d/%d): ", attempt+1, maxAttempts)
				// Prompt for password again
				c.sshPassword = promptPasswordRetry(c.sshUser, c.sshHost)
			} else {
				fmt.Printf("❌ Authentication failed after %d attempts\n", maxAttempts)
				return err
			}
		} else {
			// Non-authentication error, don't retry
			return err
		}
	}

	return fmt.Errorf("authentication failed after %d attempts", maxAttempts)
}

func (c *UniversalClient) connectSSH() error {
	config := &ssh.ClientConfig{
		User: c.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", c.sshHost, c.localPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}

	c.sshClient = client
	return nil
}

func (c *UniversalClient) startSSHSession() error {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Setup terminal
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	if err := session.RequestPty("xterm", 80, 24, ssh.TerminalModes{}); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	// Handle input with logging
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			command := scanner.Text()
			c.logCommand(command)
			stdin.Write([]byte(command + "\n"))
		}
	}()

	session.Wait()
	fmt.Println("\n🔚 SSH session ended")
	return nil
}

func (c *UniversalClient) logCommand(command string) {
	// Create logs directory
	os.MkdirAll("logs", 0755)

	// Log to file
	logFile := filepath.Join("logs", "commands.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [Client:%s] [Agent:%s] - %s\n", timestamp, c.id, c.agentID, command)
	file.WriteString(logEntry)

	// Log to relay (async)
	go c.sendSSHLogToRelay(command)
}

func (c *UniversalClient) sendSSHLogToRelay(command string) {
	logReq := UniversalSSHLogRequest{
		SessionID: c.sessionID,
		ClientID:  c.id,
		AgentID:   c.agentID,
		Direction: "command",
		User:      c.sshUser,
		Host:      c.sshHost,
		Port:      c.localPort,
		Command:   command,
		Data:      command,
	}

	jsonData, err := json.Marshal(logReq)
	if err != nil {
		return
	}

	// Build API URL
	apiURL := strings.Replace(c.relayURL, "ws://", "http://", 1)
	apiURL = strings.Replace(apiURL, "/ws/client", "/api/log-ssh", 1)

	// Send to relay (silent fail)
	resp, err := c.httpClient.Post(apiURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// ================ UTILITY FUNCTIONS ================

func promptUsername() string {
	fmt.Print("👤 Enter SSH username: ")
	var username string
	fmt.Scanln(&username)
	return username
}

func promptPassword(username, host string) string {
	fmt.Print("🔐 Enter password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("\nError reading password: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	return string(password)
}

func promptPasswordRetry(username, host string) string {
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("\nError reading password: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	return string(password)
}

func (c *UniversalClient) stop() {
	c.running = false

	if c.heartbeat != nil {
		c.heartbeat.Stop()
	}

	if c.conn != nil {
		c.conn.Close()
	}

	if c.tunnelConn != nil {
		c.tunnelConn.Close()
	}

	if c.sshClient != nil {
		c.sshClient.Close()
	}

	// Close all sessions
	c.mutex.Lock()
	for _, conn := range c.sessions {
		conn.Close()
	}
	c.mutex.Unlock()

	c.logger.Info("Universal client stopped")
}
