package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"paracletus-app/data"
	"strconv"
)

// routes define all application routes
func (a *application) routes() *chi.Mux {
	// middlewares must be defined before any routes

	// add routes here
	a.App.Routes.Get("/", a.Handlers.Home)
	a.App.Routes.Get("/go-page", a.Handlers.GoPage)
	a.App.Routes.Get("/jet-page", a.Handlers.JetPage)
	a.App.Routes.Get("/sessions", a.Handlers.SessionTest)

	a.App.Routes.Get("/users/login", a.Handlers.UserLogin)
	a.App.Routes.Post("/users/login", a.Handlers.PostUserLogin)
	a.App.Routes.Get("/users/logout", a.Handlers.UserLogout)

	a.App.Routes.Get("/create-user", func(w http.ResponseWriter, r *http.Request) {
		u := data.User{
			FirstName: "Raul",
			LastName:  "Brindus",
			Email:     "raul@xanda.net",
			Active:    1,
			Password:  "Xanda1234!",
		}

		id, err := a.Models.Users.Insert(u)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}

		fmt.Fprintf(w, "%d:%s", id, u.FirstName)
	})

	a.App.Routes.Get("/get-all-users", func(w http.ResponseWriter, r *http.Request) {
		users, err := a.Models.Users.GetAll()
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}

		for _, x := range users {
			fmt.Fprintf(w, "%d: %s\n", x.ID, x.FirstName)
		}

	})

	a.App.Routes.Get("/get-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))

		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}

		fmt.Fprintf(w, "%+v\n", user)
	})

	a.App.Routes.Get("/update-user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))

		user, err := a.Models.Users.Get(id)
		if err != nil {
			a.App.ErrorLog.Println(err)
			return
		}

		user.LastName = a.App.RandomString(10)

		err = user.Update(*user)

		fmt.Fprintf(w, "%+v\n", user)
	})

	// handler to take care of static assets
	fileServer := http.FileServer(http.Dir("./public"))
	a.App.Routes.Handle("/public/*", http.StripPrefix("/public", fileServer))

	return a.App.Routes
}
