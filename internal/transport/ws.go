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
	ws   *websocket.Conn
	ctx  context.Context
	done chan struct{}
}

func NewWSConn(ctx context.Context, ws *websocket.Conn) *WSConn {
	netConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	return &WSConn{
		Conn: netConn,
		ws:   ws,
		ctx:  ctx,
		done: make(chan struct{}),
	}
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
	opts := &websocket.DialOptions{
		HTTPHeader: http.Header{
			"X-Tunnel-Token": []string{token},
		},
	}
	
	ws, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	
	return NewWSConn(ctx, ws), nil
}

// AcceptWS accepts WebSocket connection and validates token
func AcceptWS(w http.ResponseWriter, r *http.Request, expectedToken string) (*WSConn, error) {
	token := r.Header.Get("X-Tunnel-Token")
	if token != expectedToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, fmt.Errorf("invalid token")
	}
	
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return nil, fmt.Errorf("websocket accept: %w", err)
	}
	
	return NewWSConn(r.Context(), ws), nil
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
