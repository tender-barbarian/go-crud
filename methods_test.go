package gocrud

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Tag  string `json:"tag"`
	Kind string `json:"kind"`
	IP   string `json:"ip"`
	Reflection
}

type genericRepoMock[M Model] struct {
	t      *testing.T
	model  M
	models []M
	table  string
	err    error
}

func (mock *genericRepoMock[M]) GetTable() string {
	return mock.table
}

func (mock *genericRepoMock[M]) Create(_ context.Context, in M) (int, error) {
	assert.Equal(mock.t, mock.model, in)
	return 1, nil
}

func TestMethod_Create(t *testing.T) {
	t.Run("Test generic method: Create()", func(t *testing.T) {
		want := &Item{
			ID:   1,
			Name: "test 1",
			Type: "test 2",
			Tag:  "test 3",
			Kind: "test 4",
			IP:   "test 5",
		}

		repo := &genericRepoMock[*Item]{t: t, model: want, table: "item"}
		mux := http.NewServeMux()
		RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create)

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(want)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/item", &buf)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		var got map[string]int
		err = json.NewDecoder(res.Body).Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want.ID, got["id"])
	})

	t.Run("Test generic method: Create() - empty body", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{table: "item"}
		mux := http.NewServeMux()
		RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create)

		req := httptest.NewRequest(http.MethodPost, "/item", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 400, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "invalid json\n", string(errMsg))
	})
}

func (mock *genericRepoMock[M]) Get(context.Context, int) (M, error) {
	var zero M

	if mock.err != nil {
		return zero, mock.err
	}

	return mock.model, nil
}

func TestMethod_Get(t *testing.T) {
	t.Run("Test generic method: Get()", func(t *testing.T) {
		want := &Item{
			ID:   1,
			Name: "test 1",
			Type: "test 2",
			Tag:  "test 3",
			Kind: "test 4",
			IP:   "test 5",
		}

		repo := &genericRepoMock[*Item]{t: t, model: want, table: "item"}
		mux := http.NewServeMux()
		RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get)

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(want)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/item/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		var got *Item
		err = json.NewDecoder(res.Body).Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, got)
	})

	t.Run("Test generic method: Get() - does not exist", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{t: t, table: "item", err: sql.ErrNoRows}
		mux := http.NewServeMux()
		RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get)

		req := httptest.NewRequest(http.MethodGet, "/item/10", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 404, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "resource not found\n", string(errMsg))
	})

	t.Run("Test generic method: Get() - invalid param", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{t: t, table: "item"}
		mux := http.NewServeMux()
		RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get)

		req := httptest.NewRequest(http.MethodGet, "/item/asd", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 400, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "invalid param\n", string(errMsg))
	})
}

func (mock *genericRepoMock[M]) GetAll(context.Context) ([]M, error) {
	var zero []M

	if mock.err != nil {
		return zero, mock.err
	}

	return mock.models, nil
}

func TestMethod_GetAll(t *testing.T) {
	t.Run("Test generic method: GetAll()", func(t *testing.T) {
		want := []*Item{
			{
				ID:   1,
				Name: "test 1",
				Type: "test 2",
				Tag:  "test 3",
				Kind: "test 4",
				IP:   "test 5",
			},
			{
				ID:   2,
				Name: "test 1",
				Type: "test 2",
				Tag:  "test 3",
				Kind: "test 4",
				IP:   "test 5",
			},
		}

		repo := &genericRepoMock[*Item]{t: t, models: want, table: "item"}
		mux := http.NewServeMux()
		RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll)

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(want)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/item", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		var got []*Item
		err = json.NewDecoder(res.Body).Decode(&got)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, got)
	})

	t.Run("Test generic method: GetAll() - does not exist", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{t: t, table: "item", err: sql.ErrNoRows}
		mux := http.NewServeMux()
		RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll)

		req := httptest.NewRequest(http.MethodGet, "/item", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 404, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "resource not found\n", string(errMsg))
	})
}

func (mock *genericRepoMock[M]) Delete(context.Context, int) error {
	if mock.err != nil {
		return mock.err
	}

	return nil
}

func TestMethod_Delete(t *testing.T) {
	t.Run("Test generic method: Delete()", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{t: t, table: "item"}
		mux := http.NewServeMux()
		RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/item/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()

		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("Test generic method: Delete() - does not exist", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{t: t, table: "item", err: sql.ErrNoRows}
		mux := http.NewServeMux()
		RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/item/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()

		assert.Equal(t, 404, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "resource not found\n", string(errMsg))
	})

	t.Run("Test generic method: Delete() - invalid param", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{t: t, table: "item"}
		mux := http.NewServeMux()
		RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/item/asd", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 400, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "invalid param\n", string(errMsg))
	})
}

func (mock *genericRepoMock[M]) Update(context.Context, M, int) error {
	if mock.err != nil {
		return mock.err
	}

	return nil
}

func TestMethod_Update(t *testing.T) {
	t.Run("Test generic method: Update()", func(t *testing.T) {
		want := &Item{
			ID:   1,
			Name: "test 1",
			Type: "test 2",
			Tag:  "test 3",
			Kind: "test 4",
			IP:   "test 5",
		}

		repo := &genericRepoMock[*Item]{t: t, model: want, table: "item"}
		mux := http.NewServeMux()
		RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(want)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/item/1", &buf)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()

		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("Test generic method: Update() - does not exist", func(t *testing.T) {
		want := &Item{
			ID:   1,
			Name: "test 1",
			Type: "test 2",
			Tag:  "test 3",
			Kind: "test 4",
			IP:   "test 5",
		}

		repo := &genericRepoMock[*Item]{table: "item", err: sql.ErrNoRows}
		mux := http.NewServeMux()
		RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(want)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/item/1", &buf)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 404, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "resource not found\n", string(errMsg))
	})

	t.Run("Test generic method: Update() - empty body", func(t *testing.T) {
		repo := &genericRepoMock[*Item]{table: "item"}
		mux := http.NewServeMux()
		RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)

		req := httptest.NewRequest(http.MethodPost, "/item/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 400, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "invalid json\n", string(errMsg))
	})

	t.Run("Test generic method: Update() - invalid param", func(t *testing.T) {
		want := &Item{
			ID:   1,
			Name: "test 1",
			Type: "test 2",
			Tag:  "test 3",
			Kind: "test 4",
			IP:   "test 5",
		}

		repo := &genericRepoMock[*Item]{t: t, table: "item"}
		mux := http.NewServeMux()
		RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)

		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(want)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/item/asd", &buf)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		res := rec.Result()
		assert.Equal(t, 400, res.StatusCode)

		errMsg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "invalid param\n", string(errMsg))
	})
}
