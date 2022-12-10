package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

// routes define all application routes
func (a *application) routes() *chi.Mux {
	// middlewares must be defined before any routes

	// add routes here
	a.App.Routes.Get("/", a.Handlers.Home)

	// handler to take care of static assets
	fileServer := http.FileServer(http.Dir("./public"))
	a.App.Routes.Handle("/public/*", http.StripPrefix("/public", fileServer))

	return a.App.Routes
}
