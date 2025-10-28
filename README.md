# go-crud

A lightweight, generic CRUD library for Go — designed to eliminate repetitive boilerplate code without the overhead of a full ORM.

## ✨ Overview

`go-crud` provides a simple, flexible interface for defining CRUD operations in Go.

It’s ideal if you:
* Don’t want to hand-write the same `Create`, `Read`, `Update`, and `Delete` logic for each model.
* Don’t need (or want) the complexity of large ORM frameworks.
* Prefer to stay close to standard `database/sql`.

Inspired by Andrew Pillar’s excellent article:
👉 [A Simple CRUD Library for PostgreSQL with Generics in Go](https://andrewpillar.com/archive/programming/2022/10/24/a-simple-crud-library-for-postgresql-with-generics-in-go/)

Although designed to work with any SQL database, it has primarily been tested with PostgreSQL.
If you encounter issues or have improvements, please open an issue or PR — feedback is very welcome!

## 🚀 Usage
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
`gocrud.Reflection` interface provides a `StructToMap(interface{}) map[string]any` method that converts your struct into a map of `field_name:pointer_to_field` using Go’s reflection.

The map keys, which are struct field names, are used to validate against column names returned by the query.

While map values, which are pointers to struct fields, are passed to `rows.Scan()` so your model can be populated.

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

## 🧱 Initialization
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

#### 🔄 Why the Callback?
The callback allows the repository methods to initialize a new instance of the concrete type (your model) at runtime.
For example, the `Get()` method calls the callback to create a fresh model instance to fill it with data retrieved from the database.

Case in point:

`repo := gocrud.NewGenericRepository(db, "users", func() *User { return &User{} })`

Now, whenever the repository needs to return a `User`, it executes the callback to allocate a new one.

## 🧪 Notes
Currently tested primarily with PostgreSQL but should be compatible with any SQL database supported by `database/sql`.
Contributions, bug reports, and performance improvements are highly appreciated!
