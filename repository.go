package gocrud

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type DBQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
type Validatable interface {
	validate(ctx context.Context, db DBQuerier) error
}
type Transformatable interface {
	transform(ctx context.Context) (Model, error)
}

type Model interface {
	StructToMap(d interface{}) map[string]any
}

type Repository[M Model] struct {
	mutex           sync.Mutex
	db              *sql.DB
	getConcreteType func() M
	table           string
	validate        bool
	onMutate        func(context.Context)
	transform       bool
}

func (r *Repository[M]) WithValidate() *Repository[M] {
	r.validate = true
	return r
}

func (r *Repository[M]) WithOnMutate(fn func(context.Context)) *Repository[M] {
	r.onMutate = fn
	return r
}

func (r *Repository[M]) WithTransform() *Repository[M] {
	r.transform = true
	return r
}

func NewGenericRepository[M Model](db *sql.DB, table string, callback func() M) *Repository[M] {
	r := &Repository[M]{db: db, getConcreteType: callback, table: table}

	return r
}

func (r *Repository[M]) GetTable() string {
	return r.table
}

func (r *Repository[M]) GetDB() *sql.DB {
	return r.db
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

func (r *Repository[M]) Create(ctx context.Context, model M) (int, error) { // nolint:cyclop
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.validate {
		v, ok := any(model).(Validatable)
		if ok {
			err := v.validate(ctx, r.db)
			if err != nil {
				return 0, err
			}
		}
	}

	if r.transform {
		v, ok := any(model).(Transformatable)
		if ok {
			transformedModel, err := v.transform(ctx)
			if err != nil {
				return 0, err
			}

			model, ok = transformedModel.(M)
			if !ok {
				return 0, fmt.Errorf("transform() must return model which will satisfy Model interface")
			}
		}
	}

	m := model.StructToMap(model)

	now := time.Now()
	setCreatedAt(m, now)
	setUpdatedAt(m, now)

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

	if r.onMutate != nil {
		r.onMutate(ctx)
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
		return zero, sql.ErrNoRows
	}

	fields, err := rows.Columns()
	if err != nil {
		return zero, err
	}

	model := r.getConcreteType()
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
		model := r.getConcreteType()

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

	if len(models) == 0 {
		return nil, sql.ErrNoRows
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
	if err != nil {
		return err
	}

	if r.onMutate != nil {
		r.onMutate(ctx)
	}

	return nil
}

func (r *Repository[M]) Update(ctx context.Context, model M, id int) error { // nolint:cyclop
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.validate {
		v, ok := any(model).(Validatable)
		if ok {
			err := v.validate(ctx, r.db)
			if err != nil {
				return err
			}
		}
	}

	if r.transform {
		v, ok := any(model).(Transformatable)
		if ok {
			transformedModel, err := v.transform(ctx)
			if err != nil {
				return err
			}

			model, ok = transformedModel.(M)
			if !ok {
				return fmt.Errorf("transform() must return model which will satisfy Model interface")
			}
		}
	}

	m := model.StructToMap(model)

	delete(m, "id")

	setUpdatedAt(m, time.Now())
	removeCreatedAtFields(m)

	query, args, err := sq.Update(r.table).
		SetMap(m).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if r.onMutate != nil {
		r.onMutate(ctx)
	}

	return nil
}
