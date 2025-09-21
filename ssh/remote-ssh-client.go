package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const timeFormat = "15:04:05"

type RemoteSSHClient struct {
	sshHost        string
	sshPort        string
	sshUser        string
	sshPassword    string
	sessionID      string
	retryAttempts  int
	retryDelay     time.Duration
	connectionPool map[string]*ssh.Client
	poolMutex      sync.RWMutex

	// Added for agent logging
	agentID string
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "remote-ssh-client",
		Short: "Simple Remote SSH Client",
		Run:   runRemoteSSHClient,
	}

	rootCmd.Flags().StringP("ssh-host", "H", "168.231.119.242", "SSH host")
	rootCmd.Flags().StringP("ssh-port", "P", "22", "SSH port")
	rootCmd.Flags().StringP("ssh-user", "u", "root", "SSH user")
	rootCmd.Flags().StringP("ssh-password", "p", "1qazxsw2", "SSH password")
	rootCmd.Flags().StringP("ssh-key", "k", "", "SSH private key file path (optional)")
	rootCmd.Flags().IntP("retry-attempts", "r", 3, "Number of retry attempts")
	rootCmd.Flags().IntP("retry-delay", "d", 2, "Delay between retries (seconds)")

	// Added agent parameter for logging
	rootCmd.Flags().StringP("agent", "a", "", "Target agent ID (will be logged to commands.log)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runRemoteSSHClient(cmd *cobra.Command, args []string) {
	sshHost, _ := cmd.Flags().GetString("ssh-host")
	sshPort, _ := cmd.Flags().GetString("ssh-port")
	sshUser, _ := cmd.Flags().GetString("ssh-user")
	sshPassword, _ := cmd.Flags().GetString("ssh-password")
	sshKey, _ := cmd.Flags().GetString("ssh-key")
	retryAttempts, _ := cmd.Flags().GetInt("retry-attempts")
	retryDelay, _ := cmd.Flags().GetInt("retry-delay")
	agentID, _ := cmd.Flags().GetString("agent")

	client := &RemoteSSHClient{
		sshHost:        sshHost,
		sshPort:        sshPort,
		sshUser:        sshUser,
		sshPassword:    sshPassword,
		sessionID:      fmt.Sprintf("remote_ssh_%d", time.Now().UnixNano()),
		retryAttempts:  retryAttempts,
		retryDelay:     time.Duration(retryDelay) * time.Second,
		connectionPool: make(map[string]*ssh.Client),
		agentID:        agentID,
	}

	// Print connection info
	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                🌐 Enhanced Remote SSH Client                      ║")
	fmt.Println("║           With Retry Logic & Key Authentication                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📡 Target Server: %s@%s:%s\n", sshUser, sshHost, sshPort)
	if agentID != "" {
		fmt.Printf("🏷️  Agent ID: %s\n", agentID)
	}
	fmt.Printf("🔐 Authentication: Password")
	if sshKey != "" {
		fmt.Printf(" + SSH Key (%s)", sshKey)
	}
	fmt.Printf("\n🔄 Retry Settings: %d attempts, %v delay\n", retryAttempts, client.retryDelay)
	fmt.Printf("📊 Session ID: %s\n", client.sessionID)
	fmt.Printf("📝 Commands logged to: logs/commands.log\n")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")

	if err := client.connectWithRetryAndFallback(sshKey); err != nil {
		// Error logging removed - only display to user
		fmt.Printf("\n❌ Connection Error: %v\n", err)
		fmt.Println("\n💡 Possible solutions:")
		fmt.Println("   1. Check if SSH server is running on target host")
		fmt.Println("   2. Verify username and password are correct")
		fmt.Println("   3. Check network connectivity to target host")
		fmt.Println("   4. Ensure SSH server allows password authentication")
		os.Exit(1)
	}
}

// connectWithRetryAndFallback implements retry logic and key authentication fallback
func (c *RemoteSSHClient) connectWithRetryAndFallback(sshKeyPath string) error {
	var lastErr error

	for attempt := 1; attempt <= c.retryAttempts; attempt++ {
		fmt.Printf("\n🔌 Connection Attempt %d/%d to %s:%s...\n", attempt, c.retryAttempts, c.sshHost, c.sshPort)
		fmt.Printf("⏰ Attempt started at: %s\n", time.Now().Format(timeFormat))

		// Try to connect
		sshClient, err := c.establishConnection(sshKeyPath)
		if err != nil {
			lastErr = err
			fmt.Printf("❌ Attempt %d failed: %v\n", attempt, err)
			fmt.Printf("🕒 Attempt failed at: %s\n", time.Now().Format(timeFormat))

			if attempt < c.retryAttempts {
				fmt.Printf("⏳ Waiting %v before retry...\n", c.retryDelay)
				time.Sleep(c.retryDelay)
			}
			continue
		}

		// Success - store in connection pool
		c.storeConnection(sshClient)
		fmt.Printf("✅ Connected successfully on attempt %d!\n", attempt)
		fmt.Printf("🕒 Success at: %s\n", time.Now().Format(timeFormat))

		// Start interactive session
		return c.createInteractiveShell(sshClient)
	}

	return fmt.Errorf("failed to connect after %d attempts. Last error: %v", c.retryAttempts, lastErr)
}

