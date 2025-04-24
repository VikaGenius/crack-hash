package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	mq "worker/internal/message_queue"
	"worker/internal/models"
)

type TaskService struct {
    queue *mq.WorkerQueue
    managerURL string
}

func NewTaskService(managerURL string, queue *mq.WorkerQueue) *TaskService {
    service := &TaskService{
        managerURL: managerURL,
        queue: queue,
    }

    if err := queue.ConsumeTasks(service.handleTask); err != nil {
		log.Fatalf("Failed to start task consumer: %v", err)
	}
    return service
}

func (s *TaskService) ProcessTask(req models.TaskMessage) []string {
    // вычисляем объем работы для текущего воркера
    alphabetSize := len(req.Alphabet)
    totalCombinations := totalCombinations(alphabetSize, req.MaxLength)
    partSize := int(math.Ceil(float64(totalCombinations) / float64(req.PartCount)))
    start := req.PartNumber * partSize
    end := min(start+partSize, totalCombinations)

    fmt.Println("Начинаю свою работу")
    return generateAndCheckCombinations(req.Alphabet, req.MaxLength, start, end, req.Hash)
}

func (s *TaskService) handleTask(task models.TaskMessage) error {
	answers := s.ProcessTask(task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

    fmt.Print("Отправляю результат менеджеру: ")
    fmt.Println(answers)

	return s.queue.PublishResult(ctx, models.ResultMessage{
		RequestId:  task.RequestId,
		PartNumber: task.PartNumber,
		Answers:    answers,
	})
}

func generateAndCheckCombinations(alphabet []string, maxLength int, start int, end int, targetHash string) []string {
    var answers []string
    n := len(alphabet)
    currentIndex := 0

    for length := 1; length <= maxLength; length++ {
        fmt.Println("Начинаю перебор в заданном диапазоне")

        numCombinations := int(math.Pow(float64(n), float64(length)))
        if currentIndex+numCombinations <= start {
            currentIndex += numCombinations
            continue
        }

        for i := range numCombinations {
            if currentIndex >= start && currentIndex < end {
                combination := generateCombination(alphabet, length, i)
                hash := md5Hash(combination)
                if hash == targetHash {
                    answers = append(answers, combination)
                }
            }
            currentIndex++
            if currentIndex >= end {
                return answers
            }
        }
    }

    return answers
}

func generateCombination(alphabet []string, length int, index int) string {
    n := len(alphabet)
    combination := make([]string, length)
    for i := range length {
        combination[length-i-1] = alphabet[index%n]
        index /= n
    }
    return strings.Join(combination, "")
}

func md5Hash(input string) string {
    hasher := md5.New()
    hasher.Write([]byte(input))
    return hex.EncodeToString(hasher.Sum(nil))
}

func totalCombinations(alphabetSize int, maxLength int) int {
    total := 0
    for i := 1; i <= maxLength; i++ {
        total += int(math.Pow(float64(alphabetSize), float64(i)))
    }
    return total
}