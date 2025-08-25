package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type LinuxShell struct {
	currentDir string
	history    []string
	env        map[string]string
	logFile    *os.File
	logger     *log.Logger
}

func main() {
	shell, err := NewLinuxShell()
	if err != nil {
		fmt.Printf("Failed to create shell: %v\n", err)
		os.Exit(1)
	}
	defer shell.Close()

	shell.Run()
}

func NewLinuxShell() (*LinuxShell, error) {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := filepath.Join(logsDir, fmt.Sprintf("linux-shell_%s.log", timestamp))

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %v", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)

	currentDir, _ := os.Getwd()

	shell := &LinuxShell{
		currentDir: currentDir,
		history:    make([]string, 0),
		env:        make(map[string]string),
		logFile:    logFile,
		logger:     logger,
	}

	// Log shell startup
	shell.logEvent("SHELL_START", "Linux Shell started", "")

	return shell, nil
}

func (s *LinuxShell) Close() {
	if s.logFile != nil {
		s.logEvent("SHELL_END", "Linux Shell closed", "")
		s.logFile.Close()
	}
}

func (s *LinuxShell) logEvent(eventType, description, command string) {
	if s.logger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		user := os.Getenv("USERNAME")
		if user == "" {
			user = os.Getenv("USER")
		}
		if user == "" {
			user = "unknown"
		}

		logEntry := fmt.Sprintf("[%s] [%s] User: %s | Dir: %s | Event: %s | Desc: %s",
			timestamp, eventType, user, s.currentDir, description, command)

		s.logger.Println(logEntry)
	}
}

func (s *LinuxShell) Run() {
	s.showWelcome()

	reader := bufio.NewReader(os.Stdin)

	for {
		s.showPrompt()

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		// Add to history
		s.history = append(s.history, command)

		// Log command execution
		s.logEvent("COMMAND", "Command entered", command)

		// Handle built-in commands
		if s.handleBuiltinCommand(command) {
			continue
		}

		// Execute Linux command
		s.executeLinuxCommand(command)
	}
}

func (s *LinuxShell) showWelcome() {
	fmt.Printf("Linux Shell (Logged) - %s\n", s.currentDir)
}

func (s *LinuxShell) showPrompt() {
	hostname := s.getHostname()
	user := s.getCurrentUser()
	workDir := s.getShortPath(s.currentDir)

	fmt.Printf("\033[32m%s@%s\033[0m:\033[34m%s\033[0m$ ", user, hostname, workDir)
}

func (s *LinuxShell) handleBuiltinCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return true
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help":
		s.logEvent("BUILTIN", "Help command executed", command)
		s.showHelp()
		return true

	case "exit", "quit":
		s.logEvent("BUILTIN", "Exit command executed", command)
		fmt.Println("👋 Goodbye!")
		os.Exit(0)
		return true

	case "clear", "cls":
		s.logEvent("BUILTIN", "Clear command executed", command)
		s.clearScreen()
		return true

	case "history":
		s.logEvent("BUILTIN", "History command executed", command)
		s.showHistory()
		return true

	case "logs":
		s.logEvent("BUILTIN", "Show logs command executed", command)
		s.showLogs()
		return true

	case "pwd":
		s.logEvent("BUILTIN", "PWD command executed", command)
		fmt.Println(s.currentDir)
		return true

	case "cd":
		s.logEvent("BUILTIN", "CD command executed", command)
		s.changeDirectory(args)
		return true

	case "env":
		s.logEvent("BUILTIN", "Environment command executed", command)
		s.showEnvironment()
		return true

	default:
		return false
	}
}

func (s *LinuxShell) showLogs() {
	fmt.Println("📋 Recent Command Logs:")
	fmt.Println("=======================")

	if s.logFile != nil {
		// Get log file path
		logPath := s.logFile.Name()
		fmt.Printf("Log file: %s\n\n", logPath)

		// Show last 20 lines of log
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			// On Windows, try PowerShell
			cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Get-Content '%s' | Select-Object -Last 20", logPath))
		} else {
			cmd = exec.Command("tail", "-20", logPath)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error reading log file: %v\n", err)
			// Fallback: read file manually
			s.showLogsManually(logPath)
		}
	} else {
		fmt.Println("No log file available")
	}
	fmt.Println()
}

func (s *LinuxShell) showLogsManually(logPath string) {
	file, err := os.Open(logPath)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer file.Close()

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Show last 20 lines
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		fmt.Println(lines[i])
	}
}

func (s *LinuxShell) executeLinuxCommand(command string) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Try WSL first
		if s.isWSLAvailable() {
			cmd = exec.Command("wsl", "bash", "-c", command)
		} else {
			// Convert to Windows equivalent or run in PowerShell
			cmd = s.convertToWindowsCommand(command)
		}
	} else {
		// On Linux/Unix systems
		cmd = exec.Command("bash", "-c", command)
	}

	if cmd == nil {
		fmt.Println("❌ Command not supported on this platform")
		s.logEvent("COMMAND_ERROR", "Command not supported on this platform", command)
		return
	}

	cmd.Dir = s.currentDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Command failed: %v (took %v)\n", err, duration)
		s.logEvent("COMMAND_FAILED", fmt.Sprintf("Command failed: %v (took %v)", err, duration), command)
	} else {
		fmt.Printf("✅ Command completed successfully (took %v)\n", duration)
		s.logEvent("COMMAND_SUCCESS", fmt.Sprintf("Command completed successfully (took %v)", duration), command)
	}
	fmt.Println()
}

