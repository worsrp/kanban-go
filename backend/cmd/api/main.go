package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/worsrp/kanban-go/internal/db"
	"github.com/worsrp/kanban-go/internal/handler"
	"github.com/worsrp/kanban-go/internal/ws"
	"github.com/google/uuid"
	//	"github.com/joho/godotenv"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL_DEV")
	// if dbURL is null, set it to localhost URL
	if dbURL == "" {
		dbURL = "postgres://postgres@localhost:5432/kanban_db?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	queries := db.New(pool)
	boardHandler := handler.NewBoardHandler(queries)
	hubManager := ws.NewHubManager(queries)

	originUrl := os.Getenv("VUE_APP_URL_DEV")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{originUrl},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	// Check this route first
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Get("/ws/boards/{id}", func(w http.ResponseWriter, r *http.Request) {
		boardID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid board id", http.StatusBadRequest)
			return
		}
		userName := r.URL.Query().Get("name")
		ws.ServeWS(hubManager, boardID, userName, w, r)
	})

	r.Route("/api/boards", func(r chi.Router) {
		r.Post("/", boardHandler.CreateBoard)
		r.Get("/{id}", boardHandler.GetBoardState)
	})

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}