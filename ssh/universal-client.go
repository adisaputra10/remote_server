package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	token    string
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
	localPort    string
	sshHost      string
	sshUser      string
	sshPassword  string
	sshClient    *ssh.Client
	tunnelConn   *websocket.Conn
	sessionID    string
	httpClient   *http.Client
	lastCommand  string // Store last command for OUTPUT logging
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
	IsBase64  bool   `json:"is_base64"`
}

// CommandLoggingReader wraps os.Stdin to log commands
type CommandLoggingReader struct {
	reader *bufio.Reader
	logger *common.Logger
	buffer []byte
	client *UniversalClient // Reference to client for sending to database
}

// LoggingWriter wraps output streams to log all SSH session data
type LoggingWriter struct {
	writer   io.Writer
	logger   *common.Logger
	prefix   string
	client   *UniversalClient // Reference to client for sending to database
	original io.Writer        // Keep original writer for clean output
	buffer   *bytes.Buffer    // Buffer to accumulate data
	timer    *time.Timer      // Timer to send accumulated data
	mutex    sync.Mutex       // Mutex to protect buffer access
}

func NewLoggingWriter(writer io.Writer, logger *common.Logger, prefix string, client *UniversalClient) *LoggingWriter {
	return &LoggingWriter{
		writer:   writer,
		logger:   logger,
		prefix:   prefix,
		client:   client,
		original: writer,
		buffer:   &bytes.Buffer{},
	}
}

func (lw *LoggingWriter) Write(p []byte) (n int, err error) {
	// Process the data and accumulate in buffer
	if len(p) > 0 {
		rawData := string(p)
		
		// Protect buffer access with mutex
		lw.mutex.Lock()
		
		// Add data to buffer
		lw.buffer.Write(p)
		
		// Reset or start timer to send accumulated data after 100ms of inactivity
		if lw.timer != nil {
			lw.timer.Stop()
		}
		
		lw.timer = time.AfterFunc(100*time.Millisecond, func() {
			lw.flushBuffer()
		})
		
		// Log each line immediately to file for real-time monitoring
		lines := strings.Split(rawData, "\n")
		for i, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" && !strings.Contains(trimmedLine, "INFO:") {
				lw.logger.Info("%s: %s", lw.prefix, trimmedLine)
			} else if line != "" && i < len(lines)-1 {
				cleanLine := strings.TrimRight(line, "\r\n")
				if cleanLine != "" && !strings.Contains(cleanLine, "INFO:") {
					lw.logger.Info("%s: %s", lw.prefix, cleanLine)
				}
			}
		}
		
		lw.mutex.Unlock()
	}
	// Forward to original writer for clean display
	return lw.original.Write(p)
}

// flushBuffer sends accumulated buffer data to database as one unit
func (lw *LoggingWriter) flushBuffer() {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()
	
	if lw.buffer.Len() > 0 {
		// Get accumulated data
		data := strings.TrimSpace(lw.buffer.String())
		if data != "" && !strings.Contains(data, "INFO:") {
			// Send accumulated data as one unit to database
			go lw.sendToDatabase(data)
		}
		
		// Clear buffer
		lw.buffer.Reset()
	}
}

func (lw *LoggingWriter) sendToDatabase(data string) {
	if lw.client == nil || lw.client.tunnelConn == nil {
		return
	}
	
	// For OUTPUT, use the last command executed; for other directions, leave empty
	command := ""
	if lw.prefix == "OUTPUT" && lw.client != nil {
		command = lw.client.lastCommand
	}
	
	// Encode data to base64 for multi-line support
	encodedData := base64.StdEncoding.EncodeToString([]byte(data))
	
	logRequest := UniversalSSHLogRequest{
		SessionID: lw.client.sessionID,
		ClientID:  lw.client.id,
		AgentID:   lw.client.agentID,
		Direction: lw.prefix,
		User:      lw.client.sshUser,
		Host:      lw.client.sshHost,
		Port:      lw.client.localPort,
		Command:   command,
		Data:      encodedData,
		IsBase64:  true,
	}

	logMsg := common.NewMessage(common.MsgTypeSSHLog)
	logMsg.SessionID = lw.client.sessionID
	logMsg.ClientID = lw.client.id
	logMsg.AgentID = lw.client.agentID
	
	// Convert to JSON and send with mutex protection
	if logData, err := json.Marshal(logRequest); err == nil {
		logMsg.Data = logData
		
		// Use mutex to prevent concurrent writes to websocket
		lw.client.mutex.Lock()
		lw.client.tunnelConn.WriteJSON(logMsg)
		lw.client.mutex.Unlock()
	}
}

