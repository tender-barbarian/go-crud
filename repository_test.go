package gocrud

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

type ModelWithReflection struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Chip  string `json:"chip"`
	Board string `json:"board"`
	IP    string `json:"ip"`
	Reflection
}

type ModelWithoutReflection struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Chip    string   `json:"chip"`
	Board   string   `json:"board"`
	IP      string   `json:"ip"`
	Actions []string `json:"actions"`
}

func (o *ModelWithoutReflection) StructToMap(interface{}) map[string]any {
	return map[string]any{
		"id":      &o.ID,
		"name":    &o.Name,
		"type":    &o.Type,
		"chip":    &o.Chip,
		"board":   &o.Board,
		"ip":      &o.IP,
		"actions": (*pq.StringArray)(&o.Actions),
	}
}

func TestGenericRepository_Find(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("Test Generic Repository: Get()", func(t *testing.T) {
		want := &ModelWithReflection{
			ID:    1,
			Name:  "test 1",
			Type:  "test",
			Chip:  "test",
			Board: "test",
			IP:    "test",
		}

		repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })

		rows := sqlmock.NewRows([]string{"id", "name", "type", "chip", "board", "ip"}).
			AddRow(want.ID, want.Name, want.Type, want.Chip, want.Board, want.IP)
		query := regexp.QuoteMeta("SELECT * FROM table_name WHERE id = ? LIMIT 1")
		mock.ExpectQuery(query).WithArgs(want.ID).WillReturnRows(rows)

		got, err := repo.Get(context.Background(), want.ID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, want, got)
	})

	t.Run("Test Generic Repository: Get() non-existent id", func(t *testing.T) {
		query := regexp.QuoteMeta("SELECT * FROM table_name WHERE id = ? LIMIT 1")
		mock.ExpectQuery(query).WithArgs(2).WillReturnError(sql.ErrNoRows)

		repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })
		got, err := repo.Get(context.Background(), 2)

		var want *ModelWithReflection
		assert.Equal(t, want, got)
		assert.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("Test Generic Repository: Get() - Without Reflection", func(t *testing.T) {
		want := &ModelWithoutReflection{
			ID:      1,
			Name:    "test 1",
			Type:    "test",
			Chip:    "test",
			Board:   "test",
			IP:      "test",
			Actions: []string{"d7e949b8-5c41-4972-b484-9c33b89af32c", "d7e949b8-5c41-4972-b484-9c33b89af123"},
		}

		repo := NewGenericRepository(db, "table_name", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })

		rows := sqlmock.NewRows([]string{"id", "name", "type", "chip", "board", "ip", "actions"}).
			AddRow(want.ID, want.Name, want.Type, want.Chip, want.Board, want.IP, (*pq.StringArray)(&want.Actions))
		query := regexp.QuoteMeta("SELECT * FROM table_name WHERE id = ? LIMIT 1")
		mock.ExpectQuery(query).WithArgs(want.ID).WillReturnRows(rows)

		got, err := repo.Get(context.Background(), want.ID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, want, got)
	})

	t.Run("Test Generic Repository: Get() non-existent id - Without Reflection", func(t *testing.T) {
		query := regexp.QuoteMeta("SELECT * FROM table_name WHERE id = ? LIMIT 1")
		mock.ExpectQuery(query).WithArgs(2).WillReturnError(sql.ErrNoRows)

		repo := NewGenericRepository(db, "table_name", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })
		got, err := repo.Get(context.Background(), 2)

		var want *ModelWithoutReflection
		assert.Equal(t, want, got)
		assert.Equal(t, sql.ErrNoRows, err)
	})
}

