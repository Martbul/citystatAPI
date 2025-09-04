package types

type OverpassResponse struct {
	Version   float64     `json:"version"`
	Generator string      `json:"generator"`
	Elements  []Element   `json:"elements"`
}

type Element struct {
	Type     string    `json:"type"`
	ID       int64     `json:"id"`
	Nodes    []int64   `json:"nodes,omitempty"`
	Geometry []Node    `json:"geometry,omitempty"`
	Tags     Tags      `json:"tags"`
}

type Node struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Tags struct {
	Highway string `json:"highway"`
	Name    string `json:"name"`
}

// CityStats holds the calculated statistics
type City2MainStats struct {
	City           string  `json:"city"`
	TotalStreets   int     `json:"total_streets"`
	TotalKilometers float64 `json:"total_kilometers"`
	StreetTypes    map[string]int `json:"street_types"`
}