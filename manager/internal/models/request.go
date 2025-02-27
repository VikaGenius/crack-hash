package models

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