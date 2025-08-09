package types

type SaveLocationPermitionRequest struct {
	HasLocationPermission bool `json:"hasLocationPermission"`
}

type SaveLocationPermitionResponse struct {
	Success bool `json:"success"`
}

//! ai generated check it
type SaveVisitedStreetsRequest struct {
    SessionID      string                   `json:"sessionId"`
    VisitedStreets []VisitedStreetRequest   `json:"visitedStreets"`
}

type VisitedStreetRequest struct {
    StreetID        string   `json:"streetId"`
    StreetName      string   `json:"streetName"`
    EntryTimestamp  int64    `json:"entryTimestamp"`
    ExitTimestamp   *int64   `json:"exitTimestamp,omitempty"`
    DurationSeconds *int     `json:"durationSeconds,omitempty"`
    EntryLatitude   float64  `json:"entryLatitude"`
    EntryLongitude  float64  `json:"entryLongitude"`
}