func NewCommandLoggingReader(logger *common.Logger, client *UniversalClient) *CommandLoggingReader {
	return &CommandLoggingReader{
		reader: bufio.NewReader(os.Stdin),
		logger: logger,
		buffer: make([]byte, 0),
		client: client,
	}
}

func (clr *CommandLoggingReader) Read(p []byte) (n int, err error) {
	n, err = clr.reader.Read(p)
	if n > 0 {
		// Log all input data
		data := string(p[:n])
		cleanData := strings.TrimSpace(strings.ReplaceAll(data, "\r", ""))
		if cleanData != "" && cleanData != "\n" {
			clr.logger.Info("INPUT: %s", cleanData)
			// Store last command in client
			if clr.client != nil {
				clr.client.lastCommand = cleanData
			}
			// Send to database
			go clr.sendToDatabase(cleanData)
		}
	}
	return n, err
}

func (clr *CommandLoggingReader) sendToDatabase(data string) {
	if clr.client == nil || clr.client.tunnelConn == nil {
		return
	}
	
	// Encode data to base64 for consistency
	encodedData := base64.StdEncoding.EncodeToString([]byte(data))
	
	logRequest := UniversalSSHLogRequest{
		SessionID: clr.client.sessionID,
		ClientID:  clr.client.id,
		AgentID:   clr.client.agentID,
		Direction: "INPUT",
		User:      clr.client.sshUser,
		Host:      clr.client.sshHost,
		Port:      clr.client.localPort,
		Command:   data,
		Data:      encodedData,
		IsBase64:  true,
	}

	logMsg := common.NewMessage(common.MsgTypeSSHLog)
	logMsg.SessionID = clr.client.sessionID
	logMsg.ClientID = clr.client.id
	logMsg.AgentID = clr.client.agentID
	
	// Convert to JSON and send with mutex protection
	if logData, err := json.Marshal(logRequest); err == nil {
		logMsg.Data = logData
		
		// Use mutex to prevent concurrent writes to websocket
		clr.client.mutex.Lock()
		clr.client.tunnelConn.WriteJSON(logMsg)
		clr.client.mutex.Unlock()
	}
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

// createFileOnlyLogger creates a logger that only writes to file, not console
func createFileOnlyLogger(prefix string) *common.Logger {
	logger := &common.Logger{}
	
	// Create logs directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return common.NewLogger(prefix) // fallback to default
	}
	
	// Create log file
	logFileName := fmt.Sprintf("%s.log", prefix)
	logPath := filepath.Join(logDir, logFileName)
	
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file %s: %v", logPath, err)
		return common.NewLogger(prefix) // fallback to default
	}
	
	// Setup logger with file only (no stdout)
	logger.SetFileOnly(file, prefix)
	
	log.Printf("Logging to file: %s", logPath)
	return logger
}

