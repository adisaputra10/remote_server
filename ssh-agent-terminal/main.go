package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	RelayExecutable = "relay.exe"
	AgentExecutable = "agent.exe"
	SSHNoHostKeyCheck = "StrictHostKeyChecking=no"
	SSHPasswordAuth = "PasswordAuthentication=yes"
	SSHConnectTimeout = "ConnectTimeout=5"
	SSHQuickTimeout = "ConnectTimeout=3"
)

type Config struct {
	RelayURL     string
	AgentID      string
	Token        string
	SSHHost      string
	SSHPort      string
	SSHUser      string
	SSHPassword  string
	CertFile     string
	KeyFile      string
	IsProduction bool
}

type SSHAgentTerminal struct {
	prompt       string
	history      []string
	historyPos   int
	workingDir   string
	agentRunning bool
	relayRunning bool
	sshConnected bool
	isProduction bool
	commands     map[string]func([]string) error
	config       *Config
}

func NewSSHAgentTerminal() *SSHAgentTerminal {
	workingDir, _ := os.Getwd()
	
	terminal := &SSHAgentTerminal{
		prompt:       "ssh-agent> ",
		history:      make([]string, 0, 100),
		historyPos:   0,
		workingDir:   workingDir,
		agentRunning: false,
		relayRunning: false,
		sshConnected: false,
		commands:     make(map[string]func([]string) error),
		config:       &Config{},
	}

	terminal.setupConfiguration()
	terminal.registerCommands()
	return terminal
}

