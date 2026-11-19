package ws

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Client
const (
	CmdCardCreate = "CARD_CREATE"
	CmdCardMove   = "CARD_MOVE"
	CmdCardUpdate = "CARD_UPDATE"
	CmdCardDelete = "CARD_DELETE"
)

// Server
const (
	EvtCardCreated    = "CARD_CREATE"
	EvtCardMoved      = "CARD_MOVE"
	EvtCardUpdated    = "CARD_UPDATE"
	EvtCardDeleted    = "CARD_DELETE"
	EvtPresenceUpdate = "PRESENCE_UPDATE"
	EvtError          = "ERROR"
)

// Message
type Message struct {
	Type     string          `json:"type"`
	BoardID  uuid.UUID       `json:"board_id"`
	SenderID string          `json:"sender_id,omitempty"`
	Payload  json.RawMessage `json:"payload"`
}

type CreateCardPayload struct {
	ColumnID    uuid.UUID `json:"column_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Position    float64   `json:"position"`
}

type MoveCardPayload struct {
	CardID      uuid.UUID `json:"card_id"`
	ToColumnID  uuid.UUID `json:"to_column_id"`
	NewPosition float64   `json:"new_position"`
}

type UpdateCardPayload struct {
	CardID      uuid.UUID `json:"card_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

type DeleteCardPayload struct {
	CardID uuid.UUID `json:"card_id"`
}
