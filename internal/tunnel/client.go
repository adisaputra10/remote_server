package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
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
	}
}

func (c *Client) Run() error {
	// Connect to relay
	err := c.connectToRelay()
	if err != nil {
		return fmt.Errorf("connect to relay: %w", err)
	}
	defer c.session.Close()

	// Start local listener
	err = c.startListener()
	if err != nil {
		return fmt.Errorf("start listener: %w", err)
	}
	defer c.listener.Close()

	log.Printf("Client listening on %s, forwarding to agent %s target %s", 
		c.localAddr, c.agentID, c.targetAddr)

	// Wait for context cancellation
	<-c.ctx.Done()
	return c.ctx.Err()
}

func (c *Client) connectToRelay() error {
	log.Printf("Connecting to relay: %s", c.relayURL)

	wsConn, err := transport.DialWS(c.ctx, c.relayURL, c.token)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	wsConn.StartPingPong()

	session, err := transport.NewMuxClient(wsConn)
	if err != nil {
		wsConn.Close()
		return fmt.Errorf("mux client: %w", err)
	}

	c.session = session

	log.Printf("Connected to relay")
	return nil
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

	// Send DIAL request
	err := c.session.SendControl(&proto.Control{
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
	response, err := c.waitForResponse(streamID)
	if err != nil {
		log.Printf("Failed to get response: %v", err)
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
	relayStream, err := c.session.AcceptStream()
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

func (c *Client) waitForResponse(streamID string) (*proto.Control, error) {
	timeout := time.After(30 * time.Second)
	
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for response")
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		default:
		}

		msg, err := c.session.ReceiveControl()
		if err != nil {
			return nil, fmt.Errorf("receive control: %w", err)
		}

		// Handle keep-alive messages
		switch msg.Type {
		case proto.MsgPing:
			c.session.SendControl(&proto.Control{Type: proto.MsgPong})
			continue
		case proto.MsgPong:
			continue
		}

		// Check if this is the response we're waiting for
		if msg.StreamID == streamID {
			return msg, nil
		}

		// Handle other messages
		log.Printf("Received message for different stream: %s (waiting for %s)", msg.StreamID, streamID)
	}
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
