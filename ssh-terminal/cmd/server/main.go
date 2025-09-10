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
	logger.Printf("🚀 Server starting on %s", addr)
	logger.Printf("📋 Server configuration:")
	logger.Printf("   - Host: %s", *host)
	logger.Printf("   - Port: %d", *port)
	logger.Printf("   - Agent endpoint: /agent")
	logger.Printf("   - Client endpoint: /client")
	logger.Printf("   - Health endpoint: /health")
	
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("🛑 Shutting down server...")
		server.Close()
	}()

	logger.Printf("✅ Server ready! Waiting for connections...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("❌ Server failed: %v", err)
	}

	logger.Println("👋 Server stopped")
}

func handleAgentConnection(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, logger *log.Logger) {
	logger.Printf("🔗 New agent connection attempt from %s", r.RemoteAddr)
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("❌ Agent upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	logger.Printf("✅ Agent connected successfully from %s", r.RemoteAddr)

	// Handle messages
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logger.Printf("❌ Agent read error from %s: %v", r.RemoteAddr, err)
			break
		}
		
		logger.Printf("📨 Agent message from %s [%d bytes]: %s", r.RemoteAddr, len(p), string(p))
		
		// Echo back
		if err := conn.WriteMessage(messageType, p); err != nil {
			logger.Printf("❌ Agent write error to %s: %v", r.RemoteAddr, err)
			break
		}
		
		logger.Printf("📤 Echoed message back to agent %s", r.RemoteAddr)
	}
	
	logger.Printf("🔌 Agent disconnected: %s", r.RemoteAddr)
}

func handleClientConnection(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, logger *log.Logger) {
	logger.Printf("🔗 New client connection attempt from %s", r.RemoteAddr)
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("❌ Client upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	logger.Printf("✅ Client connected successfully from %s", r.RemoteAddr)

	// Handle messages
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logger.Printf("❌ Client read error from %s: %v", r.RemoteAddr, err)
			break
		}
		
		logger.Printf("📨 Client message from %s [%d bytes]: %s", r.RemoteAddr, len(p), string(p))
		
		// Echo back
		if err := conn.WriteMessage(messageType, p); err != nil {
			logger.Printf("❌ Client write error to %s: %v", r.RemoteAddr, err)
			break
		}
		
		logger.Printf("📤 Echoed message back to client %s", r.RemoteAddr)
	}
	
	logger.Printf("🔌 Client disconnected: %s", r.RemoteAddr)
}