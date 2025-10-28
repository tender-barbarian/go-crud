# go-crud
This is a simple CRUD library.

I got tired of endlessly redefining the same boilerplate CRUD code for each item but at the same time I didn't want to use big ORM projects that handle this sort of thing.

I took main idea from: https://andrewpillar.com/archive/programming/2022/10/24/a-simple-crud-library-for-postgresql-with-generics-in-go/

This library should work with any DB but I didn't really test it very extensively so please report back any bugs/improvements/sugegstions.

# Usage

## With Reflection
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

## Without Reflection
There is also an option to do this without reflection (I've run some benchmarks and performance impact seems to be negligeble) but in such case you need to provide `StructToMap(interface{}) map[string]any` method yourself:

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

## Repo init
After model is redined you can initialize new generic repository: `gocrud.NewGenericRepository[M gocrud.Model](db *sql.DB, table string, callback func() M) *Repository[M]`. It takes three arguments:

1. DB connection
2. Name of the table you want this generic repo to call
3. Callback function

### Why the callback?
Callback function enables initialization of concrete type inside generic method.

Callback function should be passed to the repo init like this: `func() *YourModel { return &YourModel{} }`.

Now whenever a fresh concrete type is needed, for example in generic `Get()` method, the callback function will be executed to init a concrete type which can then be filled by response from DB and returned to the caller.

## Generic Handler

There is also an option to import and register generic handlers for each CRUD action:

```
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)
```

Each CRUD action register method takes three arguments:

1. Pattern
2. mux
3. Repository function callback

Checkout `examples` to check full implementation.

## Examples

TODO
