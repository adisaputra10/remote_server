package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	SeparatorLine    = "==============================================="
	MaxHistory       = 100
	Version          = "2.0.0"
	TimeFormat       = "2006-01-02 15:04:05"
	HelpFormat       = "  %-15s - %s\n"
	RelayBinary      = "relay.exe"
	AgentBinary      = "agent.exe"
	SSHPTYBinary     = "ssh-pty.exe"
	DefaultToken     = "demo-token"
	DefaultAgentID   = "demo-agent"
	RelayURL         = "wss://localhost:8080"
)

type SSHTerminal struct {
	prompt      string
	history     []string
	historyPos  int
	aliases     map[string]string
	workingDir  string
	logFile     *os.File
	commands    map[string]func([]string) error
	
	// SSH Agent related
	relayRunning    bool
	agentRunning    bool
	sshClientActive bool
}

func NewSSHTerminal() *SSHTerminal {
	workingDir, _ := os.Getwd()
	// Go back to parent directory where binaries are located
	workingDir = filepath.Dir(workingDir)
	
	logFile, _ := os.OpenFile("ssh_terminal.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	terminal := &SSHTerminal{
		prompt:          getPrompt(workingDir),
		history:         make([]string, 0, MaxHistory),
		historyPos:      0,
		aliases:         getDefaultAliases(),
		workingDir:      workingDir,
		logFile:         logFile,
		commands:        make(map[string]func([]string) error),
		relayRunning:    false,
		agentRunning:    false,
		sshClientActive: false,
	}

	terminal.registerCommands()
	return terminal
}

func getPrompt(workingDir string) string {
	dirName := filepath.Base(workingDir)
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("SSH-Term [%s]> ", dirName)
	default:
		return fmt.Sprintf("ssh-term:%s$ ", dirName)
	}
}

func getDefaultAliases() map[string]string {
	aliases := make(map[string]string)
	
	if runtime.GOOS == "windows" {
		aliases["ll"] = "dir"
		aliases["ls"] = "dir"
		aliases["cat"] = "type"
		aliases["grep"] = "findstr"
		aliases["which"] = "where"
		aliases["ps"] = "tasklist"
	} else {
		aliases["ll"] = "ls -la"
		aliases["la"] = "ls -a"
		aliases["dir"] = "ls"
		aliases["type"] = "cat"
		aliases["findstr"] = "grep"
		aliases["where"] = "which"
	}
	
	aliases[".."] = "cd .."
	aliases["..."] = "cd ../.."
	aliases["h"] = "history"
	aliases["c"] = "clear"
	aliases["q"] = "quit"

	return aliases
}

func (st *SSHTerminal) registerCommands() {
	// Basic commands
	st.commands["help"] = st.cmdHelp
	st.commands["exit"] = st.cmdExit
	st.commands["quit"] = st.cmdExit
	st.commands["clear"] = st.cmdClear
	st.commands["cls"] = st.cmdClear
	st.commands["history"] = st.cmdHistory
	st.commands["cd"] = st.cmdChangeDir
	st.commands["pwd"] = st.cmdPwd
	st.commands["alias"] = st.cmdAlias
	st.commands["env"] = st.cmdEnv
	st.commands["version"] = st.cmdVersion
	st.commands["time"] = st.cmdTime
	st.commands["whoami"] = st.cmdWhoami
	
	// SSH commands
	st.commands["ssh-status"] = st.cmdSSHStatus
	st.commands["ssh-start-relay"] = st.cmdStartRelay
	st.commands["ssh-start-agent"] = st.cmdStartAgent
	st.commands["ssh-stop-relay"] = st.cmdStopRelay
	st.commands["ssh-stop-agent"] = st.cmdStopAgent
	st.commands["ssh-stop-all"] = st.cmdStopAll
	st.commands["ssh-connect"] = st.cmdSSHConnect
	st.commands["ssh-pty"] = st.cmdSSHPTY
	st.commands["ssh-test"] = st.cmdSSHTest
	st.commands["ssh-setup"] = st.cmdSSHSetup
	st.commands["ssh-quick"] = st.cmdSSHQuick
}

