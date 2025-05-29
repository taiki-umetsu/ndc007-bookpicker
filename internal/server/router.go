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

const openapiSpecPath = "/openapi.yaml"

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(RequestLogger)
	r.Use(Recovery)
	r.Use(CORS())
	r.Use(httprate.LimitByIP(10, 1*time.Minute))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/books/random", handler.RandomBooks)
	})

	r.Get(openapiSpecPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(bundledSpec)
	})

	r.Handle("/docs", swgmdw.SwaggerUI(swgmdw.SwaggerUIOpts{SpecURL: openapiSpecPath}, nil))
	return r
}
