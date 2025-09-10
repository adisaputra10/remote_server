package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

// Simple server for now - using same structure as old server but modular
func main() {
	var (
		port = flag.Int("port", 8080, "Server port")
		host = flag.String("host", "0.0.0.0", "Server host")
	)
	flag.Parse()

	logger := log.New(os.Stdout, "[SERVER] ", log.LstdFlags|log.Lshortfile)
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	mux := http.NewServeMux()
	
	// Simple WebSocket handlers for now
	mux.HandleFunc("/agent", func(w http.ResponseWriter, r *http.Request) {
		handleAgentConnection(w, r, &upgrader, logger)
	})
	
	mux.HandleFunc("/client", func(w http.ResponseWriter, r *http.Request) {
		handleClientConnection(w, r, &upgrader, logger)
	})
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)
	logger.Printf("Server starting on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Shutting down server...")
		server.Close()
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Server failed: %v", err)
	}

	logger.Println("Server stopped")
}

func handleAgentConnection(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, logger *log.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("Agent upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	logger.Printf("Agent connected from %s", r.RemoteAddr)

	// Simple echo for now
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logger.Printf("Agent read error: %v", err)
			break
		}
		
		logger.Printf("Agent message: %s", string(p))
		
		if err := conn.WriteMessage(messageType, p); err != nil {
			logger.Printf("Agent write error: %v", err)
			break
		}
	}
}

func handleClientConnection(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, logger *log.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("Client upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	logger.Printf("Client connected from %s", r.RemoteAddr)

	// Simple echo for now
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logger.Printf("Client read error: %v", err)
			break
		}
		
		logger.Printf("Client message: %s", string(p))
		
		if err := conn.WriteMessage(messageType, p); err != nil {
			logger.Printf("Client write error: %v", err)
			break
		}
	}
}