func (st *SSHTerminal) logCommand(command, output string) {
	if st.logFile != nil {
		timestamp := time.Now().Format(TimeFormat)
		st.logFile.WriteString(fmt.Sprintf("[%s] CMD: %s\n", timestamp, command))
		if output != "" {
			st.logFile.WriteString(fmt.Sprintf("[%s] OUT: %s\n", timestamp, output))
		}
		st.logFile.Sync()
	}
}

func (st *SSHTerminal) addToHistory(command string) {
	command = strings.TrimSpace(command)
	if command == "" || (len(st.history) > 0 && st.history[len(st.history)-1] == command) {
		return
	}

	st.history = append(st.history, command)
	if len(st.history) > MaxHistory {
		st.history = st.history[1:]
	}
	st.historyPos = len(st.history)
}

func (st *SSHTerminal) processAlias(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}

	if alias, exists := st.aliases[parts[0]]; exists {
		if len(parts) > 1 {
			return alias + " " + strings.Join(parts[1:], " ")
		}
		return alias
	}
	return command
}

func (st *SSHTerminal) updatePrompt() {
	st.prompt = getPrompt(st.workingDir)
}

// Basic Commands
func (st *SSHTerminal) cmdHelp(args []string) error {
	fmt.Println(SeparatorLine)
	fmt.Println("SSH Terminal with Remote Agent Support")
	fmt.Println(SeparatorLine)
	fmt.Println("Basic Commands:")
	fmt.Printf(HelpFormat, "help", "Show this help message")
	fmt.Printf(HelpFormat, "exit/quit", "Exit the terminal")
	fmt.Printf(HelpFormat, "clear/cls", "Clear the screen")
	fmt.Printf(HelpFormat, "history", "Show command history")
	fmt.Printf(HelpFormat, "cd [dir]", "Change directory")
	fmt.Printf(HelpFormat, "pwd", "Show current directory")
	fmt.Printf(HelpFormat, "alias", "Show/manage aliases")
	fmt.Printf(HelpFormat, "env", "Show environment info")
	fmt.Printf(HelpFormat, "version", "Show terminal version")
	fmt.Printf(HelpFormat, "time", "Show current time")
	fmt.Printf(HelpFormat, "whoami", "Show current user")
	
	fmt.Println("\nSSH Remote Commands:")
	fmt.Printf(HelpFormat, "ssh-status", "Show SSH services status")
	fmt.Printf(HelpFormat, "ssh-setup", "Setup SSH environment")
	fmt.Printf(HelpFormat, "ssh-quick", "Quick start all SSH services")
	fmt.Printf(HelpFormat, "ssh-start-relay", "Start SSH relay server")
	fmt.Printf(HelpFormat, "ssh-start-agent", "Start SSH agent")
	fmt.Printf(HelpFormat, "ssh-stop-relay", "Stop SSH relay server")
	fmt.Printf(HelpFormat, "ssh-stop-agent", "Stop SSH agent")
	fmt.Printf(HelpFormat, "ssh-stop-all", "Stop all SSH services")
	fmt.Printf(HelpFormat, "ssh-connect", "Connect to SSH agent")
	fmt.Printf(HelpFormat, "ssh-pty", "Start SSH PTY session")
	fmt.Printf(HelpFormat, "ssh-test", "Test SSH connection")
	
	fmt.Println("\nQuick Start:")
	fmt.Println("  1. ssh-setup    - Check environment")
	fmt.Println("  2. ssh-quick    - Start all services")
	fmt.Println("  3. ssh-connect  - Connect to SSH")
	
	fmt.Println("\nFeatures:")
	fmt.Println("  - SSH relay server management")
	fmt.Println("  - SSH agent management")
	fmt.Println("  - Remote SSH connections")
	fmt.Println("  - PTY terminal sessions")
	fmt.Println("  - Command logging and history")
	fmt.Println(SeparatorLine)
	return nil
}

func (st *SSHTerminal) cmdExit(args []string) error {
	fmt.Println("Stopping SSH services...")
	st.cmdStopAll(nil)
	
	fmt.Println("Goodbye!")
	if st.logFile != nil {
		st.logFile.WriteString(fmt.Sprintf("[%s] Terminal session ended\n", time.Now().Format(TimeFormat)))
		st.logFile.Close()
	}
	os.Exit(0)
	return nil
}

