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

type VisitedStreetResponse struct {
	SessionID       string   `json:"session_id"`
	StreetID        string   `json:"street_id"`
	StreetName      string   `json:"street_name"`
	EntryTimestamp  int64    `json:"entry_timestamp"`
	EntryLatitude   float64  `json:"entry_latitude"`
	EntryLongitude  float64  `json:"entry_longitude"`
	ExitTimestamp   *int64   `json:"exit_timestamp,omitempty"`
	DurationSeconds *int64   `json:"duration_seconds,omitempty"`
}
