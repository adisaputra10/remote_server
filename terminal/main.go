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
	SeparatorLine  = "==============================================="
	MaxHistory     = 100
	Version        = "1.0.0"
	TimeFormat     = "2006-01-02 15:04:05"
	HelpFormat     = "  %-12s - %s\n"
)

type InteractiveTerminal struct {
	prompt      string
	history     []string
	historyPos  int
	aliases     map[string]string
	workingDir  string
	logFile     *os.File
	commands    map[string]func([]string) error
}

func NewInteractiveTerminal() *InteractiveTerminal {
	workingDir, _ := os.Getwd()
	logFile, _ := os.OpenFile("terminal_session.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	terminal := &InteractiveTerminal{
		prompt:     getPrompt(workingDir),
		history:    make([]string, 0, MaxHistory),
		historyPos: 0,
		aliases:    getDefaultAliases(),
		workingDir: workingDir,
		logFile:    logFile,
		commands:   make(map[string]func([]string) error),
	}

	terminal.registerCommands()
	return terminal
}

func getPrompt(workingDir string) string {
	dirName := filepath.Base(workingDir)
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("GoTerm [%s]> ", dirName)
	default:
		return fmt.Sprintf("goterm:%s$ ", dirName)
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

func (it *InteractiveTerminal) registerCommands() {
	it.commands["help"] = it.cmdHelp
	it.commands["exit"] = it.cmdExit
	it.commands["quit"] = it.cmdExit
	it.commands["clear"] = it.cmdClear
	it.commands["cls"] = it.cmdClear
	it.commands["history"] = it.cmdHistory
	it.commands["cd"] = it.cmdChangeDir
	it.commands["pwd"] = it.cmdPwd
	it.commands["alias"] = it.cmdAlias
	it.commands["env"] = it.cmdEnv
	it.commands["version"] = it.cmdVersion
	it.commands["time"] = it.cmdTime
	it.commands["whoami"] = it.cmdWhoami
}

func (it *InteractiveTerminal) logCommand(command, output string) {
	if it.logFile != nil {
		timestamp := time.Now().Format(TimeFormat)
		it.logFile.WriteString(fmt.Sprintf("[%s] CMD: %s\n", timestamp, command))
		if output != "" {
			it.logFile.WriteString(fmt.Sprintf("[%s] OUT: %s\n", timestamp, output))
		}
		it.logFile.Sync()
	}
}

func (it *InteractiveTerminal) addToHistory(command string) {
	command = strings.TrimSpace(command)
	if command == "" || (len(it.history) > 0 && it.history[len(it.history)-1] == command) {
		return
	}

	it.history = append(it.history, command)
	if len(it.history) > MaxHistory {
		it.history = it.history[1:]
	}
	it.historyPos = len(it.history)
}

func (it *InteractiveTerminal) processAlias(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}

	if alias, exists := it.aliases[parts[0]]; exists {
		if len(parts) > 1 {
			return alias + " " + strings.Join(parts[1:], " ")
		}
		return alias
	}
	return command
}

func (it *InteractiveTerminal) updatePrompt() {
	it.prompt = getPrompt(it.workingDir)
}

// Built-in Commands
func (it *InteractiveTerminal) cmdHelp(args []string) error {
	fmt.Println(SeparatorLine)
	fmt.Println("Interactive Terminal Help")
	fmt.Println(SeparatorLine)
	fmt.Println("Built-in Commands:")
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
	
	fmt.Println("\nFeatures:")
	fmt.Println("  - Cross-platform command execution")
	fmt.Println("  - Command history and logging")
	fmt.Println("  - Command aliases")
	fmt.Println("  - Working directory management")
	fmt.Println(SeparatorLine)
	return nil
}

func (it *InteractiveTerminal) cmdExit(args []string) error {
	fmt.Println("Goodbye!")
	if it.logFile != nil {
		it.logFile.WriteString(fmt.Sprintf("[%s] Terminal session ended\n", time.Now().Format(TimeFormat)))
		it.logFile.Close()
	}
	os.Exit(0)
	return nil
}

func (it *InteractiveTerminal) cmdClear(args []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
	return nil
}

func (it *InteractiveTerminal) cmdHistory(args []string) error {
	if len(it.history) == 0 {
		fmt.Println("No command history available")
		return nil
	}

	fmt.Println("Command History:")
	for i, cmd := range it.history {
		fmt.Printf("%3d: %s\n", i+1, cmd)
	}
	return nil
}