func TestGenericRepository_GetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("Test Generic Repository: GetAll()", func(t *testing.T) {
		want := []*ModelWithReflection{
			{
				ID:    1,
				Name:  "test 1",
				Type:  "test",
				Chip:  "test",
				Board: "test",
				IP:    "test",
			},
			{
				ID:    2,
				Name:  "test 2",
				Type:  "test",
				Chip:  "test",
				Board: "test",
				IP:    "test",
			},
		}

		values := [][]driver.Value{
			{
				want[0].ID, want[0].Name, want[0].Type, want[0].Chip, want[0].Board, want[0].IP,
			},
			{
				want[1].ID, want[1].Name, want[1].Type, want[1].Chip, want[1].Board, want[1].IP,
			},
		}

		rows := sqlmock.NewRows([]string{"id", "name", "type", "chip", "board", "ip"}).
			AddRows(values...)
		query := regexp.QuoteMeta("SELECT * FROM table_name ORDER BY id")
		mock.ExpectQuery(query).WillReturnRows(rows)

		repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })
		got, err := repo.GetAll(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, want, got)
	})

	t.Run("Test Generic Repository: GetAll() no data", func(t *testing.T) {
		query := regexp.QuoteMeta("SELECT * FROM table_name ORDER BY id")
		mock.ExpectQuery(query).WillReturnError(sql.ErrNoRows)

		repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })
		got, err := repo.GetAll(context.Background())

		var want []*ModelWithReflection
		assert.Equal(t, want, got)
		assert.Equal(t, sql.ErrNoRows, err)
	})

	t.Run("Test Generic Repository: GetAll() - Without Reflection", func(t *testing.T) {
		want := []*ModelWithoutReflection{
			{
				ID:      1,
				Name:    "test 1",
				Type:    "test",
				Chip:    "test",
				Board:   "test",
				IP:      "test",
				Actions: []string{"d7e949b8-5c41-4972-b484-9c33b89af32c", "d7e949b8-5c41-4972-b484-9c33b89af123"},
			},
			{
				ID:      2,
				Name:    "test 2",
				Type:    "test",
				Chip:    "test",
				Board:   "test",
				IP:      "test",
				Actions: []string{"d7e949b8-5c41-4972-b484-9c33b89af456", "d7e949b8-5c41-4972-b484-9c33b89af789"},
			},
		}

		values := [][]driver.Value{
			{
				want[0].ID, want[0].Name, want[0].Type, want[0].Chip, want[0].Board, want[0].IP, (*pq.StringArray)(&want[0].Actions),
			},
			{
				want[1].ID, want[1].Name, want[1].Type, want[1].Chip, want[1].Board, want[1].IP, (*pq.StringArray)(&want[1].Actions),
			},
		}

		rows := sqlmock.NewRows([]string{"id", "name", "type", "chip", "board", "ip", "actions"}).
			AddRows(values...)
		query := regexp.QuoteMeta("SELECT * FROM table_name ORDER BY id")
		mock.ExpectQuery(query).WillReturnRows(rows)

		repo := NewGenericRepository(db, "table_name", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })
		got, err := repo.GetAll(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, want, got)
	})

	t.Run("Test Generic Repository: GetAll() no data - Without Reflection", func(t *testing.T) {
		query := regexp.QuoteMeta("SELECT * FROM table_name ORDER BY id")
		mock.ExpectQuery(query).WillReturnError(sql.ErrNoRows)

		repo := NewGenericRepository(db, "table_name", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })
		got, err := repo.GetAll(context.Background())

		var want []*ModelWithoutReflection
		assert.Equal(t, want, got)
		assert.Equal(t, sql.ErrNoRows, err)
	})
}

func TestGenericRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("Test Generic Repository: Create()", func(t *testing.T) {
		want := ModelWithReflection{
			Name:  "test 1",
			Type:  "test 2",
			Chip:  "test 3",
			Board: "test 4",
			IP:    "test 5",
		}

		// Can to `.WithArgs() becasue the columns passed to query builder in Create() are taken from a map and as such their order cannot be guaranteed.
		// Below regexp allows columns to be out of order.
		mock.ExpectExec(`INSERT INTO table_name \(((ip)(,)?|(name)(,)?|(type)(,)?|(chip)(,)?|(board)(,)?){5}\) VALUES \(([?,]+)\)`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })
		_, err := repo.Create(context.Background(), &want)
		if err != nil {
			t.Fatal(err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Test Generic Repository: Create() - Without Reflection", func(t *testing.T) {
		want := ModelWithoutReflection{
			Name:    "test 1",
			Type:    "test 2",
			Chip:    "test 3",
			Board:   "test 4",
			IP:      "test 5",
			Actions: []string{"d7e949b8-5c41-4972-b484-9c33b89af456", "d7e949b8-5c41-4972-b484-9c33b89af789"},
		}

		// Can to `.WithArgs() becasue the columns passed to query builder in Create() are taken from a map and as such their order cannot be guaranteed.
		// Below regexp allows columns to be out of order.
		mock.ExpectExec(`INSERT INTO table_name \(((ip)(,)?|(actions)(,)?|(name)(,)?|(type)(,)?|(chip)(,)?|(board)(,)?){6}\) VALUES \(([?,]+)\)`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewGenericRepository(db, "table_name", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })
		_, err := repo.Create(context.Background(), &want)
		if err != nil {
			t.Fatal(err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestGenericRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("Test Generic Repository: Update()", func(t *testing.T) {
		want := ModelWithReflection{
			Name:  "test 1",
			Type:  "test 2",
			Chip:  "test 3",
			Board: "test 4",
			IP:    "test 5",
		}

		// Can to `.WithArgs() becasue the columns passed to query builder in Create() are taken from a map and as such their order cannot be guaranteed.
		// Below regexp allows columns to be out of order.
		mock.ExpectExec(`UPDATE table_name SET (\s?(board = \?)(,)?|\s?(chip = \?)(,)?|\s?(ip = \?)(,)?|\s?(name = \?)(,)?|\s?(type = \?)(,)?){5} WHERE id = \?`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })
		err := repo.Update(context.Background(), &want, 1)
		if err != nil {
			t.Fatal(err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Test Generic Repository: Update() - Without Reflection", func(t *testing.T) {
		want := ModelWithoutReflection{
			Name:    "test 1",
			Type:    "test 2",
			Chip:    "test 3",
			Board:   "test 4",
			IP:      "test 5",
			Actions: []string{"d7e949b8-5c41-4972-b484-9c33b89af456", "d7e949b8-5c41-4972-b484-9c33b89af789"},
		}

		// Can to `.WithArgs() becasue the columns passed to query builder in Create() are taken from a map and as such their order cannot be guaranteed.
		// Below regexp allows columns to be out of order.
		mock.ExpectExec(`UPDATE table_name SET (\s?(actions = \?)(,)?|\s?(board = \?)(,)?|\s?(chip = \?)(,)?|\s?(ip = \?)(,)?|\s?(name = \?)(,)?|\s?(type = \?)(,)?){6} WHERE id = \?`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewGenericRepository(db, "table_name", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })
		err := repo.Update(context.Background(), &want, 1)
		if err != nil {
			t.Fatal(err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
