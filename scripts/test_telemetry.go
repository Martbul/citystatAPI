package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
)

func main() {
    fmt.Println("🧪 Testing telemetry endpoints...")
    
    baseURL := "http://localhost:3333"
    
    // Test health endpoint
    testEndpoint(baseURL + "/health")
    
    // Test metrics endpoint
    testEndpoint(baseURL + "/metrics")
    
    // Generate some traffic
    generateTraffic(baseURL, 20)
    
    fmt.Println("✅ Test completed! Check your monitoring dashboards.")
}

func testEndpoint(url string) {
    fmt.Printf("Testing %s... ", url)
    
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        fmt.Printf("❌ Error: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    fmt.Printf("✅ Status: %d\n", resp.StatusCode)
}

func generateTraffic(baseURL string, count int) {
    fmt.Printf("🚀 Generating %d test requests...\n", count)
    
    var wg sync.WaitGroup
    client := &http.Client{Timeout: 10 * time.Second}
    
    endpoints := []string{"/health", "/api/user", "/metrics"}
    
    for i := 0; i < count; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            endpoint := endpoints[i%len(endpoints)]
            url := baseURL + endpoint
            
            req, _ := http.NewRequest("GET", url, nil)
            if endpoint != "/health" && endpoint != "/metrics" {
                req.Header.Set("Authorization", "Bearer test-token")
            }
            
            resp, err := client.Do(req)
            if err != nil {
                return
            }
            resp.Body.Close()
            
            if i%5 == 0 {
                fmt.Printf("📊 Completed %d/%d requests\n", i+1, count)
            }
        }(i)
        
        time.Sleep(100 * time.Millisecond)
    }
    
    wg.Wait()
    fmt.Println("✅ Traffic generation completed")
}
