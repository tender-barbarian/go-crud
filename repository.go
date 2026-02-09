package gocrud

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
)

// DBQuerier is an interface for database query operations.
// It provides a subset of *sql.DB functionality to enable easier testing
// and mocking in validation logic.
type DBQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Validatable is an interface that models can implement to provide custom validation logic.
// The validate method is called automatically before Create and Update operations when
// the repository is configured with WithValidate().
//
// The method receives a DBQuerier interface (compatible with *sql.DB) to allow database
// queries for validation (e.g., checking uniqueness constraints).
//
// Example:
//
//	func (u *User) validate(ctx context.Context, db DBQuerier) error {
//	    if u.Email == "" {
//	        return errors.New("email is required")
//	    }
//	    // Check uniqueness
//	    var exists bool
//	    row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", u.Email)
//	    if err := row.Scan(&exists); err != nil {
//	        return err
//	    }
//	    if exists {
//	        return errors.New("email already exists")
//	    }
//	    return nil
//	}
type Validatable interface {
	Validate(ctx context.Context, db DBQuerier) error
}

// Transformatable is an interface that models can implement to transform data before
// it is persisted to the database. The transform method is called after validation
// but before the database operation when the repository is configured with WithTransform().
//
// The method must return a Model that satisfies the same type M as the repository.
// Transformations are applied to both Create and Update operations.
//
// Common use cases:
//   - Normalizing data (e.g., lowercasing email addresses)
//   - Enriching data (e.g., setting default values)
//   - Sanitizing input (e.g., trimming whitespace)
//
// Security note: Be cautious when implementing transform. Ensure that transformations
// don't bypass intended security restrictions or modify sensitive fields unexpectedly.
//
// Example:
//
//	func (u *User) transform(ctx context.Context) (Model, error) {
//	    u.Email = strings.ToLower(strings.TrimSpace(u.Email))
//	    u.Name = strings.TrimSpace(u.Name)
//	    return u, nil
//	}
type Transformatable interface {
	Transform(ctx context.Context) (Model, error)
}

// Model is the base interface that all repository models must implement.
// It provides reflection capabilities to convert structs to maps for database operations.
type Model interface {
	StructToMap(d interface{}) map[string]any
}

// Repository is a generic CRUD repository for database operations.
//
// Thread Safety:
// The repository uses a mutex to ensure thread-safe Create, Update, Delete, Get, and GetAll
// operations. This prevents race conditions when multiple goroutines access the same
// repository instance. However, this may become a bottleneck under very high concurrency.
// Consider using separate repository instances per request if needed.
//
// Hook Execution Order:
// For Create and Update operations, hooks execute in this order:
//  1. Validate (if enabled via WithValidate)
//  2. Transform (if enabled via WithTransform)
//  3. Database operation (INSERT/UPDATE)
//  4. OnMutate callback (if configured via WithOnMutate)
//
// For Delete operations:
//  1. Database operation (DELETE)
//  2. OnMutate callback (if configured)
type Repository[M Model] struct {
	mutex           sync.Mutex
	db              *sql.DB
	getConcreteType func() M
	table           string
	validate        bool
	onMutate        func(context.Context)
	transform       bool
}

// WithValidate enables validation for Create and Update operations.
// When enabled, the repository will call the model's validate method (if it implements
// Validatable) before persisting data to the database.
//
// If validation fails, the operation is aborted and the validation error is returned.
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithValidate()
func (r *Repository[M]) WithValidate() *Repository[M] {
	r.validate = true
	return r
}

// WithOnMutate registers a callback function that will be called after successful
// Create, Update, and Delete operations. This is useful for side effects like
// cache invalidation, event publishing, or audit logging.
//
// The callback is called after the database operation completes successfully.
// It does not return an error - if the callback fails, it won't affect the
// database operation (best-effort execution).
//
// Common use cases:
//   - Cache invalidation
//   - Publishing domain events
//   - Updating search indices
//   - Audit logging
//
// Example:
//
//	cache := make(map[int]*User)
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithOnMutate(func(ctx context.Context) {
//	        // Clear cache on any mutation
//	        for k := range cache {
//	            delete(cache, k)
//	        }
//	    })
func (r *Repository[M]) WithOnMutate(fn func(context.Context)) *Repository[M] {
	r.onMutate = fn
	return r
}

// WithTransform enables transformation for Create and Update operations.
// When enabled, the repository will call the model's transform method (if it implements
// Transformatable) after validation but before persisting to the database.
//
// This allows you to normalize or enrich data before it's stored.
//
// Example:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithTransform()
func (r *Repository[M]) WithTransform() *Repository[M] {
	r.transform = true
	return r
}

// NewGenericRepository creates a new generic CRUD repository.
//
// Parameters:
//   - db: The database connection
//   - table: The name of the database table
//   - callback: A factory function that returns a new instance of the model type
//
// The repository can be configured with optional hooks using the builder pattern:
//
//	repo := NewGenericRepository(db, "users", func() *User { return &User{} }).
//	    WithValidate().
//	    WithTransform().
//	    WithOnMutate(cacheInvalidator)
//
// By default, the repository:
//   - Does NOT validate models (use WithValidate to enable)
//   - Does NOT transform models (use WithTransform to enable)
//   - Does NOT call mutation callbacks (use WithOnMutate to register)
//   - DOES automatically set created_at and updated_at timestamps if fields exist
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
			err := v.Validate(ctx, r.db)
			if err != nil {
				return 0, err
			}
		}
	}

	if r.transform {
		v, ok := any(model).(Transformatable)
		if ok {
			transformedModel, err := v.Transform(ctx)
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
		if errors.Is(err, sql.ErrNoRows) {
			return []M{}, nil
		}
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
		return []M{}, nil
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
			err := v.Validate(ctx, r.db)
			if err != nil {
				return err
			}
		}
	}

	if r.transform {
		v, ok := any(model).(Transformatable)
		if ok {
			transformedModel, err := v.Transform(ctx)
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
