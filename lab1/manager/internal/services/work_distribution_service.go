package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	//"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"manager/internal/models"

	"github.com/google/uuid"
)

type Service struct {
    requests map[string]*CrackRequest // Хранит задачи по их ID
    mu       sync.RWMutex            // Для потокобезопасного доступа к requests
    workers  []string                // Адреса воркеров
}

type CrackRequest struct {
    Hash      string
    MaxLength int
    Status    string
    Data      []string
    Workers   int
    Done      chan []string
    Progress  int
}

const (
    StatusInProgress string = "IN_PROGRESS"
    StatusReady      string = "READY"
    StatusError      string = "ERROR"
    StatusPartiallyReady string = "PARTIALLY_READY"
)

func NewService(workers []string) *Service {
    return &Service{
        requests: make(map[string]*CrackRequest),
        workers:  workers,
    }
}

func (s *Service) StartCrack(hash string, maxLength int) string {
    requestId := generateRequestId()
    cr := &CrackRequest{
        Hash:      hash,
        MaxLength: maxLength,
        Status:    StatusInProgress,
        Workers:   len(s.workers),
        Done:      make(chan []string),
    }

    s.mu.Lock()
    s.requests[requestId] = cr
    s.mu.Unlock()

    go s.distributeWork(cr, requestId)

    return requestId
}

func (s *Service) UpdateRequest(requestId string, partNumber int, answers []string) {
    fmt.Println("Получаю результат от воркера")

    s.mu.Lock()
    defer s.mu.Unlock()

    cr, exists := s.requests[requestId]
    if !exists {
        return 
    }

    cr.Done <- answers

    cr.Progress += 100 / cr.Workers
    printProgress(requestId, cr.Progress)
}

func (s *Service) GetStatus(requestId string) (string, []string, string) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    cr, exists := s.requests[requestId]
    if !exists {
        return StatusError, nil, ""
    }

    return cr.Status, cr.Data, fmt.Sprintf("%d%%", cr.Progress)
}

func (s *Service) distributeWork(cr *CrackRequest, requestId string) {
    fmt.Println("Распределяю работу между подчиненными!")

    defer close(cr.Done)
    alphabet := generateAlphabet()

    partCount := len(s.workers)
    for partNumber := 0; partNumber < partCount; partNumber++ {
        task := models.CrackHashManagerRequest{
            RequestId:  requestId,
            PartNumber: partNumber,
            PartCount:  partCount,
            Hash:       cr.Hash,
            MaxLength:  cr.MaxLength,
            Alphabet:   alphabet,
        }

        go s.sendTaskToWorker(s.workers[partNumber], task, requestId, partNumber)
    }

    results := make([][]string, partCount)
    for partNumber := 0; partNumber < partCount; partNumber++ {
        select {
        case result := <-cr.Done:
            results[partNumber] = result
        case <-time.After(3000 * time.Second): // Таймаут 5 минут
            log.Printf("Timeout for request %s, part %d", requestId, partNumber)

            s.mu.Lock()
            if len(results) > 0 {
                cr.Status = StatusPartiallyReady
            } else {
                cr.Status = StatusError
            }
            s.mu.Unlock()

            return
        }
    }

    s.mu.Lock()
    cr.Status = StatusReady
    cr.Progress = 100
    cr.Data = append(cr.Data, flattenResults(results)...)
    s.mu.Unlock()
}

func (s *Service) sendTaskToWorker(workerAddress string, task models.CrackHashManagerRequest, requestId string, partNumber int) {
    jsonData, err := json.Marshal(task)
    if err != nil {
        log.Printf("Error marshaling task: %v", err)
        return
    }
    fmt.Println("Отправляю запрос рабочему")
    resp, err := http.Post(workerAddress+"/internal/api/worker/hash/crack/task", 
						   "application/json", 
						   bytes.NewBuffer(jsonData))
    if err != nil {
        log.Printf("Error sending task to worker: %v", err)
        return
    }
    defer resp.Body.Close()
}

func generateAlphabet() []string {
    alphabet := make([]string, 0, 36)
    for ch := 'a'; ch <= 'z'; ch++ {
        alphabet = append(alphabet, string(ch))
    }
    for ch := '0'; ch <= '9'; ch++ {
        alphabet = append(alphabet, string(ch))
    }
    return alphabet
}

func flattenResults(results [][]string) []string {
    var flat []string
    for _, part := range results {
        flat = append(flat, part...)
    }
    return flat
}

func generateRequestId() string {
    return uuid.New().String()
}

func printProgress(requestId string, progress int) {
    barWidth := 50
    filled := progress * barWidth / 100
    bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
    fmt.Printf("\rЗадача %s: [%s] %d%%", requestId, bar, progress)
    fmt.Println()
}