func main() {
	// Load configuration from file, environment, or defaults
	config := loadConfig()

	var (
		clientID   string
		clientName string
		relayURL   string
		agentID    string
		token      string

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
				token:       token,
				logger:      createFileOnlyLogger("client"),
				sessions:    make(map[string]net.Conn),
				targets:     make(map[string]string),
				agentIDs:    make(map[string]string),
				localPort:   localPort,
				sshHost:     sshHost,
				sshUser:     sshUser,
				sshPassword: sshPassword,
				httpClient:  &http.Client{Timeout: 5 * time.Second},
			}

			// Debug: Log token status
			if token == "" {
				fmt.Println("🚨 WARNING: Token is empty!")
			} else {
				fmt.Printf("🔑 Token loaded: %s...%s\n", token[:min(8, len(token))], token[max(0, len(token)-8):])
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
	rootCmd.Flags().StringVarP(&token, "token", "T", "", "Client authentication token for relay server connection")

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
	c.logger.Info("Attempting to connect to relay server: %s", c.relayURL)
	
	// Add headers for debugging
	headers := http.Header{}
	headers.Set("User-Agent", "UniversalClient/1.0")
	
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	
	conn, resp, err := dialer.Dial(c.relayURL, headers)
	if err != nil {
		c.logger.Error("WebSocket connection failed: %v", err)
		if resp != nil {
			c.logger.Error("HTTP Response Status: %s", resp.Status)
			c.logger.Error("HTTP Response Headers: %v", resp.Header)
		}
		return fmt.Errorf("failed to connect to relay: %v", err)
	}
	
	c.logger.Info("Successfully connected to relay server")
	c.conn = conn
	c.running = true

	// Register with relay
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.ClientID = c.id
	registerMsg.ClientName = c.name
	registerMsg.AgentID = c.agentID
	registerMsg.Token = c.token
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
	// c.logger.Info("Creating SSH tunnel, connecting to relay: %s", c.relayURL)
	
	// Add headers for debugging
	headers := http.Header{}
	headers.Set("User-Agent", "UniversalClient-SSH/1.0")
	
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	
	// Connect to relay
	conn, resp, err := dialer.Dial(c.relayURL, headers)
	if err != nil {
		c.logger.Error("SSH Tunnel WebSocket connection failed: %v", err)
		if resp != nil {
			c.logger.Error("HTTP Response Status: %s", resp.Status)
			c.logger.Error("HTTP Response Headers: %v", resp.Header)
		}
		return fmt.Errorf("failed to create tunnel: %v", err)
	}
	
	// c.logger.Info("SSH Tunnel successfully connected to relay server")
	c.tunnelConn = conn

	// Register
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.ClientID = c.id
	registerMsg.ClientName = c.name
	registerMsg.Token = c.token  // Add token to register message
	if err := c.tunnelConn.WriteJSON(registerMsg); err != nil {
		return err
	}
	
	// c.logger.Info("Waiting for registration response...")
	
	// Wait for registration response
	var response common.Message
	if err := c.tunnelConn.ReadJSON(&response); err != nil {
		return fmt.Errorf("failed to read registration response: %v", err)
	}
	
	if response.Type == common.MsgTypeError {
		return fmt.Errorf("registration failed: %s", response.Error)
	}
	
	// c.logger.Info("Registration successful")

	// Request tunnel
	c.sessionID = fmt.Sprintf("ssh_%d", time.Now().UnixNano())
	connectMsg := common.NewMessage(common.MsgTypeConnect)
	connectMsg.SessionID = c.sessionID
	connectMsg.ClientID = c.id
	connectMsg.AgentID = c.agentID
	connectMsg.Target = "127.0.0.1:22"

	// c.logger.Info("Sending connect message to agent: %s, target: %s", c.agentID, connectMsg.Target)
	if err := c.tunnelConn.WriteJSON(connectMsg); err != nil {
		c.logger.Error("Failed to send connect message: %v", err)
		return err
	}
	// c.logger.Info("Connect message sent successfully")

	// Start tunnel listener and wait for it to be ready
	// c.logger.Info("Starting SSH tunnel listener on port: %s", c.localPort)
	listenerReady := make(chan bool)
	go c.startSSHTunnelListener(listenerReady)
	
	// Wait for listener to be ready
	<-listenerReady
	
	// c.logger.Info("SSH tunnel setup completed - ready for connections")
	
	// Create direct interactive SSH session instead of waiting
	// c.logger.Info("Creating direct SSH connection to %s@%s through tunnel", c.sshUser, c.sshHost)
	
	// Start interactive SSH session immediately
	if err := c.startInteractiveSSHSession(); err != nil {
		c.logger.Error("Failed to start interactive SSH session: %v", err)
		return err
	}
	
	return nil
}

func (c *UniversalClient) startSSHTunnelListener(ready chan bool) {
	// c.logger.Info("Attempting to create SSH tunnel listener on port: %s", c.localPort)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", c.localPort))
	if err != nil {
		c.logger.Error("SSH tunnel listener failed: %v", err)
		close(ready)
		return
	}
	defer listener.Close()
	
	// c.logger.Info("SSH tunnel listener started successfully on port: %s", c.localPort)
	// c.logger.Info("Ready to accept SSH connections on 127.0.0.1:%s", c.localPort)

	// Signal that listener is ready
	ready <- true
	close(ready)

	for {
		// c.logger.Info("Waiting for SSH connection...")
		conn, err := listener.Accept()
		if err != nil {
			c.logger.Error("Failed to accept connection: %v", err)
			continue
		}
		// c.logger.Info("New SSH connection accepted from: %s", conn.RemoteAddr())
		go c.handleSSHTunnelConnection(conn)
	}
}

func (c *UniversalClient) handleSSHTunnelConnection(conn net.Conn) {
	defer conn.Close()
	// c.logger.Info("New SSH connection established")

	// Create a dedicated WebSocket connection for this SSH session
	sessionConn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		c.logger.Error("Failed to create session WebSocket: %v", err)
		return
	}
	defer sessionConn.Close()

	// Register this session
	sessionID := fmt.Sprintf("ssh_%d", time.Now().UnixNano())
	
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.ClientID = c.id
	registerMsg.ClientName = c.name
	registerMsg.Token = c.token
	if err := sessionConn.WriteJSON(registerMsg); err != nil {
		c.logger.Error("Session registration failed: %v", err)
		return
	}

	// Wait for registration response
	var regResponse common.Message
	if err := sessionConn.ReadJSON(&regResponse); err != nil {
		c.logger.Error("Failed to read session registration response: %v", err)
		return
	}
	
	if regResponse.Type == common.MsgTypeError {
		c.logger.Error("Session registration failed: %s", regResponse.Error)
		return
	}

	// Request tunnel connection
	connectMsg := common.NewMessage(common.MsgTypeConnect)
	connectMsg.SessionID = sessionID
	connectMsg.ClientID = c.id
	connectMsg.AgentID = c.agentID
	connectMsg.Target = "127.0.0.1:22"

	if err := sessionConn.WriteJSON(connectMsg); err != nil {
		c.logger.Error("Failed to send connect message: %v", err)
		return
	}

	c.logger.Info("SSH session %s established", sessionID)

	// Forward data between local connection and tunnel
	done := make(chan bool, 2)

	// Local -> Tunnel (with command logging)
	go func() {
		defer func() { done <- true }()
		buffer := make([]byte, 4096)
		commandBuffer := make([]byte, 0)
		
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				c.logger.Debug("Local connection read error: %v", err)
				return
			}

			// Log command detection
			c.logSSHCommand(buffer[:n], &commandBuffer, sessionID, "outgoing")

			dataMsg := common.NewMessage(common.MsgTypeData)
			dataMsg.SessionID = sessionID
			dataMsg.ClientID = c.id
			dataMsg.Data = buffer[:n]
			if err := sessionConn.WriteJSON(dataMsg); err != nil {
				c.logger.Debug("WebSocket write error: %v", err)
				return
			}
		}
	}()

	// Tunnel -> Local (with command logging)
	go func() {
		defer func() { done <- true }()
		responseBuffer := make([]byte, 0)
		
		for {
			var msg common.Message
			if err := sessionConn.ReadJSON(&msg); err != nil {
				c.logger.Debug("WebSocket read error: %v", err)
				return
			}
			if msg.Type == common.MsgTypeData && msg.SessionID == sessionID {
				// Log response data
				c.logSSHCommand(msg.Data, &responseBuffer, sessionID, "incoming")
				
				if _, err := conn.Write(msg.Data); err != nil {
					c.logger.Debug("Local connection write error: %v", err)
					return
				}
			}
		}
	}()

	<-done
	c.logger.Info("SSH session %s closed", sessionID)
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