func (it *InteractiveTerminal) cmdChangeDir(args []string) error {
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
		targetDir = filepath.Join(it.workingDir, targetDir)
	}
	targetDir = filepath.Clean(targetDir)

	if info, err := os.Stat(targetDir); err != nil {
		return fmt.Errorf("cd: %s: no such directory", targetDir)
	} else if !info.IsDir() {
		return fmt.Errorf("cd: %s: not a directory", targetDir)
	}

	it.workingDir = targetDir
	it.updatePrompt()
	return nil
}

func (it *InteractiveTerminal) cmdPwd(args []string) error {
	fmt.Println(it.workingDir)
	return nil
}

func (it *InteractiveTerminal) cmdAlias(args []string) error {
	if len(args) == 1 {
		fmt.Println("Current aliases:")
		keys := make([]string, 0, len(it.aliases))
		for k := range it.aliases {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		for _, k := range keys {
			fmt.Printf("  %-10s = %s\n", k, it.aliases[k])
		}
		return nil
	}

	if len(args) >= 4 && args[2] == "=" {
		aliasName := args[1]
		aliasCommand := strings.Join(args[3:], " ")
		it.aliases[aliasName] = aliasCommand
		fmt.Printf("Alias set: %s = '%s'\n", aliasName, aliasCommand)
		return nil
	}

	fmt.Println("Usage:")
	fmt.Println("  alias                  - Show all aliases")
	fmt.Println("  alias name = command   - Set an alias")
	return nil
}

func (it *InteractiveTerminal) cmdEnv(args []string) error {
	fmt.Println("Environment Information:")
	fmt.Printf("  OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Printf("  Working Directory: %s\n", it.workingDir)
	
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

func (it *InteractiveTerminal) cmdVersion(args []string) error {
	fmt.Printf("Interactive Terminal v%s\n", Version)
	fmt.Printf("Built with Go %s for %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return nil
}

func (it *InteractiveTerminal) cmdTime(args []string) error {
	now := time.Now()
	fmt.Printf("Current time: %s\n", now.Format(TimeFormat+" Monday"))
	fmt.Printf("Unix timestamp: %d\n", now.Unix())
	return nil
}

func (it *InteractiveTerminal) cmdWhoami(args []string) error {
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

func (it *InteractiveTerminal) executeSystemCommand(command string) error {
	var cmd *exec.Cmd
	
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Dir = it.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Command failed: %v\n", err)
	}
	return err
}

func (it *InteractiveTerminal) processCommand(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	it.logCommand(input, "")
	it.addToHistory(input)

	originalInput := input
	input = it.processAlias(input)

	args := strings.Fields(input)
	if len(args) == 0 {
		return nil
	}

	commandName := strings.ToLower(args[0])

	if cmdFunc, exists := it.commands[commandName]; exists {
		return cmdFunc(args)
	}

	err := it.executeSystemCommand(input)
	if err != nil {
		it.logCommand(originalInput, fmt.Sprintf("Error: %v", err))
	} else {
		it.logCommand(originalInput, "Success")
	}
	return err
}

func (it *InteractiveTerminal) readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(it.prompt)
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(input, "\r\n"), nil
}

func (it *InteractiveTerminal) showWelcome() {
	it.cmdClear(nil)
	fmt.Println(SeparatorLine)
	fmt.Println("    Interactive Terminal with Golang")
	fmt.Println(SeparatorLine)
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Working Directory: %s\n", it.workingDir)
	fmt.Printf("Time: %s\n", time.Now().Format(TimeFormat))
	fmt.Printf("Log File: terminal_session.log\n")
	fmt.Println()
	fmt.Println("Type 'help' for available commands")
	fmt.Println("Type 'exit' or 'quit' to exit")
	fmt.Println(SeparatorLine)
	fmt.Println()

	it.logCommand("SESSION_START", fmt.Sprintf("Terminal started at %s", time.Now().Format(TimeFormat)))
}

func (it *InteractiveTerminal) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal...")
		if it.logFile != nil {
			it.logCommand("SESSION_INTERRUPT", "Terminal interrupted by user")
			it.logFile.Close()
		}
		fmt.Println("Goodbye!")
		os.Exit(0)
	}()

	it.showWelcome()

	for {
		input, err := it.readInput()
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		if err := it.processCommand(input); err != nil {
			// Error messages are already shown in processCommand
		}
	}

	if it.logFile != nil {
		it.logCommand("SESSION_END", "Terminal session ended normally")
		it.logFile.Close()
	}
}

func main() {
	terminal := NewInteractiveTerminal()
	terminal.Start()
}
