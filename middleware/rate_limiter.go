package middleware


import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// RateLimiter represents our advanced rate limiting system
type RateLimiter struct {
	redisClient    *redis.Client
	rules          map[string]*LimitRule
	globalLimits   *GlobalLimits
	userTiers      map[string]UserTier
	adaptiveMode   bool
	systemLoad     *SystemLoad
	mu             sync.RWMutex
}

// LimitRule defines rate limiting rules for specific endpoints
type LimitRule struct {
	Requests     int           `json:"requests"`     // Number of requests allowed
	Window       time.Duration `json:"window"`       // Time window
	BurstLimit   int           `json:"burstLimit"`   // Burst allowance
	Priority     Priority      `json:"priority"`     // Endpoint priority
	Adaptive     bool          `json:"adaptive"`     // Whether to use adaptive limiting
}


// Priority levels for different endpoints
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// UserTier defines different user privilege levels
type UserTier int

const (
	TierFree UserTier = iota
	TierPremium
	TierEnterprise
	TierAdmin
)

// GlobalLimits defines system-wide limits
type GlobalLimits struct {
	MaxConcurrentConnections int     `json:"maxConcurrentConnections"`
	SystemLoadThreshold      float64 `json:"systemLoadThreshold"`
	EmergencyMode            bool    `json:"emergencyMode"`
}

