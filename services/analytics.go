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
	"time"
)

type AnalyticsService struct {
	client *db.PrismaClient
}

func NewAnalyticsService(client *db.PrismaClient) *AnalyticsService {
	return &AnalyticsService{client: client}
}

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

// MonthlyIntervalData represents the data structure for chart visualization
type MonthlyIntervalData struct {
	CurrentMonth  MonthIntervals `json:"currentMonth"`
	PreviousMonth MonthIntervals `json:"previousMonth"`
}

type MonthIntervals struct {
	Month     string         `json:"month"`     // "2024-09" format
	MonthName string         `json:"monthName"` // "September 2024"
	Intervals map[string]int `json:"intervals"` // interval -> count
	Total     int            `json:"total"`
}

type IntervalRange struct {
	Start int
	End   int
	Label string
}

func (s *AnalyticsService) GetMainRadarChartData(ctx context.Context, clerkUserID string) (*MonthlyIntervalData, error) {

	// Define the intervals
	intervals := []IntervalRange{
		{Start: 1, End: 6, Label: "1-6"},
		{Start: 7, End: 11, Label: "7-11"},
		{Start: 12, End: 16, Label: "12-16"},
		{Start: 17, End: 21, Label: "17-21"},
		{Start: 22, End: 26, Label: "22-26"},
		{Start: 27, End: 31, Label: "27-31"},
	}

	// Calculate date range (past 2 months)
	now := time.Now()
	startDate := now.AddDate(0, -2, 0).Truncate(24 * time.Hour)

	// Get current and previous month info
	currentMonth := now.AddDate(0, -1, 0)  // Last month
	previousMonth := now.AddDate(0, -2, 0) // 2 months ago

	// Query visited streets for the past 2 months
	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
		db.VisitedStreet.CreatedAt.Gte(startDate),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets: %w", err)
	}

	if visitedStreets == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Initialize result structure
	result := &MonthlyIntervalData{
		CurrentMonth: MonthIntervals{
			Month:     currentMonth.Format("2006-01"),
			MonthName: currentMonth.Format("January 2006"),
			Intervals: make(map[string]int),
			Total:     0,
		},
		PreviousMonth: MonthIntervals{
			Month:     previousMonth.Format("2006-01"),
			MonthName: previousMonth.Format("January 2006"),
			Total:     0,
			Intervals: make(map[string]int),
		},
	}

	// Initialize all intervals with 0
	for _, interval := range intervals {
		result.CurrentMonth.Intervals[interval.Label] = 0
		result.PreviousMonth.Intervals[interval.Label] = 0
	}

	// Process visited streets
	for _, visitedStreet := range visitedStreets {
		visitDate := visitedStreet.CreatedAt
		day := visitDate.Day()

		// Determine which interval this day falls into
		intervalLabel := getIntervalLabel(day, intervals)

		// Determine which month this belongs to
		if isSameMonth(visitDate, currentMonth) {
			result.CurrentMonth.Intervals[intervalLabel]++
			result.CurrentMonth.Total++
		} else if isSameMonth(visitDate, previousMonth) {
			result.PreviousMonth.Intervals[intervalLabel]++
			result.PreviousMonth.Total++
		}
	}

	return result, nil
}

// Helper function to determine which interval a day belongs to
func getIntervalLabel(day int, intervals []IntervalRange) string {
	for _, interval := range intervals {
		if day >= interval.Start && day <= interval.End {
			return interval.Label
		}
	}
	return "unknown" // shouldn't happen for valid days 1-31
}

// Helper function to check if two dates are in the same month
func isSameMonth(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.Month() == date2.Month()
}

// Alternative version if you want more detailed breakdown with specific days
type DetailedMonthlyData struct {
	CurrentMonth  DetailedMonth `json:"current_month"`
	PreviousMonth DetailedMonth `json:"previous_month"`
}

type DetailedMonth struct {
	Month     string                    `json:"month"`
	MonthName string                    `json:"month_name"`
	Intervals map[string]IntervalDetail `json:"intervals"`
	Total     int                       `json:"total"`
}

type IntervalDetail struct {
	Count int            `json:"count"`
	Days  map[string]int `json:"days"` // day -> count
	Range string         `json:"range"`
}

func (s *AnalyticsService) GetMainRadarChartDataDetailed(ctx context.Context, clerkUserID string) (*DetailedMonthlyData, error) {
	intervals := []IntervalRange{
		{Start: 1, End: 6, Label: "1-6"},
		{Start: 7, End: 11, Label: "7-11"},
		{Start: 12, End: 16, Label: "12-16"},
		{Start: 17, End: 21, Label: "17-21"},
		{Start: 22, End: 26, Label: "22-26"},
		{Start: 27, End: 31, Label: "27-31"},
	}

	now := time.Now()
	startDate := now.AddDate(0, -2, 0).Truncate(24 * time.Hour)
	// Get current and previous month info

	currentMonth := now.AddDate(0, -1, 0)  // Last month
	previousMonth := now.AddDate(0, -2, 0) // 2 months ago

	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
		db.VisitedStreet.CreatedAt.Gte(startDate),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets: %w", err)
	}

	if visitedStreets == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Initialize detailed result structure
	result := &DetailedMonthlyData{
		CurrentMonth: DetailedMonth{
			Month:     currentMonth.Format("2006-01"),
			MonthName: currentMonth.Format("January 2006"),
			Intervals: make(map[string]IntervalDetail),
			Total:     0,
		},
		PreviousMonth: DetailedMonth{
			Month:     previousMonth.Format("2006-01"),
			MonthName: previousMonth.Format("January 2006"),
			Intervals: make(map[string]IntervalDetail),
			Total:     0,
		},
	}

	// Initialize intervals
	for _, interval := range intervals {
		result.CurrentMonth.Intervals[interval.Label] = IntervalDetail{
			Count: 0,
			Days:  make(map[string]int),
			Range: interval.Label,
		}
		result.PreviousMonth.Intervals[interval.Label] = IntervalDetail{
			Count: 0,
			Days:  make(map[string]int),
			Range: interval.Label,
		}
	}

	// Process visited streets
	for _, visitedStreet := range visitedStreets {
		visitDate := visitedStreet.CreatedAt
		day := visitDate.Day()
		dayStr := fmt.Sprintf("%d", day)

		intervalLabel := getIntervalLabel(day, intervals)

		if isSameMonth(visitDate, currentMonth) {
			detail := result.CurrentMonth.Intervals[intervalLabel]
			detail.Count++
			detail.Days[dayStr]++
			result.CurrentMonth.Intervals[intervalLabel] = detail
			result.CurrentMonth.Total++
		} else if isSameMonth(visitDate, previousMonth) {
			detail := result.PreviousMonth.Intervals[intervalLabel]
			detail.Count++
			detail.Days[dayStr]++
			result.PreviousMonth.Intervals[intervalLabel] = detail
			result.PreviousMonth.Total++
		}
	}

	return result, nil
}
