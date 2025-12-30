package transcripts

import (
	"context"
	"fmt"
)

// Conn is a connection to the OpenAI Realtime API.
type Conn struct {
	logger Logger
	conn   WebSocketConn
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// SendMessageRaw sends a raw message to the server.
func (c *Conn) SendMessageRaw(ctx context.Context, data []byte) error {
	return c.conn.WriteMessage(ctx, MessageText, data)
}

// SendMessage sends a client event to the server.
func (c *Conn) SendMessage(ctx context.Context, msg ClientEvent) error {
	data, err := MarshalClientEvent(msg)
	if err != nil {
		return err
	}
	return c.SendMessageRaw(ctx, data)
}

// ReadMessageRaw reads a raw message from the server.
func (c *Conn) ReadMessageRaw(ctx context.Context) ([]byte, error) {
	messageType, data, err := c.conn.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	if messageType != MessageText {
		return nil, fmt.Errorf("expected text message, got %d", messageType)
	}
	return data, nil
}

// ReadMessage reads a server event from the server.
func (c *Conn) ReadMessage(ctx context.Context) (ServerEvent, error) {
	data, err := c.ReadMessageRaw(ctx)
	if err != nil {
		return nil, err
	}
	event, err := UnmarshalServerEvent(data)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// Ping sends a ping message to the WebSocket connection.
func (c *Conn) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}