func (st *SSHTerminal) cmdClear(args []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
	return nil
}

func (st *SSHTerminal) cmdHistory(args []string) error {
	if len(st.history) == 0 {
		fmt.Println("No command history available")
		return nil
	}

	fmt.Println("Command History:")
	for i, cmd := range st.history {
		fmt.Printf("%3d: %s\n", i+1, cmd)
	}
	return nil
}

func (st *SSHTerminal) cmdChangeDir(args []string) error {
	var targetDir string

	if len(args) < 2 {
		if runtime.GOOS == "windows" {
			targetDir = os.Getenv("USERPROFILE")
		} else {
			targetDir = os.Getenv("HOME")
		}
	} else {
		targetDir = args[1]
		if targetDir == "~" {
			if runtime.GOOS == "windows" {
				targetDir = os.Getenv("USERPROFILE")
			} else {
				targetDir = os.Getenv("HOME")
			}
		}
	}

	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(st.workingDir, targetDir)
	}
	targetDir = filepath.Clean(targetDir)

	if info, err := os.Stat(targetDir); err != nil {
		return fmt.Errorf("cd: %s: no such directory", targetDir)
	} else if !info.IsDir() {
		return fmt.Errorf("cd: %s: not a directory", targetDir)
	}

	st.workingDir = targetDir
	st.updatePrompt()
	return nil
}

func (st *SSHTerminal) cmdPwd(args []string) error {
	fmt.Println(st.workingDir)
	return nil
}

