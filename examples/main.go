package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	gocrud "github.com/tender-barbarian/go-crud"

	_ "modernc.org/sqlite"
)

type genericRepo[M gocrud.Model] interface {
	Create(ctx context.Context, model M) (int, error)
	Get(ctx context.Context, id int) (M, error)
	GetAll(ctx context.Context) ([]M, error)
	Delete(ctx context.Context, id int) error
	Update(ctx context.Context, model M, id int) error
	GetTable() string
}

type Item struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	gocrud.Reflection
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT
		)
	`)
	if err != nil {
		return err
	}

	itemsRepo := gocrud.NewGenericRepository(db, "items", func() *Item { return &Item{} })
	mux := registerGenericRoutes(itemsRepo, http.NewServeMux())

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

func registerGenericRoutes[M gocrud.Model](repo genericRepo[M], mux *http.ServeMux) *http.ServeMux {
	errw := gocrud.DefaultErrorWriter{}
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create, errw)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get, errw)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll, errw)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete, errw)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update, errw)

	mux.Handle("/", http.NotFoundHandler())

	return mux
}
