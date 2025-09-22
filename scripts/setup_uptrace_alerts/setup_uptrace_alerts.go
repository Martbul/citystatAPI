package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

type UptraceAlert struct {
    Name        string            `json:"name"`
    Query       string            `json:"query"`
    Threshold   float64           `json:"threshold"`
    Duration    string            `json:"duration"`
    Severity    string            `json:"severity"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
}

func main() {
    uptraceDSN := os.Getenv("UPTRACE_DSN")
    if uptraceDSN == "" {
        fmt.Println("UPTRACE_DSN environment variable required")
        return
    }
    
    alerts := []UptraceAlert{
        {
            Name:      "High Error Rate",
            Query:     "rate(http_requests_total{status_class=\"4xx\" or status_class=\"5xx\"}[5m]) / rate(http_requests_total[5m])",
            Threshold: 0.05,
            Duration:  "5m",
            Severity:  "critical",
            Labels: map[string]string{
                "team":    "backend",
                "service": "citystat-api",
            },
            Annotations: map[string]string{
                "summary":     "High error rate detected",
                "description": "Error rate is above 5% for 5 minutes",
            },
        },
        // Add more alerts...
    }
    
    for _, alert := range alerts {
        if err := createUptraceAlert(alert); err != nil {
            fmt.Printf("❌ Failed to create alert %s: %v\n", alert.Name, err)
        } else {
            fmt.Printf("✅ Created alert: %s\n", alert.Name)
        }
    }
}

func createUptraceAlert(alert UptraceAlert) error {
    // This is pseudo-code - actual Uptrace API implementation may differ
    jsonData, _ := json.Marshal(alert)
    
    resp, err := http.Post(
        "https://api.uptrace.dev/v1/alerts", // Replace with actual Uptrace API endpoint
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 201 {
        return fmt.Errorf("API returned status: %d", resp.StatusCode)
    }
    
    return nil
}