func (st *SSHTerminal) cmdAlias(args []string) error {
	if len(args) == 1 {
		fmt.Println("Current aliases:")
		keys := make([]string, 0, len(st.aliases))
		for k := range st.aliases {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		for _, k := range keys {
			fmt.Printf("  %-10s = %s\n", k, st.aliases[k])
		}
		return nil
	}

	if len(args) >= 4 && args[2] == "=" {
		aliasName := args[1]
		aliasCommand := strings.Join(args[3:], " ")
		st.aliases[aliasName] = aliasCommand
		fmt.Printf("Alias set: %s = '%s'\n", aliasName, aliasCommand)
		return nil
	}

	fmt.Println("Usage:")
	fmt.Println("  alias                  - Show all aliases")
	fmt.Println("  alias name = command   - Set an alias")
	return nil
}

func (st *SSHTerminal) cmdEnv(args []string) error {
	fmt.Println("Environment Information:")
	fmt.Printf("  OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Printf("  Working Directory: %s\n", st.workingDir)
	
	if runtime.GOOS == "windows" {
		fmt.Printf("  Computer: %s\n", os.Getenv("COMPUTERNAME"))
		fmt.Printf("  Username: %s\n", os.Getenv("USERNAME"))
		fmt.Printf("  User Profile: %s\n", os.Getenv("USERPROFILE"))
	} else {
		fmt.Printf("  Hostname: %s\n", os.Getenv("HOSTNAME"))
		fmt.Printf("  User: %s\n", os.Getenv("USER"))
		fmt.Printf("  Home: %s\n", os.Getenv("HOME"))
	}
	return nil
}

func (st *SSHTerminal) cmdVersion(args []string) error {
	fmt.Printf("SSH Terminal v%s\n", Version)
	fmt.Printf("Built with Go %s for %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return nil
}

func (st *SSHTerminal) cmdTime(args []string) error {
	now := time.Now()
	fmt.Printf("Current time: %s\n", now.Format(TimeFormat+" Monday"))
	fmt.Printf("Unix timestamp: %d\n", now.Unix())
	return nil
}

func (st *SSHTerminal) cmdWhoami(args []string) error {
	if runtime.GOOS == "windows" {
		fmt.Printf("User: %s\n", os.Getenv("USERNAME"))
		fmt.Printf("Domain: %s\n", os.Getenv("USERDOMAIN"))
	} else {
		fmt.Printf("User: %s\n", os.Getenv("USER"))
		if uid := os.Getenv("UID"); uid != "" {
			fmt.Printf("UID: %s\n", uid)
		}
	}
	return nil
}

// SSH Commands
func (st *SSHTerminal) cmdSSHStatus(args []string) error {
	fmt.Println("SSH Services Status:")
	fmt.Println(SeparatorLine)
	
	// Check for running processes
	fmt.Println("Running Processes:")
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/fi", "imagename eq "+RelayBinary)
		output, _ := cmd.Output()
		if strings.Contains(string(output), RelayBinary) {
			fmt.Println("  ✅ relay.exe - RUNNING")
			st.relayRunning = true
		} else {
			fmt.Println("  ❌ relay.exe - STOPPED")
			st.relayRunning = false
		}
		
		cmd = exec.Command("tasklist", "/fi", "imagename eq "+AgentBinary)
		output, _ = cmd.Output()
		if strings.Contains(string(output), AgentBinary) {
			fmt.Println("  ✅ agent.exe - RUNNING")
			st.agentRunning = true
		} else {
			fmt.Println("  ❌ agent.exe - STOPPED")
			st.agentRunning = false
		}
	}
	
	// Check SSH client
	fmt.Printf("\nSSH Client: ")
	if st.sshClientActive {
		fmt.Println("ACTIVE")
	} else {
		fmt.Println("INACTIVE")
	}
	
	// Summary
	fmt.Println("\nSummary:")
	if st.relayRunning && st.agentRunning {
		fmt.Println("  ✅ Ready for SSH connections")
	} else {
		fmt.Println("  ⚠️  Services need to be started")
		fmt.Println("     Use 'ssh-quick' to start all services")
	}
	
	return nil
}

func (st *SSHTerminal) cmdSSHSetup(args []string) error {
	fmt.Println("Setting up SSH environment...")
	fmt.Println(SeparatorLine)
	
	// Check if binaries exist
	binaries := []string{RelayBinary, AgentBinary, SSHPTYBinary}
	missing := false
	
	for _, binary := range binaries {
		binaryPath := filepath.Join(st.workingDir, binary)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			fmt.Printf("❌ %s - NOT FOUND\n", binary)
			missing = true
		} else {
			fmt.Printf("✅ %s - FOUND\n", binary)
		}
	}
	
	// Check certificates
	certs := []string{"server.crt", "server.key"}
	for _, cert := range certs {
		certPath := filepath.Join(st.workingDir, cert)
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			fmt.Printf("❌ %s - NOT FOUND\n", cert)
			missing = true
		} else {
			fmt.Printf("✅ %s - FOUND\n", cert)
		}
	}
	
	// Check Windows SSH service
	if runtime.GOOS == "windows" {
		fmt.Println("\nWindows SSH Service:")
		cmd := exec.Command("sc", "query", "sshd")
		if output, err := cmd.Output(); err == nil {
			if strings.Contains(string(output), "RUNNING") {
				fmt.Println("  ✅ OpenSSH Server - RUNNING")
			} else {
				fmt.Println("  ⚠️  OpenSSH Server - NOT RUNNING")
				fmt.Println("     You may need to start it manually")
			}
		} else {
			fmt.Println("  ❌ OpenSSH Server - NOT INSTALLED")
		}
	}
	
	if missing {
		fmt.Println("\n⚠️  Some required files are missing!")
		fmt.Println("Please ensure all SSH components are built and certificates are generated.")
		return fmt.Errorf("setup incomplete")
	}
	
	fmt.Println("\n✅ SSH environment setup complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. ssh-quick     - Start all services")
	fmt.Println("  2. ssh-connect   - Connect to SSH")
	
	return nil
}

func (st *SSHTerminal) cmdSSHQuick(args []string) error {
	fmt.Println("Quick Start - Starting all SSH services...")
	fmt.Println(SeparatorLine)
	
	// Start relay
	if err := st.cmdStartRelay(nil); err != nil {
		return err
	}
	
	// Wait a bit for relay to start
	time.Sleep(2 * time.Second)
	
	// Start agent
	if err := st.cmdStartAgent(nil); err != nil {
		return err
	}
	
	// Wait a bit for agent to connect
	time.Sleep(2 * time.Second)
	
	fmt.Println("\n✅ Quick start complete!")
	fmt.Println("All SSH services are running.")
	fmt.Println("Use 'ssh-connect' to start SSH session.")
	
	return nil
}

