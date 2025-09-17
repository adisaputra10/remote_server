package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	// TimeFormat is the standard timestamp format for logs
	TimeFormat = "2006/01/02 15:04:05"
)

// GenerateID generates a random ID
func GenerateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// GetCurrentTimestamp returns current Unix timestamp
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// Logger is a simple logger wrapper
type Logger struct {
	prefix    string
	debugMode bool
	logFile   *os.File
	logger    *log.Logger
}

// NewLogger creates a new logger with prefix
func NewLogger(prefix string) *Logger {
	debugMode := os.Getenv("DEBUG") == "1" || os.Getenv("TUNNEL_DEBUG") == "1"

	logger := &Logger{
		prefix:    prefix,
		debugMode: debugMode,
	}

	// Setup file logging
	logger.setupFileLogging()

	return logger
}

// setupFileLogging initializes file logging
func (l *Logger) setupFileLogging() {
	// Create logs directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return
	}

	// Create log file with simple consistent name
	var logFileName string
	switch l.prefix {
	case "RELAY":
		logFileName = "server-relay.log"
	case "CLIENT":
		logFileName = "client.log"
	default:
		// For agents, use agent.log regardless of agent name
		if len(l.prefix) > 6 && l.prefix[:6] == "AGENT-" {
			logFileName = "agent.log"
		} else {
			logFileName = fmt.Sprintf("%s.log", l.prefix)
		}
	}

	logPath := filepath.Join(logDir, logFileName)

	// Open file in append mode to allow multiple instances to write
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file %s: %v", logPath, err)
		return
	}

	l.logFile = file

	// Create multi-writer to write to both console and file
	multiWriter := io.MultiWriter(os.Stdout, file)
	l.logger = log.New(multiWriter, "", 0)

	log.Printf("Logging to file: %s", logPath)
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// Info logs info message
func (l *Logger) Info(format string, args ...interface{}) {
	timestamp := time.Now().Format(TimeFormat)
	message := fmt.Sprintf("%s [%s] INFO: "+format, append([]interface{}{timestamp, l.prefix}, args...)...)

	if l.logger != nil {
		l.logger.Println(message)
	} else {
		log.Println(message)
	}
}

// Error logs error message
func (l *Logger) Error(format string, args ...interface{}) {
	timestamp := time.Now().Format(TimeFormat)
	message := fmt.Sprintf("%s [%s] ERROR: "+format, append([]interface{}{timestamp, l.prefix}, args...)...)

	if l.logger != nil {
		l.logger.Println(message)
	} else {
		log.Println(message)
	}
}

// Debug logs debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debugMode {
		timestamp := time.Now().Format(TimeFormat)
		message := fmt.Sprintf("%s [%s] DEBUG: "+format, append([]interface{}{timestamp, l.prefix}, args...)...)

		if l.logger != nil {
			l.logger.Println(message)
		} else {
			log.Println(message)
		}
	}
}

// Warn logs warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	timestamp := time.Now().Format(TimeFormat)
	message := fmt.Sprintf("%s [%s] WARN: "+format, append([]interface{}{timestamp, l.prefix}, args...)...)

	if l.logger != nil {
		l.logger.Println(message)
	} else {
		log.Println(message)
	}
}

// Fatal logs fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	timestamp := time.Now().Format(TimeFormat)
	message := fmt.Sprintf("%s [%s] FATAL: "+format, append([]interface{}{timestamp, l.prefix}, args...)...)

	if l.logger != nil {
		l.logger.Println(message)
	} else {
		log.Println(message)
	}

	if l.logFile != nil {
		l.logFile.Close()
	}
	os.Exit(1)
}

// IsPortOpen checks if a port is open on the given host
func IsPortOpen(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetLocalIP returns the local IP address
func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
