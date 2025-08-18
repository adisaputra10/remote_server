package ssh

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type PTYClient struct {
	tunnelConn   net.Conn
	sshClient    *ssh.Client
	session      *ssh.Session
	logFile      *os.File
	cmdLogFile   *os.File
	outLogFile   *os.File
	logEnabled   bool
	logDirectory string
	terminalFd   int
	originalState *term.State
	mutex        sync.Mutex
}

type PTYConfig struct {
	Username     string
	Password     string
	PrivateKey   string
	TunnelConn   net.Conn
	LogEnabled   bool
	LogDirectory string
}

func NewPTYClient(config *PTYConfig) (*PTYClient, error) {
	client := &PTYClient{
		tunnelConn:   config.TunnelConn,
		logEnabled:   config.LogEnabled,
		logDirectory: config.LogDirectory,
		terminalFd:   int(os.Stdin.Fd()),
	}

	// Setup logging if enabled
	if client.logEnabled {
		if err := client.setupLogging(); err != nil {
			return nil, fmt.Errorf("setup logging: %w", err)
		}
	}

	// Create SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 30 * time.Second,
	}

	// Add authentication methods
	if config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.Password))
	}

	if config.PrivateKey != "" {
		key, err := ssh.ParsePrivateKey([]byte(config.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(key))
	}

	// Connect via tunnel
	sshConn, chans, reqs, err := ssh.NewClientConn(config.TunnelConn, "remote", sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh connection: %w", err)
	}

	client.sshClient = ssh.NewClient(sshConn, chans, reqs)
	return client, nil
}

func (c *PTYClient) setupLogging() error {
	if c.logDirectory == "" {
		c.logDirectory = "ssh-logs"
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(c.logDirectory, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	
	// Session log file
	sessionLogPath := filepath.Join(c.logDirectory, fmt.Sprintf("pty-session_%s.log", timestamp))
	sessionLog, err := os.OpenFile(sessionLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("create session log file: %w", err)
	}
	c.logFile = sessionLog

	// Command log file
	cmdLogPath := filepath.Join(c.logDirectory, fmt.Sprintf("pty-commands_%s.log", timestamp))
	cmdLog, err := os.OpenFile(cmdLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("create command log file: %w", err)
	}
	c.cmdLogFile = cmdLog

	// Output log file
	outLogPath := filepath.Join(c.logDirectory, fmt.Sprintf("pty-output_%s.log", timestamp))
	outLog, err := os.OpenFile(outLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("create output log file: %w", err)
	}
	c.outLogFile = outLog

	log.Printf("PTY session logging to: %s", sessionLogPath)
	c.logSession("PTY session started")
	
	return nil
}

func (c *PTYClient) logSession(message string) {
	if c.logEnabled && c.logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		c.logFile.WriteString(fmt.Sprintf("%s [INFO] %s\n", timestamp, message))
		c.logFile.Sync()
	}
}

func (c *PTYClient) logCommand(command string) {
	if c.logEnabled && c.cmdLogFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		c.cmdLogFile.WriteString(fmt.Sprintf("%s [CMD] %s\n", timestamp, strings.TrimSpace(command)))
		c.cmdLogFile.Sync()
	}
}

func (c *PTYClient) logOutput(output string) {
	if c.logEnabled && c.outLogFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				c.outLogFile.WriteString(fmt.Sprintf("%s [OUT] %s\n", timestamp, line))
			}
		}
		c.outLogFile.Sync()
	}
}

func (c *PTYClient) StartInteractivePTY() error {
	// Create new SSH session
	session, err := c.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()
	c.session = session

	// Setup terminal raw mode
	if term.IsTerminal(c.terminalFd) {
		state, err := term.MakeRaw(c.terminalFd)
		if err != nil {
			return fmt.Errorf("make terminal raw: %w", err)
		}
		c.originalState = state
		defer c.restoreTerminal()
	}

	// Get terminal size
	termWidth, termHeight := c.getTerminalSize()

	// Request PTY
	err = session.RequestPty("xterm-256color", termHeight, termWidth, ssh.TerminalModes{
		ssh.ECHO:          1,     // Enable echo
		ssh.TTY_OP_ISPEED: 14400, // Input speed
		ssh.TTY_OP_OSPEED: 14400, // Output speed
		ssh.ICRNL:         1,     // Map CR to NL on input
		ssh.OPOST:         1,     // Enable output processing
		ssh.ONLCR:         1,     // Map NL to CR-NL on output
	})
	if err != nil {
		return fmt.Errorf("request PTY: %w", err)
	}

	// Setup I/O pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("start shell: %w", err)
	}

	c.logSession("Interactive PTY shell started")

	// Setup signal handling
	c.setupSignalHandling()

	// Setup I/O goroutines
	var wg sync.WaitGroup

	// Handle stdin (user input)
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.handleInput(stdin)
	}()

	// Handle stdout (command output)
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.handleOutput(stdout)
	}()

	// Handle stderr (error output)
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.handleError(stderr)
	}()

	// Handle terminal resize
	go c.handleResize()

	// Wait for session to end
	err = session.Wait()
	
	// Wait for I/O goroutines to finish
	wg.Wait()

	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			c.logSession(fmt.Sprintf("Shell exited with code: %d", exitError.ExitStatus()))
		} else {
			c.logSession(fmt.Sprintf("Session error: %v", err))
		}
	} else {
		c.logSession("Shell session ended normally")
	}

	return nil
}

