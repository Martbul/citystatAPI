package types

type StreetVisitApiResponse struct {
    Status  string      `json:"status"`           // "success" or "error"
    Message *string     `json:"message,omitempty"` // optional
    Data    interface{} `json:"data,omitempty"`    // optional, can be any type
}


type StreetStat struct {
    StreetID        string  `json:"streetId"`
    StreetName      string  `json:"streetName"`
    VisitCount      int64   `json:"visitCount"`
    FirstVisit      int64   `json:"firstVisit"`
    LastVisit       int64   `json:"lastVisit"`
    TotalTimeSpent  int64   `json:"totalTimeSpent"`
    AverageTimeSpent int64  `json:"averageTimeSpent"`
}

type SaveStreetVisitStatsRequest struct {
    StreetStats []StreetStat `json:"streetStats"`
}
