package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"remote-tunnel/internal/tunnel"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var (
		id            = flag.String("id", "", "Agent ID")
		relayURL      = flag.String("relay-url", "", "Relay WebSocket URL (e.g., wss://relay.example.com/ws/agent)")
		token         = flag.String("token", "", "Auth token (or set TUNNEL_TOKEN env)")
		insecure      = flag.Bool("insecure", false, "Skip TLS certificate verification (for self-signed certificates)")
		compress      = flag.Bool("compress", false, "Enable gzip compression for data transfer")
		allowed       arrayFlags
	)
	flag.Var(&allowed, "allow", "Allowed target addresses (can be specified multiple times)")
	flag.Parse()

	// Validate required flags
	if *id == "" {
		log.Fatal("Agent ID required: use -id flag")
	}
	if *relayURL == "" {
		log.Fatal("Relay URL required: use -relay-url flag")
	}

	// Get token from env if not provided
	if *token == "" {
		*token = os.Getenv("TUNNEL_TOKEN")
	}
	if *token == "" {
		log.Fatal("Token required: use -token flag or TUNNEL_TOKEN env var")
	}

	// Default allowed hosts if none specified
	if len(allowed) == 0 {
		allowed = append(allowed, "127.0.0.1:")
		log.Printf("No -allow flags specified, defaulting to 127.0.0.1:")
	}

	log.Printf("Starting agent with ID: %s", *id)
	log.Printf("Relay URL: %s", *relayURL)
	log.Printf("Allowed targets: %v", []string(allowed))
	if *insecure {
		log.Printf("TLS certificate verification disabled (insecure mode)")
	}
	if *compress {
		log.Printf("Gzip compression enabled")
	}

	// Create agent
	agent := tunnel.NewAgent(*id, *relayURL, *token, []string(allowed))
	if *insecure {
		agent.SetInsecure(true)
	}
	if *compress {
		agent.SetCompression(true)
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Printf("Shutting down agent...")
		agent.Close()
	}()

	// Run agent
	if err := agent.Run(); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	log.Printf("Agent stopped")
}
