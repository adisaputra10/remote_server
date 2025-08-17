package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"remote-tunnel/internal/relay"
	"remote-tunnel/internal/transport"
)

func main() {
	var (
		addr     = flag.String("addr", ":443", "Server address")
		certFile = flag.String("cert", "server.crt", "TLS certificate file")
		keyFile  = flag.String("key", "server.key", "TLS private key file")
		token    = flag.String("token", "", "Auth token (or set TUNNEL_TOKEN env)")
	)
	flag.Parse()

	// Get token from env if not provided
	if *token == "" {
		*token = os.Getenv("TUNNEL_TOKEN")
	}
	if *token == "" {
		log.Fatal("Token required: use -token flag or TUNNEL_TOKEN env var")
	}

	// Generate self-signed cert if not exists
	if _, err := os.Stat(*certFile); os.IsNotExist(err) {
		log.Printf("Generating self-signed certificate...")
		err := generateSelfSignedCert(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate certificate: %v", err)
		}
	}

	// Load TLS config
	tlsConfig, err := transport.CreateTLSConfig(*certFile, *keyFile)
	if err != nil {
		log.Fatalf("Failed to load TLS config: %v", err)
	}

	// Create relay server
	server := relay.NewServer(*token)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/agent", server.HandleAgent)
	mux.HandleFunc("/ws/client", server.HandleClient)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create HTTPS server
	httpServer := &http.Server{
		Addr:      *addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Printf("Shutting down server...")
		server.Close()
		httpServer.Close()
	}()

	log.Printf("Relay server starting on %s", *addr)
	log.Printf("Agent endpoint: wss://localhost%s/ws/agent", *addr)
	log.Printf("Client endpoint: wss://localhost%s/ws/client", *addr)

	if err := httpServer.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	log.Printf("Server stopped")
}

func generateSelfSignedCert(certFile, keyFile string) error {
	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Remote Tunnel"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Save certificate
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Save private key
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	return nil
}
