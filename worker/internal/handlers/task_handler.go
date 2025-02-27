package handlers

import (
    "encoding/json"
    "net/http"

    "worker/internal/models"
    "worker/internal/services"
)

type TaskHandler struct {
    taskService *services.TaskService
}

func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
    return &TaskHandler{taskService: taskService}
}

func (h *TaskHandler) TaskHandler(w http.ResponseWriter, r *http.Request) {
    var req models.CrackHashManagerRequest

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    answers := h.taskService.ProcessTask(req)

    h.taskService.SendResultsToManager(req.RequestId, req.PartNumber, answers)

    // response := models.CrackHashWorkerResponse{
    //     RequestId:  req.RequestId,
    //     PartNumber: req.PartNumber,
    //     Answers:    answers,
    // }

    // w.Header().Set("Content-Type", "application/json")
    // w.WriteHeader(http.StatusOK)
    // json.NewEncoder(w).Encode(response)
}