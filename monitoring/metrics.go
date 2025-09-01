// monitoring/metrics.go
package monitoring

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP Metrics
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ResponseSize      *prometheus.HistogramVec
	RequestsInFlight  prometheus.Gauge
	
	// Database Metrics
	DBConnectionsTotal    prometheus.Gauge
	DBConnectionsIdle     prometheus.Gauge
	DBQueriesTotal        *prometheus.CounterVec
	DBQueryDuration       *prometheus.HistogramVec
	DBConnectionErrors    *prometheus.CounterVec
	
	// Redis Metrics
	RedisConnectionsTotal prometheus.Gauge
	RedisOperationsTotal  *prometheus.CounterVec
	RedisOperationDuration *prometheus.HistogramVec
	RedisCacheHits        *prometheus.CounterVec
	
	// Rate Limiting Metrics
	RateLimitHits         *prometheus.CounterVec
	RateLimitBlocks       *prometheus.CounterVec
	AdaptiveLimitAdjustments *prometheus.CounterVec
	
	// Business Metrics
	UsersTotal            prometheus.Gauge
	ActiveUsers           prometheus.Gauge
	StreetsVisited        *prometheus.CounterVec
	PointsAwarded         *prometheus.CounterVec
	FriendRequests        *prometheus.CounterVec
	
	// System Metrics
	CPUUsage              prometheus.Gauge
	MemoryUsage           prometheus.Gauge
	GoroutinesCount       prometheus.Gauge
	GCDuration            prometheus.Histogram
	
	// Error Metrics
	ErrorsTotal           *prometheus.CounterVec
	PanicTotal            *prometheus.CounterVec
	
	// Custom Application Metrics
	UserRegistrations     *prometheus.CounterVec
	LocationUpdates       *prometheus.CounterVec
	SearchQueries         *prometheus.CounterVec
	LeaderboardRequests   *prometheus.CounterVec
}

// NewMetrics initializes all Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// HTTP Metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code", "user_tier"},
		),
		
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint", "status_code"},
		),
		
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 2, 10),
			},
			[]string{"method", "endpoint"},
		),
		
		RequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),
		
		// Database Metrics
		DBConnectionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_total",
				Help: "Total number of database connections",
			},
		),
		
		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"operation", "table"},
		),
		
		DBConnectionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_connection_errors_total",
				Help: "Total number of database connection errors",
			},
			[]string{"error_type"},
		),
		
		// Redis Metrics
		RedisConnectionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_connections_total",
				Help: "Total number of Redis connections",
			},
		),
		
		RedisOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_operations_total",
				Help: "Total number of Redis operations",
			},
			[]string{"operation", "status"},
		),
		
		RedisOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_operation_duration_seconds",
				Help:    "Redis operation duration in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"operation"},
		),
		
		RedisCacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_cache_operations_total",
				Help: "Total number of Redis cache operations",
			},
			[]string{"operation", "result"}, // hit, miss, set, delete
		),
		
		// Rate Limiting Metrics
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limit_hits_total",
				Help: "Total number of rate limit checks",
			},
			[]string{"endpoint", "user_tier", "limit_type"},
		),
		
		RateLimitBlocks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limit_blocks_total",
				Help: "Total number of rate limit blocks",
			},
			[]string{"endpoint", "user_tier", "reason"},
		),
		
		AdaptiveLimitAdjustments: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "adaptive_rate_limit_adjustments_total",
				Help: "Total number of adaptive rate limit adjustments",
			},
			[]string{"endpoint", "direction"}, // increase, decrease
		),
		
		// Business Metrics
		UsersTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "users_total",
				Help: "Total number of registered users",
			},
		),
		
		ActiveUsers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_users",
				Help: "Number of active users in last 24 hours",
			},
		),
		
		StreetsVisited: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "streets_visited_total",
				Help: "Total number of streets visited",
			},
			[]string{"city", "user_tier"},
		),
		
		PointsAwarded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "points_awarded_total",
				Help: "Total points awarded to users",
			},
			[]string{"reason", "user_tier"},
		),
		
		FriendRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "friend_requests_total",
				Help: "Total number of friend requests",
			},
			[]string{"status"}, // sent, accepted, rejected
		),
		
		// System Metrics
		CPUUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),
		
		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),
		
		GoroutinesCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "goroutines_count",
				Help: "Number of goroutines",
			},
		),
		
		GCDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gc_duration_seconds",
				Help:    "Garbage collection duration in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
		),
		
		// Error Metrics
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors",
			},
			[]string{"service", "error_type", "severity"},
		),
		
		PanicTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "panics_total",
				Help: "Total number of panics",
			},
			[]string{"service", "endpoint"},
		),
		
		// Custom Application Metrics
		UserRegistrations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"source"}, // clerk_sync, direct, invite
		),
		
		LocationUpdates: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "location_updates_total",
				Help: "Total number of location updates",
			},
			[]string{"city", "update_type"},
		),
		
		SearchQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "search_queries_total",
				Help: "Total number of search queries",
			},
			[]string{"search_type", "results_count_range"},
		),
		
		LeaderboardRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "leaderboard_requests_total",
				Help: "Total number of leaderboard requests",
			},
			[]string{"leaderboard_type"}, // global, local
		),
	}
	
	// Start system metrics collection
	go m.collectSystemMetrics()
	
	return m
}

