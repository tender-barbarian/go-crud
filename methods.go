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

// ErrorHandler is an interface for handling HTTP errors in a consistent way.
// Implement this interface to customize error responses across all CRUD endpoints.
type ErrorHandler interface {
	WriteError(w http.ResponseWriter, r *http.Request, err error, customMsg string, statusCode int)
}

// DefaultErrorHandler is the default implementation of ErrorHandler.
// It writes errors as plain text HTTP responses.
type DefaultErrorHandler struct{}

// WriteError writes an error response to the HTTP response writer.
// If customMsg is provided, it's used instead of the error message.
func (d DefaultErrorHandler) WriteError(w http.ResponseWriter, _ *http.Request, err error, customMsg string, statusCode int) {
	if customMsg != "" {
		http.Error(w, customMsg, statusCode)
		return
	}
	http.Error(w, err.Error(), statusCode)
}

// RegisterCreate registers an HTTP handler for creating resources.
//
// The handler:
//  1. Decodes JSON from the request body into the model type
//  2. Calls the provided function (typically Repository.Create)
//  3. Returns the created resource ID as JSON with 201 status
//
// Repository hooks are executed during the create operation:
//   - Validation (if configured with WithValidate)
//   - Transformation (if configured with WithTransform)
//   - OnMutate callback (if configured with WithOnMutate)
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithValidate().
//	    WithTransform().
//	    WithOnMutate(cacheInvalidator)
//	mux := http.NewServeMux()
//	RegisterCreate("POST /users", mux, repo.Create, DefaultErrorHandler{})
func RegisterCreate[In Model](pattern string, mux *http.ServeMux, f func(context.Context, In) (int, error), eh ErrorHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			eh.WriteError(w, r, err, "invalid json", http.StatusBadRequest)
			return
		}

		out, err := f(r.Context(), in)
		if err != nil {
			eh.WriteError(w, r, err, "", http.StatusBadRequest)
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

// RegisterGet registers an HTTP handler for retrieving a single resource by ID.
//
// The handler:
//  1. Extracts the ID from the URL path parameter
//  2. Calls the provided function (typically Repository.Get)
//  3. Returns the resource as JSON with 200 status
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} })
//	mux := http.NewServeMux()
//	RegisterGet("GET /users/{id}", mux, repo.Get, DefaultErrorHandler{})
func RegisterGet[Out Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, id int) (Out, error), eh ErrorHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			eh.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		out, err := f(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				eh.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			eh.WriteError(w, r, err, "", http.StatusBadRequest)
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

// RegisterGetAll registers an HTTP handler for retrieving all resources.
//
// The handler:
//  1. Calls the provided function (typically Repository.GetAll)
//  2. Returns the resources as a JSON array with 200 status
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} })
//	mux := http.NewServeMux()
//	RegisterGetAll("GET /users", mux, repo.GetAll, DefaultErrorHandler{})
func RegisterGetAll[Out any](pattern string, mux *http.ServeMux, f func(context.Context) ([]Out, error), eh ErrorHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		out, err := f(r.Context())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				eh.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			eh.WriteError(w, r, err, "", http.StatusBadRequest)
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

// RegisterDelete registers an HTTP handler for deleting a resource by ID.
//
// The handler:
//  1. Extracts the ID from the URL path parameter
//  2. Calls the provided function (typically Repository.Delete)
//  3. Returns 200 status on success
//
// Repository hooks are executed during the delete operation:
//   - OnMutate callback (if configured with WithOnMutate)
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithOnMutate(cacheInvalidator)
//	mux := http.NewServeMux()
//	RegisterDelete("DELETE /users/{id}", mux, repo.Delete, DefaultErrorHandler{})
func RegisterDelete(pattern string, mux *http.ServeMux, f func(context.Context, int) error, eh ErrorHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			eh.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				eh.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			eh.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}

// RegisterUpdate registers an HTTP handler for updating a resource.
//
// The handler:
//  1. Decodes JSON from the request body into the model type
//  2. Extracts the ID from the URL path parameter
//  3. Calls the provided function (typically Repository.Update)
//  4. Returns 200 status on success
//
// Repository hooks are executed during the update operation:
//   - Validation (if configured with WithValidate)
//   - Transformation (if configured with WithTransform)
//   - OnMutate callback (if configured with WithOnMutate)
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithValidate().
//	    WithTransform().
//	    WithOnMutate(cacheInvalidator)
//	mux := http.NewServeMux()
//	RegisterUpdate("POST /users/{id}", mux, repo.Update, DefaultErrorHandler{})
func RegisterUpdate[In Model](pattern string, mux *http.ServeMux, f func(ctx context.Context, in In, id int) error, eh ErrorHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In

		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			eh.WriteError(w, r, err, "invalid json", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			eh.WriteError(w, r, err, "invalid param", http.StatusBadRequest)
			return
		}

		err = f(r.Context(), in, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				eh.WriteError(w, r, err, "resource not found", http.StatusNotFound)
				return
			}
			eh.WriteError(w, r, err, "", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})
}
