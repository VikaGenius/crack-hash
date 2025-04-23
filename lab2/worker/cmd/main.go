package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"worker/internal/services"
	mq "worker/internal/message_queue"

)

func main() {
    managerURL := os.Getenv("MANAGER_URL")
	rabbitURL := os.Getenv("RABBITMQ_URL")	
	queue, err := mq.NewWorkerQueue(rabbitURL, "crack_hash_tasks")
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}
	defer func() {
		if err := queue.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}()
    services.NewTaskService(managerURL, queue)

    r := mux.NewRouter()

	http.ListenAndServe(":8081", r)
}