package models

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CrackHashRequest struct {
    Hash      string `json:"hash"`
    MaxLength int    `json:"maxLength"`
}

type CrackHashManagerRequest struct {
    RequestId  string   `json:"requestId"`
    PartNumber int      `json:"partNumber"`
    PartCount  int      `json:"partCount"`
    Hash       string   `json:"hash"`
    MaxLength  int      `json:"maxLength"`
    Alphabet   []string `json:"alphabet"`
}

type CrackRequest struct {
	ID        		   primitive.ObjectID `bson:"_id,omitempty"`
	RequestID 		   string             `bson:"request_id"`
	Hash      		   string             `bson:"hash"`
	MaxLength          int                `bson:"max_length"`
	Status             string             `bson:"status"`
	Data               []string           `bson:"data"`
	Workers            int                `bson:"workers"`
	Progress           int                `bson:"progress"`
	CompletedParts     map[int]bool       `bson:"completed_parts"`
	CreatedAt          time.Time          `bson:"created_at"`
	UpdatedAt 	  	   time.Time          `bson:"updated_at"`
}

type RequestPart struct {
	PartNumber int      `bson:"part_number"`
	Status     string   `bson:"status"` // "pending", "processing", "completed", "failed"
	Answers    []string `bson:"answers"`
}

func NewCrackRequest(hash string, maxLength int, workers int) *CrackRequest {
	return &CrackRequest{
		RequestID: generateRequestID(),
		Hash:      hash,
		MaxLength: maxLength,
		Status:    "IN_PROGRESS",
		Data:      make([]string, 0),
		Workers:   workers,
		Progress:  0,
		CompletedParts: make(map[int]bool),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func generateRequestID() string {
	return uuid.New().String()
}