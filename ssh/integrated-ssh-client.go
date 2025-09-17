package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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

const timeFormat = "15:04:05"

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

type OutputCapture struct {
	client *CombinedSSHClient
	writer io.Writer
}

func (oc *OutputCapture) Write(p []byte) (n int, err error) {
	// Write to original stdout
	n, err = oc.writer.Write(p)

	// Log server response if it contains meaningful output
	output := string(p)
	if len(strings.TrimSpace(output)) > 0 && !strings.Contains(output, "\x1b[") {
		// Skip escape sequences and empty lines
		go oc.client.sendSSHLogToRelay(strings.TrimSpace(output), "response")
	}

	return n, err
}

type CombinedSSHClient struct {
	// Tunnel client fields
	clientID   string
	clientName string
	relayURL   string
	agentID    string
	localPort  string
	tunnelConn *websocket.Conn
	sessionID  string
	logger     *common.Logger

	// SSH client fields
	sshHost     string
	sshPort     string
	sshUser     string
	sshPassword string
	sshClient   *ssh.Client
	sshSession  *ssh.Session

	// Shared fields
	retryAttempts int
	retryDelay    time.Duration
	isConnected   bool
	mutex         sync.RWMutex

	// Logging fields
	httpClient *http.Client
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "integrated-ssh-client",
		Short: "Integrated SSH Tunnel + Direct SSH Client",
		Long:  "Combines SSH tunnel creation with direct SSH connection for seamless remote access",
		Run:   runIntegratedSSHClient,
	}

	// Tunnel parameters
	rootCmd.Flags().StringP("client-id", "c", "integrated-client", "Client ID for tunnel")
	rootCmd.Flags().StringP("client-name", "n", "Integrated SSH Client", "Client name")
	rootCmd.Flags().StringP("relay", "r", "ws://168.231.119.242:8080/ws/client", "Relay server WebSocket URL")
	rootCmd.Flags().StringP("agent", "a", "agent-linux", "Target agent ID")
	rootCmd.Flags().StringP("local-port", "p", "2222", "Local tunnel port")

	// SSH connection parameters
	rootCmd.Flags().StringP("ssh-user", "u", "root", "SSH username")
	rootCmd.Flags().StringP("ssh-password", "P", "", "SSH password (leave empty for interactive prompt)")
	rootCmd.Flags().StringP("ssh-host", "H", "127.0.0.1", "SSH host (usually 127.0.0.1 for tunnel)")
	rootCmd.Flags().StringP("target-host", "t", "127.0.0.1", "Target SSH server host")
	rootCmd.Flags().StringP("target-port", "T", "22", "Target SSH server port")

	// Optional parameters
	rootCmd.Flags().IntP("retry-attempts", "R", 3, "Connection retry attempts")
	rootCmd.Flags().IntP("retry-delay", "d", 2, "Delay between retries (seconds)")
	rootCmd.Flags().BoolP("tunnel-only", "x", false, "Only create tunnel, don't auto-connect SSH")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// promptPassword prompts user for password securely
