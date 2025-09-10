package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	
	"ssh-terminal/internal/agent"
)

func main() {
	var (
		configFile = flag.String("config", "agent-config.json", "Path to agent configuration file")
		logFile    = flag.String("log", "", "Path to log file (default: stdout)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()
	
	// Setup logger
	var logger *log.Logger
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer file.Close()
		logger = log.New(file, "[AGENT] ", log.LstdFlags|log.Lmicroseconds)
	} else {
		logger = log.New(os.Stdout, "[AGENT] ", log.LstdFlags|log.Lmicroseconds)
	}
	
	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	
	// Override log file from config if not set via flag
	if *logFile == "" && config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Fatalf("Failed to open config log file: %v", err)
		}
		defer file.Close()
		logger = log.New(file, "[AGENT] ", log.LstdFlags|log.Lmicroseconds)
	}
	
	if *verbose {
		logger.Printf("🔧 Loaded configuration: %+v", config)
	}
	
	// Create agent
	ag := agent.NewAgent(config, logger)
	
	// Start agent
	if err := ag.Start(); err != nil {
		logger.Fatalf("Failed to start agent: %v", err)
	}
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	logger.Printf("🚀 Agent started successfully. Press Ctrl+C to stop.")
	
	// Wait for signal
	sig := <-sigChan
	logger.Printf("📡 Received signal: %v", sig)
	
	// Stop agent
	ag.Stop()
	logger.Printf("👋 Agent stopped")
}

// loadConfig loads agent configuration from file
func loadConfig(filename string) (*agent.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()
	
	var config agent.Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	
	// Set defaults
	if config.Platform == "" {
		config.Platform = "unknown"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.Metadata == nil {
		config.Metadata = make(map[string]string)
	}
	
	return &config, nil
}
