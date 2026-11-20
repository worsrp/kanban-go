export interface Board {
    id: string
    title: string
    created_at: string
}

export interface Column {
    id: string
    board_id: string
    title: string
    position: number
}

export interface Card {
    id: string
    column_id: string
    title: string
    description: string
    position: number
    created_at: string
    updated_at: string
}

export interface BoardStateResponse {
    board: Board
    columns: Column[]
    cards: Card[]
}

// Inbound Server Events (matches internal/ws/protocol.go)
export type ServerEvent =
    | { type: 'CARD_CREATED'; board_id: string; sender_id: string; payload: Card }
    | { type: 'CARD_MOVED'; board_id: string; sender_id: string; payload: Card }
    | { type: 'CARD_UPDATED'; board_id: string; sender_id: string; payload: Card }
    | { type: 'CARD_DELETED'; board_id: string; sender_id: string; payload: { card_id: string } }
    | { type: 'PRESENCE_UPDATE'; board_id: string; sender_id: string; payload: { active_users: string[]; count: number } }
    | { type: 'ERROR'; board_id: string; payload: { error: string } }

// Outbound Client Commands
export type ClientCommand =
    | { type: 'CARD_CREATE'; payload: { column_id: string; title: string; description: string; position: number } }
    | { type: 'CARD_MOVE'; payload: { card_id: string; to_column_id: string; new_position: number } }
    | { type: 'CARD_UPDATE'; payload: { card_id: string; title: string; description: string } }
    | { type: 'CARD_DELETE'; payload: { card_id: string } }