func promptPassword(username, host string) string {
	fmt.Printf("🔐 Enter password for %s@%s: ", username, host)

	// Get password without echoing
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("\nError reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Println() // New line after password input
	return string(password)
}

func runIntegratedSSHClient(cmd *cobra.Command, args []string) {
	// Get parameters
	clientID, _ := cmd.Flags().GetString("client-id")
	clientName, _ := cmd.Flags().GetString("client-name")
	relayURL, _ := cmd.Flags().GetString("relay")
	agentID, _ := cmd.Flags().GetString("agent")
	localPort, _ := cmd.Flags().GetString("local-port")
	sshUser, _ := cmd.Flags().GetString("ssh-user")
	sshPassword, _ := cmd.Flags().GetString("ssh-password")
	sshHost, _ := cmd.Flags().GetString("ssh-host")
	targetHost, _ := cmd.Flags().GetString("target-host")
	targetPort, _ := cmd.Flags().GetString("target-port")
	retryAttempts, _ := cmd.Flags().GetInt("retry-attempts")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")
	tunnelOnly, _ := cmd.Flags().GetBool("tunnel-only")

	// If password is empty, prompt for it interactively
	if sshPassword == "" {
		sshPassword = promptPassword(sshUser, sshHost)
	}

	client := &CombinedSSHClient{
		clientID:      clientID,
		clientName:    clientName,
		relayURL:      relayURL,
		agentID:       agentID,
		localPort:     localPort,
		sshHost:       sshHost,
		sshPort:       localPort, // Use tunnel port
		sshUser:       sshUser,
		sshPassword:   sshPassword,
		retryAttempts: retryAttempts,
		retryDelay:    time.Duration(retryDelay) * time.Second,
		logger:        common.NewLogger(fmt.Sprintf("INTEGRATED-SSH-%s", clientID)),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              🚀 Integrated SSH Tunnel + Client                   ║")
	fmt.Println("║          Auto-Connect to Remote Server via Tunnel                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n🔗 Step 1: Creating SSH tunnel...\n")
	fmt.Printf("   📡 Relay: %s\n", strings.Replace(relayURL, "ws://", "", 1))
	fmt.Printf("   🏷️  Agent: %s\n", agentID)
	fmt.Printf("   🔌 Local Port: %s\n", localPort)
	fmt.Printf("   🎯 Target: %s:%s\n", targetHost, targetPort)

	// Step 1: Create tunnel connection
	if err := client.createTunnel(targetHost, targetPort); err != nil {
		client.logger.Error("Failed to create tunnel: %v", err)
		fmt.Printf("❌ Tunnel creation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ SSH tunnel established on port %s\n", localPort)

	if tunnelOnly {
		fmt.Printf("\n🔧 Tunnel-only mode: SSH tunnel is ready\n")
		fmt.Printf("💡 Connect manually: ssh %s@%s -p %s\n", sshUser, sshHost, localPort)

		// Keep tunnel alive
		select {}
	}

	// Step 2: Auto-connect to SSH
	fmt.Printf("\n🔗 Step 2: Auto-connecting to SSH server...\n")
	fmt.Printf("   👤 User: %s\n", sshUser)
	fmt.Printf("   🔐 Auth: Password\n")

	// Wait a moment for tunnel to be fully ready
	time.Sleep(2 * time.Second)

	if err := client.connectSSH(); err != nil {
		client.logger.Error("Failed to connect SSH: %v", err)
		fmt.Printf("❌ SSH connection failed: %v\n", err)
		fmt.Printf("💡 Tunnel is still running. Try manual connection:\n")
		fmt.Printf("   ssh %s@%s -p %s\n", sshUser, sshHost, localPort)

		// Keep tunnel alive for manual connection
		select {}
	}

	fmt.Printf("✅ SSH connection established\n")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Printf("🎉 Success! You are now connected to remote server\n")
	fmt.Printf("📝 Commands will be logged to: logs/commands.log\n")
	fmt.Println("───────────────────────────────────────────────────────────────────")

	// Step 3: Start interactive session
	client.startInteractiveSession()
}

func (c *CombinedSSHClient) createTunnel(targetHost, targetPort string) error {
	// Connect to relay server
	conn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %v", err)
	}
	c.tunnelConn = conn

	// Register as client
	registerMsg := common.NewMessage(common.MsgTypeRegister)
	registerMsg.ClientID = c.clientID
	registerMsg.ClientName = c.clientName

	if err := c.tunnelConn.WriteJSON(registerMsg); err != nil {
		return fmt.Errorf("failed to register client: %v", err)
	}

	// Request tunnel
	c.sessionID = fmt.Sprintf("ssh_%d", time.Now().UnixNano())
	connectMsg := common.NewMessage(common.MsgTypeConnect)
	connectMsg.SessionID = c.sessionID
	connectMsg.ClientID = c.clientID
	connectMsg.AgentID = c.agentID
	connectMsg.Target = fmt.Sprintf("%s:%s", targetHost, targetPort)

	if err := c.tunnelConn.WriteJSON(connectMsg); err != nil {
		return fmt.Errorf("failed to request tunnel: %v", err)
	}

	// Start local tunnel listener
	go c.startLocalTunnelListener()

	// Wait for tunnel to be ready
	time.Sleep(1 * time.Second)

	return nil
}

func (c *CombinedSSHClient) startLocalTunnelListener() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", c.localPort))
	if err != nil {
		c.logger.Error("Failed to start local listener: %v", err)
		return
	}
	defer listener.Close()

	c.logger.Info("SSH tunnel listening on port %s", c.localPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			c.logger.Error("Failed to accept connection: %v", err)
			continue
		}

		go c.handleTunnelConnection(conn)
	}
}

