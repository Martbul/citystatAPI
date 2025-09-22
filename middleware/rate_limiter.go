package middleware

import (
    "encoding/json"
    "net/http"
    "strconv"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

// Client holds rate limiter and last seen time for each identifier
type Client struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

// RateLimiter manages multiple clients with different tiers
type RateLimiter struct {
    clients map[string]*Client
    mu      sync.RWMutex
    
    // Default limits
    defaultRPS   rate.Limit
    defaultBurst int
    
    // User tier limits (since you use Clerk auth)
    premiumRPS   rate.Limit
    premiumBurst int
}

// RateLimitConfig holds configuration for different rate limits
type RateLimitConfig struct {
    DefaultRPS   rate.Limit
    DefaultBurst int
    PremiumRPS   rate.Limit
    PremiumBurst int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
    rl := &RateLimiter{
        clients:      make(map[string]*Client),
        defaultRPS:   config.DefaultRPS,
        defaultBurst: config.DefaultBurst,
        premiumRPS:   config.PremiumRPS,
        premiumBurst: config.PremiumBurst,
    }
    
    // Clean up old clients every 5 minutes
    go rl.cleanupClients()
    
    return rl
}

// GetLimiter returns rate limiter for specific identifier
func (rl *RateLimiter) GetLimiter(identifier string, isPremium bool) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    client, exists := rl.clients[identifier]
    if !exists {
        var rps rate.Limit
        var burst int
        
        if isPremium {
            rps = rl.premiumRPS
            burst = rl.premiumBurst
        } else {
            rps = rl.defaultRPS
            burst = rl.defaultBurst
        }
        
        client = &Client{
            limiter: rate.NewLimiter(rps, burst),
        }
        rl.clients[identifier] = client
    }
    
    client.lastSeen = time.Now()
    return client.limiter
}

// cleanupClients removes inactive clients
func (rl *RateLimiter) cleanupClients() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        rl.mu.Lock()
        for identifier, client := range rl.clients {
            if time.Since(client.lastSeen) > 10*time.Minute {
                delete(rl.clients, identifier)
            }
        }
        rl.mu.Unlock()
    }
}



// RateLimitMiddleware creates the rate limiting middleware
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            identifier := getIdentifier(r)
            isPremium := checkUserPremiumStatus(r)
            
            limiter := rl.GetLimiter(identifier, isPremium)
            
            if !limiter.Allow() {
                // Record rate limit hit for monitoring
				                //! add this rate limitert hit tp the metrics

                // telemetry.RecordRateLimitHit(r.Context())
                
                // Set rate limit headers
                w.Header().Set("X-RateLimit-Limit", getRateLimitHeader(isPremium, "limit"))
                w.Header().Set("X-RateLimit-Remaining", "0")
                w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
                w.Header().Set("Retry-After", "60")
                
                // Return JSON error response
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                
                errorResponse := map[string]interface{}{
                    "error":   "Rate limit exceeded",
                    "message": "Too many requests. Please try again later.",
                    "retryAfter": 60,
                }
                
                json.NewEncoder(w).Encode(errorResponse)
                return
            }
            
            // Add rate limit headers to successful requests
            w.Header().Set("X-RateLimit-Limit", getRateLimitHeader(isPremium, "limit"))
            
            next.ServeHTTP(w, r)
        })
    }
}
