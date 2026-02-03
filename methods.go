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

// Validatable is an optional interface that models can implement
// to enable automatic validation after JSON decoding.
type Validatable interface {
	Validate() error
}

// ValidationError is an optional interface that validation errors can implement
// to provide custom HTTP status codes and messages.
type ValidationError interface {
	error
	Message() string
	StatusCode() int
}

// Row represents a single database row that can be scanned.
type Row interface {
	Scan(dest ...any) error
}

// DBQuerier provides database query capabilities for validation.
type DBQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
}

// ValidatableWithDB is an optional interface that models can implement
// to enable database-dependent validation after JSON decoding.
type ValidatableWithDB interface {
	ValidateWithDB(ctx context.Context, db DBQuerier) error
}

func handleValidationError(err error, w http.ResponseWriter, r *http.Request, errw ErrorWriter) {
	msg := "validation error"
	code := http.StatusBadRequest
	if ve, ok := err.(ValidationError); ok {
		msg = ve.Message()
		code = ve.StatusCode()
	}
	errw.WriteError(w, r, err, msg, code)
}

type ErrorWriter interface {
	WriteError(w http.ResponseWriter, r *http.Request, err error, customMsg string, statusCode int)
}

type DefaultErrorWriter struct{}

func (d DefaultErrorWriter) WriteError(w http.ResponseWriter, _ *http.Request, err error, customMsg string, statusCode int) {
	if customMsg != "" {
		http.Error(w, customMsg, statusCode)
		return
	}
	http.Error(w, err.Error(), statusCode)
}

func RegisterCreate[In Model](pattern string, mux *http.ServeMux, f func(context.Context, In) (int, error), errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			errw.WriteError(w, r, err, "invalid json", http.StatusBadRequest)
			return
		}

		if v, ok := any(in).(Validatable); ok {
			if err := v.Validate(); err != nil {
				handleValidationError(err, w, r, errw)
				return
			}
		}

		out, err := f(r.Context(), in)
		if err != nil {
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(map[string]interface{}{"id": out})
		if err != nil {
			log.Printf("failed to encode output: %v", err)
			return
		}
	})
}

func RegisterCreateWithDB[In Model](pattern string, mux *http.ServeMux, f func(context.Context, In) (int, error), db DBQuerier, errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			errw.WriteError(w, r, err, "invalid json", http.StatusBadRequest)
			return
		}

		if v, ok := any(in).(Validatable); ok {
			if err := v.Validate(); err != nil {
				handleValidationError(err, w, r, errw)
				return
			}
		}

		if v, ok := any(in).(ValidatableWithDB); ok {
			if err := v.ValidateWithDB(r.Context(), db); err != nil {
				handleValidationError(err, w, r, errw)
				return
			}
		}

		out, err := f(r.Context(), in)
		if err != nil {
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(map[string]interface{}{"id": out})
		if err != nil {
			log.Printf("failed to encode output: %v", err)
			return
		}
	})
}

func RegisterGet[Out Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, id int) (Out, error), errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			errw.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		out, err := f(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errw.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(out)
		if err != nil {
			log.Printf("failed to encode output: %v", err)
			return
		}
	})
}

func RegisterGetAll[Out any](pattern string, mux *http.ServeMux, f func(context.Context) ([]Out, error), errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		out, err := f(r.Context())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errw.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(out)
		if err != nil {
			log.Printf("failed to encode output: %v", err)
			return
		}
	})
}

func RegisterDelete(pattern string, mux *http.ServeMux, f func(context.Context, int) error, errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			errw.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errw.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}

func RegisterUpdate[In Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, in In, id int) error, errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			errw.WriteError(w, r, err, "invalid json", http.StatusBadRequest)
			return
		}

		if v, ok := any(in).(Validatable); ok {
			if err := v.Validate(); err != nil {
				handleValidationError(err, w, r, errw)
				return
			}
		}

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			errw.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), in, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errw.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}

func RegisterUpdateWithDB[In Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, in In, id int) error, db DBQuerier, errw ErrorWriter) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			errw.WriteError(w, r, err, "invalid json", http.StatusBadRequest)
			return
		}

		if v, ok := any(in).(Validatable); ok {
			if err := v.Validate(); err != nil {
				handleValidationError(err, w, r, errw)
				return
			}
		}

		if v, ok := any(in).(ValidatableWithDB); ok {
			if err := v.ValidateWithDB(r.Context(), db); err != nil {
				handleValidationError(err, w, r, errw)
				return
			}
		}

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			errw.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), in, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errw.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			errw.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}
