package app

import (
	"net/http"

	"github.com/gorilla/mux"

	app "github.com/tprynn/goread"
)

func init() {
	router := mux.NewRouter()
	app.RegisterHandlers(router)
	http.Handle("/", router)
}
