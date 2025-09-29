/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package trans

import (
	"context"
	"sync"
	"time"

	"seata-go-ai-txn-agent/pkg/utils"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	ID        string
	SessionID string
	Conn      *websocket.Conn
	Send      chan WebSocketMessage
	Hub       *Hub
	mu        sync.RWMutex
	logger    *utils.Logger
}

// NewClient creates a new WebSocket client
func NewClient(id, sessionID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:        id,
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan WebSocketMessage, 256),
		Hub:       hub,
		logger:    utils.GetLogger("websocket-client"),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline and pong handler
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg WebSocketMessage
			err := c.Conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.WithError(err).WithField("client_id", c.ID).Error("WebSocket read error")
				}
				return
			}

			msg.Timestamp = time.Now()
			c.Hub.Broadcast <- BroadcastMessage{
				Message: msg,
				Client:  c,
			}
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				c.logger.WithError(err).WithField("client_id", c.ID).Error("WebSocket write error")
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msgType MessageType, data interface{}) {
	message := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case c.Send <- message:
	default:
		close(c.Send)
		delete(c.Hub.clients, c)
	}
}

// GetSessionID returns the client's session ID
func (c *Client) GetSessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SessionID
}

// SetSessionID sets the client's session ID
func (c *Client) SetSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SessionID = sessionID
}
