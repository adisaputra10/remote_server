package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	*log.Logger
	component string
	logFile   *os.File
}

func New(component string) *Logger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("logs/%s-%s.log", component, timestamp)
	
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		logFile = nil
	}

	logger := &Logger{
		Logger:    log.New(os.Stdout, fmt.Sprintf("[%s] ", component), log.LstdFlags|log.Lshortfile),
		component: component,
		logFile:   logFile,
	}

	// Also write to file if available
	if logFile != nil {
		logger.Logger.SetOutput(os.Stdout)
		// Create a separate file logger
		go logger.startFileLogging()
	}

	return logger
}

func (l *Logger) startFileLogging() {
	if l.logFile != nil {
		fileLogger := log.New(l.logFile, fmt.Sprintf("[%s] ", l.component), log.LstdFlags|log.Lshortfile)
		// This is a simplified approach; in production you'd want to use a proper logging library
		_ = fileLogger
	}
}

func (l *Logger) Logf(level string, format string, v ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, v...)
	logMessage := fmt.Sprintf("%s [%s] %s", timestamp, level, message)
	
	// Print to console
	fmt.Println(logMessage)
	
	// Write to file
	if l.logFile != nil {
		l.logFile.WriteString(logMessage + "\n")
		l.logFile.Sync()
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.Logf("INFO", format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.Logf("WARN", format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.Logf("ERROR", format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.Logf("DEBUG", format, v...)
}

func (l *Logger) Command(cmd string, args ...interface{}) {
	if len(args) > 0 {
		l.Logf("COMMAND", "%s %v", cmd, args)
	} else {
		l.Logf("COMMAND", "%s", cmd)
	}
}

func (l *Logger) Connection(action, remote string) {
	l.Logf("CONNECTION", "%s: %s", action, remote)
}

func (l *Logger) Tunnel(action, tunnelID string, details ...string) {
	if len(details) > 0 {
		l.Logf("TUNNEL", "%s [%s]: %s", action, tunnelID, details[0])
	} else {
		l.Logf("TUNNEL", "%s [%s]", action, tunnelID)
	}
}

func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