// collectSystemMetrics runs in background to collect system metrics
func (m *Metrics) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// Collect Go runtime metrics
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		
		m.MemoryUsage.Set(float64(memStats.Alloc))
		m.GoroutinesCount.Set(float64(runtime.NumGoroutine()))
		
		// GC metrics
		if memStats.NumGC > 0 {
			// Get the most recent GC pause time
			gcPause := float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e9
			m.GCDuration.Observe(gcPause)
		}
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration, responseSize int64, userTier string) {
	statusStr := strconv.Itoa(statusCode)
	
	m.RequestsTotal.WithLabelValues(method, endpoint, statusStr, userTier).Inc()
	m.RequestDuration.WithLabelValues(method, endpoint, statusStr).Observe(duration.Seconds())
	m.ResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	
	m.DBQueriesTotal.WithLabelValues(operation, table, status).Inc()
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	
	if err != nil {
		m.DBConnectionErrors.WithLabelValues("query_error").Inc()
	}
}

// RecordRedisOperation records Redis operation metrics
func (m *Metrics) RecordRedisOperation(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	
	m.RedisOperationsTotal.WithLabelValues(operation, status).Inc()
	m.RedisOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordRateLimit records rate limiting metrics
func (m *Metrics) RecordRateLimit(endpoint, userTier, limitType string, blocked bool, reason string) {
	m.RateLimitHits.WithLabelValues(endpoint, userTier, limitType).Inc()
	
	if blocked {
		m.RateLimitBlocks.WithLabelValues(endpoint, userTier, reason).Inc()
	}
}

// RecordError records error metrics
func (m *Metrics) RecordError(service, errorType, severity string) {
	m.ErrorsTotal.WithLabelValues(service, errorType, severity).Inc()
}

// RecordBusinessMetric records various business metrics
func (m *Metrics) RecordStreetVisit(city, userTier string) {
	m.StreetsVisited.WithLabelValues(city, userTier).Inc()
}

func (m *Metrics) RecordPointsAwarded(reason, userTier string, points int) {
	m.PointsAwarded.WithLabelValues(reason, userTier).Add(float64(points))
}

func (m *Metrics) RecordUserRegistration(source string) {
	m.UserRegistrations.WithLabelValues(source).Inc()
}

func (m *Metrics) RecordSearchQuery(searchType string, resultCount int) {
	var resultRange string
	switch {
	case resultCount == 0:
		resultRange = "0"
	case resultCount <= 5:
		resultRange = "1-5"
	case resultCount <= 10:
		resultRange = "6-10"
	case resultCount <= 20:
		resultRange = "11-20"
	default:
		resultRange = "20+"
	}
	
	m.SearchQueries.WithLabelValues(searchType, resultRange).Inc()
}

func (m *Metrics) RecordLeaderboardRequest(leaderboardType string) {
	m.LeaderboardRequests.WithLabelValues(leaderboardType).Inc()
}

// StartMetricsServer starts the Prometheus metrics server
func (m *Metrics) StartMetricsServer(port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Metrics summary endpoint
	mux.HandleFunc("/metrics/summary", m.handleMetricsSummary)
	
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	log.Printf("Starting metrics server on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Metrics server error: %v", err)
	}
}

// handleMetricsSummary provides a human-readable metrics summary
func (m *Metrics) handleMetricsSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	summary := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"system": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  getBytesInMB(getMemoryUsage()),
		},
		"note": "For detailed metrics, use /metrics endpoint with Prometheus format",
	}
	
	fmt.Fprintf(w, "%+v", summary)
}

// Helper functions
func getBytesInMB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}

func getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

