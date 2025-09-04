package services

import (
	"citystatAPI/prisma/db"
	"citystatAPI/types"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

type AnalyticsService struct {
	client *db.PrismaClient
}

func NewAnalyticsService(client *db.PrismaClient) *AnalyticsService {
	return &AnalyticsService{client: client}
}

// ! edit the return type
func (s *AnalyticsService) GetMain2Stats(ctx context.Context, clerkUserID string) (*types.City2MainStats, error) {
	currUserCity, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Select(
		db.User.City.Field(),
	).Exec(ctx)

	if err != nil {
		fmt.Printf("Error: failed to get total user city: %v\n", err)

	}

	cityName, ok := currUserCity.City()
	if !ok {
		fmt.Printf("Error: failed to get total user city: %v\n", err)

	}

	bbox, err := getCityBoundingBox(ctx, cityName)
	if err != nil {
		return nil, fmt.Errorf("failed to get city boundaries: %w", err)
	}

	// Create Overpass query for the city
	overpassQuery := fmt.Sprintf(`
[out:json][timeout:30];
(
  way["highway"~"^(primary|secondary|tertiary|residential|trunk|motorway|unclassified|living_street|service|footway|path|cycleway|track)$"]
    (%f,%f,%f,%f);
);
out geom;
`, bbox.South, bbox.West, bbox.North, bbox.East)

	// Make request to Overpass API
	overpassData, err := queryOverpassAPI(ctx, overpassQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query Overpass API: %w", err)
	}

	// Calculate statistics
	stats := calculateStreetStats(cityName, overpassData)

	return stats, nil
}



// getCityBoundingBox fetches city boundaries from Nominatim
func getCityBoundingBox(ctx context.Context, cityName string) (*BoundingBox, error) {
	nominatimURL := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1",
		url.QueryEscape(cityName),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", nominatimURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CityStreetAnalyzer/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nominatimResults []struct {
		BoundingBox []string `json:"boundingbox"`
	}

	if err := json.Unmarshal(body, &nominatimResults); err != nil {
		return nil, err
	}

	if len(nominatimResults) == 0 {
		return nil, fmt.Errorf("city not found: %s", cityName)
	}

	bbox := nominatimResults[0].BoundingBox
	if len(bbox) != 4 {
		return nil, fmt.Errorf("invalid bounding box data")
	}

	// Parse bounding box coordinates
	south := parseFloat(bbox[0])
	north := parseFloat(bbox[1])
	west := parseFloat(bbox[2])
	east := parseFloat(bbox[3])

	return &BoundingBox{
		North: north,
		South: south,
		East:  east,
		West:  west,
	}, nil
}

// queryOverpassAPI makes the POST request to Overpass API (recreating your JS fetch)
func queryOverpassAPI(ctx context.Context, overpassQuery string) (*types.OverpassResponse, error) {
	// Prepare form data (equivalent to your JS body)
	formData := url.Values{}
	formData.Set("data", overpassQuery)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://overpass-api.de/api/interpreter",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, err
	}

	// Set headers (equivalent to your JS headers)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "CityStreetAnalyzer/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("overpass API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var overpassResp types.OverpassResponse
	if err := json.Unmarshal(body, &overpassResp); err != nil {
		return nil, fmt.Errorf("failed to parse Overpass response: %w", err)
	}

	return &overpassResp, nil
}

// calculateStreetStats processes the Overpass data and calculates statistics
func calculateStreetStats(cityName string, data *types.OverpassResponse) *types.City2MainStats {
	stats := &types.City2MainStats{
		City:        cityName,
		StreetTypes: make(map[string]int),
	}

	var totalDistance float64

	for _, element := range data.Elements {
		if element.Type == "way" && len(element.Geometry) > 1 {
			stats.TotalStreets++
			
			// Count street types
			highway := element.Tags.Highway
			if highway != "" {
				stats.StreetTypes[highway]++
			}

			// Calculate distance for this way
			wayDistance := 0.0
			for i := 0; i < len(element.Geometry)-1; i++ {
				dist := haversineDistance(
					element.Geometry[i].Lat, element.Geometry[i].Lon,
					element.Geometry[i+1].Lat, element.Geometry[i+1].Lon,
				)
				wayDistance += dist
			}
			totalDistance += wayDistance
		}
	}

	stats.TotalKilometers = totalDistance
	return stats
}

// haversineDistance calculates the distance between two points in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// Helper function to parse float from string
func parseFloat(s string) float64 {
	if val, err := fmt.Sscanf(s, "%f", new(float64)); err == nil && val == 1 {
		var result float64
		fmt.Sscanf(s, "%f", &result)
		return result
	}
	return 0.0
}



type BoundingBox struct {
	North, South, East, West float64
}