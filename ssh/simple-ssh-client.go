package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"ssh-tunnel/internal/common"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

type SSHClient struct {
	clientID    string
	clientName  string
	relayURL    string
	conn        *websocket.Conn
	sessionID   string
	agentID     string
	sshHost     string
	sshPort     string
	sshUser     string
	sshPassword string
	localPort   string
	logger      *common.Logger
}

type Message struct {
	Type       string `json:"type"`
	ClientID   string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`
	AgentID    string `json:"agent_id,omitempty"`
	Target     string `json:"target,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Data       []byte `json:"data,omitempty"`
}

type SSHLogRequest struct {
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

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ssh-tunnel-client",
		Short: "SSH Tunnel Client with Enhanced Command Logging",
		Run:   runSSHClient,
	}

	rootCmd.Flags().StringP("client-id", "c", "ssh-client-1", "Client ID")
	rootCmd.Flags().StringP("client-name", "n", "SSH Client", "Client name")
	rootCmd.Flags().StringP("relay", "r", "ws://168.231.119.242:8080/ws/client", "Relay server WebSocket URL")
	rootCmd.Flags().StringP("agent", "a", "agent-linux", "Target agent ID")
	rootCmd.Flags().StringP("local-port", "p", "2222", "Local port to listen")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runSSHClient(cmd *cobra.Command, args []string) {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientName, _ := cmd.Flags().GetString("client-name")
	relayURL, _ := cmd.Flags().GetString("relay")
	agentID, _ := cmd.Flags().GetString("agent")
	localPort, _ := cmd.Flags().GetString("local-port")

	// Use default SSH settings
	sshHost := "127.0.0.1"
	sshPort := "22"
	sshUser := "root"
	sshPassword := "1qazxsw2"

	client := &SSHClient{
		clientID:    clientID,
		clientName:  clientName,
		relayURL:    relayURL,
		agentID:     agentID,
		sshHost:     sshHost,
		sshPort:     sshPort,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		localPort:   localPort,
		logger:      common.NewLogger(fmt.Sprintf("SSH-CLIENT-%s", clientID)),
	}

	if err := client.connect(); err != nil {
		client.logger.Error("Failed to connect: %v", err)
		os.Exit(1)
	}

	client.startLocalSSHServer()
}

func (c *SSHClient) connect() error {
	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %v", err)
	}

	// Register client
	registerMsg := Message{
		Type:       "register",
		ClientID:   c.clientID,
		ClientName: c.clientName,
	}

	if err := c.conn.WriteJSON(registerMsg); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	c.logger.Info("Connected to relay server as %s (%s)", c.clientID, c.clientName)
	return nil
}

func (c *SSHClient) startLocalSSHServer() {
	listener, err := net.Listen("tcp", ":"+c.localPort)
	if err != nil {
		c.logger.Error("Failed to listen on port %s: %v", c.localPort, err)
		os.Exit(1)
	}
	defer listener.Close()

	c.logger.Info("SSH tunnel with enhanced logging listening on port %s", c.localPort)
	c.logger.Info("Connect using: ssh %s@localhost -p %s", c.sshUser, c.localPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			c.logger.Error("Failed to accept connection: %v", err)
			continue
		}

		go c.handleSSHConnection(conn)
	}
}

func (c *SSHClient) handleSSHConnection(conn net.Conn) {
	defer conn.Close()

	c.logger.Info("New SSH connection from %s - Enhanced logging enabled", conn.RemoteAddr())

	// Create tunnel session
	c.sessionID = fmt.Sprintf("ssh_%d", time.Now().UnixNano())
	target := fmt.Sprintf("%s:%s", c.sshHost, c.sshPort)

	// Request tunnel through relay
	connectMsg := Message{
		Type:      "connect",
		ClientID:  c.clientID,
		AgentID:   c.agentID,
		Target:    target,
		SessionID: c.sessionID,
	}

	if err := c.conn.WriteJSON(connectMsg); err != nil {
		c.logger.Error("Failed to request tunnel: %v", err)
		return
	}

	c.logger.Info("✅ Requested tunnel for session %s to target %s with command logging", c.sessionID, target)

	// Handle data forwarding with enhanced SSH command logging
	c.forwardData(conn)
}

