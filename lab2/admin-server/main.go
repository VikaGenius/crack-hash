package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/mux"
)

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/crack", CrackHashPageHandler).Methods("GET")
	r.HandleFunc("/admin", AdminPageHandler).Methods("GET")
	r.HandleFunc("/admin/stop", StopContainerHandler).Methods("POST")
	r.HandleFunc("/admin/start", StartContainerHandler).Methods("POST")

	log.Println("Admin server is starting...")
	if err := http.ListenAndServe(":9090", r); err != nil {
		log.Fatalf("Error starting admin server: %v", err)
	}
}

func AdminPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/admin.html")
}

func CrackHashPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/ui.html")
}

func StopContainerHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Service string `json:"service"`
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	cmd := exec.Command("docker", "stop", req.Service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Command failed: "+err.Error()+"\nOutput: "+string(output), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Stopped " + req.Service))
}

func StartContainerHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Service string `json:"service"`
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	cmd := exec.Command("docker", "start", req.Service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, string(output), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Started " + req.Service))
}