// SystemLoad tracks current system performance
type SystemLoad struct {
	CPUUsage    float64   `json:"cpuUsage"`
	MemoryUsage float64   `json:"memoryUsage"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// RateLimitResult contains the result of rate limit check
type RateLimitResult struct {
	Allowed         bool          `json:"allowed"`
	Remaining       int           `json:"remaining"`
	ResetTime       time.Time     `json:"resetTime"`
	RetryAfter      time.Duration `json:"retryAfter"`
	Reason          string        `json:"reason"`
	AdaptiveApplied bool          `json:"adaptiveApplied"`
}

// NewRateLimiter creates a new advanced rate limiter
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	rl := &RateLimiter{
		redisClient:  redisClient,
		rules:        make(map[string]*LimitRule),
		userTiers:    make(map[string]UserTier),
		adaptiveMode: true,
		systemLoad: &SystemLoad{
			LastUpdated: time.Now(),
		},
		globalLimits: &GlobalLimits{
			MaxConcurrentConnections: 10000,
			SystemLoadThreshold:      0.8,
			EmergencyMode:           false,
		},
	}

	// Initialize default rules
	rl.initializeDefaultRules()
	
	// Start background monitoring
	go rl.monitorSystemLoad()
	go rl.cleanupExpiredKeys()

	return rl
}

// initializeDefaultRules sets up default rate limiting rules
func (rl *RateLimiter) initializeDefaultRules() {
	// Authentication endpoints - stricter limits
	rl.rules["/webhooks"] = &LimitRule{
		Requests:   100,
		Window:     time.Hour,
		BurstLimit: 5,
		Priority:   PriorityCritical,
		Adaptive:   false,
	}

	// User profile endpoints - medium limits
	rl.rules["/api/user"] = &LimitRule{
		Requests:   1000,
		Window:     time.Hour,
		BurstLimit: 20,
		Priority:   PriorityMedium,
		Adaptive:   true,
	}

	// Search endpoints - higher limits but adaptive
	rl.rules["/api/users/search"] = &LimitRule{
		Requests:   500,
		Window:     time.Hour,
		BurstLimit: 10,
		Priority:   PriorityMedium,
		Adaptive:   true,
	}

	// Location tracking - highest limits
	rl.rules["/api/visitor/streets"] = &LimitRule{
		Requests:   5000,
		Window:     time.Hour,
		BurstLimit: 100,
		Priority:   PriorityHigh,
		Adaptive:   true,
	}

	// Friend operations - medium limits
	rl.rules["/api/friends"] = &LimitRule{
		Requests:   2000,
		Window:     time.Hour,
		BurstLimit: 30,
		Priority:   PriorityMedium,
		Adaptive:   true,
	}

	// Settings - lower limits
	rl.rules["/api/settings"] = &LimitRule{
		Requests:   200,
		Window:     time.Hour,
		BurstLimit: 5,
		Priority:   PriorityLow,
		Adaptive:   false,
	}

	// Ranking/Leaderboard - medium limits
	rl.rules["/api/rank"] = &LimitRule{
		Requests:   1500,
		Window:     time.Hour,
		BurstLimit: 25,
		Priority:   PriorityMedium,
		Adaptive:   true,
	}
}

// SmartRateLimiter returns the middleware function
func (rl *RateLimiter) SmartRateLimiter() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user information
			userID := rl.getUserID(r)
			clientIP := rl.getClientIP(r)
			endpoint := rl.normalizeEndpoint(r.URL.Path)

			// Check global limits first
			if !rl.checkGlobalLimits(r) {
				rl.handleRateLimitExceeded(w, &RateLimitResult{
					Allowed: false,
					Reason:  "System overloaded - emergency mode active",
				})
				return
			}

			// Perform rate limit check
			result := rl.checkRateLimit(r.Context(), userID, clientIP, endpoint, r.Method)
			
			// Add rate limit headers
			rl.addRateLimitHeaders(w, result)

			if !result.Allowed {
				rl.handleRateLimitExceeded(w, result)
				return
			}

			// Log successful request for analytics
			go rl.logRequest(userID, clientIP, endpoint, r.Method)

			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit performs the core rate limiting logic
func (rl *RateLimiter) checkRateLimit(ctx context.Context, userID, clientIP, endpoint, method string) *RateLimitResult {
	// Get applicable rule
	rule := rl.getApplicableRule(endpoint, method)
	if rule == nil {
		return &RateLimitResult{Allowed: true}
	}

	// Get user tier for multiplier
	tier := rl.getUserTier(userID)
	multiplier := rl.getTierMultiplier(tier)

	// Apply adaptive adjustments
	adaptiveRule := rule
	if rule.Adaptive && rl.adaptiveMode {
		adaptiveRule = rl.applyAdaptiveAdjustments(rule)
	}

	// Check multiple rate limit keys
	keys := rl.generateLimitKeys(userID, clientIP, endpoint, method)
	
	for _, key := range keys {
		result := rl.checkSingleLimit(ctx, key, adaptiveRule, multiplier)
		if !result.Allowed {
			return result
		}
	}

	return &RateLimitResult{
		Allowed:         true,
		AdaptiveApplied: rule.Adaptive && rl.adaptiveMode,
	}
}

// checkSingleLimit checks rate limit for a single key using sliding window
func (rl *RateLimiter) checkSingleLimit(ctx context.Context, key string, rule *LimitRule, multiplier float64) *RateLimitResult {
	now := time.Now()
	windowStart := now.Add(-rule.Window)
	
	// Use Redis sliding window algorithm
	pipe := rl.redisClient.TxPipeline()
	
	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	
	// Count current entries
	countCmd := pipe.ZCard(ctx, key)
	
	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	
	// Set expiration
	pipe.Expire(ctx, key, rule.Window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		// On Redis error, allow request but log
		fmt.Printf("Redis error in rate limiter: %v", err)
		return &RateLimitResult{Allowed: true}
	}

	currentCount := int(countCmd.Val())
	limit := int(float64(rule.Requests) * multiplier)

	if currentCount >= limit {
		return &RateLimitResult{
			Allowed:    false,
			Remaining:  0,
			ResetTime:  windowStart.Add(rule.Window),
			RetryAfter: rule.Window,
			Reason:     fmt.Sprintf("Rate limit exceeded: %d/%d requests", currentCount, limit),
		}
	}

	return &RateLimitResult{
		Allowed:   true,
		Remaining: limit - currentCount,
		ResetTime: windowStart.Add(rule.Window),
	}
}

// applyAdaptiveAdjustments modifies rate limits based on system load
func (rl *RateLimiter) applyAdaptiveAdjustments(rule *LimitRule) *LimitRule {
	rl.mu.RLock()
	load := rl.systemLoad
	rl.mu.RUnlock()

	// Create a copy to avoid modifying the original
	adaptiveRule := *rule

	// Adjust based on system load
	loadFactor := 1.0
	if load.CPUUsage > 0.8 || load.MemoryUsage > 0.8 {
		loadFactor = 0.5 // Reduce limits by 50% under high load
	} else if load.CPUUsage > 0.6 || load.MemoryUsage > 0.6 {
		loadFactor = 0.7 // Reduce limits by 30% under medium load
	}

	adaptiveRule.Requests = int(float64(adaptiveRule.Requests) * loadFactor)
	adaptiveRule.BurstLimit = int(float64(adaptiveRule.BurstLimit) * loadFactor)

	return &adaptiveRule
}

// generateLimitKeys creates different keys for different limiting strategies
func (rl *RateLimiter) generateLimitKeys(userID, clientIP, endpoint, method string) []string {
	var keys []string

	// User-based limiting (if user is authenticated)
	if userID != "" {
		keys = append(keys, fmt.Sprintf("rate_limit:user:%s:%s", userID, endpoint))
	}

	// IP-based limiting (always applied)
	keys = append(keys, fmt.Sprintf("rate_limit:ip:%s:%s", clientIP, endpoint))

	// Global endpoint limiting
	keys = append(keys, fmt.Sprintf("rate_limit:global:%s", endpoint))

	return keys
}

// getUserTier determines the user's subscription tier
func (rl *RateLimiter) getUserTier(userID string) UserTier {
	if userID == "" {
		return TierFree
	}

	rl.mu.RLock()
	tier, exists := rl.userTiers[userID]
	rl.mu.RUnlock()

	if !exists {
		// In a real implementation, you'd fetch this from your database
		// For now, default to free tier
		return TierFree
	}

	return tier
}

// getTierMultiplier returns the rate limit multiplier for each tier
func (rl *RateLimiter) getTierMultiplier(tier UserTier) float64 {
	switch tier {
	case TierFree:
		return 1.0
	case TierPremium:
		return 2.0
	case TierEnterprise:
		return 5.0
	case TierAdmin:
		return 10.0
	default:
		return 1.0
	}
}

// getApplicableRule finds the most specific rule for an endpoint
func (rl *RateLimiter) getApplicableRule(endpoint, method string) *LimitRule {
	// Check for exact match first
	if rule, exists := rl.rules[endpoint]; exists {
		return rule
	}

	// Check for pattern matches
	for pattern, rule := range rl.rules {
		if rl.matchesPattern(endpoint, pattern) {
			return rule
		}
	}

	return nil
}

// matchesPattern checks if an endpoint matches a pattern
func (rl *RateLimiter) matchesPattern(endpoint, pattern string) bool {
	// Simple pattern matching - can be enhanced with regex
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(endpoint, prefix)
	}
	
	return endpoint == pattern
}

// normalizeEndpoint normalizes the endpoint for consistent rule matching
func (rl *RateLimiter) normalizeEndpoint(path string) string {
	// Remove trailing slashes
	path = strings.TrimSuffix(path, "/")
	
	// Replace dynamic segments (like user IDs) with placeholders
	// Example: /api/users/12345 -> /api/users/*
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		// If segment looks like an ID (all digits or UUID pattern)
		if rl.isLikelyID(segment) {
			segments[i] = "*"
		}
	}
	
	return strings.Join(segments, "/")
}

// isLikelyID checks if a segment looks like an ID
func (rl *RateLimiter) isLikelyID(segment string) bool {
	// Check if it's all digits
	if _, err := strconv.Atoi(segment); err == nil && len(segment) > 3 {
		return true
	}
	
	// Check if it's a UUID pattern
	if len(segment) == 36 && strings.Count(segment, "-") == 4 {
		return true
	}
	
	// Check if it's a Clerk user ID pattern
	if strings.HasPrefix(segment, "user_") && len(segment) > 10 {
		return true
	}
	
	return false
}

// checkGlobalLimits checks system-wide limits
func (rl *RateLimiter) checkGlobalLimits(r *http.Request) bool {
	rl.mu.RLock()
	limits := rl.globalLimits
	load := rl.systemLoad
	rl.mu.RUnlock()

	// Check if system is in emergency mode
	if limits.EmergencyMode {
		return false
	}

	// Check system load
	if load.CPUUsage > limits.SystemLoadThreshold || 
	   load.MemoryUsage > limits.SystemLoadThreshold {
		return false
	}

	return true
}

// addRateLimitHeaders adds standard rate limit headers to the response
func (rl *RateLimiter) addRateLimitHeaders(w http.ResponseWriter, result *RateLimitResult) {
	if result.Remaining > 0 {
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	}
	
	if !result.ResetTime.IsZero() {
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	}
	
	if result.AdaptiveApplied {
		w.Header().Set("X-RateLimit-Adaptive", "true")
	}
	
	if result.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	}
}

// handleRateLimitExceeded sends rate limit exceeded response
func (rl *RateLimiter) handleRateLimitExceeded(w http.ResponseWriter, result *RateLimitResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	
	response := map[string]interface{}{
		"error":   "Rate limit exceeded",
		"message": result.Reason,
	}
	
	if result.RetryAfter > 0 {
		response["retryAfter"] = int(result.RetryAfter.Seconds())
	}
	
	json.NewEncoder(w).Encode(response)
}

// getUserID extracts user ID from request (from your existing auth middleware)
func (rl *RateLimiter) getUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// getClientIP extracts the real client IP
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for load balancers/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to remote address
	ip := strings.Split(r.RemoteAddr, ":")[0]
	return ip
}

// monitorSystemLoad runs in background to monitor system performance
func (rl *RateLimiter) monitorSystemLoad() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// In a real implementation, you'd get actual system metrics
		// For now, we'll simulate some values
		rl.mu.Lock()
		rl.systemLoad.CPUUsage = rl.getCurrentCPUUsage()
		rl.systemLoad.MemoryUsage = rl.getCurrentMemoryUsage()
		rl.systemLoad.LastUpdated = time.Now()
		
		// Update emergency mode based on load
		if rl.systemLoad.CPUUsage > 0.95 || rl.systemLoad.MemoryUsage > 0.95 {
			rl.globalLimits.EmergencyMode = true
		} else if rl.systemLoad.CPUUsage < 0.7 && rl.systemLoad.MemoryUsage < 0.7 {
			rl.globalLimits.EmergencyMode = false
		}
		rl.mu.Unlock()
	}
}

// getCurrentCPUUsage gets current CPU usage (implement with actual system monitoring)
func (rl *RateLimiter) getCurrentCPUUsage() float64 {
	// Placeholder - implement with actual CPU monitoring
	return 0.3 // 30% usage
}

// getCurrentMemoryUsage gets current memory usage (implement with actual system monitoring)
func (rl *RateLimiter) getCurrentMemoryUsage() float64 {
	// Placeholder - implement with actual memory monitoring
	return 0.4 // 40% usage
}

// cleanupExpiredKeys removes expired rate limit keys from Redis
func (rl *RateLimiter) cleanupExpiredKeys() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		
		// Get all rate limit keys
		keys, err := rl.redisClient.Keys(ctx, "rate_limit:*").Result()
		if err != nil {
			continue
		}

		// Check and clean expired keys
		for _, key := range keys {
			ttl := rl.redisClient.TTL(ctx, key).Val()
			if ttl < 0 {
				rl.redisClient.Del(ctx, key)
			}
		}
	}
}

// logRequest logs successful requests for analytics
func (rl *RateLimiter) logRequest(userID, clientIP, endpoint, method string) {
	// Implement request logging for analytics
	// This could go to a separate analytics system
	fmt.Printf("Request logged: user=%s, ip=%s, endpoint=%s, method=%s, time=%s\n",
		userID, clientIP, endpoint, method, time.Now().Format(time.RFC3339))
}

// SetUserTier allows updating user tiers (call this when user upgrades/downgrades)
func (rl *RateLimiter) SetUserTier(userID string, tier UserTier) {
	rl.mu.Lock()
	rl.userTiers[userID] = tier
	rl.mu.Unlock()
}

// AddCustomRule allows adding custom rate limiting rules
func (rl *RateLimiter) AddCustomRule(endpoint string, rule *LimitRule) {
	rl.mu.Lock()
	rl.rules[endpoint] = rule
	rl.mu.Unlock()
}

// GetStats returns current rate limiting statistics
func (rl *RateLimiter) GetStats(ctx context.Context) *RateLimitStats {
	rl.mu.RLock()
	load := *rl.systemLoad
	emergencyMode := rl.globalLimits.EmergencyMode
	rl.mu.RUnlock()

	// Get Redis key count
	keyCount, _ := rl.redisClient.DBSize(ctx).Result()

	return &RateLimitStats{
		SystemLoad:    load,
		EmergencyMode: emergencyMode,
		ActiveKeys:    keyCount,
		RulesCount:    len(rl.rules),
	}
}

// RateLimitStats represents current system statistics
type RateLimitStats struct {
	SystemLoad    SystemLoad `json:"systemLoad"`
	EmergencyMode bool       `json:"emergencyMode"`
	ActiveKeys    int64      `json:"activeKeys"`
	RulesCount    int        `json:"rulesCount"`
}