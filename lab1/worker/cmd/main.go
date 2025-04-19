package main

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"worker/internal/handlers"
	"worker/internal/services"
)

func main() {
    managerURL := os.Getenv("MANAGER_URL")
    taskService := services.NewTaskService(managerURL)
    taskHandler := handlers.NewTaskHandler(taskService)

    r := mux.NewRouter()
    r.HandleFunc("/internal/api/worker/hash/crack/task", taskHandler.TaskHandler).Methods("POST")

	http.ListenAndServe(":8081", r)
}