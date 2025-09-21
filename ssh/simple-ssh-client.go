package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
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

	// Performance optimizations
	logBuffer  []SSHLogRequest
	logMutex   sync.Mutex
	httpClient *http.Client
	perfStats  *PerformanceStats
	config     *ClientConfig

	// File logging optimizations
	fileLogBuffer []string
	fileLogMutex  sync.Mutex
	lastFileFlush time.Time
}

// Performance configuration
type ClientConfig struct {
	BufferSize       int
	LogLevel         string
	AsyncLogging     bool
	LogBatchSize     int
	LogFlushInterval time.Duration
	DisableAnalysis  bool
}

// Performance statistics
type PerformanceStats struct {
	BytesSent       int64
	BytesReceived   int64
	PacketsSent     int64
	PacketsReceived int64
	LogsBuffered    int64
	LogsSent        int64
	LastUpdate      time.Time
	mutex           sync.RWMutex
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

		// Performance optimizations
		logBuffer:     make([]SSHLogRequest, 0, 100),
		httpClient:    &http.Client{Timeout: 2 * time.Second},
		perfStats:     &PerformanceStats{LastUpdate: time.Now()},
		fileLogBuffer: make([]string, 0, 50),
		lastFileFlush: time.Now(),
		config: &ClientConfig{
			BufferSize:       8192,
			LogLevel:         "INFO",
			AsyncLogging:     os.Getenv("ASYNC_LOGGING") == "true",
			LogBatchSize:     100,
			LogFlushInterval: 5 * time.Second,
			DisableAnalysis:  os.Getenv("DISABLE_SSH_ANALYSIS") == "true",
		},
	}

	// Start async log flusher if enabled
	if client.config.AsyncLogging {
		go client.startLogFlusher()
	}

	if err := client.connect(); err != nil {
		client.logger.Error("Failed to connect: %v", err)
		os.Exit(1)
	}

	// Start performance monitoring
	client.startPerformanceMonitor()

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

	// Use larger buffer for better performance
	buffer := make([]byte, c.config.BufferSize)

	// Forward from local connection to relay (client to server)
	go func() {
		defer func() { done <- true }()

		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF && c.config.LogLevel == "DEBUG" {
					c.logger.Error("Error reading from local connection: %v", err)
				}
				return
			}

			// Update performance stats
			c.updatePerfStats(int64(n), 0, 1, 0)

			// Minimal logging for performance
			if c.config.LogLevel == "DEBUG" {
				c.logger.Debug("📤 READ %d bytes from local SSH client", n)
			}

			// Optimize: Skip analysis for large data packets (likely file transfers)
			if !c.config.DisableAnalysis && n < 1024 {
				c.analyzeAndLogSSHDataAsync(buffer[:n], "outbound")
			}

			// Send using optimized JSON format
			dataMsg := Message{
				Type:      "data",
				ClientID:  c.clientID,
				SessionID: c.sessionID,
				Data:      make([]byte, n),
			}
			copy(dataMsg.Data, buffer[:n])

			if err := c.conn.WriteJSON(dataMsg); err != nil {
				if c.config.LogLevel == "DEBUG" {
					c.logger.Error("Error sending data to relay: %v", err)
				}
				return
			}

			if c.config.LogLevel == "DEBUG" {
				c.logger.Debug("✅ Sent %d bytes to relay via JSON", n)
			}
		}
	}()

	// Forward from relay to local connection (server to client)
	go func() {
		defer func() { done <- true }()

		for {
			var msg Message
			if err := c.conn.ReadJSON(&msg); err != nil {
				if c.config.LogLevel == "DEBUG" {
					c.logger.Error("Error reading from relay: %v", err)
				}
				return
			}

			if msg.Type == "data" && msg.SessionID == c.sessionID {
				// Update performance stats
				c.updatePerfStats(0, int64(len(msg.Data)), 0, 1)

				if c.config.LogLevel == "DEBUG" {
					c.logger.Debug("📥 RECEIVED %d bytes from relay", len(msg.Data))
				}

				// Optimize: Skip analysis for large data packets
				if !c.config.DisableAnalysis && len(msg.Data) < 1024 {
					c.analyzeAndLogSSHDataAsync(msg.Data, "inbound")
				}

				if _, err := conn.Write(msg.Data); err != nil {
					if c.config.LogLevel == "DEBUG" {
						c.logger.Error("Error writing to local connection: %v", err)
					}
					return
				}

				if c.config.LogLevel == "DEBUG" {
					c.logger.Debug("✅ Wrote %d bytes to local SSH client", len(msg.Data))
				}
			}
		}
	}()

	// Wait for either direction to complete
	<-done
	c.logger.Info("SSH session %s ended", c.sessionID)
}

// Performance optimization functions
func (c *SSHClient) startLogFlusher() {
	ticker := time.NewTicker(c.config.LogFlushInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.flushLogs()
		c.flushFileLogBuffer() // Also flush file logs periodically
	}
}

