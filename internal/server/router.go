package server

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	swaggermiddleware "github.com/go-openapi/runtime/middleware"
)

//go:embed openapi.yaml
var specFS embed.FS

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/v1/books/random", handler.RandomBooks)

	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		data, err := specFS.ReadFile("openapi.yaml")
		if err != nil {
			http.Error(w, "cannot read spec", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(data)
	})

	opts := swaggermiddleware.SwaggerUIOpts{SpecURL: "/openapi.yaml"}
	r.Handle("/docs", swaggermiddleware.SwaggerUI(opts, nil))
	return r
}
