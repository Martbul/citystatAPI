package middleware

import (
	"citystatAPI/monitoring"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture response size and status
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += int64(size)
	return size, err
}

// HTTPMetricsMiddleware creates middleware for HTTP metrics collection
func HTTPMetricsMiddleware(metrics *monitoring.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Increment in-flight requests
			metrics.RequestsInFlight.Inc()
			defer metrics.RequestsInFlight.Dec()
			
			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
				size:          0,
			}
			
			// Get user tier from context (set by your auth middleware)
			userTier := "free" // default
			if tier := r.Context().Value("user_tier"); tier != nil {
				if tierStr, ok := tier.(string); ok {
					userTier = tierStr
				}
			}
			
			// Process request
			next.ServeHTTP(rw, r)
			
			// Record metrics
			duration := time.Since(start)
			endpoint := normalizeEndpoint(r.URL.Path)
			
			metrics.RecordHTTPRequest(
				r.Method,
				endpoint,
				rw.statusCode,
				duration,
				rw.size,
				userTier,
			)
		})
	}
}

// normalizeEndpoint normalizes URL paths for consistent metrics
func normalizeEndpoint(path string) string {
	// This should match the logic in your rate limiter
	// For now, simplified version
	if len(path) > 100 {
		return path[:100] + "..."
	}
	return path
}

// PanicRecoveryMiddleware wraps handlers with panic recovery and metrics
func PanicRecoveryMiddleware(metrics *monitoring.Metrics, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					metrics.PanicTotal.WithLabelValues(serviceName, r.URL.Path).Inc()
					metrics.RecordError(serviceName, "panic", "critical")
					
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}