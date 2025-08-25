package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type GoTeleportAgent struct {
	config     *AgentConfig
	conn       *websocket.Conn
	logger     *log.Logger
	agentID    string
	sessions   map[string]*AgentSession
}

type AgentConfig struct {
	ServerURL    string            `json:"server_url"`
	AgentName    string            `json:"agent_name"`
	Platform     string            `json:"platform"`
	LogFile      string            `json:"log_file"`
	AuthToken    string            `json:"auth_token"`
	Metadata     map[string]string `json:"metadata"`
	WorkingDir   string            `json:"working_dir"`
	AllowedUsers []string          `json:"allowed_users"`
}

type AgentSession struct {
	ID          string
	ClientID    string
	WorkingDir  string
	Environment map[string]string
	CreatedAt   time.Time
	LastUsed    time.Time
}

type Message struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Command   string                 `json:"command,omitempty"`
	Data      string                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type CommandResult struct {
	Command   string        `json:"command"`
	Output    string        `json:"output"`
	Error     string        `json:"error"`
	ExitCode  int           `json:"exit_code"`
	Duration  time.Duration `json:"duration"`
	WorkingDir string       `json:"working_dir"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: goteleport-agent.exe <config-file>")
	}

	agent, err := NewGoTeleportAgent(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	agent.Start()
}

func NewGoTeleportAgent(configFile string) (*GoTeleportAgent, error) {
	// Read config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Set defaults
	if config.Platform == "" {
		config.Platform = runtime.GOOS
	}
	if config.WorkingDir == "" {
		config.WorkingDir, _ = os.Getwd()
	}

	// Setup logger
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "", log.LstdFlags)

	agent := &GoTeleportAgent{
		config:   &config,
		logger:   logger,
		sessions: make(map[string]*AgentSession),
	}

	return agent, nil
}

func (a *GoTeleportAgent) Start() {
	a.logEvent("AGENT_START", "GoTeleport Agent starting", a.config.AgentName)

	for {
		if err := a.connect(); err != nil {
			a.logEvent("ERROR", "Connection failed", err.Error())
			fmt.Printf("❌ Connection failed: %v\n", err)
			fmt.Println("🔄 Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}

		// Connection lost, retry
		fmt.Println("🔄 Connection lost, retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func (a *GoTeleportAgent) connect() error {
	fmt.Printf("🔗 Connecting to server: %s\n", a.config.ServerURL)

	// Connect to server
	conn, _, err := websocket.DefaultDialer.Dial(a.config.ServerURL+"/ws/agent", nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	a.conn = conn

	// Register with server
	if err := a.register(); err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	fmt.Printf("✅ Connected and registered as: %s\n", a.config.AgentName)
	a.logEvent("AGENT_CONNECT", "Connected to server", a.config.ServerURL)

	// Start heartbeat
	go a.heartbeat()

	// Handle messages
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			a.logEvent("AGENT_DISCONNECT", "Disconnected from server", err.Error())
			return err
		}

		go a.handleMessage(&msg)
	}
}

func (a *GoTeleportAgent) register() error {
	regMsg := Message{
		Type: "register",
		Metadata: map[string]interface{}{
			"name":        a.config.AgentName,
			"platform":    a.config.Platform,
			"working_dir": a.config.WorkingDir,
			"auth_token":  a.config.AuthToken,
			"metadata":    a.config.Metadata,
		},
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(regMsg); err != nil {
		return err
	}

	// Wait for registration response
	var response Message
	if err := a.conn.ReadJSON(&response); err != nil {
		return err
	}

	if response.Type != "registered" {
		return fmt.Errorf("registration failed: %s", response.Type)
	}

	a.agentID = response.AgentID
	return nil
}

func (a *GoTeleportAgent) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if a.conn == nil {
			return
		}

		heartbeatMsg := Message{
			Type:      "heartbeat",
			AgentID:   a.agentID,
			Timestamp: time.Now(),
		}

		if err := a.conn.WriteJSON(heartbeatMsg); err != nil {
			a.logEvent("ERROR", "Failed to send heartbeat", err.Error())
			return
		}
	}
}

func (a *GoTeleportAgent) handleMessage(msg *Message) {
	switch msg.Type {
	case "command":
		a.executeCommand(msg)
	case "session_start":
		a.createSession(msg)
	case "session_end":
		a.endSession(msg.SessionID)
	default:
		a.logEvent("UNKNOWN_MSG", "Unknown message type", msg.Type)
	}
}

func (a *GoTeleportAgent) createSession(msg *Message) {
	session := &AgentSession{
		ID:          msg.SessionID,
		ClientID:    msg.ClientID,
		WorkingDir:  a.config.WorkingDir,
		Environment: make(map[string]string),
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
	}

	a.sessions[msg.SessionID] = session
	a.logEvent("SESSION_CREATE", "Session created", msg.SessionID)
}

func (a *GoTeleportAgent) endSession(sessionID string) {
	delete(a.sessions, sessionID)
	a.logEvent("SESSION_END", "Session ended", sessionID)
}

func (a *GoTeleportAgent) executeCommand(msg *Message) {
	startTime := time.Now()
	
	// Get or create session
	session, exists := a.sessions[msg.SessionID]
	if !exists {
		session = &AgentSession{
			ID:          msg.SessionID,
			WorkingDir:  a.config.WorkingDir,
			Environment: make(map[string]string),
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
		}
		a.sessions[msg.SessionID] = session
	}

	session.LastUsed = time.Now()

	a.logEvent("COMMAND_START", "Executing command", fmt.Sprintf("Session: %s, Command: %s", msg.SessionID, msg.Command))

	// Execute command
	result := a.runCommand(msg.Command, session)
	result.Duration = time.Since(startTime)

	// Send result back to server
	response := Message{
		Type:      "command_result",
		SessionID: msg.SessionID,
		AgentID:   a.agentID,
		Data:      a.formatCommandResult(result),
		Metadata: map[string]interface{}{
			"result": result,
		},
		Timestamp: time.Now(),
	}

	if err := a.conn.WriteJSON(response); err != nil {
		a.logEvent("ERROR", "Failed to send command result", err.Error())
	}

	a.logEvent("COMMAND_COMPLETE", "Command completed", fmt.Sprintf("Command: %s, Exit: %d, Duration: %v", msg.Command, result.ExitCode, result.Duration))
}

func (a *GoTeleportAgent) runCommand(command string, session *AgentSession) *CommandResult {
	result := &CommandResult{
		Command:    command,
		WorkingDir: session.WorkingDir,
	}

	// Handle built-in commands
	if strings.HasPrefix(command, "cd ") {
		return a.handleCD(command, session)
	}

	// Convert to appropriate command based on platform
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}

	// Set working directory
	cmd.Dir = session.WorkingDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range session.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

func (a *GoTeleportAgent) handleCD(command string, session *AgentSession) *CommandResult {
	parts := strings.Fields(command)
	if len(parts) < 2 {
		return &CommandResult{
			Command:    command,
			Error:      "cd: missing directory argument",
			ExitCode:   1,
			WorkingDir: session.WorkingDir,
		}
	}

	newDir := parts[1]
	if !strings.HasPrefix(newDir, "/") && !strings.Contains(newDir, ":") {
		// Relative path
		newDir = session.WorkingDir + string(os.PathSeparator) + newDir
	}

	// Check if directory exists
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		return &CommandResult{
			Command:    command,
			Error:      fmt.Sprintf("cd: %s: No such file or directory", newDir),
			ExitCode:   1,
			WorkingDir: session.WorkingDir,
		}
	}

	// Change directory
	session.WorkingDir = newDir
	
	return &CommandResult{
		Command:    command,
		Output:     fmt.Sprintf("Changed directory to: %s", newDir),
		ExitCode:   0,
		WorkingDir: newDir,
	}
}

func (a *GoTeleportAgent) formatCommandResult(result *CommandResult) string {
	var output strings.Builder
	
	if result.Output != "" {
		output.WriteString(result.Output)
	}
	
	if result.Error != "" {
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		output.WriteString("Error: " + result.Error)
	}
	
	output.WriteString(fmt.Sprintf("\n[Exit: %d, Duration: %v, Dir: %s]", 
		result.ExitCode, result.Duration, result.WorkingDir))
	
	return output.String()
}

func (a *GoTeleportAgent) logEvent(eventType, description, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "system"
	}

	logEntry := fmt.Sprintf("[%s] [%s] User: %s | Agent: %s | Event: %s | Details: %s",
		timestamp, eventType, user, a.config.AgentName, description, details)

	if a.logger != nil {
		a.logger.Println(logEntry)
	}

	// Also print to stdout for important events
	if eventType == "AGENT_START" || eventType == "AGENT_CONNECT" || eventType == "ERROR" {
		fmt.Printf("📝 %s\n", logEntry)
	}
}