func (c *SSHClient) forwardData(conn net.Conn) {
	done := make(chan bool, 2)

	// Forward from local connection to relay (client to server)
	go func() {
		defer func() { done <- true }()

		buffer := make([]byte, 1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					c.logger.Error("Error reading from local connection: %v", err)
				}
				return
			}

			c.logger.Debug("📤 READ %d bytes from local SSH client", n)

			// Analyze and log SSH commands (outbound - commands from user)
			c.analyzeAndLogSSHData(buffer[:n], "outbound")

			// Send using JSON format
			dataMsg := Message{
				Type:      "data",
				ClientID:  c.clientID,
				SessionID: c.sessionID,
				Data:      make([]byte, n),
			}
			copy(dataMsg.Data, buffer[:n])

			if err := c.conn.WriteJSON(dataMsg); err != nil {
				c.logger.Error("Error sending data to relay: %v", err)
				return
			}

			c.logger.Debug("✅ Sent %d bytes to relay via JSON", n)
		}
	}()

	// Forward from relay to local connection (server to client)
	go func() {
		defer func() { done <- true }()

		for {
			var msg Message
			if err := c.conn.ReadJSON(&msg); err != nil {
				c.logger.Error("Error reading from relay: %v", err)
				return
			}

			if msg.Type == "data" && msg.SessionID == c.sessionID {
				c.logger.Debug("📥 RECEIVED %d bytes from relay", len(msg.Data))

				// Analyze and log SSH responses (inbound - responses from server)
				c.analyzeAndLogSSHData(msg.Data, "inbound")

				if _, err := conn.Write(msg.Data); err != nil {
					c.logger.Error("Error writing to local connection: %v", err)
					return
				}

				c.logger.Debug("✅ Wrote %d bytes to local SSH client", len(msg.Data))
			} else {
				c.logger.Debug("Ignoring message - Type: %s, SessionID match: %t", msg.Type, msg.SessionID == c.sessionID)
			}
		}
	}()

	// Wait for either direction to complete
	<-done
	c.logger.Info("SSH session %s ended", c.sessionID)
}

// Enhanced SSH Command Detection and Logging Functions
func (c *SSHClient) analyzeAndLogSSHData(data []byte, direction string) {
	dataStr := string(data)

	// Log raw data for debugging (only small packets)
	if len(data) < 100 {
		c.logger.Debug("SSH %s HEX: %s", direction, hex.EncodeToString(data))
		c.logger.Debug("SSH %s TXT: %q", direction, dataStr)
	}

	// Detect SSH protocol and commands
	if strings.Contains(dataStr, "SSH-") {
		// SSH version exchange
		c.logSSHCommand("SSH_VERSION", direction, dataStr)
	} else if c.isTerminalCommand(data) {
		// Detect terminal commands (after SSH authentication)
		command := c.extractTerminalCommand(data)
		if command != "" {
			c.logSSHCommand(command, direction, string(data))
		}
	} else if c.isInteractiveCommand(dataStr) {
		// Detect interactive shell commands
		command := c.extractInteractiveCommand(dataStr)
		if command != "" {
			c.logSSHCommand(command, direction, dataStr)
		}
	} else if len(data) > 0 && direction == "outbound" {
		// Log any outbound data as potential command
		cleaned := c.cleanData(dataStr)
		if len(cleaned) > 0 && len(cleaned) < 200 && c.isPrintableCommand(cleaned) {
			c.logSSHCommand("USER_INPUT", direction, cleaned)
		}
	} else if direction == "inbound" && len(data) > 0 {
		// Log server responses
		cleaned := c.cleanData(dataStr)
		if len(cleaned) > 0 && len(cleaned) < 500 && c.isPrintableResponse(cleaned) {
			c.logSSHCommand("SERVER_RESPONSE", direction, cleaned)
		}
	}
}

func (c *SSHClient) isTerminalCommand(data []byte) bool {
	// Detect common terminal commands in SSH session
	dataStr := strings.ToLower(string(data))
	commands := []string{
		"ls", "cd", "pwd", "cat", "grep", "ps", "top", "vim", "nano", "tail", "head", 
		"mkdir", "rm", "cp", "mv", "chmod", "chown", "sudo", "su", "exit", "logout", 
		"whoami", "id", "uname", "df", "du", "free", "history", "systemctl", "service", 
		"apt", "yum", "wget", "curl", "ping", "netstat", "ss", "iptables", "find", 
		"which", "whereis", "locate", "man", "info", "help", "clear", "reset", "date", 
		"uptime", "w", "who", "last", "lastlog", "hostname", "passwd", "mount", "umount", 
		"fdisk", "lsblk", "lscpu", "lsmem", "lsof", "kill", "killall", "jobs", "nohup", 
		"screen", "tmux", "tar", "gzip", "unzip", "zip", "rsync", "scp", "sftp", "ssh",
		"mysql", "psql", "sqlite3", "redis-cli", "mongo", "docker", "kubectl", "git",
		"make", "gcc", "python", "python3", "node", "npm", "yarn", "php", "ruby", "go",
	}
	
	// Clean data first
	cleaned := c.cleanTerminalData(dataStr)
	
	for _, cmd := range commands {
		if strings.HasPrefix(cleaned, cmd+" ") || cleaned == cmd || strings.HasPrefix(cleaned, cmd+"\n") || strings.HasPrefix(cleaned, cmd+"\r") {
			return true
		}
	}
	return false
}

