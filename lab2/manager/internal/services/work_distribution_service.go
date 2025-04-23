package services

import (
	"context"
	"fmt"

	"log"
	"strings"
	"sync"
	"time"

	"manager/internal/models"
    "manager/internal/repository"
	mq "manager/internal/message_queue"
)

type Service struct {
    requests map[string]*models.CrackRequest //хранит задачи по их ID
    mu       sync.RWMutex            //для потокобезопасного доступа к requests
    workers  []string                //адреса воркеров
    repo     *repository.MongoRepository
	queue     *mq.ManagerQueue
}

const (
    StatusInProgress string = "IN_PROGRESS"
    StatusReady      string = "READY"
    StatusError      string = "ERROR"
    StatusPartiallyReady string = "PARTIALLY_READY"
)

func NewService(workers []string, repo *repository.MongoRepository, queue *mq.ManagerQueue,) *Service {
    service := &Service{
		repo:     repo,
        workers:  workers,
		requests: make(map[string]*models.CrackRequest),
		queue: queue,
	}

	//запускаем consumer для результатов
	if err := queue.ConsumeResults(service.handleResult); err != nil {
		log.Fatalf("Failed to start result consumer: %v", err)
	}

	//восстановление незавершенных задач при старте
	service.restorePendingRequests()

	return service
}

func (s *Service) StartCrack(hash string, maxLength int) (string, error) {
	req := models.NewCrackRequest(hash, maxLength)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//сохраняем запрос в БД с подтверждением репликации
	if err := s.repo.SaveRequest(ctx, req); err != nil {
		return "", err
	}

	s.mu.Lock()
	s.requests[req.RequestID] = req
	s.mu.Unlock()

	go s.distributeWork(req)
	return req.RequestID, nil
}

func (s *Service) UpdateRequest(requestID string, partNumber int, answers []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, exists := s.requests[requestID]
	if !exists {
		log.Printf("Request %s not found", requestID)
		return
	}

	fmt.Print("Получаю результат от воркера: ")
	fmt.Println(answers)
	req.Data = append(req.Data, answers...)
	req.Progress += 100 / len(s.workers)
	if req.Progress >= 99 {
		req.Progress = 100
        req.Status = StatusReady
    }

	//обновляем в БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.repo.UpdateRequest(ctx, req); err != nil {
		log.Printf("Failed to update request %s: %v", requestID, err)
	}

	printProgress(requestID, req.Progress)
}

func (s *Service) GetStatus(requestID string) (string, []string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if req, exists := s.requests[requestID]; exists {
		return req.Status, req.Data, fmt.Sprintf("%d%%", req.Progress)
	}

	//если нет в памяти, проверяем в БД
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return StatusError, nil, "0%"
	}

	return req.Status, req.Data, fmt.Sprintf("%d%%", req.Progress)
}

func (s *Service) distributeWork(req *models.CrackRequest) {
	alphabet := generateAlphabet()
	partCount := len(s.workers)

	for partNumber := 0; partNumber < partCount; partNumber++ {
		task := models.TaskMessage{
			RequestId:  req.RequestID,
			PartNumber: partNumber,
			PartCount:  partCount,
			Hash:       req.Hash,
			MaxLength:  req.MaxLength,
			Alphabet:   alphabet,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.queue.PublishTask(ctx, task); err != nil {
			log.Printf("Failed to publish task for request %s, part %d: %v", req.RequestID, partNumber, err)
		}
	}
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

func printProgress(requestId string, progress int) {
    barWidth := 50
    filled := progress * barWidth / 100
    bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
    fmt.Printf("\rЗадача %s: [%s] %d%%", requestId, bar, progress)
    fmt.Println()
}

func (s *Service) restorePendingRequests() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pendingRequests, err := s.repo.GetPendingRequests(ctx)
	if err != nil {
		log.Printf("Failed to restore pending requests: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, req := range pendingRequests {
		s.requests[req.RequestID] = &req
		go s.distributeWork(&req)
	}
}

func (s *Service) handleResult(result models.ResultMessage) error {
	s.UpdateRequest(result.RequestId, result.PartNumber, result.Answers)
	return nil
}