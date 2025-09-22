package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "sync"
    "time"
)

func main() {
    fmt.Println("🧪 Testing telemetry and generating metrics...")
    
    baseURL := "http://localhost:3333"
    
    // Test basic connectivity
    if !testConnectivity(baseURL) {
        log.Fatal("❌ Cannot connect to the API")
    }
    
    // Generate various types of traffic to trigger metrics
    var wg sync.WaitGroup
    
    // Test 1: Normal requests
    wg.Add(1)
    go func() {
        defer wg.Done()
        generateNormalTraffic(baseURL, 50)
    }()
    
    // Test 2: Error conditions
    wg.Add(1)
    go func() {
        defer wg.Done()
        generateErrorTraffic(baseURL, 10)
    }()
    
    // Test 3: Slow requests
    wg.Add(1)
    go func() {
        defer wg.Done()
        generateSlowRequests(baseURL, 5)
    }()
    
    // Test 4: High frequency requests (potential rate limiting)
    wg.Add(1)
    go func() {
        defer wg.Done()
        generateHighFrequencyTraffic(baseURL, 100)
    }()
    
    wg.Wait()
    
    fmt.Println("\n✅ Test completed! Check your monitoring dashboards:")
    fmt.Printf("📊 Metrics: %s/metrics\n", baseURL)
    fmt.Printf("🏥 Health: %s/health\n", baseURL)
    fmt.Println("📈 Check Uptrace for traces and metrics")
    
    // Test metrics endpoint
    testMetricsEndpoint(baseURL)
}

func testConnectivity(baseURL string) bool {
    resp, err := http.Get(baseURL + "/health")
    if err != nil {
        fmt.Printf("❌ Health check failed: %v\n", err)
        return false
    }
    defer resp.Body.Close()
    
    fmt.Printf("✅ Health check passed: %d\n", resp.StatusCode)
    return resp.StatusCode == 200
}

func generateNormalTraffic(baseURL string, count int) {
    fmt.Printf("🚀 Generating %d normal requests...\n", count)
    
    client := &http.Client{Timeout: 10 * time.Second}
    
    for i := 0; i < count; i++ {
        // Test different endpoints
        endpoints := []string{
            "/health",
            "/api/user",
            "/api/settings",
            "/api/friends/list",
        }
        
        endpoint := endpoints[rand.Intn(len(endpoints))]
        req, _ := http.NewRequest("GET", baseURL+endpoint, nil)
        
        // Add auth header for protected endpoints
        if endpoint != "/health" {
            req.Header.Set("Authorization", "Bearer test-token-"+fmt.Sprintf("%d", i))
        }
        
        resp, err := client.Do(req)
        if err != nil {
            fmt.Printf("❌ Request to %s failed: %v\n", endpoint, err)
            continue
        }
        resp.Body.Close()
        
        if i%10 == 0 {
            fmt.Printf("📈 Completed %d/%d requests\n", i+1, count)
        }
        
        // Random delay between requests
        time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
    }
    
    fmt.Printf("✅ Normal traffic generation completed\n")
}

func generateErrorTraffic(baseURL string, count int) {
    fmt.Printf("💥 Generating %d error conditions...\n", count)
    
    client := &http.Client{Timeout: 5 * time.Second}
    
    for i := 0; i < count; i++ {
        // Test various error conditions
        errorEndpoints := []string{
            "/api/nonexistent",           // 404
            "/api/user",                  // 401 (no auth)
            "/api/users/search",          // 400 (missing params)
        }
        
        endpoint := errorEndpoints[rand.Intn(len(errorEndpoints))]
        req, _ := http.NewRequest("GET", baseURL+endpoint, nil)
        
        resp, err := client.Do(req)
        if err != nil {
            fmt.Printf("🔥 Expected error for %s: %v\n", endpoint, err)
            continue
        }
        resp.Body.Close()
        
        fmt.Printf("📊 Error response %d for %s\n", resp.StatusCode, endpoint)
        time.Sleep(200 * time.Millisecond)
    }
    
    fmt.Printf("✅ Error traffic generation completed\n")
}

func generateSlowRequests(baseURL string, count int) {
    fmt.Printf("🐌 Generating %d slow requests...\n", count)
    
    // Use a longer timeout to simulate slow requests
    client := &http.Client{Timeout: 30 * time.Second}
    
    for i := 0; i < count; i++ {
        // Make requests that might be slower
        req, _ := http.NewRequest("GET", baseURL+"/api/users/sameCity", nil)
        req.Header.Set("Authorization", "Bearer test-token")
        
        start := time.Now()
        resp, err := client.Do(req)
        duration := time.Since(start)
        
        if err != nil {
            fmt.Printf("❌ Slow request failed: %v (took %v)\n", err, duration)
            continue
        }
        resp.Body.Close()
        
        fmt.Printf("⏱️  Request took %v\n", duration)
        time.Sleep(1 * time.Second)
    }
    
    fmt.Printf("✅ Slow request generation completed\n")
}

func generateHighFrequencyTraffic(baseURL string, count int) {
    fmt.Printf("⚡ Generating %d high-frequency requests...\n", count)
    
    client := &http.Client{Timeout: 5 * time.Second}
    
    // Send requests rapidly
    for i := 0; i < count; i++ {
        go func(i int) {
            req, _ := http.NewRequest("GET", baseURL+"/health", nil)
            resp, err := client.Do(req)
            if err != nil {
                return
            }
            resp.Body.Close()
        }(i)
        
        // Very short delay between requests
        time.Sleep(10 * time.Millisecond)
    }
    
    // Wait a bit for all requests to complete
    time.Sleep(2 * time.Second)
    fmt.Printf("✅ High-frequency traffic generation completed\n")
}

func testMetricsEndpoint(baseURL string) {
    fmt.Println("\n📊 Testing metrics endpoint...")
    
    resp, err := http.Get(baseURL + "/metrics")
    if err != nil {
        fmt.Printf("❌ Metrics endpoint failed: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 200 {
        fmt.Printf("✅ Metrics endpoint responding: %d\n", resp.StatusCode)
        fmt.Println("📈 Metrics are being collected and exposed")
    } else {
        fmt.Printf("⚠️  Metrics endpoint returned: %d\n", resp.StatusCode)
    }
}

// TestUserRegistration simulates user registration for business metrics
func TestUserRegistration(baseURL string) {
    fmt.Println("👤 Testing user registration metrics...")
    
    client := &http.Client{Timeout: 10 * time.Second}
    
    // Simulate user registration
    userData := map[string]interface{}{
        "firstName": "Test",
        "lastName":  "User",
        "userName":  fmt.Sprintf("testuser_%d", time.Now().Unix()),
    }
    
    jsonData, _ := json.Marshal(userData)
    req, _ := http.NewRequest("POST", baseURL+"/api/user/details", bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer test-token")
    
    resp, err := client.Do(req)
    if err != nil {
        fmt.Printf("❌ User registration test failed: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    fmt.Printf("👤 User registration test completed: %d\n", resp.StatusCode)
}