func (s *LinuxShell) isWSLAvailable() bool {
	cmd := exec.Command("wsl", "--version")
	err := cmd.Run()
	return err == nil
}

func (s *LinuxShell) convertToWindowsCommand(command string) *exec.Cmd {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	mainCmd := parts[0]

	// Common Linux to Windows command conversions
	switch mainCmd {
	case "ls":
		if len(parts) == 1 {
			return exec.Command("dir")
		}
		// Convert ls flags to dir flags
		dirArgs := []string{}
		for _, arg := range parts[1:] {
			if strings.HasPrefix(arg, "-") {
				// Convert common ls flags
				if strings.Contains(arg, "l") {
					dirArgs = append(dirArgs, "/L")
				}
				if strings.Contains(arg, "a") {
					dirArgs = append(dirArgs, "/A")
				}
			} else {
				dirArgs = append(dirArgs, arg)
			}
		}
		return exec.Command("dir", dirArgs...)

	case "cat":
		if len(parts) > 1 {
			return exec.Command("type", parts[1:]...)
		}

	case "grep":
		// Use findstr instead of grep
		return exec.Command("findstr", parts[1:]...)

	case "ps":
		return exec.Command("tasklist")

	case "kill":
		if len(parts) > 1 {
			return exec.Command("taskkill", "/PID", parts[1])
		}

	case "which":
		if len(parts) > 1 {
			return exec.Command("where", parts[1])
		}

	case "top":
		return exec.Command("tasklist")

	case "df":
		return exec.Command("wmic", "logicaldisk", "get", "size,freespace,caption")

	case "free":
		return exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory")

	case "uname":
		return exec.Command("systeminfo")

	default:
		// Try to run as PowerShell command
		return exec.Command("powershell", "-Command", command)
	}

	return nil
}

func (s *LinuxShell) showHelp() {
	fmt.Println("📖 Linux Command Shell Help")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println("Built-in Commands:")
	fmt.Println("  help           - Show this help")
	fmt.Println("  exit/quit      - Exit the shell")
	fmt.Println("  clear/cls      - Clear screen")
	fmt.Println("  history        - Show command history")
	fmt.Println("  logs           - Show recent command logs")
	fmt.Println("  pwd            - Show current directory")
	fmt.Println("  cd <dir>       - Change directory")
	fmt.Println("  env            - Show environment variables")
	fmt.Println()
	fmt.Println("Linux Commands (examples):")
	fmt.Println("  ls -la         - List files with details")
	fmt.Println("  ps aux         - Show running processes")
	fmt.Println("  grep pattern   - Search for pattern")
	fmt.Println("  find . -name   - Find files by name")
	fmt.Println("  cat file.txt   - Display file contents")
	fmt.Println("  top            - Show system processes")
	fmt.Println("  df -h          - Show disk usage")
	fmt.Println("  free -h        - Show memory usage")
	fmt.Println()
	fmt.Println("📝 All commands are automatically logged!")
	fmt.Println()
}

func (s *LinuxShell) clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func (s *LinuxShell) showHistory() {
	fmt.Println("📜 Command History:")
	fmt.Println("===================")
	for i, cmd := range s.history {
		fmt.Printf("%3d: %s\n", i+1, cmd)
	}
	fmt.Printf("\nTotal commands: %d\n\n", len(s.history))
}

func (s *LinuxShell) changeDirectory(args []string) {
	var newDir string

	if len(args) == 0 {
		// Go to home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("❌ Error getting home directory: %v\n", err)
			return
		}
		newDir = homeDir
	} else {
		newDir = args[0]
	}

	// Expand relative path
	if !filepath.IsAbs(newDir) {
		newDir = filepath.Join(s.currentDir, newDir)
	}

	err := os.Chdir(newDir)
	if err != nil {
		fmt.Printf("❌ Error changing directory: %v\n", err)
		s.logEvent("CD_FAILED", fmt.Sprintf("Failed to change directory: %v", err), strings.Join(args, " "))
		return
	}

	s.currentDir = newDir
	fmt.Printf("📁 Changed to: %s\n", newDir)
	s.logEvent("CD_SUCCESS", fmt.Sprintf("Changed directory to: %s", newDir), strings.Join(args, " "))
}

func (s *LinuxShell) showEnvironment() {
	fmt.Println("🌍 Environment Variables:")
	fmt.Println("=========================")

	envVars := os.Environ()
	for _, env := range envVars {
		fmt.Println(env)
	}
	fmt.Printf("\nTotal variables: %d\n\n", len(envVars))
}

func (s *LinuxShell) getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return hostname
}

func (s *LinuxShell) getCurrentUser() string {
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "user"
	}
	return user
}

func (s *LinuxShell) getShortPath(path string) string {
	homeDir, _ := os.UserHomeDir()
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}

	// Truncate long paths
	if len(path) > 30 {
		return "..." + path[len(path)-27:]
	}

	return path
}

func (s *LinuxShell) getTruncatedPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-(maxLen-3):]
}
