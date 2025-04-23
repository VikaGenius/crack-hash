package models

type TaskMessage struct {
    RequestId  string   `json:"request_id"`
    PartNumber int      `json:"part_number"`
    PartCount  int      `json:"part_count"`
    Hash       string   `json:"hash"`
    MaxLength  int      `json:"max_length"`
    Alphabet   []string `json:"alphabet"`
}

type ResultMessage struct {
    RequestId  string   `json:"request_id"`
    PartNumber int      `json:"part_number"`
    Answers    []string `json:"answers"`
}