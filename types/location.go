package types

type OverpassResponse struct {
	Version   float64   `json:"version"`
	Generator string    `json:"generator"`
	Elements  []Element `json:"elements"`
}

type Element struct {
	Type     string  `json:"type"`
	ID       int64   `json:"id"`
	Nodes    []int64 `json:"nodes,omitempty"`
	Geometry []Node  `json:"geometry,omitempty"`
	Tags     Tags    `json:"tags"`
}

type Node struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Tags struct {
	Highway string `json:"highway"`
	Name    string `json:"name"`
}

type City2MainStats struct {
	City                       string  `json:"city"`
	TotalStreetsCity           int     `json:"totalStreetsCity"`
	TotalKilometersCity        float64 `json:"totalKilometersCity"`
	TotalStreetsCovered        int     `json:"totalStreetsCovered"`
	TotalKilometersCovered     float64 `json:"totalKilometersCovered"`
	PercentCityStreetCouverage float64 `json:"PercentCityStreetCouverage"`

	StreetTypes map[string]int `json:"street_types"`
}
