package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"os"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	// "github.com/joho/godotenv"
)

type Client struct {
	id string
	name string
	hub *BoardHub
	conn *websocket.Conn
	send chan []byte
}

// ServeWS upgrades an HTTP connection to a WebSocket and attaches it to the BoardHub.
func ServeWS(manager *HubManager, boardID uuid.UUID, userName string, w http.ResponseWriter, r *http.Request) {
	originUrl := os.Getenv("VUE_APP_URL_DEV")
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{originUrl},
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	if userName == "" {
		userName = "Guest-" + uuid.NewString()[:4]
	}

	hub := manager.GetOrCreateHub(boardID)
	client := &Client{
		id: uuid.NewString(),
		name: userName,
		hub: hub,
		conn: conn,
		send: make(chan []byte, 64),
	}

	hub.register <- client

	go client.writePump()
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, data, err := c.conn.Read(context.Background())
		if err != nil {
			break // Client disconnected
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid JSON message")
			continue
		}
		msg.BoardID = c.hub.boardID
		msg.SenderID = c.id

		c.hub.commands <- InboundCommand{
			Client: c,
			Message: msg,
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return // Channel closed by hub
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}

		case <- ticker.C:
			// Keep connection alive through proxies/load balancers
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) sendError(errMsg string) {
	raw, _ := json.Marshal(map[string]string{"error": errMsg})
	out, _ := json.Marshal(Message{
		Type: EvtError,
		BoardID: c.hub.boardID,
		Payload: raw,
	})
	select {
	case c.send <- out:
	default:
	}
}