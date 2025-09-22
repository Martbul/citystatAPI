package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
)

func main() {
    fmt.Println("🔥 Starting load test to trigger alerts...")
    
    baseURL := "http://localhost:3333"
    
    // Create high error rate to trigger alert
    var wg sync.WaitGroup
    
    fmt.Println("🚨 Generating high error rate (should trigger alert)...")
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            http.Get(baseURL + "/api/nonexistent-endpoint")
        }()
    }
    
    wg.Wait()
    fmt.Println("✅ High error rate generated - check your alerts!")
    
    // Generate high response times
    fmt.Println("🐌 Generating slow responses (should trigger latency alert)...")
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            client := &http.Client{Timeout: 30 * time.Second}
            req, _ := http.NewRequest("GET", baseURL+"/api/users/sameCity", nil)
            req.Header.Set("Authorization", "Bearer test-token")
            client.Do(req)
        }()
        
        time.Sleep(100 * time.Millisecond)
    }
    
    wg.Wait()
    fmt.Println("✅ Load test completed - monitor your dashboards!")
}