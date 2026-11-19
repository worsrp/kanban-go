package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/worsrp/kanban-go/internal/db"
)

// InboundCommand pairs a parsed message with the client that sent it.
type InboundCommand struct {
	Client  *Client
	Message Message
}

// HubManager manages active BoardHubs safely across concurrent HTTP upgrades.
type HubManager struct {
	mu      sync.RWMutex
	hubs    map[uuid.UUID]*BoardHub
	queries *db.Queries
}

func NewHubManager(q *db.Queries) *HubManager {
	return &HubManager{
		hubs:    make(map[uuid.UUID]*BoardHub),
		queries: q,
	}
}

// GetOrCreateHub returns the active hub for a board, spinning one up if needed.
func (m *HubManager) GetOrCreateHub(boardID uuid.UUID) *BoardHub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, exists := m.hubs[boardID]; exists {
		return hub
	}

	hub := newBoardHub(boardID, m.queries, func() {
		m.removeHub(boardID)
	})
	m.hubs[boardID] = hub
	go hub.run() // Start the hub's event loop in its own goroutine
	return hub
}

func (m *HubManager) removeHub(boardID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hubs, boardID)
	log.Printf("Hub closed for empty board %s", boardID)
}

// BoardHub coordinates all connected clients and mutations for a single board.
type BoardHub struct {
	boardID    uuid.UUID
	queries    *db.Queries
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	commands   chan InboundCommand
	onEmpty    func()
}

func newBoardHub(boardID uuid.UUID, q *db.Queries, onEmpty func()) *BoardHub {
	return &BoardHub{
		boardID:    boardID,
		queries:    q,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		commands:   make(chan InboundCommand, 64),
		onEmpty:    onEmpty,
	}
}

func (h *BoardHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client %s joined board %s (total: %d)", client.id, h.boardID, len(h.clients))
			h.broadcastPresence()

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client %s left board %s (total: %d)", client.id, h.boardID, len(h.clients))
				if len(h.clients) == 0 {
					h.onEmpty()
					return // Exit goroutine cleanly when no clients remain
				}
				h.broadcastPresence()
			}

		case cmd := <-h.commands:
			h.handleCommand(cmd)
		}
	}
}

func (h *BoardHub) handleCommand(cmd InboundCommand) {
	ctx := context.Background()

	switch cmd.Message.Type {
	case CmdCardCreate:
		var p CreateCardPayload
		if err := json.Unmarshal(cmd.Message.Payload, &p); err != nil {
			cmd.Client.sendError("invalid CARD_CREATE payload")
			return
		}
		card, err := h.queries.CreateCard(ctx, db.CreateCardParams{
			ColumnID:    pgtype.UUID{Bytes: p.ColumnID, Valid: true},
			Title:       p.Title,
			Description: p.Description,
			Position:    p.Position,
		})
		if err != nil {
			cmd.Client.sendError("failed to create card")
			return
		}
		h.broadcastEvent(EvtCardCreated, cmd.Client.id, card)

	case CmdCardMove:
		var p MoveCardPayload
		if err := json.Unmarshal(cmd.Message.Payload, &p); err != nil {
			cmd.Client.sendError("invalid CARD_MOVE payload")
			return
		}
		card, err := h.queries.MoveCard(ctx, db.MoveCardParams{
			ID:       pgtype.UUID{Bytes: p.CardID, Valid: true},
			ColumnID: pgtype.UUID{Bytes: p.ToColumnID, Valid: true},
			Position: p.NewPosition,
		})
		if err != nil {
			cmd.Client.sendError("failed to move card")
			return
		}
		h.broadcastEvent(EvtCardMoved, cmd.Client.id, card)

	case CmdCardUpdate:
		var p UpdateCardPayload
		if err := json.Unmarshal(cmd.Message.Payload, &p); err != nil {
			cmd.Client.sendError("invalid CARD_UPDATE payload")
			return
		}
		card, err := h.queries.UpdateCard(ctx, db.UpdateCardParams{
			ID:          pgtype.UUID{Bytes: p.CardID, Valid: true},
			Title:       p.Title,
			Description: p.Description,
		})
		if err != nil {
			cmd.Client.sendError("failed to update card")
			return
		}
		h.broadcastEvent(EvtCardUpdated, cmd.Client.id, card)

	case CmdCardDelete:
		var p DeleteCardPayload
		if err := json.Unmarshal(cmd.Message.Payload, &p); err != nil {
			cmd.Client.sendError("invalid CARD_DELETE payload")
			return
		}
		if err := h.queries.DeleteCard(ctx, pgtype.UUID{Bytes: p.CardID, Valid: true}); err != nil {
			cmd.Client.sendError("failed to delete card")
			return
		}
		h.broadcastEvent(EvtCardDeleted, cmd.Client.id, map[string]any{"card_id": p.CardID})

	default:
		cmd.Client.sendError("unknown command type: " + cmd.Message.Type)
	}
}

func (h *BoardHub) broadcastEvent(evtType string, senderID string, payload any) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msgBytes, err := json.Marshal(Message{
		Type:     evtType,
		BoardID:  h.boardID,
		SenderID: senderID,
		Payload:  rawPayload,
	})
	if err != nil {
		return
	}

	for client := range h.clients {
		select {
		case client.send <- msgBytes:
		default:
			// Drop slow/stuck client rather than blocking the entire board hub
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *BoardHub) broadcastPresence() {
	users := make([]string, 0, len(h.clients))
	for c := range h.clients {
		users = append(users, c.name)
	}
	h.broadcastEvent(EvtPresenceUpdate, "server", map[string]any{
		"active_users": users,
		"count":        len(users),
	})
}
