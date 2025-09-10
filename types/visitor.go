package types

import "github.com/shopspring/decimal"

type VisitedStreet struct {
	StreetID         string  `json:"street_id"`
	StreetName       string  `json:"street_name"`
	EntryLat         float64 `json:"entry_lat,omitempty"`
	EntryLng         float64 `json:"entry_lng,omitempty"`
	Duration         *int    `json:"duration,omitempty"` // in seconds, can be nil
	VisitCount       int     `json:"visit_count"`        // total number of visits
	FirstVisit       int64   `json:"first_visit"`        // unix timestamp
	LastVisit        int64   `json:"last_visit"`         // unix timestamp
	TotalTimeSpent   int     `json:"total_time_spent"`   // seconds
	AverageTimeSpent int     `json:"average_time_spent"` // seconds
}

type SaveLocationPermitionRequest struct {
	HasLocationPermission bool `json:"hasLocationPermission"`
}

type SaveLocationPermitionResponse struct {
	Success bool `json:"success"`
}

type SaveVisitedStreetsRequest struct {
	SessionID      string                 `json:"sessionId"`
	VisitedStreets []VisitedStreetRequest `json:"visitedStreets"`
}

type VisitedStreetRequest struct {
	StreetID        string  `json:"streetId"`
	StreetName      string  `json:"streetName"`
	EntryTimestamp  int64   `json:"entryTimestamp"`
	ExitTimestamp   *int64  `json:"exitTimestamp,omitempty"`
	DurationSeconds *int    `json:"durationSeconds,omitempty"`
	EntryLatitude   float64 `json:"entryLatitude"`
	EntryLongitude  float64 `json:"entryLongitude"`
}

type GetVisitedStreetsResponse struct {
	Data    []VisitedStreet `json:"data"`
	Message string          `json:"message"`
	Status  string          `json:"status"`
}

// VisitedStreetsResponse is the main response type for GetVisitedStreetsWithDetails
type VisitedStreetsResponse struct {
	Status string                 `json:"status"`
	Data   []VisitedStreetDetails `json:"data"`
	Count  int                    `json:"count"`
}

// VisitedStreetDetails combines individual visit details with aggregated statistics
type VisitedStreetDetails struct {
	// Core identifiers
	StreetID   string  `json:"streetId"`
	StreetName string  `json:"streetName"`
	SessionID  *string `json:"sessionId,omitempty"`

	// Latest visit details
	EntryTimestamp  int64            `json:"entryTimestamp"`
	ExitTimestamp   *int64           `json:"exitTimestamp,omitempty"`
	DurationSeconds *int             `json:"durationSeconds,omitempty"`
	EntryLatitude   *decimal.Decimal `json:"entryLatitude,omitempty"`
	EntryLongitude  *decimal.Decimal `json:"entryLongitude,omitempty"`

	// Aggregated statistics
	VisitCount       int64 `json:"visitCount"`
	FirstVisit       int64 `json:"firstVisit"`
	LastVisit        int64 `json:"lastVisit"`
	TotalTimeSpent   int64 `json:"totalTimeSpent"`   // in seconds
	AverageTimeSpent int64 `json:"averageTimeSpent"` // in seconds
}