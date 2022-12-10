package handlers

import (
	"github.com/development-raul/paracletus"
	"net/http"
)

type Handlers struct {
	App *paracletus.Paracletus
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	err := h.App.Render.Page(w, r, "home", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println("error rendering home :", err)
	}
}
