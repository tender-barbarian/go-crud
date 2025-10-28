# go-crud
This is a simple CRUD library.

# How It Works

I got tired of endlessly redefining the same boilerplate CRUD code for each item but at the same time I didn't want to use big ORM projects that handle this sort of thing.

I took main idea from: https://andrewpillar.com/archive/programming/2022/10/24/a-simple-crud-library-for-postgresql-with-generics-in-go/

# Usage

## Generic Repository

### With Reflection
Each CRUD action is mapped to generic repository method. All you need to do is provide your model and embed `Reflection` interface.

```
type ModelWithReflection struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Chip  string `json:"chip"`
	Board string `json:"board"`
	IP    string `json:"ip"`
	gocrud.Reflection
}

repo := NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })
got, err := repo.Get(ctx, id)
```

`Reflection` interface provides `StructToMap(interface{}) map[string]any` method which converts your struct into a map of struct fields pointers using Reflection. The struct fields pointers are then used in `rows.Scan()` to scan directly into struct.

### Without Reflection
There is also an option to do this without reflection (even though performance impact seems to be negligeble in benchmarks) but in such case you need to provide `StructToMap(interface{}) map[string]any` method yourself:

```
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
		"actions": &o.Actions,
	}
}

repo := NewGenericRepository(db, "devices", func() *ModelWithoutReflection { return &ModelWithoutReflection{} })
got, err := repo.Get(ctx, id)
```

### Model constructor

Repo init function `NewGenericRepository[M Model](db *sql.DB, table string, callback func() M) *Repository[M]`  takes a callback function which allows to initialize concrete type inside generic method.

For example: `NewGenericRepository(db, "table_name", func() *ModelWithReflection { return &ModelWithReflection{} })`

Now whenever a fresh concrete type is needed, for example in generic `Get()` method, the callback function will be executed to get an empty concrete type which can then be filled by response from DB and returned to the caller.

## Generic Handler

There is also an option to import generic handlers for each CRUD action:

```
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)
```

You need to provide your own pattern.

## Examples

TODO