func (st *SSHTerminal) cmdStartRelay(args []string) error {
	if st.relayRunning {
		fmt.Println("Relay server is already running")
		return nil
	}
	
	fmt.Println("Starting SSH relay server...")
	
	relayPath := filepath.Join(st.workingDir, RelayBinary)
	certPath := filepath.Join(st.workingDir, "server.crt")
	keyPath := filepath.Join(st.workingDir, "server.key")
	
	cmd := exec.Command(relayPath, "-addr", ":8080", "-token", DefaultToken, "-cert", certPath, "-key", keyPath)
	cmd.Dir = st.workingDir
	
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start relay: %v", err)
	}
	
	st.relayRunning = true
	fmt.Printf("✅ Relay server started (PID: %d)\n", cmd.Process.Pid)
	fmt.Println("   Listening on: https://localhost:8080")
	
	return nil
}

func (st *SSHTerminal) cmdStartAgent(args []string) error {
	if st.agentRunning {
		fmt.Println("SSH agent is already running")
		return nil
	}
	
	if !st.relayRunning {
		fmt.Println("Starting relay server first...")
		if err := st.cmdStartRelay(nil); err != nil {
			return err
		}
		time.Sleep(2 * time.Second) // Wait for relay to start
	}
	
	fmt.Println("Starting SSH agent...")
	
	agentPath := filepath.Join(st.workingDir, AgentBinary)
	cmd := exec.Command(agentPath, 
		"-relay-url", RelayURL+"/ws/agent",
		"-id", DefaultAgentID,
		"-token", DefaultToken,
		"-allow", "127.0.0.1:22",
		"-insecure")
	cmd.Dir = st.workingDir
	
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start agent: %v", err)
	}
	
	st.agentRunning = true
	fmt.Printf("✅ SSH agent started (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("   Connected to relay at: %s/ws/agent\n", RelayURL)
	fmt.Println("   Allowing connections to: 127.0.0.1:22")
	
	return nil
}

func (st *SSHTerminal) cmdStopRelay(args []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/f", "/im", RelayBinary)
		cmd.Run()
	}
	st.relayRunning = false
	fmt.Println("✅ Relay server stopped")
	return nil
}

func (st *SSHTerminal) cmdStopAgent(args []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/f", "/im", AgentBinary)
		cmd.Run()
	}
	st.agentRunning = false
	fmt.Println("✅ SSH agent stopped")
	return nil
}

func (st *SSHTerminal) cmdStopAll(args []string) error {
	st.cmdStopAgent(nil)
	st.cmdStopRelay(nil)
	fmt.Println("✅ All SSH services stopped")
	return nil
}

func (st *SSHTerminal) cmdSSHConnect(args []string) error {
	if !st.relayRunning || !st.agentRunning {
		fmt.Println("❌ Both relay and agent must be running")
		fmt.Println("Use 'ssh-quick' to start all services")
		return fmt.Errorf("services not running")
	}
	
	fmt.Println("Connecting to SSH agent...")
	fmt.Println("Enter your SSH credentials when prompted")
	fmt.Println("Press Ctrl+C to disconnect")
	fmt.Println(SeparatorLine)
	
	sshPTYPath := filepath.Join(st.workingDir, SSHPTYBinary)
	cmd := exec.Command(sshPTYPath,
		"-relay-url", RelayURL+"/ws/client",
		"-agent", DefaultAgentID,
		"-token", DefaultToken,
		"-user", "john",
		"-insecure")
	
	cmd.Dir = st.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	st.sshClientActive = true
	err := cmd.Run()
	st.sshClientActive = false
	
	fmt.Println("\nSSH session ended")
	
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	
	return nil
}

func (st *SSHTerminal) cmdSSHPTY(args []string) error {
	return st.cmdSSHConnect(args)
}

