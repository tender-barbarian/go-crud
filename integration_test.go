package gocrud

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

type TestItem struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Params      string   `json:"params"`
	CreatedAt   NullTime `json:"created_at" db:"created_at"`
	UpdatedAt   NullTime `json:"updated_at" db:"updated_at"`
	Reflection
}

func (ti *TestItem) Validate(_ context.Context, _ DBQuerier) error {
	if ti.Params != "" {
		if !json.Valid([]byte(ti.Params)) {
			return fmt.Errorf("params must be valid JSON")
		}
	}

	return nil
}

func (ti *TestItem) Transform(_ context.Context) (Model, error) {
	if ti.Params != "" {
		ti.Params = "transformed!"
	}

	return ti, nil
}

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
			category TEXT,
			params TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func seedTestData(t *testing.T, db *sql.DB) []TestItem {
	items := []TestItem{
		{Name: "Item 1", Description: "First item", Category: "A", Params: "{\"fist\":\"item\"}"},
		{Name: "Item 2", Description: "Second item", Category: "B", Params: "{\"second\":\"item\"}"},
		{Name: "Item 3", Description: "Third item", Category: "A", Params: "{\"third\":\"item\"}"},
	}

	for i := range items {
		result, err := db.Exec(
			"INSERT INTO items (name, description, category, params) VALUES (?, ?, ?, ?)",
			items[i].Name, items[i].Description, items[i].Category, items[i].Params,
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

func removeTestData(t *testing.T, db *sql.DB) {
	_, err := db.Exec(
		"DELETE FROM items",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_HTTP_Create(t *testing.T) { // nolint:cyclop
	t.Run("new item", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
		mux := http.NewServeMux()
		RegisterCreate("POST /items", mux, repo.Create, DefaultErrorHandler{})

		body := `{"name": "HTTP Item", "description": "Created via HTTP", "category": "HTTP"}`
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

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
	})

	t.Run("new item with validation", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		testItem := &TestItem{}
		repo := NewGenericRepository(db, "items", func() *TestItem { return testItem }).WithValidate()
		mux := http.NewServeMux()
		RegisterCreate("POST /items", mux, repo.Create, DefaultErrorHandler{})

		body := `{"name": "HTTP Item", "description": "Created via HTTP", "category": "HTTP", "params": "{\"test1\":\"test\"}"}`
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

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
	})

	t.Run("new item with failed validation", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		testItem := &TestItem{}
		repo := NewGenericRepository(db, "items", func() *TestItem { return testItem }).WithValidate()
		mux := http.NewServeMux()
		RegisterCreate("POST /items", mux, repo.Create, DefaultErrorHandler{})

		body := `{"name": "HTTP Item", "description": "Created via HTTP", "category": "HTTP", "params": "{\"test1\"\"test\"}"}`
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)

		assert.Equal(t, "params must be valid JSON\n", rec.Body.String())
	})

	t.Run("new item with failed invalid JSON", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
		mux := http.NewServeMux()
		RegisterCreate("POST /items", mux, repo.Create, DefaultErrorHandler{})

		body := `{invalid json}`
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("new item with timestamps", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })

		item := &TestItem{Name: "Test Item"}

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
	})

	t.Run("new item with transformation", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		testItem := &TestItem{}
		repo := NewGenericRepository(db, "items", func() *TestItem { return testItem }).WithTransform()
		mux := http.NewServeMux()
		RegisterCreate("POST /items", mux, repo.Create, DefaultErrorHandler{})

		body := `{"name": "HTTP Item", "description": "Created via HTTP", "category": "HTTP", "params": "{\"test1\":\"test\"}"}`
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

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

		// Check that the transformation was applied
		item, err := repo.Get(context.Background(), int(id))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "transformed!", item.Params)
	})

	t.Run("onMutate for cache invalidation with Create", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		cache := make(map[int]*TestItem)
		cache[1] = &TestItem{ID: 1, Name: "Cached"}

		testItem := &TestItem{}
		repo := NewGenericRepository(db, "items", func() *TestItem { return testItem }).
			WithOnMutate(func(_ context.Context) {
				for k := range cache {
					delete(cache, k)
				}
			})

		item := &TestItem{Name: "New Item", Category: "test"}
		_, err := repo.Create(context.Background(), item)
		if err != nil {
			t.Fatal(err)
		}

		assert.Empty(t, cache, "cache should be cleared after mutation")
	})
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

