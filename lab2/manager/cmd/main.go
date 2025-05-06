package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"manager/internal/handlers"
	"manager/internal/repository"
	"manager/internal/services"
	mq "manager/internal/message_queue"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")

	repo, err := repository.NewMongoRepository(mongoURI)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB repository: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		}
	}()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	queue, err := mq.NewManagerQueue(rabbitURL, "tasks", "results")
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}
	defer func() {
		if err := queue.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		}
	}()

	workerURLs := os.Getenv("WORKER_URLS")
    workers := strings.Split(workerURLs, ",")
	
	hashService := services.NewService(workers, repo, queue)
    hashHandler := handlers.NewHashHandler(hashService)

    r := mux.NewRouter()
	r.Use(corsMiddleware())

    r.HandleFunc("/api/hash/crack", hashHandler.StartCrackHandler).Methods("POST")
	r.HandleFunc("/api/hash/status", hashHandler.StatusHandler).Methods("GET")

	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.ListenAndServe(":8080", r)
}

func corsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