func (c *SSHClient) updatePerfStats(bytesSent, bytesReceived, packetsSent, packetsReceived int64) {
	c.perfStats.mutex.Lock()
	c.perfStats.BytesSent += bytesSent
	c.perfStats.BytesReceived += bytesReceived
	c.perfStats.PacketsSent += packetsSent
	c.perfStats.PacketsReceived += packetsReceived
	c.perfStats.LastUpdate = time.Now()
	c.perfStats.mutex.Unlock()
}

func (c *SSHClient) analyzeAndLogSSHDataAsync(data []byte, direction string) {
	if c.config.DisableAnalysis {
		return
	}

	// Run analysis in background to avoid blocking data forwarding
	go func() {
		c.analyzeAndLogSSHDataOptimized(data, direction)
	}()
}

func (c *SSHClient) analyzeAndLogSSHDataOptimized(data []byte, direction string) {
	// Quick checks to avoid expensive analysis
	if len(data) == 0 || len(data) > 2048 {
		return
	}

	dataStr := string(data)

	// Quick SSH protocol detection
	if strings.Contains(dataStr, "SSH-") {
		c.logSSHCommandAsync("SSH_VERSION", direction, dataStr)
		return
	}

	// Quick command detection for outbound data
	if direction == "outbound" && len(data) < 200 {
		if command := c.quickExtractCommand(dataStr); command != "" {
			c.logSSHCommandAsync(command, direction, dataStr)
		}
	}
}

func (c *SSHClient) quickExtractCommand(data string) string {
	// Fast command extraction without heavy string processing
	cleaned := strings.TrimSpace(data)
	if len(cleaned) == 0 {
		return ""
	}

	// Remove obvious control characters quickly
	var result strings.Builder
	printable := 0
	for _, r := range cleaned {
		if r >= 32 && r <= 126 {
			result.WriteRune(r)
			printable++
		}
	}

	if printable < len(cleaned)/2 {
		return "" // Too many control characters
	}

	words := strings.Fields(result.String())
	if len(words) > 0 {
		// Quick check against common commands
		command := strings.ToLower(words[0])
		commonCommands := map[string]bool{
			"ls": true, "cd": true, "pwd": true, "cat": true, "grep": true,
			"ps": true, "top": true, "vim": true, "nano": true, "tail": true,
			"mkdir": true, "rm": true, "cp": true, "mv": true, "chmod": true,
			"sudo": true, "su": true, "exit": true, "whoami": true,
		}

		if commonCommands[command] {
			return command
		}
	}

	return ""
}

func (c *SSHClient) logSSHCommandAsync(command, direction, data string) {
	if !c.config.AsyncLogging {
		c.logSSHCommandSync(command, direction, data)
		return
	}

	// Buffer the log for batch processing
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

	c.logMutex.Lock()
	c.logBuffer = append(c.logBuffer, logReq)
	c.perfStats.LogsBuffered++

	// Flush if buffer is full
	if len(c.logBuffer) >= c.config.LogBatchSize {
		go c.flushLogs()
	}
	c.logMutex.Unlock()

	// Add to file log buffer for commands.log
	c.addToFileLogBuffer(command, direction, data)

	// Log to local logger
	if c.config.LogLevel == "DEBUG" {
		c.logger.Info("🔍 SSH Command: [%s] %s -> %s:%s (User: %s)", direction, command, c.sshHost, c.sshPort, c.sshUser)
	}
}

func (c *SSHClient) flushLogs() {
	c.logMutex.Lock()
	if len(c.logBuffer) == 0 {
		c.logMutex.Unlock()
		return
	}

	// Copy buffer and clear it
	logs := make([]SSHLogRequest, len(c.logBuffer))
	copy(logs, c.logBuffer)
	c.logBuffer = c.logBuffer[:0] // Reset buffer
	c.logMutex.Unlock()

	// Send batch to relay API
	go c.sendLogBatch(logs)
}