func (c *CombinedSSHClient) handleTunnelConnection(conn net.Conn) {
	defer conn.Close()

	// Create data forwarding between local connection and tunnel
	done := make(chan bool, 2)

	// Forward local -> tunnel
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
			dataMsg.ClientID = c.clientID
			dataMsg.AgentID = c.agentID
			dataMsg.Data = buffer[:n]

			if err := c.tunnelConn.WriteJSON(dataMsg); err != nil {
				return
			}
		}
	}()

	// Forward tunnel -> local
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

func (c *CombinedSSHClient) connectSSH() error {
	// SSH client configuration
	config := &ssh.ClientConfig{
		User: c.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// Connect via tunnel
	addr := fmt.Sprintf("%s:%s", c.sshHost, c.sshPort)

	var err error
	for i := 0; i < c.retryAttempts; i++ {
		c.sshClient, err = ssh.Dial("tcp", addr, config)
		if err == nil {
			break
		}

		if i < c.retryAttempts-1 {
			fmt.Printf("🔄 Connection attempt %d/%d failed, retrying in %v...\n", i+1, c.retryAttempts, c.retryDelay)
			time.Sleep(c.retryDelay)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect after %d attempts: %v", c.retryAttempts, err)
	}

	c.isConnected = true
	return nil
}

func (c *CombinedSSHClient) startInteractiveSession() {
	defer func() {
		if c.sshClient != nil {
			c.sshClient.Close()
		}
		if c.tunnelConn != nil {
			c.tunnelConn.Close()
		}
	}()

	// Create SSH session
	session, err := c.sshClient.NewSession()
	if err != nil {
		fmt.Printf("❌ Failed to create SSH session: %v\n", err)
		return
	}
	defer session.Close()

	// Setup terminal with output capture
	outputWriter := &OutputCapture{
		client: c,
		writer: os.Stdout,
	}
	session.Stdout = outputWriter
	session.Stderr = os.Stderr

	// Create stdin pipe for command logging
	stdin, err := session.StdinPipe()
	if err != nil {
		fmt.Printf("❌ Failed to create stdin pipe: %v\n", err)
		return
	}

	// Request PTY
	if err := session.RequestPty("xterm", 80, 24, ssh.TerminalModes{}); err != nil {
		fmt.Printf("❌ Failed to request PTY: %v\n", err)
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		fmt.Printf("❌ Failed to start shell: %v\n", err)
		return
	}

	// Handle user input with command logging
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			command := scanner.Text()

			// Log command
			c.logCommandToFile(command)

			// Send to SSH session
			stdin.Write([]byte(command + "\n"))
		}
	}()

	// Wait for session to complete
	session.Wait()
	fmt.Println("\n🔚 SSH session ended")
}

func (c *CombinedSSHClient) logCommandToFile(command string) {
	// Create logs directory
	os.MkdirAll("logs", 0755)

	// Open/create commands log file
	logFile := filepath.Join("logs", "commands.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		c.logger.Error("Failed to open commands.log: %v", err)
		return
	}
	defer file.Close()

	// Write command with timestamp, client, and agent to file
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [Client:%s] [Agent:%s] - %s\n", timestamp, c.clientID, c.agentID, command)
	file.WriteString(logEntry)

	// Also send to relay server database
	go c.sendSSHLogToRelay(command, "command")
}

func (c *CombinedSSHClient) sendSSHLogToRelay(command, direction string) {
	// Prepare SSH log request
	logReq := SSHLogRequest{
		SessionID: c.sessionID,
		ClientID:  c.clientID,
		AgentID:   c.agentID,
		Direction: direction,
		User:      c.sshUser,
		Host:      c.sshHost,
		Port:      c.sshPort,
		Command:   command,
		Data:      command,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(logReq)
	if err != nil {
		c.logger.Error("Failed to marshal SSH log: %v", err)
		return
	}

	// Build relay API URL
	relayAPIURL := strings.Replace(c.relayURL, "ws://", "http://", 1)
	relayAPIURL = strings.Replace(relayAPIURL, "/ws/client", "/api/log-ssh", 1)

	// Send to relay server
	resp, err := c.httpClient.Post(relayAPIURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		// Silently fail for relay logging - don't spam error logs
		// c.logger.Debug("Failed to send SSH log to relay: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		c.logger.Debug("✅ SSH command logged to database: [%s] %s", direction, command)
	} else {
		// Only log HTTP errors as debug, not error
		c.logger.Debug("❌ Failed to log SSH command to database: HTTP %d", resp.StatusCode)
	}
}
