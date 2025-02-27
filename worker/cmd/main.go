package main

import (
    "net/http"
    "github.com/gorilla/mux"

    "worker/internal/handlers"
    "worker/internal/services"
)

func main() {
    taskService := services.NewTaskService()
    taskHandler := handlers.NewTaskHandler(taskService)

    r := mux.NewRouter()
    r.HandleFunc("/internal/api/worker/hash/crack/task", taskHandler.TaskHandler).Methods("POST")

	http.ListenAndServe(":8081", r)
}