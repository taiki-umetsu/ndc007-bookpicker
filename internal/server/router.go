package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/v1/books/random", handler.RandomBooks)
	return r
}
