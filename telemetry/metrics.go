package telemetry

import (
    "context"
    "fmt"
    "runtime"
    "time"

    "github.com/gorilla/mux"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

var (
    meter      = otel.Meter("citystat-api/metrics")
    tracer     = otel.Tracer("citystat-api/metrics")
    
    // HTTP Metrics (handled by middleware)
    httpRequestsTotal    metric.Int64Counter
    httpRequestDuration  metric.Float64Histogram
    activeRequests       metric.Int64UpDownCounter
    httpRequestSize      metric.Int64Histogram
    httpResponseSize     metric.Int64Histogram
    
    // Business Metrics
    userRegistrations      metric.Int64Counter
    cityDataProcessed      metric.Int64Counter
    friendRequestsSent     metric.Int64Counter
    searchQueriesTotal     metric.Int64Counter
    userOperationsTotal    metric.Int64Counter
    
    // System Metrics
    goroutineCount         metric.Int64ObservableGauge
    memoryUsage           metric.Int64ObservableGauge
    heapObjects           metric.Int64ObservableGauge
    
    // Performance Metrics
    cityProcessingDuration metric.Float64Histogram
    databaseOperations     metric.Int64Counter
    responseTime          metric.Float64Histogram
    
    // Error Metrics
    errorsTotal           metric.Int64Counter
)


//! leter add the rate-limiter to the metrics
// var (
//     // ... your existing metrics ...
    
//     RateLimitHitsTotal = promauto.NewCounterVec(
//         prometheus.CounterOpts{
//             Name: "rate_limit_hits_total",
//             Help: "Total number of rate limit hits",
//         },
//         []string{"identifier_type", "endpoint", "user_tier"},
//     )
    
//     ActiveRateLimitClients = promauto.NewGauge(
//         prometheus.GaugeOpts{
//             Name: "active_rate_limit_clients",
//             Help: "Number of active rate limit clients",
//         },
//     )
// )


// InitAllMetrics initializes all metrics at once
func InitAllMetrics() error {
    var err error
    
    // HTTP Metrics (used by middleware)
    httpRequestsTotal, err = meter.Int64Counter(
        "http_requests_total",
        metric.WithDescription("Total number of HTTP requests"),
    )
    if err != nil {
        return fmt.Errorf("failed to create http_requests_total: %w", err)
    }
    
    httpRequestDuration, err = meter.Float64Histogram(
        "http_request_duration_seconds",
        metric.WithDescription("Duration of HTTP requests in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return fmt.Errorf("failed to create http_request_duration_seconds: %w", err)
    }
    
    activeRequests, err = meter.Int64UpDownCounter(
        "http_active_requests",
        metric.WithDescription("Number of active HTTP requests"),
    )
    if err != nil {
        return fmt.Errorf("failed to create http_active_requests: %w", err)
    }
    
    httpRequestSize, err = meter.Int64Histogram(
        "http_request_size_bytes",
        metric.WithDescription("Size of HTTP requests in bytes"),
        metric.WithUnit("bytes"),
    )
    if err != nil {
        return fmt.Errorf("failed to create http_request_size_bytes: %w", err)
    }
    
    httpResponseSize, err = meter.Int64Histogram(
        "http_response_size_bytes",
        metric.WithDescription("Size of HTTP responses in bytes"),
        metric.WithUnit("bytes"),
    )
    if err != nil {
        return fmt.Errorf("failed to create http_response_size_bytes: %w", err)
    }
    
    // Business Metrics
    userRegistrations, err = meter.Int64Counter(
        "user_registrations_total",
        metric.WithDescription("Total number of user registrations"),
    )
    if err != nil {
        return fmt.Errorf("failed to create user_registrations_total: %w", err)
    }
    
    cityDataProcessed, err = meter.Int64Counter(
        "city_data_processed_total",
        metric.WithDescription("Total number of city data processing operations"),
    )
    if err != nil {
        return fmt.Errorf("failed to create city_data_processed_total: %w", err)
    }
    
    friendRequestsSent, err = meter.Int64Counter(
        "friend_requests_sent_total",
        metric.WithDescription("Total number of friend requests sent"),
    )
    if err != nil {
        return fmt.Errorf("failed to create friend_requests_sent_total: %w", err)
    }
    
    searchQueriesTotal, err = meter.Int64Counter(
        "search_queries_total",
        metric.WithDescription("Total number of search queries"),
    )
    if err != nil {
        return fmt.Errorf("failed to create search_queries_total: %w", err)
    }
    
    userOperationsTotal, err = meter.Int64Counter(
        "user_operations_total",
        metric.WithDescription("Total number of user operations"),
    )
    if err != nil {
        return fmt.Errorf("failed to create user_operations_total: %w", err)
    }
    
    // Performance Metrics
    cityProcessingDuration, err = meter.Float64Histogram(
        "city_processing_duration_seconds",
        metric.WithDescription("Duration of city data processing operations"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return fmt.Errorf("failed to create city_processing_duration_seconds: %w", err)
    }
    
    databaseOperations, err = meter.Int64Counter(
        "database_operations_total",
        metric.WithDescription("Total number of database operations"),
    )
    if err != nil {
        return fmt.Errorf("failed to create database_operations_total: %w", err)
    }
    
    responseTime, err = meter.Float64Histogram(
        "response_time_seconds",
        metric.WithDescription("HTTP response time"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return fmt.Errorf("failed to create response_time_seconds: %w", err)
    }
    
    // Error Metrics
    errorsTotal, err = meter.Int64Counter(
        "errors_total",
        metric.WithDescription("Total number of errors"),
    )
    if err != nil {
        return fmt.Errorf("failed to create errors_total: %w", err)
    }
    
    // System Metrics (Observable Gauges)
    goroutineCount, err = meter.Int64ObservableGauge(
        "goroutine_count",
        metric.WithDescription("Number of goroutines"),
    )
    if err != nil {
        return fmt.Errorf("failed to create goroutine_count: %w", err)
    }
    
    memoryUsage, err = meter.Int64ObservableGauge(
        "memory_usage_bytes",
        metric.WithDescription("Memory usage in bytes"),
        metric.WithUnit("bytes"),
    )
    if err != nil {
        return fmt.Errorf("failed to create memory_usage_bytes: %w", err)
    }
    
    heapObjects, err = meter.Int64ObservableGauge(
        "heap_objects",
        metric.WithDescription("Number of objects on the heap"),
    )
    if err != nil {
        return fmt.Errorf("failed to create heap_objects: %w", err)
    }
    
    // Register runtime metrics callback
    _, err = meter.RegisterCallback(
        func(ctx context.Context, o metric.Observer) error {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            
            o.ObserveInt64(goroutineCount, int64(runtime.NumGoroutine()))
            o.ObserveInt64(memoryUsage, int64(m.Alloc))
            o.ObserveInt64(heapObjects, int64(m.HeapObjects))
            
            return nil
        },
        goroutineCount,
        memoryUsage,
        heapObjects,
    )
    
    if err != nil {
        return fmt.Errorf("failed to register runtime metrics callback: %w", err)
    }
    
    return nil
}

// GetHTTPMetrics returns HTTP metrics for middleware use
func GetHTTPMetrics() (metric.Int64Counter, metric.Float64Histogram, metric.Int64UpDownCounter, metric.Int64Histogram, metric.Int64Histogram) {
    return httpRequestsTotal, httpRequestDuration, activeRequests, httpRequestSize, httpResponseSize
}

// Business Metric Recording Functions
func RecordUserRegistration(ctx context.Context, method string) {
    if userRegistrations != nil {
        userRegistrations.Add(ctx, 1, metric.WithAttributes(
            attribute.String("registration_method", method),
        ))
    }
}

func RecordCityDataProcessing(ctx context.Context, cityName string, duration time.Duration) {
    if cityDataProcessed != nil && cityProcessingDuration != nil {
        cityDataProcessed.Add(ctx, 1, metric.WithAttributes(
            attribute.String("city", cityName),
        ))
        cityProcessingDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
            attribute.String("city", cityName),
        ))
    }
}

func RecordSearchQuery(ctx context.Context, queryType string, resultCount int) {
    if searchQueriesTotal != nil {
        searchQueriesTotal.Add(ctx, 1, metric.WithAttributes(
            attribute.String("query_type", queryType),
            attribute.Int("result_count", resultCount),
        ))
    }
}

func RecordFriendRequest(ctx context.Context, status string) {
    if friendRequestsSent != nil {
        friendRequestsSent.Add(ctx, 1, metric.WithAttributes(
            attribute.String("status", status),
        ))
    }
}

func RecordUserOperation(ctx context.Context, operation string) {
    if userOperationsTotal != nil {
        userOperationsTotal.Add(ctx, 1, metric.WithAttributes(
            attribute.String("operation", operation),
        ))
    }
}

func RecordDatabaseOperation(ctx context.Context, operation, table string, duration time.Duration) {
    if databaseOperations != nil {
        databaseOperations.Add(ctx, 1, metric.WithAttributes(
            attribute.String("operation", operation),
            attribute.String("table", table),
            attribute.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
        ))
    }
}

func RecordError(ctx context.Context, errorType, component string) {
    if errorsTotal != nil {
        errorsTotal.Add(ctx, 1, metric.WithAttributes(
            attribute.String("error_type", errorType),
            attribute.String("component", component),
        ))
    }
}

// Error tracking with traces
func RecordErrorWithTrace(ctx context.Context, err error, component string, additionalAttrs ...attribute.KeyValue) {
    span := trace.SpanFromContext(ctx)
    
    if span.IsRecording() {
        span.SetStatus(codes.Error, err.Error())
        
        attrs := []attribute.KeyValue{
            attribute.String("error.type", fmt.Sprintf("%T", err)),
            attribute.String("error.message", err.Error()),
            attribute.String("component", component),
        }
        
        attrs = append(attrs, additionalAttrs...)
        span.SetAttributes(attrs...)
    }
    
    // Also record to metrics
    RecordError(ctx, fmt.Sprintf("%T", err), component)
}

// Panic recovery
func RecordPanic(ctx context.Context, component string) {
    if r := recover(); r != nil {
        span := trace.SpanFromContext(ctx)
        if span.IsRecording() {
            span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", r))
            span.SetAttributes(
                attribute.String("error.type", "panic"),
                attribute.String("error.message", fmt.Sprintf("%v", r)),
                attribute.String("component", component),
            )
        }
        
        RecordError(ctx, "panic", component)
        
        // Re-panic after recording
        panic(r)
    }
}

// Timer for measuring execution time
type Timer struct {
    start time.Time
    ctx   context.Context
    name  string
    attrs []attribute.KeyValue
}

func StartTimer(ctx context.Context, name string, attrs ...attribute.KeyValue) *Timer {
    return &Timer{
        start: time.Now(),
        ctx:   ctx,
        name:  name,
        attrs: attrs,
    }
}

func (t *Timer) Stop() time.Duration {
    duration := time.Since(t.start)
    
    if responseTime != nil {
        responseTime.Record(t.ctx, duration.Seconds(), metric.WithAttributes(t.attrs...))
    }
    
    return duration
}

// Prometheus metrics setup (optional - if you want both OTLP and Prometheus)
var (
    promRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "citystat_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    promRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "citystat_http_request_duration_seconds",
            Help:    "Duration of HTTP requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(promRequestsTotal)
    prometheus.MustRegister(promRequestDuration)
}

// SetupMetricsEndpoint adds a /metrics endpoint for Prometheus scraping
func SetupMetricsEndpoint(r *mux.Router) {
    r.Handle("/metrics", promhttp.Handler())
}

// Wrapper function for error handling with recording
func WrapWithErrorRecording(ctx context.Context, component string, fn func() error) error {
    defer RecordPanic(ctx, component)
    
    err := fn()
    if err != nil {
        RecordErrorWithTrace(ctx, err, component)
    }
    
    return err
}