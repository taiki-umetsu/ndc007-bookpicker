package server

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	swgmdw "github.com/go-openapi/runtime/middleware"
)

//go:embed openapi/openapi.bundle.yaml
var bundledSpec []byte

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(RequestLogger)
	r.Use(Recovery)
	r.Use(CORS())
	r.Use(httprate.LimitByIP(10, 1*time.Minute))

	r.Get("/api/v1/books/random", handler.RandomBooks)

	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(bundledSpec)
	})

	opts := swgmdw.SwaggerUIOpts{SpecURL: "/openapi.yaml"}
	r.Handle("/docs", swgmdw.SwaggerUI(opts, nil))
	return r
}
