package gocrud

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestIntegration_HTTP_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterCreate("POST /items", mux, repo.Create)

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
	RegisterGet("GET /items/{id}", mux, repo.Get)

	t.Run("existing item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/items/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var got TestItem
		err := json.NewDecoder(rec.Body).Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, seeded[0].Name, got.Name)
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
	RegisterGetAll("GET /items", mux, repo.GetAll)

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

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
}

func TestIntegration_HTTP_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedTestData(t, db)
	repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
	mux := http.NewServeMux()
	RegisterUpdate("POST /items/{id}", mux, repo.Update)
	RegisterGet("GET /items/{id}", mux, repo.Get)

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
	RegisterDelete("DELETE /items/{id}", mux, repo.Delete)
	RegisterGetAll("GET /items", mux, repo.GetAll)

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
	RegisterCreate("POST /items", mux, repo.Create)

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
	RegisterUpdate("POST /items/{id}", mux, repo.Update)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/items/1", strings.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
