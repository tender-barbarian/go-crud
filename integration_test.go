package gocrud

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "modernc.org/sqlite"
)

// TestItem is the model used for integration tests
type TestItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Reflection
}

// setupTestDB creates an in-memory SQLite database with the test table
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

// seedTestData inserts test data into the database
func seedTestData(t *testing.T, db *sql.DB) []TestItem {
	items := []TestItem{
		{Name: "Item 1", Description: "First item", Category: "A"},
		{Name: "Item 2", Description: "Second item", Category: "B"},
		{Name: "Item 3", Description: "Third item", Category: "A"},
	}

	for i := range items {
		result, err := db.Exec(
			"INSERT INTO items (name, description, category) VALUES (?, ?, ?)",
			items[i].Name, items[i].Description, items[i].Category,
		)
		if err != nil {
			t.Fatal(err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}

		items[i].ID = int(id)
	}

	return items
}

// removeTestData deletes test data from the database
func removeTestData(t *testing.T, db *sql.DB) {
	_, err := db.Exec(
		"DELETE FROM items",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_HTTP_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterCreate("POST /items", mux, repo.Create, nil, DefaultErrorHandler{})

	body := `{"name": "HTTP Item", "description": "Created via HTTP", "category": "HTTP"}`
	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := response["id"]; !exists {
		t.Fatal("response should contain id")
	}

	id, ok := response["id"].(float64)
	if !ok {
		t.Fatal("id should be a number")
	}

	assert.Equal(t, float64(1), id)
}

func TestIntegration_HTTP_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seeded := seedTestData(t, db)
	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterGet("GET /items/{id}", mux, repo.Get, DefaultErrorHandler{})

	t.Run("existing item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/items/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var got TestItem
		err := json.NewDecoder(rec.Body).Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, seeded[0].Name, got.Name)
	})

	t.Run("non existing item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/items/10", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/items/invalid", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestIntegration_HTTP_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seeded := seedTestData(t, db)
	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterGetAll("GET /items", mux, repo.GetAll, DefaultErrorHandler{})

	t.Run("existing items", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/items", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var got []TestItem
		err := json.NewDecoder(rec.Body).Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		assert.Len(t, got, len(seeded))
		for i, item := range got {
			assert.Equal(t, seeded[i].ID, item.ID)
			assert.Equal(t, seeded[i].Name, item.Name)
			assert.Equal(t, seeded[i].Description, item.Description)
			assert.Equal(t, seeded[i].Category, item.Category)
		}
	})

	t.Run("non existing items", func(t *testing.T) {
		removeTestData(t, db)
		req := httptest.NewRequest(http.MethodGet, "/items", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestIntegration_HTTP_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedTestData(t, db)
	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterUpdate("POST /items/{id}", mux, repo.Update, nil, DefaultErrorHandler{})
	RegisterGet("GET /items/{id}", mux, repo.Get, DefaultErrorHandler{})

	body := `{"name": "Updated via HTTP", "description": "Updated", "category": "UPD"}`
	req := httptest.NewRequest(http.MethodPost, "/items/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/items/1", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var got TestItem
	err := json.NewDecoder(rec.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Updated via HTTP", got.Name)
	assert.Equal(t, "Updated", got.Description)
	assert.Equal(t, "UPD", got.Category)
}

func TestIntegration_HTTP_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedTestData(t, db)
	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterDelete("DELETE /items/{id}", mux, repo.Delete, DefaultErrorHandler{})
	RegisterGetAll("GET /items", mux, repo.GetAll, DefaultErrorHandler{})

	req := httptest.NewRequest(http.MethodDelete, "/items/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/items", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var got []TestItem
	err := json.NewDecoder(rec.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, got, 2)
}

func TestIntegration_HTTP_CreateInvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterCreate("POST /items", mux, repo.Create, nil, DefaultErrorHandler{})

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIntegration_HTTP_UpdateInvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedTestData(t, db)
	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterUpdate("POST /items/{id}", mux, repo.Update, nil, DefaultErrorHandler{})

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/items/1", strings.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestItemWithTimestamps is a model with timestamp fields for testing
type TestItemWithTimestamps struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	CreatedAt NullTime `json:"created_at" db:"created_at"`
	UpdatedAt NullTime `json:"updated_at" db:"updated_at"`
	Reflection
}

// setupTestDBWithTimestamps creates an in-memory SQLite database with timestamp columns
func setupTestDBWithTimestamps(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE items_ts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestIntegration_Create_SetsTimestamps(t *testing.T) {
	db := setupTestDBWithTimestamps(t)
	defer db.Close()

	repo := NewGenericRepository(db, "items_ts", func() *TestItemWithTimestamps {
		return &TestItemWithTimestamps{}
	})

	item := &TestItemWithTimestamps{Name: "Test Item"}

	before := time.Now().Add(-time.Second)
	id, err := repo.Create(context.Background(), item)
	after := time.Now().Add(time.Second)

	if err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}

	if got.CreatedAt.Time.Before(before) || got.CreatedAt.Time.After(after) {
		t.Errorf("created_at %v not in expected range [%v, %v]",
			got.CreatedAt.Time, before, after)
	}

	if got.UpdatedAt.Time.Before(before) || got.UpdatedAt.Time.After(after) {
		t.Errorf("updated_at %v not in expected range [%v, %v]",
			got.UpdatedAt.Time, before, after)
	}
}

func TestIntegration_Update_SetsUpdatedAt_PreservesCreatedAt(t *testing.T) {
	db := setupTestDBWithTimestamps(t)
	defer db.Close()

	repo := NewGenericRepository(db, "items_ts", func() *TestItemWithTimestamps {
		return &TestItemWithTimestamps{}
	})

	// Create initial item
	item := &TestItemWithTimestamps{Name: "Original"}
	id, err := repo.Create(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}

	// Get original timestamps
	original, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	originalCreatedAt := original.CreatedAt.Time

	// Wait to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update the item
	updateItem := &TestItemWithTimestamps{Name: "Updated"}
	beforeUpdate := time.Now().Add(-time.Second)
	err = repo.Update(context.Background(), updateItem, id)
	afterUpdate := time.Now().Add(time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Verify timestamps
	updated, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}

	// created_at should be preserved (unchanged from original)
	if !updated.CreatedAt.Time.Equal(originalCreatedAt) {
		t.Errorf("created_at changed: was %v, now %v",
			originalCreatedAt, updated.CreatedAt.Time)
	}

	// updated_at should be updated
	if updated.UpdatedAt.Time.Before(beforeUpdate) || updated.UpdatedAt.Time.After(afterUpdate) {
		t.Errorf("updated_at %v not in expected range [%v, %v]",
			updated.UpdatedAt.Time, beforeUpdate, afterUpdate)
	}
}
