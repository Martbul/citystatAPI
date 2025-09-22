package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisRateLimiter implements distributed rate limiting using Redis
type RedisRateLimiter struct {
    client       *redis.Client
    defaultLimit int
    premiumLimit int
    window       time.Duration
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(client *redis.Client, defaultLimit, premiumLimit int, window time.Duration) *RedisRateLimiter {
    return &RedisRateLimiter{
        client:       client,
        defaultLimit: defaultLimit,
        premiumLimit: premiumLimit,
        window:       window,
    }
}

// CheckRateLimit checks if request is within rate limit using sliding window
func (r *RedisRateLimiter) CheckRateLimit(ctx context.Context, identifier string, isPremium bool) (bool, error) {
    limit := r.defaultLimit
    if isPremium {
        limit = r.premiumLimit
    }

    now := time.Now()
    windowStart := now.Add(-r.window)
    
    pipe := r.client.Pipeline()
    
    // Remove old entries
    pipe.ZRemRangeByScore(ctx, identifier, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
    
    // Count current requests in window
    pipe.ZCard(ctx, identifier)
    
    // Add current request
    pipe.ZAdd(ctx, identifier, &redis.Z{
        Score:  float64(now.UnixNano()),
        Member: fmt.Sprintf("%d", now.UnixNano()),
    })
    
    // Set expiration
    pipe.Expire(ctx, identifier, r.window*2)
    
    results, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    
    // Get count from second command
    count := results[1].(*redis.IntCmd).Val()
    
    return count < int64(limit), nil
}

// RedisRateLimitMiddleware creates Redis-based rate limiting middleware
func RedisRateLimitMiddleware(rl *RedisRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            identifier := getIdentifier(r)
            isPremium := checkUserPremiumStatus(r)
            
            allowed, err := rl.CheckRateLimit(r.Context(), identifier, isPremium)
            if err != nil {
                // If Redis fails, log error but allow request (fallback mode)
                log	.Printf("Redis rate limit error: %v", err)
                next.ServeHTTP(w, r)
                return
            }
            
            if !allowed {
                //! add this rate limitert hit tp the metrics
                // telemetry.RecordRateLimitHit(r.Context())
                
                limit := rl.defaultLimit
                if isPremium {
                    limit = rl.premiumLimit
                }
                
                w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
                w.Header().Set("X-RateLimit-Remaining", "0")
                w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(rl.window).Unix(), 10))
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                
                errorResponse := map[string]interface{}{
                    "error":      "Rate limit exceeded",
                    "message":    "Too many requests. Please try again later.",
                    "retryAfter": int(rl.window.Seconds()),
                }
                
                json.NewEncoder(w).Encode(errorResponse)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}


func getIdentifier(r *http.Request) string {
    // First try to get user ID from context (set by ClerkMiddleware)
    if userID, ok := r.Context().Value("userID").(string); ok && userID != "" {
        return "user:" + userID
    }
    
    // Fall back to IP address
    return "ip:" + getIP(r)
}



// getIP extracts IP from request
func getIP(r *http.Request) string {
    // Check X-Forwarded-For header first (for load balancers/proxies)
    forwarded := r.Header.Get("X-Forwarded-For")
    if forwarded != "" {
        return forwarded
    }
    
    // Check X-Real-IP header
    realIP := r.Header.Get("X-Real-IP")
    if realIP != "" {
        return realIP
    }
    
    // Fall back to RemoteAddr
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    
    return ip
}

// checkUserPremiumStatus checks if user has premium status
func checkUserPremiumStatus(r *http.Request) bool {
    // You can implement this based on your user service
    // For now, return false. You can enhance this later.
    userID, ok := r.Context().Value("userID").(string)
    if !ok {
        return false
    }
    
    // TODO: Check user's subscription status from your database
    // Example:
    // user, err := userService.GetUser(userID)
    // return user.IsPremium || user.IsEnterprise
    
    _ = userID // Suppress unused variable warning
    return false
}


// getRateLimitHeader returns appropriate header value based on user tier
func getRateLimitHeader(isPremium bool, headerType string) string {
    if isPremium {
        switch headerType {
        case "limit":
            return "1000" // Premium limit per minute
        default:
            return "1000"
        }
    }
    return "100" // Default limit per minute
}