func TestIntegration_HTTP_Update(t *testing.T) { // nolint:cyclop
	t.Run("update item", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		seedTestData(t, db)
		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
		mux := http.NewServeMux()
		RegisterUpdate("POST /items/{id}", mux, repo.Update, DefaultErrorHandler{})
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
	})

	t.Run("update item with validation", func(t *testing.T) { // nolint:cyclop,dupl
		db := setupTestDB(t)
		defer db.Close()

		seedTestData(t, db)
		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} }).WithValidate()
		mux := http.NewServeMux()
		RegisterUpdate("POST /items/{id}", mux, repo.Update, DefaultErrorHandler{})
		RegisterGet("GET /items/{id}", mux, repo.Get, DefaultErrorHandler{})

		body := `{"name": "Updated via HTTP", "description": "Updated", "category": "UPD", "params": "{\"test1\":\"test\"}"}`
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
		assert.Equal(t, "{\"test1\":\"test\"}", got.Params)
	})

	t.Run("update item with failed validation", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		seedTestData(t, db)
		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} }).WithValidate()
		mux := http.NewServeMux()
		RegisterUpdate("POST /items/{id}", mux, repo.Update, DefaultErrorHandler{})
		RegisterGet("GET /items/{id}", mux, repo.Get, DefaultErrorHandler{})

		body := `{"name": "Updated via HTTP", "description": "Updated", "category": "UPD", "params": "{\"test1\"\"test\"}"}`
		req := httptest.NewRequest(http.MethodPost, "/items/1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)

		assert.Equal(t, "params must be valid JSON\n", rec.Body.String())
	})

	t.Run("update item with invalid JSON", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		seedTestData(t, db)
		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} })
		mux := http.NewServeMux()
		RegisterUpdate("POST /items/{id}", mux, repo.Update, DefaultErrorHandler{})

		body := `{invalid json}`
		req := httptest.NewRequest(http.MethodPost, "/items/1", strings.NewReader(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("update item with timestamps", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewGenericRepository(db, "items", func() *TestItem {
			return &TestItem{}
		})

		// Create initial item
		item := &TestItem{Name: "Original"}
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
		updateItem := &TestItem{Name: "Updated"}
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
	})

	t.Run("update item with transformation", func(t *testing.T) { // nolint:cyclop,dupl
		db := setupTestDB(t)
		defer db.Close()

		seedTestData(t, db)
		repo := NewGenericRepository(db, "items", func() *TestItem { return &TestItem{} }).WithTransform()
		mux := http.NewServeMux()
		RegisterUpdate("POST /items/{id}", mux, repo.Update, DefaultErrorHandler{})
		RegisterGet("GET /items/{id}", mux, repo.Get, DefaultErrorHandler{})

		body := `{"name": "Updated via HTTP", "description": "Updated", "category": "UPD", "params": "{\"test1\":\"test\"}"}`
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
		assert.Equal(t, "transformed!", got.Params)
	})

	t.Run("onMutate for cache invalidation with Update", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		cache := make(map[int]*TestItem)
		cache[1] = &TestItem{ID: 1, Name: "Cached"}

		testItem := &TestItem{}
		repo := NewGenericRepository(db, "items", func() *TestItem { return testItem }).
			WithOnMutate(func(_ context.Context) {
				for k := range cache {
					delete(cache, k)
				}
			})

		item := &TestItem{Name: "Initial Item", Category: "test"}
		id, err := repo.Create(context.Background(), item)
		require.NoError(t, err)

		cache[1] = &TestItem{ID: 1, Name: "Cached"}

		item.Name = "Updated Item"
		err = repo.Update(context.Background(), item, id)
		require.NoError(t, err)

		assert.Empty(t, cache, "cache should be cleared after Update")
	})
}

func TestIntegration_HTTP_Delete(t *testing.T) {
	t.Run("delete successful", func(t *testing.T) {
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
	})

	t.Run("onMutate for cache invalidation with Delete", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		cache := make(map[int]*TestItem)
		cache[1] = &TestItem{ID: 1, Name: "Cached"}

		testItem := &TestItem{}
		repo := NewGenericRepository(db, "items", func() *TestItem { return testItem }).
			WithOnMutate(func(_ context.Context) {
				for k := range cache {
					delete(cache, k)
				}
			})

		item := &TestItem{Name: "Item to Delete", Category: "test"}
		id, err := repo.Create(context.Background(), item)
		require.NoError(t, err)

		cache[1] = &TestItem{ID: 1, Name: "Cached"}

		err = repo.Delete(context.Background(), id)
		require.NoError(t, err)

		assert.Empty(t, cache, "cache should be cleared after Delete")
	})

}
