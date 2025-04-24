package handlers

import (
	"encoding/json"
	"net/http"

	"manager/internal/models"
	"manager/internal/services"
)

type Handler struct {
    service *services.Service
}

func NewHashHandler(service *services.Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) StartCrackHandler(w http.ResponseWriter, r *http.Request) {
    var req models.CrackHashRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if req.Hash == "" || req.MaxLength <= 0 {
        http.Error(w, "Hash and maxLength are required", http.StatusBadRequest)
        return
    }
    
    requestId, error := h.service.StartCrack(req.Hash, req.MaxLength)
    if error != nil {
        http.Error(w, "Service failed", http.StatusBadRequest)
        return
    }

    response := models.StartWorkResponse{
        RequestId: requestId,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
    requestId := r.URL.Query().Get("requestId")
    if requestId == "" {
        http.Error(w, "requestId is required", http.StatusBadRequest)
        return
    }

    status, data, progress := h.service.GetStatus(requestId)
    
    response := models.CrackStatusResponse{
        Status: status,
        Data:   data,
        Progress: progress,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}