// logSSHCommand logs SSH commands and responses for monitoring
func (c *UniversalClient) startInteractiveSSHSession() error {
	c.logger.Info("Starting interactive SSH session...")
	
	// Connect to local tunnel port
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", c.localPort))
	if err != nil {
		return fmt.Errorf("failed to connect to tunnel: %v", err)
	}
	defer conn.Close()

	c.logger.Info("Connected to tunnel, establishing SSH client...")

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: c.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// Create SSH connection through tunnel
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, c.sshHost, config)
	if err != nil {
		return fmt.Errorf("SSH handshake failed: %v", err)
	}
	defer sshConn.Close()

	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	c.logger.Info("SSH client connected successfully!")

	// Create interactive session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Setup terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request PTY
	if err := session.RequestPty("xterm-256color", 40, 80, modes); err != nil {
		return fmt.Errorf("failed to request PTY: %v", err)
	}

	// Set up input/output with logging
	session.Stdout = NewLoggingWriter(os.Stdout, c.logger, "OUTPUT", c)
	session.Stderr = NewLoggingWriter(os.Stderr, c.logger, "ERROR", c)
	
	// Use command logging reader for stdin
	commandReader := NewCommandLoggingReader(c.logger, c)
	session.Stdin = commandReader

	// c.logger.Info("Starting interactive shell...")
	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║       🎯 Connected to %s@%s via agent %s                    ║\n", c.sshUser, c.sshHost, c.agentID)
	fmt.Println("║       Type 'exit' to disconnect                                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// Wait for session to end
	if err := session.Wait(); err != nil {
		c.logger.Info("SSH session ended: %v", err)
	}

	c.logger.Info("Interactive SSH session completed")
	return nil
}

