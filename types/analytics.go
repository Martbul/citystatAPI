package types

type StreetVisitApiResponse struct {
    Status  string      `json:"status"`           // "success" or "error"
    Message *string     `json:"message,omitempty"` // optional
    Data    interface{} `json:"data,omitempty"`    // optional, can be any type
}



