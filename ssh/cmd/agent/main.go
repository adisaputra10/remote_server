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

	logger.Printf("🔧 Loaded config: ID=%s, ServerURL=%s", config.ID, config.ServerURL)

	if *verbose {
		logger.Printf("🔧 Loaded configuration: %+v", config)
	}

	// Create agent
	logger.Printf("🔧 Creating agent...")
	ag := agent.NewAgent(config, logger)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent in goroutine
	logger.Printf("🚀 Starting agent...")
	go func() {
		if err := ag.Run(); err != nil {
			logger.Printf("❌ Agent error: %v", err)
		}
	}()

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

	return &config, nil
}
