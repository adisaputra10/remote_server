package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"remote-tunnel/internal/ssh"
	"remote-tunnel/internal/tunnel"
)

func main() {
	config := parseFlags()
	validateConfig(config)
	
	log.Printf("Starting SSH tunnel client")
	logConfig(config)

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create and connect tunnel
	tunnelConn, err := establishTunnel(config)
	if err != nil {
		log.Fatalf("Failed to connect to tunnel: %v", err)
	}
	defer tunnelConn.Close()

	log.Printf("Tunnel connected, establishing SSH connection...")

	// Get SSH credentials and create client
	sshConfig := createSSHConfig(config, tunnelConn)
	sshClient, err := ssh.NewSSHClient(sshConfig)
	if err != nil {
		log.Fatalf("Failed to create SSH client: %v", err)
	}
	defer sshClient.Close()

	log.Printf("SSH connection established!")
	log.Printf("Starting interactive session...")
	log.Printf("Press Ctrl+C to exit")

	// Handle graceful shutdown
	go handleShutdown(sigCh, sshClient)

	// Start interactive SSH session
	if err := sshClient.StartInteractiveSession(); err != nil {
		log.Fatalf("SSH session error: %v", err)
	}

	log.Printf("SSH session ended")
}

type SSHClientConfig struct {
	RelayURL    string
	AgentID     string
	Token       string
	Insecure    bool
	Compress    bool
	Username    string
	Password    string
	PrivateKey  string
	LogEnabled  bool
	LogDir      string
}

func parseFlags() *SSHClientConfig {
	config := &SSHClientConfig{}
	
	flag.StringVar(&config.RelayURL, "relay-url", "", "Relay WebSocket URL (e.g., wss://relay.example.com/ws/client)")
	flag.StringVar(&config.AgentID, "agent", "", "Target agent ID")
	flag.StringVar(&config.Token, "token", "", "Auth token (or set TUNNEL_TOKEN env)")
	flag.BoolVar(&config.Insecure, "insecure", false, "Skip TLS certificate verification")
	flag.BoolVar(&config.Compress, "compress", false, "Enable gzip compression")
	flag.StringVar(&config.Username, "user", "", "SSH username")
	flag.StringVar(&config.Password, "password", "", "SSH password (not recommended, use key)")
	flag.StringVar(&config.PrivateKey, "key", "", "Path to SSH private key file")
	flag.BoolVar(&config.LogEnabled, "log", true, "Enable command logging")
	flag.StringVar(&config.LogDir, "log-dir", "ssh-logs", "Directory for SSH session logs")
	
	flag.Parse()

	// Get token from environment if not provided
	if config.Token == "" {
		config.Token = os.Getenv("TUNNEL_TOKEN")
	}

	return config
}

func validateConfig(config *SSHClientConfig) {
	if config.RelayURL == "" || config.AgentID == "" || config.Token == "" {
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("Usage: ssh-client -relay-url <url> -agent <id> -token <token> [options]")
	fmt.Println("\nRequired:")
	fmt.Println("  -relay-url    Relay WebSocket URL")
	fmt.Println("  -agent        Target agent ID") 
	fmt.Println("  -token        Authentication token")
	fmt.Println("\nSSH Options:")
	fmt.Println("  -user         SSH username")
	fmt.Println("  -password     SSH password (not recommended)")
	fmt.Println("  -key          Path to SSH private key file")
	fmt.Println("  -log          Enable command logging (default: true)")
	fmt.Println("  -log-dir      Log directory (default: ssh-logs)")
	fmt.Println("\nTunnel Options:")
	fmt.Println("  -insecure     Skip TLS verification")
	fmt.Println("  -compress     Enable compression")
	fmt.Println("\nExample:")
	fmt.Println("  ssh-client -relay-url wss://relay.example.com/ws/client -agent my-agent -token secret123 -user admin")
}

func logConfig(config *SSHClientConfig) {
	log.Printf("Relay URL: %s", config.RelayURL)
	log.Printf("Agent ID: %s", config.AgentID)
	log.Printf("SSH User: %s", config.Username)
	if config.LogEnabled {
		log.Printf("Command logging: enabled (directory: %s)", config.LogDir)
	} else {
		log.Printf("Command logging: disabled")
	}
}

func establishTunnel(config *SSHClientConfig) (net.Conn, error) {
	// Create tunnel client with correct parameters
	client := tunnel.NewClient("127.0.0.1:0", config.RelayURL, config.AgentID, "127.0.0.1:22", config.Token)
	if config.Insecure {
		client.SetInsecure(true)
	}
	if config.Compress {
		client.SetCompression(true)
	}

	log.Printf("Connecting to relay...")
	return connectToTunnel(client)
}

func createSSHConfig(config *SSHClientConfig, tunnelConn net.Conn) *ssh.SSHConfig {
	sshUsername := getSSHUsername(config.Username)
	sshPassword := config.Password
	var privateKeyData string

	// Read private key if provided
	if config.PrivateKey != "" {
		keyData, err := os.ReadFile(config.PrivateKey)
		if err != nil {
			log.Fatalf("Failed to read private key: %v", err)
		}
		privateKeyData = string(keyData)
	} else if sshPassword == "" {
		// Prompt for password if no key provided
		sshPassword = getSSHPassword()
	}

	return &ssh.SSHConfig{
		Username:     sshUsername,
		Password:     sshPassword,
		PrivateKey:   privateKeyData,
		TunnelConn:   tunnelConn,
		LogEnabled:   config.LogEnabled,
		LogDirectory: config.LogDir,
	}
}

func getSSHUsername(username string) string {
	if username == "" {
		fmt.Print("SSH Username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}
	return username
}

func getSSHPassword() string {
	fmt.Print("SSH Password: ")
	passwordBytes, err := readPassword()
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	return string(passwordBytes)
}

func handleShutdown(sigCh chan os.Signal, sshClient *ssh.SSHClient) {
	<-sigCh
	log.Printf("\nShutting down SSH client...")
	sshClient.Close()
	os.Exit(0)
}

func connectToTunnel(client *tunnel.Client) (net.Conn, error) {
	// Connect to relay directly and get a stream for SSH
	conn, err := client.CreateDirectConnection()
	if err != nil {
		return nil, fmt.Errorf("create direct connection: %w", err)
	}
	return conn, nil
}

func createTunnelConnection(client *tunnel.Client) (net.Conn, error) {
	// This function is now unused - keeping for compatibility
	return nil, fmt.Errorf("deprecated: use connectToTunnel directly")
}

func readPassword() ([]byte, error) {
	// Simple password reading - in production use golang.org/x/term.ReadPassword
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return []byte(strings.TrimSpace(password)), nil
}
