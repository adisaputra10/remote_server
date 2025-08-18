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
	SSHTarget       = "john@127.0.0.1"
)

type SSHAgentTerminal struct {
	prompt      string
	history     []string
	historyPos  int
	workingDir  string
	agentRunning bool
	relayRunning bool
	sshConnected bool
	commands    map[string]func([]string) error
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
	}

	terminal.registerCommands()
	return terminal
}

func (t *SSHAgentTerminal) registerCommands() {
	t.commands = map[string]func([]string) error{
		"help":         t.cmdHelp,
		"exit":         t.cmdExit,
		"quit":         t.cmdExit,
		"clear":        t.cmdClear,
		"status":       t.cmdStatus,
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
	fmt.Printf("SSH Target: john@127.0.0.1:22\n")
	fmt.Printf("Relay URL: wss://localhost:8080/ws/agent\n")
	fmt.Println(strings.Repeat("=", 50))
	
	return nil
}

func (t *SSHAgentTerminal) cmdStartRelay(args []string) error {
	if t.relayRunning {
		fmt.Println("⚠ Relay server is already running")
		return nil
	}
	
	fmt.Println("🚀 Starting relay server on port 8080...")
	
	cmd := exec.Command("cmd", "/c", "start", "\"Relay Server\"", 
		RelayExecutable, "-addr", ":8080", "-token", "demo-token", 
		"-cert", "server.crt", "-key", "server.key")
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
	
	if !t.relayRunning {
		fmt.Println("⚠ Relay server not running. Starting relay first...")
		t.cmdStartRelay(nil)
		time.Sleep(2 * time.Second)
	}
	
	fmt.Println("🚀 Starting SSH agent with port 22 forwarding...")
	
	cmd := exec.Command("cmd", "/c", "start", "\"SSH Agent\"",
		AgentExecutable, "-relay-url", "wss://localhost:8080/ws/agent",
		"-id", "demo-agent", "-token", "demo-token",
		"-allow", "127.0.0.1:22", "-insecure")
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
		fmt.Println("  - Agent ID: demo-agent")
		fmt.Println("  - Forwarding: 127.0.0.1:22")
		fmt.Println("  - SSH Target: john@localhost")
		t.agentRunning = true
	} else {
		fmt.Println("✗ SSH Agent failed to start")
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
	fmt.Println("Login: " + SSHTarget)
	fmt.Println("Password: john123")
	fmt.Println("Type 'exit' to return to terminal")
	fmt.Println()
	
	cmd := exec.Command("ssh", SSHTarget)
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
	
	cmd := exec.Command("ssh", "-o", "ConnectTimeout=5", 
		"-o", "StrictHostKeyChecking=no", 
		SSHTarget, command)
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
	fmt.Println("SSH Agent Terminal v1.0")
	fmt.Println("Built for SSH remote access through agent")
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
	cmd := exec.Command("ssh", "-o", "ConnectTimeout=3",
		"-o", "StrictHostKeyChecking=no",
		"-o", "PasswordAuthentication=yes",
		SSHTarget, "echo", "test")
	
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
	fmt.Println("Welcome! This terminal provides SSH remote access through agent.")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("1. status        - Check system status")
	fmt.Println("2. start-relay   - Start relay server") 
	fmt.Println("3. start-agent   - Start SSH agent")
	fmt.Println("4. ssh-test      - Test SSH connection")
	fmt.Println("5. ssh-connect   - Connect to SSH server")
	fmt.Println()
	fmt.Println("Type 'help' for full command list")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}

func main() {
	terminal := NewSSHAgentTerminal()
	terminal.Run()
}
