package gocrud

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	sq "github.com/Masterminds/squirrel"
)

type Model interface {
	StructToMap(d interface{}) map[string]any
}

type Repository[M Model] struct {
	mutex sync.Mutex
	db    *sql.DB
	new   func() M
	table string
}

func NewGenericRepository[M Model](db *sql.DB, table string, new func() M) *Repository[M] {
	return &Repository[M]{
		db:    db,
		new:   new,
		table: table,
	}
}

func (r *Repository[M]) GetTable() string {
	return r.table
}

func (r *Repository[M]) set(fields []string, scan func(dest ...any) error, model M) error {
	validate := model.StructToMap(model)

	dest := make([]any, 0, len(fields))

	for _, field := range fields {
		if p, ok := validate[field]; ok {
			dest = append(dest, p)
		}
	}

	return scan(dest...)
}

func (r *Repository[M]) Create(ctx context.Context, model M) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := model.StructToMap(model)

	columns := make([]string, 0, len(m))
	values := make([]any, 0, len(m))

	for key, value := range m {
		if key == "id" {
			continue
		}
		columns = append(columns, key)
		values = append(values, value)
	}

	query, args, err := sq.
		Insert(r.table).
		Columns(columns...).
		Values(values...).
		ToSql()
	if err != nil {
		return 0, err
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (r *Repository[M]) Get(ctx context.Context, id int) (M, error) {
	var zero M

	r.mutex.Lock()
	defer r.mutex.Unlock()

	query, args, err := sq.
		Select("*").
		From(r.table).
		Where(sq.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return zero, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return zero, err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zero, err
		}
		return zero, err
	}

	fields, err := rows.Columns()
	if err != nil {
		return zero, err
	}

	model := r.new()
	if err := r.set(fields, rows.Scan, model); err != nil {
		return zero, err
	}

	if err = rows.Close(); err != nil {
		return zero, err
	}

	if err = rows.Err(); err != nil {
		return zero, err
	}

	return model, nil
}

func (r *Repository[M]) GetAll(ctx context.Context) ([]M, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	query, args, err := sq.
		Select("*").
		From(r.table).
		OrderBy("id").ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	fields, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var models []M
	for rows.Next() {
		model := r.new()

		if err := r.set(fields, rows.Scan, model); err != nil {
			return nil, err
		}

		models = append(models, model)
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

func (r *Repository[M]) Delete(ctx context.Context, id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	query, args, err := sq.Delete(r.table).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)

	return err
}

func (r *Repository[M]) Update(ctx context.Context, model M, id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := model.StructToMap(model)
	delete(m, "id")

	query, args, err := sq.Update(r.table).
		SetMap(m).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}

	what, _ := r.db.ExecContext(ctx, query, args...)
	ids, _ := what.LastInsertId()
	rows, _ := what.RowsAffected()
	fmt.Printf("%d %d", ids, rows)

	return err
}