func (c *SSHClient) sendLogBatch(logs []SSHLogRequest) {
	if len(logs) == 0 {
		return
	}

	// Build API URL
	relayAPIURL := strings.Replace(c.relayURL, "ws://", "http://", 1)
	relayAPIURL = strings.Replace(relayAPIURL, "/ws/client", "/api/log-ssh-batch", 1)

	// Prepare batch data
	batchData := map[string]interface{}{
		"logs": logs,
	}

	jsonData, err := json.Marshal(batchData)
	if err != nil {
		if c.config.LogLevel == "DEBUG" {
			c.logger.Error("Failed to marshal log batch: %v", err)
		}
		return
	}

	// Send with timeout
	resp, err := c.httpClient.Post(relayAPIURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		if c.config.LogLevel == "DEBUG" {
			c.logger.Error("Failed to send log batch to relay: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	c.perfStats.mutex.Lock()
	c.perfStats.LogsSent += int64(len(logs))
	c.perfStats.mutex.Unlock()

	if c.config.LogLevel == "DEBUG" {
		if resp.StatusCode == 200 {
			c.logger.Debug("✅ Sent %d SSH logs to database", len(logs))
		} else {
			c.logger.Error("❌ Failed to send log batch: HTTP %d", resp.StatusCode)
		}
	}
}

// File logging functions for commands.log
func (c *SSHClient) addToFileLogBuffer(command, direction, data string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [%s] Client:%s -> %s:%s (User:%s) Command: %s",
		timestamp, direction, c.clientID, c.sshHost, c.sshPort, c.sshUser, command)

	c.fileLogMutex.Lock()
	c.fileLogBuffer = append(c.fileLogBuffer, logEntry)

	// Flush if buffer is full or enough time has passed
	if len(c.fileLogBuffer) >= 50 || time.Since(c.lastFileFlush) > 5*time.Second {
		go c.flushFileLogBuffer()
	}
	c.fileLogMutex.Unlock()
}

func (c *SSHClient) flushFileLogBuffer() {
	c.fileLogMutex.Lock()
	if len(c.fileLogBuffer) == 0 {
		c.fileLogMutex.Unlock()
		return
	}

	// Copy buffer and clear it
	logs := make([]string, len(c.fileLogBuffer))
	copy(logs, c.fileLogBuffer)
	c.fileLogBuffer = c.fileLogBuffer[:0]
	c.lastFileFlush = time.Now()
	c.fileLogMutex.Unlock()

	// Write to commands.log file
	c.writeToCommandsLog(logs)
}

func (c *SSHClient) writeToCommandsLog(logs []string) {
	if len(logs) == 0 {
		return
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		if c.config.LogLevel == "DEBUG" {
			c.logger.Error("Failed to create logs directory: %v", err)
		}
		return
	}

	// Open commands.log file in append mode
	file, err := os.OpenFile("logs/commands.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		if c.config.LogLevel == "DEBUG" {
			c.logger.Error("Failed to open commands.log: %v", err)
		}
		return
	}
	defer file.Close()

	// Write all logs at once
	for _, logEntry := range logs {
		if _, err := file.WriteString(logEntry + "\n"); err != nil {
			if c.config.LogLevel == "DEBUG" {
				c.logger.Error("Failed to write to commands.log: %v", err)
			}
			return
		}
	}

	// Sync to disk
	file.Sync()
}

func (c *SSHClient) logSSHCommandSync(command, direction, data string) {
	// Fallback to synchronous logging (not recommended for performance)
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

	jsonData, err := json.Marshal(logReq)
	if err != nil {
		c.logger.Error("Failed to marshal SSH log: %v", err)
		return
	}

	relayAPIURL := strings.Replace(c.relayURL, "ws://", "http://", 1)
	relayAPIURL = strings.Replace(relayAPIURL, "/ws/client", "/api/log-ssh", 1)

	resp, err := c.httpClient.Post(relayAPIURL, "application/json", bytes.NewReader(jsonData))
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

// Enhanced SSH Command Detection and Logging Functions (Optimized)
func (c *SSHClient) analyzeAndLogSSHData(data []byte, direction string) {
	// Simplified for compatibility - use async version for performance
	c.analyzeAndLogSSHDataOptimized(data, direction)
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
		"$ ",        // Shell prompt
		"# ",        // Root prompt
		"> ",        // Continuation prompt
		"? ",        // Help prompt
		"[y/n]",     // Yes/no prompt
		"[Y/n]",     // Yes/no prompt
		"(yes/no)",  // SSH host verification
		"password:", // Password prompt
		"Password:", // Password prompt
		"Enter",     // Enter prompt
		"Press",     // Press key prompt
		"Continue",  // Continue prompt
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
	// Use async logging for better performance
	c.logSSHCommandAsync(command, direction, data)
}

// Performance monitoring functions
func (c *SSHClient) startPerformanceMonitor() {
	if c.config.LogLevel != "DEBUG" {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			c.logPerformanceStats()
		}
	}()
}

func (c *SSHClient) logPerformanceStats() {
	c.perfStats.mutex.RLock()
	stats := *c.perfStats
	c.perfStats.mutex.RUnlock()

	c.logger.Info("📊 Performance Stats - Sent: %d bytes (%d packets), Received: %d bytes (%d packets), Logs: %d buffered, %d sent",
		stats.BytesSent, stats.PacketsSent, stats.BytesReceived, stats.PacketsReceived,
		stats.LogsBuffered, stats.LogsSent)
}

func (c *SSHClient) getPerformanceStats() PerformanceStats {
	c.perfStats.mutex.RLock()
	defer c.perfStats.mutex.RUnlock()
	return *c.perfStats
}
