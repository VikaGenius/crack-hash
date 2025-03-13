package services

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"

	"worker/internal/models"
)

type TaskService struct {
    managerURL string
}

func NewTaskService(managerURL string) *TaskService {
    return &TaskService{managerURL: managerURL}
}

func (s *TaskService) ProcessTask(req models.CrackHashManagerRequest) []string {
    // вычисляем объем работы для текущего воркера
    alphabetSize := len(req.Alphabet)
    totalCombinations := totalCombinations(alphabetSize, req.MaxLength)
    partSize := int(math.Ceil(float64(totalCombinations) / float64(req.PartCount)))
    start := req.PartNumber * partSize
    end := min(start+partSize, totalCombinations)

    fmt.Println("Начинаю свою работу")
    return generateAndCheckCombinations(req.Alphabet, req.MaxLength, start, end, req.Hash)
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

    // отправка PATCH-запроса менеджеру
    req, err := http.NewRequest(http.MethodPatch, 
                                s.managerURL+"/internal/api/manager/hash/crack/request", 
                                bytes.NewBuffer(jsonData))
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

        for i := 0; i < numCombinations; i++ {
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
    for i := 0; i < length; i++ {
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