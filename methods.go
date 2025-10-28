package gocrud

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

func RegisterCreate[In Model](pattern string, mux *http.ServeMux, f func(context.Context, In) (int, error)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		out, err := f(r.Context(), in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(map[string]interface{}{"id": out})
		if err != nil {
			log.Printf("failed to encode created note: %v", err)
			return
		}
	})
}

func RegisterGet[Out Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, id int) (Out, error)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid param", http.StatusBadRequest)
			return
		}

		out, err := f(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "resource not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(out)
		if err != nil {
			log.Printf("failed to encode created note: %v", err)
			return
		}
	})
}

func RegisterGetAll[Out any](pattern string, mux *http.ServeMux, f func(context.Context) ([]Out, error)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		out, err := f(r.Context())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "resource not found", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(out)
		if err != nil {
			log.Printf("failed to encode created note: %v", err)
			return
		}
	})
}

func RegisterDelete(pattern string, mux *http.ServeMux, f func(context.Context, int) error) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "resource not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}

func RegisterUpdate[In Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, in In, id int) error) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), in, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "resource not found", http.StatusNotFound)
				return
			}
			http.Error(w, "resource not found", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}