// establishConnection tries to establish SSH connection with auth fallback
func (c *RemoteSSHClient) establishConnection(sshKeyPath string) (*ssh.Client, error) {
	authMethods := c.buildAuthMethods(sshKeyPath)

	config := &ssh.ClientConfig{
		User:            c.sshUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second, // Reduced timeout for faster retries
		ClientVersion:   "SSH-2.0-OpenSSH_9.5",
	}

	addr := fmt.Sprintf("%s:%s", c.sshHost, c.sshPort)
	fmt.Printf("🔧 Attempting TCP dial to %s (timeout: 10s)...\n", addr)

	// Add channel for timeout detection
	type result struct {
		client *ssh.Client
		err    error
	}

	resultChan := make(chan result, 1)

	go func() {
		client, err := ssh.Dial("tcp", addr, config)
		resultChan <- result{client: client, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			fmt.Printf("❌ SSH Dial failed: %v\n", res.err)
		} else {
			fmt.Printf("✅ TCP connection established\n")
		}
		return res.client, res.err
	case <-time.After(12 * time.Second): // Slightly longer than config timeout
		return nil, fmt.Errorf("connection timeout after 12 seconds")
	}
}

// buildAuthMethods creates authentication methods with fallback
func (c *RemoteSSHClient) buildAuthMethods(sshKeyPath string) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// 1. Try SSH Key first if provided
	if sshKeyPath != "" {
		if keyAuth := c.loadSSHKey(sshKeyPath); keyAuth != nil {
			methods = append(methods, keyAuth)
			fmt.Printf("🔑 SSH Key loaded: %s\n", sshKeyPath)
		}
	}

	// 2. Try SSH Key from default locations
	defaultKeys := []string{
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519"),
		filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "id_rsa"),
		filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "id_ed25519"),
	}

	for _, keyPath := range defaultKeys {
		if keyAuth := c.loadSSHKey(keyPath); keyAuth != nil {
			methods = append(methods, keyAuth)
			fmt.Printf("🔑 Default SSH Key found: %s\n", keyPath)
			break // Only use the first default key found
		}
	}

	// 3. Password authentication
	if c.sshPassword != "" {
		methods = append(methods, ssh.Password(c.sshPassword))
		fmt.Printf("🔐 Password authentication enabled\n")
	}

	// 4. Keyboard Interactive (fallback for password)
	methods = append(methods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
		answers = make([]string, len(questions))
		for i := range questions {
			answers[i] = c.sshPassword
		}
		return answers, nil
	}))

	return methods
}

// loadSSHKey loads SSH private key from file
func (c *RemoteSSHClient) loadSSHKey(keyPath string) ssh.AuthMethod {
	if keyPath == "" {
		return nil
	}

	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil
	}

	// Try parsing key without passphrase first
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		// Key might be encrypted - for now skip (could add passphrase prompt)
		return nil
	}

	return ssh.PublicKeys(signer)
}

// storeConnection stores connection in pool for reuse
func (c *RemoteSSHClient) storeConnection(client *ssh.Client) {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	connKey := fmt.Sprintf("%s:%s@%s", c.sshUser, c.sshHost, c.sshPort)
	c.connectionPool[connKey] = client
}

// getConnection retrieves connection from pool
func (c *RemoteSSHClient) getConnection() *ssh.Client {
	c.poolMutex.RLock()
	defer c.poolMutex.RUnlock()

	connKey := fmt.Sprintf("%s:%s@%s", c.sshUser, c.sshHost, c.sshPort)
	return c.connectionPool[connKey]
}

func (c *RemoteSSHClient) createInteractiveShell(sshClient *ssh.Client) error {
	// Create a single persistent session
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 24, modes); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %v", err)
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("unable to setup stdin: %v", err)
	}
	defer stdin.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("unable to setup stdout: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("unable to setup stderr: %v", err)
	}

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	fmt.Printf("\n💻 Interactive SSH Shell Started\n")
	fmt.Printf("You now have a persistent shell session. Commands like 'cd' will work correctly.\n")
	fmt.Printf("Type 'exit' to quit the session.\n")
	fmt.Println("───────────────────────────────────────────────────────────────────")

	// Handle input/output concurrently with command logging
	commandCount := 0
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()

			// Log command if it's not empty
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" {
				commandCount++
				c.logLinuxCommand(trimmedLine, commandCount)
			}

			fmt.Fprintf(stdin, "%s\n", line)
		}
	}()

	// Handle stdout
	go func() {
		io.Copy(os.Stdout, stdout)
	}()

	// Handle stderr
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	// Wait for session to end
	return session.Wait()
}

func (c *RemoteSSHClient) logLinuxCommand(command string, commandNum int) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Log command with timestamp to file
	c.logCommandToFile(command)
}

func (c *RemoteSSHClient) logCommandToFile(command string) {
	// Create logs directory if it doesn't exist
	logDir := "logs"
	os.MkdirAll(logDir, 0755)

	// Open/create commands log file
	logFile := filepath.Join(logDir, "commands.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return // Silently fail if cannot log
	}
	defer file.Close()

	// Write command with timestamp and agent ID (simplified format)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	agentInfo := ""
	if c.agentID != "" {
		agentInfo = fmt.Sprintf(" [Agent:%s]", c.agentID)
	}
	logEntry := fmt.Sprintf("[%s]%s - %s\n", timestamp, agentInfo, command)
	file.WriteString(logEntry)
}

// closeConnections closes all connections in the pool
func (c *RemoteSSHClient) closeConnections() {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	for key, client := range c.connectionPool {
		if client != nil {
			client.Close()
			fmt.Printf("🔌 Closed connection: %s\n", key)
		}
	}
	c.connectionPool = make(map[string]*ssh.Client)
}

// addSessionCleanup adds cleanup for graceful shutdown
func (c *RemoteSSHClient) addSessionCleanup() {
	// This could be called from main to ensure cleanup on exit
	defer c.closeConnections()
}
