package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"remote-tunnel/internal/proto"
	"remote-tunnel/internal/transport"
)

type Client struct {
	localAddr  string
	relayURL   string
	agentID    string
	targetAddr string
	token      string
	session    *transport.MuxSession
	listener   net.Listener
	ctx        context.Context
	cancel     context.CancelFunc
	insecure   bool
	compress   bool

	controlMutex   sync.RWMutex
	connected      bool
	responses      map[string]chan *proto.Control
	responsesMutex sync.RWMutex
}

func NewClient(localAddr, relayURL, agentID, targetAddr, token string) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		localAddr:  localAddr,
		relayURL:   relayURL,
		agentID:    agentID,
		targetAddr: targetAddr,
		token:      token,
		ctx:        ctx,
		cancel:     cancel,
		insecure:   false,
		compress:   false,
		responses:  make(map[string]chan *proto.Control),
	}
}

func (c *Client) SetInsecure(insecure bool) {
	c.insecure = insecure
}

func (c *Client) SetCompression(compress bool) {
	c.compress = compress
}

func (c *Client) Run() error {
	// Start local listener
	err := c.startListener()
	if err != nil {
		return fmt.Errorf("start listener: %w", err)
	}
	defer c.listener.Close()

	log.Printf("Client listening on %s, forwarding to agent %s target %s", 
		c.localAddr, c.agentID, c.targetAddr)

	// Start connection and message handling
	go c.connectionLoop()

	// Wait for context cancellation
	<-c.ctx.Done()
	return c.ctx.Err()
}

func (c *Client) connectToRelay() error {
	log.Printf("Connecting to relay: %s (goroutine starting)", c.relayURL)

	// Close any existing session first
	c.controlMutex.Lock()
	if c.session != nil {
		c.session.Close()
		c.session = nil
	}
	c.controlMutex.Unlock()

	// Add a small delay to ensure cleanup
	time.Sleep(200 * time.Millisecond)

	wsConn, err := transport.DialWSInsecureWithCompression(c.ctx, c.relayURL, c.token, c.insecure, c.compress)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	wsConn.StartPingPong()

	// Add delay before creating mux session
	time.Sleep(100 * time.Millisecond)

	session, err := transport.NewMuxClientWithCompression(wsConn, c.compress)
	if err != nil {
		wsConn.Close()
		return fmt.Errorf("mux client: %w", err)
	}

	c.session = session

	log.Printf("Connected to relay (goroutine completed)")
	return nil
}

func (c *Client) connectionLoop() {
	log.Printf("Starting connection loop")
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	
	for {
		select {
		case <-c.ctx.Done():
			log.Printf("Connection loop exiting due to context cancellation")
			return
		default:
		}

		// Connect to relay
		log.Printf("Attempting connection to relay...")
		err := c.connectToRelay()
		if err != nil {
			log.Printf("Failed to connect to relay: %v", err)
			log.Printf("Retrying in %v...", backoff)
			time.Sleep(backoff)
			
			// Exponential backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on successful connection
		backoff = 1 * time.Second

		c.controlMutex.Lock()
		c.connected = true
		session := c.session
		c.controlMutex.Unlock()

		log.Printf("Control message handler started")

		// Handle control messages until connection fails
		c.handleControlMessages(session)

		// Connection failed, mark as disconnected
		c.markDisconnected()
		log.Printf("Disconnected from relay, will retry...")
		time.Sleep(2 * time.Second)
	}
}

func (c *Client) handleControlMessages(session *transport.MuxSession) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		msg, err := session.ReceiveControl()
		if err != nil {
			log.Printf("Control message receive error: %v", err)
			return
		}

		// Handle control messages
		switch msg.Type {
		case proto.MsgPing:
			session.SendControl(&proto.Control{Type: proto.MsgPong})
		case proto.MsgPong:
			// Ignore pong messages
		case proto.MsgAccept, proto.MsgRefuse:
			// Route to waiting handler
			c.responsesMutex.RLock()
			respChan, exists := c.responses[msg.StreamID]
			c.responsesMutex.RUnlock()
			
			if exists {
				select {
				case respChan <- msg:
				default:
					log.Printf("Response channel full for stream %s", msg.StreamID)
				}
			} else {
				log.Printf("No waiting handler for stream %s", msg.StreamID)
			}
		default:
			log.Printf("Unknown control message type: %s", msg.Type)
		}
	}
}

func (c *Client) markDisconnected() {
	c.controlMutex.Lock()
	defer c.controlMutex.Unlock()
	
	if c.connected {
		c.connected = false
		if c.session != nil {
			c.session.Close()
			c.session = nil
		}
	}
}

func (c *Client) startListener() error {
	listener, err := net.Listen("tcp", c.localAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	c.listener = listener

	go c.acceptLoop()
	return nil
}

func (c *Client) acceptLoop() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		go c.handleConnection(conn)
	}
}

func (c *Client) handleConnection(localConn net.Conn) {
	defer localConn.Close()

	streamID := uuid.New().String()
	log.Printf("New connection, stream ID: %s", streamID)

	// Wait for connection to be established
	var session *transport.MuxSession
	for i := 0; i < 10; i++ { // Wait up to 10 seconds
		c.controlMutex.RLock()
		if c.connected && c.session != nil {
			session = c.session
			c.controlMutex.RUnlock()
			break
		}
		c.controlMutex.RUnlock()
		
		log.Printf("Waiting for relay connection... (%d/10)", i+1)
		time.Sleep(1 * time.Second)
	}

	if session == nil {
		log.Printf("No connection to relay available, dropping connection")
		return
	}

	// Create response channel
	respChan := make(chan *proto.Control, 1)
	c.responsesMutex.Lock()
	c.responses[streamID] = respChan
	c.responsesMutex.Unlock()

	defer func() {
		c.responsesMutex.Lock()
		delete(c.responses, streamID)
		c.responsesMutex.Unlock()
		close(respChan)
	}()

	// Send DIAL request
	err := session.SendControl(&proto.Control{
		Type:       proto.MsgDial,
		AgentID:    c.agentID,
		StreamID:   streamID,
		TargetAddr: c.targetAddr,
	})
	if err != nil {
		log.Printf("Failed to send dial: %v", err)
		return
	}

	// Wait for ACCEPT/REFUSE
	var response *proto.Control
	select {
	case response = <-respChan:
	case <-time.After(30 * time.Second):
		log.Printf("Timeout waiting for response for stream %s", streamID)
		return
	case <-c.ctx.Done():
		return
	}

	if response.Type == proto.MsgRefuse {
		log.Printf("Dial refused: %s", response.Error)
		return
	}

	if response.Type != proto.MsgAccept {
		log.Printf("Unexpected response: %s", response.Type)
		return
	}

	log.Printf("Dial accepted for stream %s", streamID)

	// Accept stream from relay
	relayStream, err := session.AcceptStream()
	if err != nil {
		log.Printf("Failed to accept relay stream: %v", err)
		return
	}
	defer relayStream.Close()

	log.Printf("Bridging local connection with relay stream %s", streamID)

	// Bridge connections
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = transport.Bridge(ctx, localConn, relayStream)
	if err != nil {
		log.Printf("Bridge error: %v", err)
	}

	log.Printf("Connection closed for stream %s", streamID)
}

func (c *Client) Close() error {
	c.cancel()
	
	if c.listener != nil {
		c.listener.Close()
	}
	
	if c.session != nil {
		return c.session.Close()
	}
	
	return nil
}
