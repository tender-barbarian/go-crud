# go-crud

A lightweight, generic CRUD library for Go — designed to eliminate repetitive boilerplate code without the overhead of a full ORM.

## Overview

`go-crud` provides a simple, flexible interface for defining CRUD operations in Go.

It’s ideal if you:
* Don’t want to hand-write the same `Create`, `Read`, `Update`, and `Delete` logic for each model.
* Don’t need (or want) the complexity of large ORM frameworks.
* Prefer to stay close to standard `database/sql`.

Inspired by Andrew Pillar’s excellent article:
👉 [A Simple CRUD Library for PostgreSQL with Generics in Go](https://andrewpillar.com/archive/programming/2022/10/24/a-simple-crud-library-for-postgresql-with-generics-in-go/)

Although designed to work with any SQL database, it has primarily been tested with PostgreSQL.
If you encounter issues or have improvements, please open an issue or PR — feedback is very welcome!

## Usage
You can use `go-crud` in two ways:
1. **With reflection** (simpler setup, slightly slower)
2. **Without reflection** (manual setup, marginally faster)

### 1. Using Reflection
Each CRUD action maps to a generic repository method.
To get started, define a new model and embed the `gocrud.Reflection` interface in your model.

```
type ModelWithReflection struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	IP    string `json:"ip"`
	Actions []string `json:"actions"`
	gocrud.Reflection
}

repo := gocrud.NewGenericRepository(db, "table_name", func() *ModelWithReflection {
	return &ModelWithReflection{}
})

got, err := repo.Get(ctx, id)
```

### Why the embedded interface?
`gocrud.Reflection` interface provides a `StructToMap(interface{}) map[string]any` method that converts your struct into a map of `field_name:pointer_to_field` using Go's reflection.

The map keys, which are struct field names, are used to validate against column names returned by the query.

While map values, which are pointers to struct fields, are passed to `rows.Scan()` so your model can be populated.

### Custom Column Names with `db` Tag

By default, field names are lowercased to match database columns. You can override this behavior using the `db` struct tag:

```go
type User struct {
    ID        int    `db:"user_id"`      // maps to "user_id" column
    FirstName string `db:"first_name"`   // maps to "first_name" column
    LastName  string                     // maps to "lastname" (lowercase field name)
    gocrud.Reflection
}
```

This is useful when your database column names don't match your Go field names (e.g., snake_case columns with PascalCase fields).

### Automatic Timestamps

`go-crud` automatically handles `created_at` and `updated_at` timestamp fields:

| Operation | `created_at` | `updated_at` |
|-----------|--------------|--------------|
| Create    | Set to now   | Set to now   |
| Update    | Preserved    | Set to now   |

For **PostgreSQL** (native timestamp support), use `time.Time`:

```go
type Item struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    gocrud.Reflection
}
```

For **SQLite** (stores timestamps as strings), use `gocrud.NullTime`:

```go
type Item struct {
    ID        int            `json:"id"`
    Name      string         `json:"name"`
    CreatedAt gocrud.NullTime `json:"created_at" db:"created_at"`
    UpdatedAt gocrud.NullTime `json:"updated_at" db:"updated_at"`
    gocrud.Reflection
}

// Access the time value via .Time field
fmt.Println(item.CreatedAt.Time)
```

`NullTime` implements `sql.Scanner` and `driver.Valuer`, handling both native time values and string representations.

**Recognized field names:**
- `created_at`, `createdat`, `created`
- `updated_at`, `updatedat`, `updated`, `modified_at`, `modifiedat`

Models without these fields continue to work unchanged.

---

### 2. Without Reflection
If you prefer to avoid reflection, you can implement `StructToMap` yourself.
In benchmarks, the performance difference is negligible but this approach gives you full control.

```
type ModelWithoutReflection struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	IP      string   `json:"ip"`
	Actions []string `json:"actions"`
}

func (o *ModelWithoutReflection) StructToMap(interface{}) map[string]any {
	return map[string]any{
		"id":      &o.ID,
		"name":    &o.Name,
		"type":    &o.Type,
		"ip":      &o.IP,
		"actions": &o.Actions,
	}
}

repo := gocrud.NewGenericRepository(db, "table_name", func() *ModelWithoutReflection {
	return &ModelWithoutReflection{}
})

got, err := repo.Get(ctx, id)
```

## Initialization
You can create a new generic repository using:

```
gocrud.NewGenericRepository[M gocrud.Model](
	db *sql.DB,
	table string,
	callback func() M,
) *Repository[M]
```

### Parameters
1. `db` — your `*sql.DB` connection
2. `table` — the name of the database table to target
3. `callback` — a function returning a new instance of your model

#### Why the Callback?
The callback allows the repository methods to initialize a new instance of the concrete type (your model) at runtime.
For example, the `Get()` method calls the callback to create a fresh model instance to fill it with data retrieved from the database.

Case in point:

`repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} })`

Now, whenever the repository needs to return a `User`, it executes the callback to allocate a new one.

## Repository Hooks

`go-crud` supports an opt-in hook system for validation, transformation, and mutation callbacks. Configure hooks using the builder pattern:

```go
repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} }).
    WithValidate().
    WithTransform().
    WithOnMutate(cacheInvalidator)
```

### Hook Execution Order

For Create and Update operations:
1. **Validate** - validates the model before persisting
2. **Transform** - transforms/normalizes data
3. **Database operation** - INSERT or UPDATE
4. **OnMutate** - callback for side effects

For Delete operations:
1. **Database operation** - DELETE
2. **OnMutate** - callback for side effects

### Validation Hook

Models can implement the `Validatable` interface to enable automatic validation:

```go
type Validatable interface {
    Validate(ctx context.Context, db DBQuerier) error
}
```

Example with field validation and uniqueness check:

```go
func (u *User) Validate(ctx context.Context, db gocrud.DBQuerier) error {
    if u.Email == "" {
        return errors.New("email is required")
    }

    // Check uniqueness
    var exists bool
    row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", u.Email)
    if err := row.Scan(&exists); err != nil {
        return err
    }
    if exists {
        return errors.New("email already exists")
    }

    return nil
}
```

Enable validation with `WithValidate()`:

```go
repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} }).
    WithValidate()
```

### Transformation Hook

Models can implement the `Transformatable` interface to normalize/enrich data before persistence:

```go
type Transformatable interface {
    Transform(ctx context.Context) (Model, error)
}
```

Example for normalizing email addresses:

```go
func (u *User) Transform(ctx context.Context) (gocrud.Model, error) {
    u.Email = strings.ToLower(strings.TrimSpace(u.Email))
    u.Name = strings.TrimSpace(u.Name)
    return u, nil
}
```

Enable transformation with `WithTransform()`:

```go
repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} }).
    WithTransform()
```

### OnMutate Callback

Register a callback to run after successful Create, Update, or Delete operations:

```go
cache := make(map[int]*User)

repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} }).
    WithOnMutate(func(ctx context.Context) {
        // Clear cache on any mutation
        for k := range cache {
            delete(cache, k)
        }
    })
```

Common use cases:
- Cache invalidation
- Publishing domain events
- Updating search indices
- Audit logging

The callback does not return an error (best-effort execution).


## HTTP Route Registration

`go-crud` provides helper functions to register HTTP handlers for your CRUD operations:

```go
mux := http.NewServeMux()
repo := gocrud.NewGenericRepository(db, "items", func() *Item { return &Item{} })
eh := gocrud.DefaultErrorHandler{}

gocrud.RegisterCreate("POST /items", mux, repo.Create, eh)
gocrud.RegisterGet("GET /items/{id}", mux, repo.Get, eh)
gocrud.RegisterGetAll("GET /items", mux, repo.GetAll, eh)
gocrud.RegisterDelete("DELETE /items/{id}", mux, repo.Delete, eh)
gocrud.RegisterUpdate("POST /items/{id}", mux, repo.Update, eh)
```

### Custom Error Handling

The `ErrorHandler` interface allows you to customize how errors are returned to clients:

```go
type ErrorHandler interface {
    WriteError(w http.ResponseWriter, r *http.Request, err error, customMsg string, statusCode int)
}
```

The `DefaultErrorHandler` uses the custom message if provided, otherwise falls back to `err.Error()`:

```go
type DefaultErrorHandler struct{}

func (d DefaultErrorHandler) WriteError(w http.ResponseWriter, _ *http.Request, err error, customMsg string, statusCode int) {
    if customMsg != "" {
        http.Error(w, customMsg, statusCode)
        return
    }
    http.Error(w, err.Error(), statusCode)
}
```

To implement custom error handling (e.g., JSON responses, logging), create your own type that implements `ErrorHandler`:

```go
type JSONErrorHandler struct {
    Logger *log.Logger
}

func (j JSONErrorHandler) WriteError(w http.ResponseWriter, r *http.Request, err error, customMsg string, statusCode int) {
    j.Logger.Printf("error: %v", err)

    msg := customMsg
    if msg == "" && err != nil {
        msg = err.Error()
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

## Complete Example

Here's a full example combining hooks, HTTP handlers, and automatic timestamps:

```go
type User struct {
    ID        int            `json:"id"`
    Email     string         `json:"email"`
    Name      string         `json:"name"`
    CreatedAt gocrud.NullTime `json:"created_at" db:"created_at"`
    UpdatedAt gocrud.NullTime `json:"updated_at" db:"updated_at"`
    gocrud.Reflection
}

// Validation hook
func (u *User) Validate(ctx context.Context, db gocrud.DBQuerier) error {
    if u.Email == "" {
        return errors.New("email is required")
    }

    var exists bool
    row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", u.Email)
    if err := row.Scan(&exists); err != nil {
        return err
    }
    if exists {
        return errors.New("email already exists")
    }

    return nil
}

// Transformation hook
func (u *User) Transform(ctx context.Context) (gocrud.Model, error) {
    u.Email = strings.ToLower(strings.TrimSpace(u.Email))
    u.Name = strings.TrimSpace(u.Name)
    return u, nil
}

func main() {
    db, _ := sql.Open("sqlite3", "app.db")

    // Setup repository with all hooks
    cache := make(map[int]*User)
    repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} }).
        WithValidate().
        WithTransform().
        WithOnMutate(func(ctx context.Context) {
            for k := range cache {
                delete(cache, k)
            }
        })

    // Register HTTP handlers
    mux := http.NewServeMux()
    eh := gocrud.DefaultErrorHandler{}

    gocrud.RegisterCreate("POST /users", mux, repo.Create, eh)
    gocrud.RegisterGet("GET /users/{id}", mux, repo.Get, eh)
    gocrud.RegisterGetAll("GET /users", mux, repo.GetAll, eh)
    gocrud.RegisterUpdate("POST /users/{id}", mux, repo.Update, eh)
    gocrud.RegisterDelete("DELETE /users/{id}", mux, repo.Delete, eh)

    http.ListenAndServe(":8080", mux)
}
```

## Notes
Tested with both SQLite and PostgreSQL. Should be compatible with any SQL database supported by `database/sql`.
Contributions, bug reports, and performance improvements are highly appreciated!
