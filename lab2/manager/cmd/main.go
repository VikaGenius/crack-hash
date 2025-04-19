package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"manager/internal/handlers"
	"manager/internal/services"
)

func main() {
	workerURLs := os.Getenv("WORKER_URLS")
    workers := strings.Split(workerURLs, ",")
	
	hashService := services.NewService(workers)
    hashHandler := handlers.NewHashHandler(hashService)

    r := mux.NewRouter()
    r.HandleFunc("/api/hash/crack", hashHandler.StartCrackHandler).Methods("POST")
	r.HandleFunc("/api/hash/status", hashHandler.StatusHandler).Methods("GET")
	r.HandleFunc("/internal/api/manager/hash/crack/request", hashHandler.UpdateRequestHandler).Methods("PATCH")

	http.ListenAndServe(":8080", r)
}