func (t *SSHAgentTerminal) setupConfiguration() {
	fmt.Println("===============================================")
	fmt.Println("SSH Agent Terminal Configuration")
	fmt.Println("===============================================")
	fmt.Println()
	
	// Ask for mode
	fmt.Print("Select mode (1=Local, 2=Production): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	mode := strings.TrimSpace(scanner.Text())
	
	if mode == "2" || strings.ToLower(mode) == "production" || strings.ToLower(mode) == "prod" {
		t.setupProductionConfig()
	} else {
		t.setupLocalConfig()
	}
	
	fmt.Println("Configuration completed!")
	fmt.Println("===============================================")
	fmt.Println()
}

func (t *SSHAgentTerminal) setupLocalConfig() {
	fmt.Println("🏠 Local Development Mode")
	fmt.Println()
	
	// Default local configuration
	t.config.RelayURL = "wss://localhost:8080/ws/agent"
	t.config.AgentID = "demo-agent"
	t.config.Token = "demo-token"
	t.config.SSHHost = "127.0.0.1"
	t.config.SSHPort = "22"
	t.config.CertFile = "server.crt"
	t.config.KeyFile = "server.key"
	t.config.IsProduction = false
	t.isProduction = false  // Set terminal local flag
	
	// Prompt for SSH user
	fmt.Print("SSH Username (default: john): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())
	if username == "" {
		username = "john"
	}
	t.config.SSHUser = username
	
	// Prompt for SSH password
	fmt.Print("SSH Password (default: john123): ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())
	if password == "" {
		password = "john123"
	}
	t.config.SSHPassword = password
	
	fmt.Printf("✓ Local mode configured: %s@%s:%s\n", t.config.SSHUser, t.config.SSHHost, t.config.SSHPort)
}

func (t *SSHAgentTerminal) setupProductionConfig() {
	fmt.Println("🌐 Production Server Mode")
	fmt.Println()
	
	scanner := bufio.NewScanner(os.Stdin)
	
	// Server domain
	fmt.Print("Server domain (default: sh.adisaputra.online): ")
	scanner.Scan()
	domain := strings.TrimSpace(scanner.Text())
	if domain == "" {
		domain = "sh.adisaputra.online"
	}
	
	// Relay URL
	t.config.RelayURL = fmt.Sprintf("wss://%s:8443/ws/agent", domain)
	
	// Agent ID
	fmt.Print("Agent ID (default: server-agent): ")
	scanner.Scan()
	agentID := strings.TrimSpace(scanner.Text())
	if agentID == "" {
		agentID = "server-agent"
	}
	t.config.AgentID = agentID
	
	// Token
	fmt.Print("Token (default: production-secure-token-2024): ")
	scanner.Scan()
	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		token = "production-secure-token-2024"
	}
	t.config.Token = token
	
	// SSH Host
	fmt.Print("SSH Host (default: 127.0.0.1): ")
	scanner.Scan()
	sshHost := strings.TrimSpace(scanner.Text())
	if sshHost == "" {
		sshHost = "127.0.0.1"
	}
	t.config.SSHHost = sshHost
	
	// SSH Port
	fmt.Print("SSH Port (default: 22): ")
	scanner.Scan()
	sshPort := strings.TrimSpace(scanner.Text())
	if sshPort == "" {
		sshPort = "22"
	}
	t.config.SSHPort = sshPort
	
	// SSH User
	fmt.Print("SSH Username: ")
	scanner.Scan()
	t.config.SSHUser = strings.TrimSpace(scanner.Text())
	
	// SSH Password
	fmt.Print("SSH Password: ")
	scanner.Scan()
	t.config.SSHPassword = strings.TrimSpace(scanner.Text())
	
	t.config.CertFile = "/etc/tunnel-certs/server.crt"
	t.config.KeyFile = "/etc/tunnel-certs/server.key"
	t.config.IsProduction = true
	t.isProduction = true  // Set terminal production flag
	
	fmt.Printf("✓ Production mode configured: %s@%s:%s\n", t.config.SSHUser, t.config.SSHHost, t.config.SSHPort)
	fmt.Printf("✓ Relay: %s\n", t.config.RelayURL)
}

func (t *SSHAgentTerminal) registerCommands() {
	t.commands = map[string]func([]string) error{
		"help":         t.cmdHelp,
		"exit":         t.cmdExit,
		"quit":         t.cmdExit,
		"clear":        t.cmdClear,
		"status":       t.cmdStatus,
		"config":       t.cmdConfig,
		"start-relay":  t.cmdStartRelay,
		"start-agent":  t.cmdStartAgent,
		"stop-relay":   t.cmdStopRelay,
		"stop-agent":   t.cmdStopAgent,
		"ssh-connect":  t.cmdSSHConnect,
		"ssh-test":     t.cmdSSHTest,
		"ssh-exec":     t.cmdSSHExec,
		"check-port":   t.cmdCheckPort,
		"check-ssh":    t.cmdCheckSSH,
		"restart-all":  t.cmdRestartAll,
		"reconnect":    t.cmdReconnect,
		"version":      t.cmdVersion,
	}
}

func (t *SSHAgentTerminal) cmdHelp(args []string) error {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SSH Agent Terminal - Command Help")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Println("\nConnection Management:")
	fmt.Printf("  %-15s - Start relay server (port 8080)\n", "start-relay")
	fmt.Printf("  %-15s - Start agent with SSH forwarding\n", "start-agent")
	fmt.Printf("  %-15s - Stop relay server\n", "stop-relay")
	fmt.Printf("  %-15s - Stop agent\n", "stop-agent")
	fmt.Printf("  %-15s - Restart relay and agent\n", "restart-all")
	
	fmt.Println("\nSSH Operations:")
	fmt.Printf("  %-15s - Connect to SSH server\n", "ssh-connect")
	fmt.Printf("  %-15s - Test SSH connectivity\n", "ssh-test")
	fmt.Printf("  %-15s - Execute SSH command\n", "ssh-exec <cmd>")
	
	fmt.Println("\nSystem Checks:")
	fmt.Printf("  %-15s - Show system status\n", "status")
	fmt.Printf("  %-15s - Check port availability\n", "check-port <port>")
	fmt.Printf("  %-15s - Check SSH service\n", "check-ssh")
	
	fmt.Println("\nConfiguration:")
	fmt.Printf("  %-15s - Show current configuration\n", "config")
	fmt.Printf("  %-15s - Reconnect with new settings\n", "reconnect")
	
	fmt.Println("\nGeneral:")
	fmt.Printf("  %-15s - Show this help\n", "help")
	fmt.Printf("  %-15s - Clear screen\n", "clear")
	fmt.Printf("  %-15s - Show version\n", "version")
	fmt.Printf("  %-15s - Exit terminal\n", "exit/quit")
	
	fmt.Println(strings.Repeat("=", 50))
	return nil
}

func (t *SSHAgentTerminal) cmdExit(args []string) error {
	fmt.Println("\nStopping all services before exit...")
	t.cmdStopAgent(nil)
	t.cmdStopRelay(nil)
	fmt.Println("SSH Agent Terminal - Goodbye!")
	os.Exit(0)
	return nil
}

func (t *SSHAgentTerminal) cmdClear(args []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	return nil
}

func (t *SSHAgentTerminal) cmdStatus(args []string) error {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SSH Agent Terminal Status")
	fmt.Println(strings.Repeat("=", 50))
	
	// Check relay
	if t.isProcessRunning(RelayExecutable) {
		fmt.Println("✓ Relay Server: RUNNING (port 8080)")
		t.relayRunning = true
	} else {
		fmt.Println("✗ Relay Server: STOPPED")
		t.relayRunning = false
	}
	
	// Check agent
	if t.isProcessRunning(AgentExecutable) {
		fmt.Println("✓ SSH Agent: RUNNING (forwarding port 22)")
		t.agentRunning = true
	} else {
		fmt.Println("✗ SSH Agent: STOPPED")
		t.agentRunning = false
	}
	
	// Check SSH service
	if t.isSSHServiceRunning() {
		fmt.Println("✓ SSH Service: RUNNING")
	} else {
		fmt.Println("✗ SSH Service: STOPPED")
	}
	
	// Check SSH connectivity
	if t.testSSHConnection() {
		fmt.Println("✓ SSH Connection: WORKING")
		t.sshConnected = true
	} else {
		fmt.Println("✗ SSH Connection: FAILED")
		t.sshConnected = false
	}
	
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Working Directory: %s\n", t.workingDir)
	fmt.Printf("SSH Target: %s@%s:%s\n", t.config.SSHUser, t.config.SSHHost, t.config.SSHPort)
	fmt.Printf("Relay URL: %s\n", t.config.RelayURL)
	fmt.Println(strings.Repeat("=", 50))
	
	return nil
}

func (t *SSHAgentTerminal) cmdStartRelay(args []string) error {
	if t.relayRunning {
		fmt.Println("⚠ Relay server is already running")
		return nil
	}
	
	fmt.Println("🚀 Starting relay server on port 8080...")
	
	var cmd *exec.Cmd
	if t.config.IsProduction {
		// Production mode - connect to existing relay server
		fmt.Println("⚠ Production mode: Connecting to existing relay server")
		fmt.Printf("  Relay URL: %s\n", t.config.RelayURL)
		return nil
	} else {
		// Local mode - start local relay
		cmd = exec.Command("cmd", "/c", "start", "\"Relay Server\"", 
			RelayExecutable, "-addr", ":8080", "-token", t.config.Token, 
			"-cert", t.config.CertFile, "-key", t.config.KeyFile)
	}
	cmd.Dir = t.workingDir
	
	err := cmd.Start()
	if err != nil {
		fmt.Printf("✗ Failed to start relay: %v\n", err)
		return err
	}
	
	// Wait a moment and check if it started
	time.Sleep(2 * time.Second)
	if t.isProcessRunning(RelayExecutable) {
		fmt.Println("✓ Relay server started successfully")
		t.relayRunning = true
	} else {
		fmt.Println("✗ Relay server failed to start")
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdStartAgent(args []string) error {
	if t.agentRunning {
		fmt.Println("⚠ SSH Agent is already running")
		return nil
	}
	
	// For production mode, don't try to start relay locally
	if t.isProduction {
		fmt.Println("⚠ Production mode: Connecting to existing relay server")
		fmt.Printf("  Relay URL: %s\n", t.config.RelayURL)
	} else {
		if !t.relayRunning {
			fmt.Println("⚠ Relay server not running. Starting relay first...")
			t.cmdStartRelay(nil)
			time.Sleep(2 * time.Second)
		}
	}
	
	fmt.Println("🚀 Starting SSH agent with port 22 forwarding...")
	
	cmd := exec.Command("cmd", "/c", "start", "\"SSH Agent\"",
		AgentExecutable, "-relay-url", t.config.RelayURL,
		"-id", t.config.AgentID, "-token", t.config.Token,
		"-allow", fmt.Sprintf("%s:%s", t.config.SSHHost, t.config.SSHPort), "-insecure")
	cmd.Dir = t.workingDir
	
	err := cmd.Start()
	if err != nil {
		fmt.Printf("✗ Failed to start agent: %v\n", err)
		return err
	}
	
	// Wait a moment and check if it started
	time.Sleep(3 * time.Second)
	if t.isProcessRunning(AgentExecutable) {
		fmt.Println("✓ SSH Agent started successfully")
		fmt.Printf("  - Agent ID: %s\n", t.config.AgentID)
		fmt.Printf("  - Forwarding: %s:%s\n", t.config.SSHHost, t.config.SSHPort)
		fmt.Printf("  - SSH Target: %s@%s\n", t.config.SSHUser, t.config.SSHHost)
		t.agentRunning = true
	} else {
		fmt.Println("✗ SSH Agent failed to start")
		if t.isProduction {
			fmt.Println("  Troubleshooting:")
			fmt.Println("  1. Check network connectivity to relay server")
			fmt.Println("  2. Verify token and agent ID configuration")
			fmt.Println("  3. Ensure relay server is running on production")
			fmt.Println("  4. Check firewall settings")
		}
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdStopRelay(args []string) error {
	fmt.Println("🛑 Stopping relay server...")
	cmd := exec.Command("taskkill", "/f", "/im", RelayExecutable)
	cmd.Run()
	time.Sleep(1 * time.Second)
	t.relayRunning = false
	fmt.Println("✓ Relay server stopped")
	return nil
}

func (t *SSHAgentTerminal) cmdStopAgent(args []string) error {
	fmt.Println("🛑 Stopping SSH agent...")
	cmd := exec.Command("taskkill", "/f", "/im", AgentExecutable)
	cmd.Run()
	time.Sleep(1 * time.Second)
	t.agentRunning = false
	fmt.Println("✓ SSH Agent stopped")
	return nil
}

func (t *SSHAgentTerminal) cmdSSHConnect(args []string) error {
	if !t.sshConnected && !t.testSSHConnection() {
		fmt.Println("✗ SSH connection test failed")
		fmt.Println("  Please check SSH service and credentials")
		return fmt.Errorf("SSH connection not available")
	}
	
	fmt.Println("🔗 Connecting to SSH server...")
	sshTarget := fmt.Sprintf("%s@%s", t.config.SSHUser, t.config.SSHHost)
	fmt.Println("Login: " + sshTarget)
	fmt.Printf("Password: %s\n", t.config.SSHPassword)
	fmt.Println("Type 'exit' to return to terminal")
	fmt.Println()
	
	var cmd *exec.Cmd
	if t.config.SSHPort == "22" {
		cmd = exec.Command("ssh", sshTarget)
	} else {
		cmd = exec.Command("ssh", "-p", t.config.SSHPort, sshTarget)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("SSH connection ended with error: %v\n", err)
	} else {
		fmt.Println("SSH connection closed normally")
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdSSHTest(args []string) error {
	fmt.Println("🧪 Testing SSH connectivity...")
	
	if t.testSSHConnection() {
		fmt.Println("✓ SSH connection test successful")
		t.sshConnected = true
	} else {
		fmt.Println("✗ SSH connection test failed")
		t.sshConnected = false
		
		fmt.Println("\nTroubleshooting steps:")
		fmt.Println("1. Check if SSH service is running: check-ssh")
		fmt.Println("2. Verify user exists: net user john")
		fmt.Println("3. Test manual connection: ssh john@127.0.0.1")
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdSSHExec(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: ssh-exec <command>")
		fmt.Println("Example: ssh-exec \"echo hello && whoami\"")
		return nil
	}
	
	command := strings.Join(args, " ")
	fmt.Printf("🔧 Executing SSH command: %s\n", command)
	
	sshTarget := fmt.Sprintf("%s@%s", t.config.SSHUser, t.config.SSHHost)
	var cmd *exec.Cmd
	if t.config.SSHPort == "22" {
		cmd = exec.Command("ssh", "-o", SSHConnectTimeout, 
			"-o", SSHNoHostKeyCheck, 
			sshTarget, command)
	} else {
		cmd = exec.Command("ssh", "-p", t.config.SSHPort, "-o", SSHConnectTimeout, 
			"-o", SSHNoHostKeyCheck, 
			sshTarget, command)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("SSH command failed: %v\n", err)
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdCheckPort(args []string) error {
	port := "22"
	if len(args) > 0 {
		port = args[0]
	}
	
	fmt.Printf("🔍 Checking if port %s is available...\n", port)
	
	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Printf("✗ Port %s is not accessible: %v\n", port, err)
	} else {
		conn.Close()
		fmt.Printf("✓ Port %s is accessible\n", port)
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdCheckSSH(args []string) error {
	fmt.Println("🔍 Checking SSH service...")
	
	// Check SSH service status
	cmd := exec.Command("sc", "query", "sshd")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("✗ SSH service not found or not accessible")
		return err
	}
	
	if strings.Contains(string(output), "RUNNING") {
		fmt.Println("✓ SSH service is running")
	} else {
		fmt.Println("✗ SSH service is not running")
		fmt.Println("Try: net start sshd")
	}
	
	return nil
}

func (t *SSHAgentTerminal) cmdRestartAll(args []string) error {
	fmt.Println("🔄 Restarting all services...")
	
	t.cmdStopAgent(nil)
	t.cmdStopRelay(nil)
	
	time.Sleep(2 * time.Second)
	
	t.cmdStartRelay(nil)
	time.Sleep(2 * time.Second)
	t.cmdStartAgent(nil)
	
	fmt.Println("✓ All services restarted")
	return nil
}

func (t *SSHAgentTerminal) cmdVersion(args []string) error {
	fmt.Println("SSH Agent Terminal v2.0")
	fmt.Println("Built for SSH remote access through agent")
	if t.config.IsProduction {
		fmt.Println("Mode: Production")
	} else {
		fmt.Println("Mode: Local Development")
	}
	return nil
}

func (t *SSHAgentTerminal) cmdConfig(args []string) error {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Current Configuration")
	fmt.Println(strings.Repeat("=", 50))
	
	if t.config.IsProduction {
		fmt.Println("🌐 Mode: Production")
	} else {
		fmt.Println("🏠 Mode: Local Development")
	}
	
	fmt.Printf("Relay URL: %s\n", t.config.RelayURL)
	fmt.Printf("Agent ID: %s\n", t.config.AgentID)
	fmt.Printf("Token: %s\n", strings.Repeat("*", len(t.config.Token)-4) + t.config.Token[len(t.config.Token)-4:])
	fmt.Printf("SSH Host: %s\n", t.config.SSHHost)
	fmt.Printf("SSH Port: %s\n", t.config.SSHPort)
	fmt.Printf("SSH User: %s\n", t.config.SSHUser)
	fmt.Printf("SSH Password: %s\n", strings.Repeat("*", len(t.config.SSHPassword)))
	fmt.Printf("Certificate: %s\n", t.config.CertFile)
	fmt.Printf("Private Key: %s\n", t.config.KeyFile)
	
	fmt.Println(strings.Repeat("=", 50))
	return nil
}

func (t *SSHAgentTerminal) cmdReconnect(args []string) error {
	fmt.Println("🔄 Reconfiguring connection...")
	
	// Stop current services
	t.cmdStopAgent(nil)
	if !t.config.IsProduction {
		t.cmdStopRelay(nil)
	}
	
	// Reset configuration
	t.config = &Config{}
	t.setupConfiguration()
	
	fmt.Println("✓ Configuration updated. Use 'start-relay' and 'start-agent' to reconnect.")
	return nil
}

func (t *SSHAgentTerminal) isProcessRunning(processName string) bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+processName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), processName)
}

func (t *SSHAgentTerminal) isSSHServiceRunning() bool {
	cmd := exec.Command("sc", "query", "sshd")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "RUNNING")
}

func (t *SSHAgentTerminal) testSSHConnection() bool {
	sshTarget := fmt.Sprintf("%s@%s", t.config.SSHUser, t.config.SSHHost)
	var cmd *exec.Cmd
	if t.config.SSHPort == "22" {
		cmd = exec.Command("ssh", "-o", SSHQuickTimeout,
			"-o", SSHNoHostKeyCheck,
			"-o", SSHPasswordAuth,
			sshTarget, "echo", "test")
	} else {
		cmd = exec.Command("ssh", "-p", t.config.SSHPort, "-o", SSHQuickTimeout,
			"-o", SSHNoHostKeyCheck,
			"-o", SSHPasswordAuth,
			sshTarget, "echo", "test")
	}
	
	err := cmd.Run()
	return err == nil
}

func (t *SSHAgentTerminal) addToHistory(command string) {
	if command = strings.TrimSpace(command); command != "" {
		t.history = append(t.history, command)
		if len(t.history) > 100 {
			t.history = t.history[1:]
		}
		t.historyPos = len(t.history)
	}
}

func (t *SSHAgentTerminal) processCommand(input string) error {
	args := strings.Fields(strings.TrimSpace(input))
	if len(args) == 0 {
		return nil
	}
	
	command := strings.ToLower(args[0])
	
	if cmd, exists := t.commands[command]; exists {
		return cmd(args[1:])
	}
	
	// If not a built-in command, try to execute as system command
	fmt.Printf("Unknown command: %s\n", command)
	fmt.Println("Type 'help' for available commands")
	return nil
}

func (t *SSHAgentTerminal) Run() {
	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		t.cmdExit(nil)
	}()
	
	// Display welcome message
	t.displayWelcome()
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print(t.prompt)
		
		if !scanner.Scan() {
			break
		}
		
		input := scanner.Text()
		t.addToHistory(input)
		
		if err := t.processCommand(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

func (t *SSHAgentTerminal) displayWelcome() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("SSH Agent Terminal - Interactive SSH Remote Access")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	
	if t.config.IsProduction {
		fmt.Printf("🌐 Production Mode: %s@%s:%s\n", t.config.SSHUser, t.config.SSHHost, t.config.SSHPort)
		fmt.Printf("🔗 Relay: %s\n", t.config.RelayURL)
	} else {
		fmt.Printf("🏠 Local Mode: %s@%s:%s\n", t.config.SSHUser, t.config.SSHHost, t.config.SSHPort)
		fmt.Println("🔗 Relay: wss://localhost:8080/ws/agent")
	}
	
	fmt.Println()
	fmt.Println("Quick Start:")
	if t.config.IsProduction {
		fmt.Println("1. start-agent   - Connect to production server")
		fmt.Println("2. ssh-test      - Test SSH connection")
		fmt.Println("3. ssh-connect   - Connect to SSH server")
	} else {
		fmt.Println("1. start-relay   - Start local relay server") 
		fmt.Println("2. start-agent   - Start SSH agent")
		fmt.Println("3. ssh-test      - Test SSH connection")
		fmt.Println("4. ssh-connect   - Connect to SSH server")
	}
	fmt.Println()
	fmt.Println("Type 'help' for full command list")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}

func main() {
	terminal := NewSSHAgentTerminal()
	terminal.Run()
}
