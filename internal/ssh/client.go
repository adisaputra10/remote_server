package ssh

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SSHClient struct {
	tunnelConn   net.Conn
	sshClient    *ssh.Client
	session      *ssh.Session
	logFile      *os.File
	logEnabled   bool
	logDirectory string
}

type SSHConfig struct {
	Username     string
	Password     string
	PrivateKey   string
	TunnelConn   net.Conn
	LogEnabled   bool
	LogDirectory string
}

func NewSSHClient(config *SSHConfig) (*SSHClient, error) {
	client := &SSHClient{
		tunnelConn:   config.TunnelConn,
		logEnabled:   config.LogEnabled,
		logDirectory: config.LogDirectory,
	}

	// Setup logging if enabled
	if client.logEnabled {
		if err := client.setupLogging(); err != nil {
			return nil, fmt.Errorf("setup logging: %w", err)
		}
	}

	// Create SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For tunnel connections
		Timeout:         30 * time.Second,
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

func (c *SSHClient) setupLogging() error {
	if c.logDirectory == "" {
		c.logDirectory = "ssh-logs"
	}

	// Create log directory
	if err := os.MkdirAll(c.logDirectory, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFilename := filepath.Join(c.logDirectory, fmt.Sprintf("ssh-session_%s.log", timestamp))

	logFile, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}

	c.logFile = logFile

	// Write session header
	header := fmt.Sprintf("=== SSH Session Log ===\n")
	header += fmt.Sprintf("Start Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	header += fmt.Sprintf("Log File: %s\n", logFilename)
	header += strings.Repeat("=", 50) + "\n\n"

	if _, err := c.logFile.WriteString(header); err != nil {
		return fmt.Errorf("write log header: %w", err)
	}

	log.Printf("SSH session logging to: %s", logFilename)
	return nil
}

func (c *SSHClient) StartInteractiveSession() error {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	c.session = session

	// Get terminal size
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("make terminal raw: %w", err)
	}
	defer term.Restore(fd, state)

	termWidth, termHeight, err := term.GetSize(fd)
	if err != nil {
		termWidth, termHeight = 80, 24 // Default size
	}

	// Setup PTY
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		return fmt.Errorf("request pty: %w", err)
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

	// Handle terminal size changes
	go c.handleTerminalResize(session)

	// Handle I/O with logging
	go c.handleStdin(stdin)
	go c.handleStdout(stdout)
	go c.handleStderr(stderr)

	// Wait for session to end
	if err := session.Wait(); err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			log.Printf("SSH session ended with exit code: %d", exitError.ExitStatus())
		} else {
			log.Printf("SSH session error: %v", err)
		}
	}

	return nil
}

func (c *SSHClient) handleStdin(stdin io.WriteCloser) {
	defer stdin.Close()

	reader := bufio.NewReader(os.Stdin)
	var inputBuffer strings.Builder

	for {
		char := make([]byte, 1)
		n, err := reader.Read(char)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading stdin: %v", err)
			}
			return
		}

		if n > 0 {
			// Write to SSH session
			if _, err := stdin.Write(char); err != nil {
				log.Printf("Error writing to SSH: %v", err)
				return
			}

			// Build command for logging
			if char[0] == '\r' || char[0] == '\n' {
				// Command completed, log it
				command := strings.TrimSpace(inputBuffer.String())
				if command != "" && c.logEnabled {
					c.logCommand(command)
				}
				inputBuffer.Reset()
			} else if char[0] == 127 || char[0] == 8 { // Backspace or DEL
				// Handle backspace
				current := inputBuffer.String()
				if len(current) > 0 {
					inputBuffer.Reset()
					inputBuffer.WriteString(current[:len(current)-1])
				}
			} else if char[0] >= 32 { // Printable characters
				inputBuffer.WriteByte(char[0])
			}
		}
	}
}

func (c *SSHClient) handleStdout(stdout io.Reader) {
	buffer := make([]byte, 1024)
	
	for {
		n, err := stdout.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading stdout: %v", err)
			}
			return
		}

		if n > 0 {
			// Write to terminal
			os.Stdout.Write(buffer[:n])

			// Log output if enabled
			if c.logEnabled {
				c.logOutput("stdout", buffer[:n])
			}
		}
	}
}

func (c *SSHClient) handleStderr(stderr io.Reader) {
	buffer := make([]byte, 1024)
	
	for {
		n, err := stderr.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading stderr: %v", err)
			}
			return
		}

		if n > 0 {
			// Write to terminal
			os.Stderr.Write(buffer[:n])

			// Log output if enabled
			if c.logEnabled {
				c.logOutput("stderr", buffer[:n])
			}
		}
	}
}

func (c *SSHClient) handleTerminalResize(session *ssh.Session) {
	// Monitor terminal size changes (simplified implementation)
	// In production, you'd want to handle SIGWINCH signal
	for {
		time.Sleep(1 * time.Second)
		
		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			width, height, err := term.GetSize(fd)
			if err == nil {
				session.WindowChange(height, width)
			}
		}
	}
}

func (c *SSHClient) logCommand(command string) {
	if c.logFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] COMMAND: %s\n", timestamp, command)

	if _, err := c.logFile.WriteString(logEntry); err != nil {
		log.Printf("Error writing to log file: %v", err)
	}

	// Flush to ensure immediate write
	c.logFile.Sync()
}

func (c *SSHClient) logOutput(stream string, data []byte) {
	if c.logFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// Clean output for logging (remove control characters for readability)
	cleaned := strings.Map(func(r rune) rune {
		if r >= 32 && r < 127 || r == '\n' || r == '\t' {
			return r
		}
		return -1
	}, string(data))

	if strings.TrimSpace(cleaned) != "" {
		logEntry := fmt.Sprintf("[%s] %s: %s", timestamp, strings.ToUpper(stream), cleaned)
		if !strings.HasSuffix(logEntry, "\n") {
			logEntry += "\n"
		}

		if _, err := c.logFile.WriteString(logEntry); err != nil {
			log.Printf("Error writing to log file: %v", err)
		}
	}
}

func (c *SSHClient) ExecuteCommand(command string) (string, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Log command execution
	if c.logEnabled {
		c.logCommand(fmt.Sprintf("EXEC: %s", command))
	}

	output, err := session.CombinedOutput(command)
	if err != nil {
		if c.logEnabled {
			c.logOutput("error", []byte(fmt.Sprintf("Command failed: %v", err)))
		}
		return string(output), fmt.Errorf("execute command: %w", err)
	}

	// Log output
	if c.logEnabled {
		c.logOutput("exec_output", output)
	}

	return string(output), nil
}

func (c *SSHClient) Close() error {
	if c.logFile != nil {
		// Write session footer
		footer := fmt.Sprintf("\n%s\n", strings.Repeat("=", 50))
		footer += fmt.Sprintf("End Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		footer += "=== Session Ended ===\n"
		
		c.logFile.WriteString(footer)
		c.logFile.Close()
	}

	if c.sshClient != nil {
		return c.sshClient.Close()
	}

	return nil
}
