package services

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"worker/internal/models"

	combinations "github.com/mxschmitt/golang-combinations"
)

type TaskService struct {
    managerURL string
}

func NewTaskService(managerURL string) *TaskService {
    return &TaskService{managerURL: managerURL}
}

func (s *TaskService) ProcessTask(req models.CrackHashManagerRequest) []string {
    var answers []string

    combinations := generateCombinations(req.Alphabet, req.MaxLength)

    // вычисление диапазона для текущего воркера
    totalCombinations := len(combinations)
    partSize := int(math.Ceil(float64(totalCombinations) / float64(req.PartCount)))
    start := req.PartNumber * partSize
    end := min(start + partSize, totalCombinations)

    // перебор комбинаций в заданном диапазоне
    for _, word := range combinations[start:end] {
        hash := md5Hash(word)
        if hash == req.Hash {
            answers = append(answers, word)
        }
    }

    return answers
}

func generateCombinations(alphabet []string, maxLength int) []string {
    var allCombinations []string

    for length := 1; length <= maxLength; length++ {
        combinations := combinations.Combinations(alphabet, length)
        for _, combo := range combinations {
            word := strings.Join(combo, "") // Объединяем элементы массива в строку
            allCombinations = append(allCombinations, word)
        }
    }

    return allCombinations
}

func md5Hash(input string) string {
    hasher := md5.New()
    hasher.Write([]byte(input))
    return hex.EncodeToString(hasher.Sum(nil))
}

func (s *TaskService) SendResultsToManager(requestId string, partNumber int, answers []string) {
    response := models.CrackHashWorkerResponse{
        RequestId:  requestId,
        PartNumber: partNumber,
        Answers:    answers,
    }

    jsonData, err := json.Marshal(response)
    if err != nil {
        log.Printf("Error marshaling response: %v", err)
        return
    }

    // Отправка PATCH-запроса менеджеру
    req, err := http.NewRequest(http.MethodPatch, s.managerURL+"/internal/api/manager/hash/crack/request", bytes.NewBuffer(jsonData))
    if err != nil {
        log.Printf("Error creating request: %v", err)
        return
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Error sending results to manager: %v", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("Manager returned status code: %d", resp.StatusCode)
    }
}