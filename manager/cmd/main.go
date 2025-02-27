package cmd

import (
	"net/http"
    "github.com/gorilla/mux"	
	
	"manager/internal/services"
	"manager/internal/handlers"
)

func main() {
	hashService := services.NewService()
    hashHandler := handlers.NewHashHandler(hashService)

    r := mux.NewRouter()
    r.HandleFunc("/api/hash/crack", hashHandler.StartCrackHandler).Methods("POST")
	r.HandleFunc("/api/hash/status", hashHandler.StatusHandler).Methods("GET")
	r.HandleFunc("/internal/api/manager/hash/crack/request", hashHandler.UpdateRequestHandler).Methods("PATCH")

	http.ListenAndServe(":8080", r)
}