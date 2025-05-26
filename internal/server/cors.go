package server

import (
	"net/http"

	"github.com/go-chi/cors"
)

func CORS() func(next http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Accept"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           300,
	}).Handler
}
