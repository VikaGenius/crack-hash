package models

type CrackHashWorkerResponse struct {
    RequestId  string   `json:"requestId"`
    PartNumber int      `json:"partNumber"`
    Answers    []string `json:"answers"`
}