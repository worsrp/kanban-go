package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/worsrp/kanban-go/internal/db"
)

type BoardHandler struct {
	queries *db.Queries
}

func NewBoardHandler(q *db.Queries) *BoardHandler {
	return &BoardHandler{queries: q}
}

func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	// Require Board Title
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		http.Error(w, "invalid title", http.StatusBadRequest)
		return
	}

	// Create Board	
	board, err := h.queries.CreateBoard(r.Context(), req.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create Default Columns	
	defaultCols := []string{"To Do", "In Progress", "Done"}
	for i, title := range defaultCols {
		_, err := h.queries.CreateColumn(r.Context(), db.CreateColumnParams{
			BoardID: board.ID,
			Title: title,
			Position: float64(i + 1),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusCreated, board)
}

func (h *BoardHandler) GetBoardState(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid board id", http.StatusBadRequest)
		return
	}
	pgID := pgtype.UUID{Bytes: parsedID, Valid: true}
	
	board, err := h.queries.GetBoard(r.Context(), pgID)
	if err != nil {
		http.Error(w, "board not found", http.StatusNotFound)
		return
	}

	columns, err := h.queries.ListColumnsByBoard(r.Context(), pgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cards, err := h.queries.ListCardsByBoard(r.Context(), pgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"board": board,
		"columns": columns,
		"cards": cards,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}