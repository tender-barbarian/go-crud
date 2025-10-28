package gocrud

import (
	"context"
	"fmt"
	"net/http"

	gocrud "github.com/tender-barbarian/go-crud"
)

type genericRepo[M gocrud.Model] interface {
	Create(ctx context.Context, model M) (int, error)
	Get(ctx context.Context, id int) (M, error)
	GetAll(ctx context.Context) ([]M, error)
	Delete(ctx context.Context, id int) error
	Update(ctx context.Context, model M, id int) error
	GetTable() string
}

func RegisterGenericRoutes[M gocrud.Model](repo genericRepo[M], mux *http.ServeMux) *http.ServeMux {
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update)

	mux.Handle("/", http.NotFoundHandler())

	return mux
}
