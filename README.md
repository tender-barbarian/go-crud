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

Simply add `time.Time` fields to your model:

```go
type Item struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    gocrud.Reflection
}
```

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

## HTTP Route Registration

`go-crud` provides helper functions to register HTTP handlers for your CRUD operations:

```go
mux := http.NewServeMux()
repo := gocrud.NewGenericRepository(db, "items", func() *Item { return &Item{} })
eh := gocrud.DefaultErrorHandler{}

gocrud.RegisterCreate("POST /items", mux, repo.Create, db, eh)
gocrud.RegisterGet("GET /items/{id}", mux, repo.Get, eh)
gocrud.RegisterGetAll("GET /items", mux, repo.GetAll, eh)
gocrud.RegisterDelete("DELETE /items/{id}", mux, repo.Delete, eh)
gocrud.RegisterUpdate("POST /items/{id}", mux, repo.Update, db, eh)
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

### Input Validation

Models can optionally implement the `Validatable` interface to enable automatic validation after JSON decoding in `RegisterCreate` and `RegisterUpdate`:

```go
type Validatable interface {
    Validate(ctx context.Context, db DBQuerier) error
}
```

The `DBQuerier` interface provides database query capabilities for validations that need them:

```go
type DBQuerier interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

Example — basic validation and uniqueness check:

```go
type Item struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    gocrud.Reflection
}

func (i *Item) Validate(ctx context.Context, db gocrud.DBQuerier) error {
    if i.Name == "" {
        return errors.New("name is required")
    }

    // Database validation (only if db is provided)
    if db != nil {
        var exists bool
        row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM items WHERE name = ?)", i.Name)
        if err := row.Scan(&exists); err != nil {
            return err
        }
        if exists {
            return errors.New("name already exists")
        }
    }

    return nil
}
```

Pass `nil` for the `db` parameter if your model doesn't need database validation:

```go
gocrud.RegisterCreate("POST /items", mux, repo.Create, nil, eh)  // no DB validation
gocrud.RegisterCreate("POST /items", mux, repo.Create, db, eh)   // with DB validation
```

You can also use the repository's `GetDB()` helper to retrieve the database connection:

```go
gocrud.RegisterCreate("POST /items", mux, repo.Create, repo.GetDB(), eh)
```

When validation fails, the handler returns HTTP 400 Bad Request with "validation error" as the response body by default. Models that don't implement `Validatable` skip validation.

To customize the error message and HTTP status code, implement the `ValidationError` interface on your error:

```go
type ValidationError interface {
    error
    Message() string
    StatusCode() int
}

type NameRequiredError struct{}

func (e NameRequiredError) Error() string      { return "name is required" }
func (e NameRequiredError) Message() string    { return "name field cannot be empty" }
func (e NameRequiredError) StatusCode() int    { return http.StatusUnprocessableEntity }

func (i *Item) Validate(ctx context.Context, db gocrud.DBQuerier) error {
    if i.Name == "" {
        return NameRequiredError{}
    }
    return nil
}
```

## Notes
Currently tested primarily with SQLite but should be compatible with any SQL database supported by `database/sql`.
Contributions, bug reports, and performance improvements are highly appreciated!
