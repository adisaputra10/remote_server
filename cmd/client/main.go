package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"remote-tunnel/internal/tunnel"
)

func main() {
	var (
		localAddr = flag.String("L", "", "Local listen address (e.g., :2222)")
		relayURL  = flag.String("relay-url", "", "Relay WebSocket URL (e.g., wss://relay.example.com/ws/client)")
		agentID   = flag.String("agent", "", "Target agent ID")
		target    = flag.String("target", "", "Target address (e.g., 127.0.0.1:22)")
		token     = flag.String("token", "", "Auth token (or set TUNNEL_TOKEN env)")
		insecure  = flag.Bool("insecure", false, "Skip TLS certificate verification (for self-signed certificates)")
	)
	flag.Parse()

	// Validate required flags
	if *localAddr == "" {
		log.Fatal("Local address required: use -L flag")
	}
	if *relayURL == "" {
		log.Fatal("Relay URL required: use -relay-url flag")
	}
	if *agentID == "" {
		log.Fatal("Agent ID required: use -agent flag")
	}
	if *target == "" {
		log.Fatal("Target address required: use -target flag")
	}

	// Get token from env if not provided
	if *token == "" {
		*token = os.Getenv("TUNNEL_TOKEN")
	}
	if *token == "" {
		log.Fatal("Token required: use -token flag or TUNNEL_TOKEN env var")
	}

	log.Printf("Starting client")
	log.Printf("Local address: %s", *localAddr)
	log.Printf("Relay URL: %s", *relayURL)
	log.Printf("Agent ID: %s", *agentID)
	log.Printf("Target: %s", *target)
	if *insecure {
		log.Printf("TLS certificate verification disabled (insecure mode)")
	}

	// Create client
	client := tunnel.NewClient(*localAddr, *relayURL, *agentID, *target, *token)
	if *insecure {
		client.SetInsecure(true)
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Printf("Shutting down client...")
		client.Close()
	}()

	// Run client
	if err := client.Run(); err != nil {
		log.Fatalf("Client failed: %v", err)
	}

	log.Printf("Client stopped")
}