func (c *PTYClient) getTerminalSize() (int, int) {
	if term.IsTerminal(c.terminalFd) {
		width, height, err := term.GetSize(c.terminalFd)
		if err == nil {
			return width, height
		}
	}
	return 80, 24 // Default size
}

func (c *PTYClient) restoreTerminal() {
	if c.originalState != nil {
		term.Restore(c.terminalFd, c.originalState)
	}
}

func (c *PTYClient) handleInput(stdin io.WriteCloser) {
	defer stdin.Close()

	buf := make([]byte, 1024)
	var commandBuffer strings.Builder

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading stdin: %v", err)
			}
			return
		}

		data := buf[:n]
		
		// Write to SSH session
		_, err = stdin.Write(data)
		if err != nil {
			log.Printf("Error writing to SSH session: %v", err)
			return
		}

		// Process input for command logging
		for _, b := range data {
			if b == '\n' || b == '\r' {
				// Command completed
				cmd := commandBuffer.String()
				if strings.TrimSpace(cmd) != "" {
					c.logCommand(cmd)
				}
				commandBuffer.Reset()
			} else if b == 127 || b == 8 { // Backspace
				s := commandBuffer.String()
				if len(s) > 0 {
					commandBuffer.Reset()
					commandBuffer.WriteString(s[:len(s)-1])
				}
			} else if b >= 32 && b <= 126 { // Printable characters
				commandBuffer.WriteByte(b)
			}
		}
	}
}

func (c *PTYClient) handleOutput(stdout io.Reader) {
	buf := make([]byte, 4096)
	
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading stdout: %v", err)
			}
			return
		}

		data := buf[:n]
		
		// Write to local stdout
		os.Stdout.Write(data)
		
		// Log output
		c.logOutput(string(data))
	}
}

func (c *PTYClient) handleError(stderr io.Reader) {
	buf := make([]byte, 4096)
	
	for {
		n, err := stderr.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading stderr: %v", err)
			}
			return
		}

		data := buf[:n]
		
		// Write to local stderr
		os.Stderr.Write(data)
		
		// Log error output
		c.logOutput(fmt.Sprintf("[STDERR] %s", string(data)))
	}
}

func (c *PTYClient) handleResize() {
	if c.session == nil {
		return
	}

	width, height := c.getTerminalSize()
	
	err := c.session.WindowChange(height, width)
	if err != nil {
		log.Printf("Error changing window size: %v", err)
	} else {
		c.logSession(fmt.Sprintf("Terminal resized to %dx%d", width, height))
	}
}

func (c *PTYClient) ExecuteCommand(command string) error {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Setup PTY for command execution
	termWidth, termHeight := c.getTerminalSize()
	err = session.RequestPty("xterm-256color", termHeight, termWidth, ssh.TerminalModes{
		ssh.ECHO:          0, // Disable echo for command execution
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	})
	if err != nil {
		return fmt.Errorf("request PTY for command: %w", err)
	}

	// Log command
	c.logCommand(command)
	c.logSession(fmt.Sprintf("Executing command: %s", command))

	// Run command and capture output
	output, err := session.CombinedOutput(command)
	if err != nil {
		c.logSession(fmt.Sprintf("Command failed: %v", err))
		return fmt.Errorf("command execution failed: %w", err)
	}

	// Display and log output
	fmt.Print(string(output))
	c.logOutput(string(output))

	return nil
}

func (c *PTYClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logSession("Closing PTY client")

	// Restore terminal
	c.restoreTerminal()

	// Close log files
	if c.logFile != nil {
		c.logFile.Close()
	}
	if c.cmdLogFile != nil {
		c.cmdLogFile.Close()
	}
	if c.outLogFile != nil {
		c.outLogFile.Close()
	}

	// Close SSH session
	if c.session != nil {
		c.session.Close()
	}

	// Close SSH client
	if c.sshClient != nil {
		return c.sshClient.Close()
	}

	return nil
}