func (st *SSHTerminal) cmdSSHTest(args []string) error {
	fmt.Println("Testing SSH connection...")
	fmt.Println(SeparatorLine)
	
	// Test direct SSH
	fmt.Println("1. Testing direct SSH connection...")
	cmd := exec.Command("ssh", "john@127.0.0.1", "exit")
	if err := cmd.Run(); err != nil {
		fmt.Println("   ❌ Direct SSH failed - this is expected if using password auth")
	} else {
		fmt.Println("   ✅ Direct SSH successful")
	}
	
	// Test Windows SSH service
	fmt.Println("2. Checking Windows SSH service...")
	if runtime.GOOS == "windows" {
		cmd := exec.Command("sc", "query", "sshd")
		if output, err := cmd.Output(); err == nil {
			if strings.Contains(string(output), "RUNNING") {
				fmt.Println("   ✅ SSH service is running")
			} else {
				fmt.Println("   ❌ SSH service is not running")
			}
		} else {
			fmt.Println("   ❌ Could not check SSH service")
		}
	}
	
	// Check SSH services
	fmt.Println("3. Checking SSH remote services...")
	st.cmdSSHStatus(nil)
	
	// Test relay connectivity
	fmt.Println("4. Testing relay connectivity...")
	if st.relayRunning {
		fmt.Println("   ✅ Relay server is running")
	} else {
		fmt.Println("   ❌ Relay server is not running")
	}
	
	return nil
}

func (st *SSHTerminal) executeSystemCommand(command string) error {
	var cmd *exec.Cmd
	
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Dir = st.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Command failed: %v\n", err)
	}
	return err
}

func (st *SSHTerminal) processCommand(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	st.logCommand(input, "")
	st.addToHistory(input)

	originalInput := input
	input = st.processAlias(input)

	args := strings.Fields(input)
	if len(args) == 0 {
		return nil
	}

	commandName := strings.ToLower(args[0])

	if cmdFunc, exists := st.commands[commandName]; exists {
		return cmdFunc(args)
	}

	err := st.executeSystemCommand(input)
	if err != nil {
		st.logCommand(originalInput, fmt.Sprintf("Error: %v", err))
	} else {
		st.logCommand(originalInput, "Success")
	}
	return err
}

func (st *SSHTerminal) readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(st.prompt)
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(input, "\r\n"), nil
}

func (st *SSHTerminal) showWelcome() {
	st.cmdClear(nil)
	fmt.Println(SeparatorLine)
	fmt.Println("    SSH Terminal with Remote Agent Support")
	fmt.Println(SeparatorLine)
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Working Directory: %s\n", st.workingDir)
	fmt.Printf("Time: %s\n", time.Now().Format(TimeFormat))
	fmt.Printf("Log File: ssh_terminal.log\n")
	fmt.Println()
	fmt.Println("SSH Features:")
	fmt.Println("  - Remote SSH agent management")
	fmt.Println("  - SSH relay server control") 
	fmt.Println("  - PTY terminal sessions")
	fmt.Println("  - Command logging and history")
	fmt.Println()
	fmt.Println("Quick Start Guide:")
	fmt.Println("  1. ssh-setup   - Check environment")
	fmt.Println("  2. ssh-quick   - Start all services")
	fmt.Println("  3. ssh-connect - Connect to SSH")
	fmt.Println()
	fmt.Println("Type 'help' for all available commands")
	fmt.Println("Type 'exit' or 'quit' to exit")
	fmt.Println(SeparatorLine)
	fmt.Println()

	st.logCommand("SESSION_START", fmt.Sprintf("SSH Terminal started at %s", time.Now().Format(TimeFormat)))
}

func (st *SSHTerminal) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal...")
		st.cmdStopAll(nil)
		if st.logFile != nil {
			st.logCommand("SESSION_INTERRUPT", "Terminal interrupted by user")
			st.logFile.Close()
		}
		fmt.Println("Goodbye!")
		os.Exit(0)
	}()

	st.showWelcome()

	for {
		input, err := st.readInput()
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		if err := st.processCommand(input); err != nil {
			// Error messages are already shown in processCommand
		}
	}

	st.cmdStopAll(nil)
	
	if st.logFile != nil {
		st.logCommand("SESSION_END", "SSH Terminal session ended normally")
		st.logFile.Close()
	}
}

func main() {
	terminal := NewSSHTerminal()
	terminal.Start()
}
