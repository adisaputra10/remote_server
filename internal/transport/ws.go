package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

const (
	PingInterval = 15 * time.Second
	PongTimeout  = 30 * time.Second
)

// WSConn wraps websocket connection as net.Conn
type WSConn struct {
	net.Conn
	ws         *websocket.Conn
	ctx        context.Context
	done       chan struct{}
	compressed bool
}

func NewWSConn(ctx context.Context, ws *websocket.Conn) *WSConn {
	netConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	return &WSConn{
		Conn:       netConn,
		ws:         ws,
		ctx:        ctx,
		done:       make(chan struct{}),
		compressed: false,
	}
}

func NewWSConnWithCompression(ctx context.Context, ws *websocket.Conn, enableCompression bool) *WSConn {
	netConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	
	wsConn := &WSConn{
		ws:         ws,
		ctx:        ctx,
		done:       make(chan struct{}),
		compressed: enableCompression,
	}
	
	if enableCompression {
		wsConn.Conn = EnableCompression(netConn)
	} else {
		wsConn.Conn = netConn
	}
	
	return wsConn
}

func (w *WSConn) Close() error {
	select {
	case <-w.done:
		return nil
	default:
		close(w.done)
	}
	
	w.ws.Close(websocket.StatusNormalClosure, "")
	return w.Conn.Close()
}

// DialWS connects to WebSocket server with auth token
func DialWS(ctx context.Context, url, token string) (*WSConn, error) {
	return DialWSInsecure(ctx, url, token, false)
}

// DialWSWithCompression connects to WebSocket server with compression
func DialWSWithCompression(ctx context.Context, url, token string, enableCompression bool) (*WSConn, error) {
	return DialWSInsecureWithCompression(ctx, url, token, false, enableCompression)
}

// DialWSInsecure connects to WebSocket server with optional TLS skip verification
func DialWSInsecure(ctx context.Context, url, token string, insecure bool) (*WSConn, error) {
	return DialWSInsecureWithCompression(ctx, url, token, insecure, false)
}

// DialWSInsecureWithCompression connects to WebSocket server with optional TLS skip verification and compression
// DialWSInsecureWithCompression connects to WebSocket server with optional TLS skip verification and compression
func DialWSInsecureWithCompression(ctx context.Context, url, token string, insecure bool, enableCompression bool) (*WSConn, error) {
	opts := &websocket.DialOptions{
		HTTPHeader: http.Header{
			"X-Tunnel-Token": []string{token},
		},
	}
	
	// Add compression header if enabled
	if enableCompression {
		opts.HTTPHeader.Set("X-Tunnel-Compression", "gzip")
	}
	
	// Skip TLS verification if insecure mode is enabled
	if insecure {
		opts.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	
	ws, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	
	return NewWSConnWithCompression(ctx, ws, enableCompression), nil
}

// AcceptWS accepts WebSocket connection and validates token
func AcceptWS(w http.ResponseWriter, r *http.Request, expectedToken string) (*WSConn, error) {
	return AcceptWSWithCompression(w, r, expectedToken, false)
}

// AcceptWSWithCompression accepts WebSocket connection with compression support
func AcceptWSWithCompression(w http.ResponseWriter, r *http.Request, expectedToken string, enableCompression bool) (*WSConn, error) {
	token := r.Header.Get("X-Tunnel-Token")
	if token != expectedToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, fmt.Errorf("invalid token")
	}
	
	// Check if client supports compression
	clientCompression := r.Header.Get("X-Tunnel-Compression") == "gzip"
	useCompression := enableCompression && clientCompression
	
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return nil, fmt.Errorf("websocket accept: %w", err)
	}
	
	return NewWSConnWithCompression(r.Context(), ws, useCompression), nil
}

// StartPingPong starts ping/pong keepalive
func (w *WSConn) StartPingPong() {
	ticker := time.NewTicker(PingInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-w.done:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(w.ctx, PongTimeout)
				err := w.ws.Ping(ctx)
				cancel()
				if err != nil {
					w.Close()
					return
				}
			}
		}
	}()
}

// CreateTLSConfig creates TLS config for development
func CreateTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load cert: %w", err)
	}
	
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
