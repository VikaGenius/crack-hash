package models

type StartWorkResponse struct {
	RequestId string `json:"requestId"`
}

type CrackStatusResponse struct {
    Status    string   `json:"status"`
    Data      []string `json:"data"`
    Progress  string   `json:"progress"`
}

type CrackHashWorkerResponse struct {
    RequestId  string   `json:"requestId"`
    PartNumber int      `json:"partNumber"`
    Answers    []string `json:"answers"`
}