func (c *SSHClient) isInteractiveCommand(data string) bool {
	// Detect interactive shell patterns
	patterns := []string{
		"$ ",       // Shell prompt
		"# ",       // Root prompt
		"> ",       // Continuation prompt
		"? ",       // Help prompt
		"[y/n]",    // Yes/no prompt
		"[Y/n]",    // Yes/no prompt
		"(yes/no)", // SSH host verification
		"password:", // Password prompt
		"Password:", // Password prompt
		"Enter",    // Enter prompt
		"Press",    // Press key prompt
		"Continue", // Continue prompt
	}
	
	lowerData := strings.ToLower(data)
	for _, pattern := range patterns {
		if strings.Contains(lowerData, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func (c *SSHClient) extractInteractiveCommand(data string) string {
	// Extract command from interactive shell data
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "$ ") || strings.Contains(line, "# ") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "$" || part == "#") && i+1 < len(parts) {
					return parts[i+1]
				}
			}
		}
	}
	return "INTERACTIVE_PROMPT"
}

func (c *SSHClient) extractTerminalCommand(data []byte) string {
	// Extract terminal command from data
	cleaned := c.cleanTerminalData(string(data))
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		return words[0]
	}
	return ""
}

func (c *SSHClient) cleanData(data string) string {
	// Remove control characters and clean data
	var result strings.Builder
	for _, r := range data {
		if r >= 32 && r <= 126 { // Printable ASCII
			result.WriteRune(r)
		} else if r == 10 || r == 13 { // LF or CR
			result.WriteRune(' ')
		}
	}
	return strings.TrimSpace(result.String())
}

func (c *SSHClient) cleanTerminalData(data string) string {
	// Remove ANSI escape sequences and control characters
	var result strings.Builder
	inEscape := false
	
	for _, r := range data {
		if r == 27 { // ESC character (start of ANSI sequence)
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEscape = false
			}
			continue
		}
		if r >= 32 && r <= 126 { // Printable ASCII
			result.WriteRune(r)
		} else if r == 10 || r == 13 { // LF or CR
			result.WriteRune(' ')
		}
	}
	
	return strings.TrimSpace(result.String())
}

func (c *SSHClient) isPrintableCommand(data string) bool {
	// Check if data looks like a printable command
	if len(data) == 0 {
		return false
	}
	
	// Skip if too many non-printable characters
	printableCount := 0
	for _, r := range data {
		if r >= 32 && r <= 126 {
			printableCount++
		}
	}
	
	return float64(printableCount)/float64(len(data)) > 0.7 // At least 70% printable
}

func (c *SSHClient) isPrintableResponse(data string) bool {
	// Check if data looks like a printable server response
	if len(data) == 0 {
		return false
	}
	
	// Common server response patterns
	responsePatterns := []string{
		"total ", "drwx", "-rw-", "permission denied", "command not found",
		"directory", "file", "error", "warning", "success", "completed",
		"root@", "user@", "login", "logout", "welcome", "bye", "exit",
	}
	
	lowerData := strings.ToLower(data)
	for _, pattern := range responsePatterns {
		if strings.Contains(lowerData, pattern) {
			return true
		}
	}
	
	// Check if mostly printable
	printableCount := 0
	for _, r := range data {
		if r >= 32 && r <= 126 {
			printableCount++
		}
	}
	
	return float64(printableCount)/float64(len(data)) > 0.5 // At least 50% printable
}

func (c *SSHClient) logSSHCommand(command, direction, data string) {
	// Log to local logger first with enhanced info
	c.logger.Info("🔍 SSH Command: [%s] %s -> %s:%s (User: %s)", direction, command, c.sshHost, c.sshPort, c.sshUser)

	logReq := SSHLogRequest{
		SessionID: c.sessionID,
		ClientID:  c.clientID,
		AgentID:   c.agentID,
		Direction: direction,
		User:      c.sshUser,
		Host:      c.sshHost,
		Port:      c.sshPort,
		Command:   command,
		Data:      data,
	}

	// Send to relay server
	jsonData, err := json.Marshal(logReq)
	if err != nil {
		c.logger.Error("Failed to marshal SSH log: %v", err)
		return
	}

	// Post to relay API
	relayAPIURL := strings.Replace(c.relayURL, "ws://", "http://", 1)
	relayAPIURL = strings.Replace(relayAPIURL, "/ws/client", "/api/log-ssh", 1)

	resp, err := http.Post(relayAPIURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		c.logger.Error("Failed to send SSH log to relay: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		c.logger.Debug("✅ SSH command logged to database: [%s] %s", direction, command)
	} else {
		c.logger.Error("❌ Failed to log SSH command: HTTP %d", resp.StatusCode)
	}
}