func (c *UniversalClient) logSSHCommand(data []byte, buffer *[]byte, sessionID, direction string) {
	// Append data to buffer
	*buffer = append(*buffer, data...)
	
	// Convert to string for analysis
	str := string(*buffer)
	
	// Detect commands (look for command patterns)
	if direction == "outgoing" {
		// Look for SSH commands being sent
		if strings.Contains(str, "\r") || strings.Contains(str, "\n") {
			// Clean the command
			command := strings.TrimSpace(str)
			command = strings.ReplaceAll(command, "\r", "")
			command = strings.ReplaceAll(command, "\n", "")
			
			// Filter out non-printable characters and SSH protocol data
			if len(command) > 0 && isPrintableCommand(command) {
				c.logger.Info("SSH Command [%s]: %s", sessionID, command)
				
				// Log to database through relay server
				c.logCommandToDatabase(sessionID, command, "command")
			}
			
			// Clear buffer after processing
			*buffer = (*buffer)[:0]
		}
	} else if direction == "incoming" {
		// Log response data (optional, can be large)
		if len(*buffer) > 10240 { // Limit buffer size to 10KB
			responseSize := len(*buffer)
			c.logger.Debug("SSH Response [%s]: %d bytes received", sessionID, responseSize)
			*buffer = (*buffer)[:0] // Clear large buffer
		}
	}
}

// isPrintableCommand checks if the command contains mostly printable characters
func isPrintableCommand(command string) bool {
	if len(command) < 2 {
		return false
	}
	
	// Skip SSH protocol data and binary data
	if strings.HasPrefix(command, "SSH-") || 
	   strings.Contains(command, "\x00") ||
	   strings.Contains(command, "\x01") ||
	   strings.Contains(command, "\x02") {
		return false
	}
	
	// Count printable characters
	printableCount := 0
	for _, r := range command {
		if r >= 32 && r <= 126 { // Printable ASCII range
			printableCount++
		}
	}
	
	// Command should be mostly printable
	return float64(printableCount)/float64(len(command)) > 0.7
}

// logCommandToDatabase sends command log to relay server for database storage
func (c *UniversalClient) logCommandToDatabase(sessionID, command, operation string) {
	// Create a log message to send to relay server
	logMsg := common.NewMessage(common.MsgTypeDBQuery)
	logMsg.SessionID = sessionID
	logMsg.ClientID = c.id
	logMsg.AgentID = c.agentID
	logMsg.DBOperation = operation
	logMsg.DBQuery = command
	logMsg.DBProtocol = "ssh"
	logMsg.DBTable = "ssh_commands"
	
	// Send via main connection if available
	if c.conn != nil {
		if err := c.sendMessage(logMsg); err != nil {
			c.logger.Debug("Failed to log command to database: %v", err)
		}
	}
}
