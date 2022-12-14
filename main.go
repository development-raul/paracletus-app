package main

import (
	"github.com/development-raul/paracletus"
	"paracletus-app/data"
	"paracletus-app/handlers"
)

// application holds all the things our app needs
// and will share with parts of our application that
// needs that kind of functionality
type application struct {
	App      *paracletus.Paracletus
	Handlers *handlers.Handlers
	Models   data.Models
}

func main() {
	p := initApplication()
	p.App.ListenAndServe()
}
