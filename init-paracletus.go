package main

import (
	"github.com/development-raul/paracletus"
	"log"
	"os"
	"paracletus-app/handlers"
)

func initApplication() *application {
	// find the working dir
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// initialize paracletus
	par := &paracletus.Paracletus{}
	err = par.New(path)
	if err != nil {
		log.Fatal(err)
	}

	par.AppName = "paracletusApp"

	myHandlers := &handlers.Handlers{
		App: par,
	}

	// build our application variable
	app := &application{
		App:      par,
		Handlers: myHandlers,
	}

	// add our application routes to the default Paracletus routes
	app.App.Routes = app.routes()

	return app
}
