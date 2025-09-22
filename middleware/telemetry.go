package middleware

import (
	"net/http"
	"strconv"
	"time"

	"citystatAPI/telemetry"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
    tracer = otel.Tracer("citystat-api/middleware")
)

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

// TelemetryMiddleware adds comprehensive telemetry to HTTP requests
func TelemetryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create span
        ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
            trace.WithAttributes(
                attribute.String("http.method", r.Method),
                attribute.String("http.url", r.URL.String()),
                attribute.String("http.scheme", r.URL.Scheme),
                attribute.String("http.host", r.Host),
                attribute.String("http.user_agent", r.UserAgent()),
                attribute.String("http.remote_addr", r.RemoteAddr),
            ),
        )
        defer span.End()

        // Add user ID to span if available
        if userID, ok := GetUserID(r); ok {
            span.SetAttributes(attribute.String("user.id", userID))
        }

        // Get HTTP metrics from telemetry package
        httpRequestsTotal, httpRequestDuration, activeRequests, httpRequestSize, httpResponseSize := telemetry.GetHTTPMetrics()

        // Increment active requests
        activeRequests.Add(ctx, 1)
        defer func() {
            activeRequests.Add(ctx, -1)
        }()

        // Record request size
        if r.ContentLength > 0 {
            httpRequestSize.Record(ctx, r.ContentLength,
                metric.WithAttributes(
                    attribute.String("method", r.Method),
                    attribute.String("route", r.URL.Path),
                ),
            )
        }

        // Wrap response writer
        rw := &responseWriter{
            ResponseWriter: w,
            statusCode:     http.StatusOK,
        }

        // Process request
        next.ServeHTTP(rw, r.WithContext(ctx))

        // Calculate duration
        duration := time.Since(start)

        // Set span attributes
        span.SetAttributes(
            attribute.Int("http.status_code", rw.statusCode),
            attribute.Int64("http.response_size", rw.size),
            attribute.Float64("http.duration_ms", float64(duration.Nanoseconds())/1e6),
        )

        // Record metrics using telemetry package functions
        labels := metric.WithAttributes(
            attribute.String("method", r.Method),
            attribute.String("route", r.URL.Path),
            attribute.String("status", strconv.Itoa(rw.statusCode)),
            attribute.String("status_class", getStatusClass(rw.statusCode)),
        )

        httpRequestsTotal.Add(ctx, 1, labels)
        httpRequestDuration.Record(ctx, duration.Seconds(), labels)

        if rw.size > 0 {
            httpResponseSize.Record(ctx, rw.size, labels)
        }

        // Mark span as error if status code >= 400
        if rw.statusCode >= 400 {
            span.SetAttributes(attribute.Bool("error", true))
        }
    })
}

func getStatusClass(statusCode int) string {
    switch {
    case statusCode >= 200 && statusCode < 300:
        return "2xx"
    case statusCode >= 300 && statusCode < 400:
        return "3xx"
    case statusCode >= 400 && statusCode < 500:
        return "4xx"
    case statusCode >= 500:
        return "5xx"
    default:
        return "1xx"
    }
}