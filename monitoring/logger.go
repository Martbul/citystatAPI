
// monitoring/logger.go
package monitoring

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// StructuredLogger provides structured logging with metrics integration
type StructuredLogger struct {
	logger  *log.Logger
	metrics *Metrics
	level   LogLevel
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Duration  float64                `json:"duration_ms,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(metrics *Metrics, serviceName string, level LogLevel) *StructuredLogger {
	return &StructuredLogger{
		logger:  log.New(os.Stdout, "", 0),
		metrics: metrics,
		level:   level,
	}
}

// Debug logs debug level messages
func (sl *StructuredLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	if sl.level <= LogLevelDebug {
		sl.log(ctx, LogLevelDebug, "debug", message, fields, nil)
	}
}

// Info logs info level messages
func (sl *StructuredLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	if sl.level <= LogLevelInfo {
		sl.log(ctx, LogLevelInfo, "info", message, fields, nil)
	}
}

// Warn logs warning messages
func (sl *StructuredLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	if sl.level <= LogLevelWarn {
		sl.log(ctx, LogLevelWarn, "warning", message, fields, nil)
		if sl.metrics != nil {
			sl.metrics.RecordError("api", "warning", "warning")
		}
	}
}

// Error logs error messages
func (sl *StructuredLogger) Error(ctx context.Context, message string, err error, fields map[string]interface{}) {
	if sl.level <= LogLevelError {
		sl.log(ctx, LogLevelError, "error", message, fields, err)
		if sl.metrics != nil {
			sl.metrics.RecordError("api", "application_error", "error")
		}
	}
}

// Fatal logs fatal messages and exits
func (sl *StructuredLogger) Fatal(ctx context.Context, message string, err error, fields map[string]interface{}) {
	sl.log(ctx, LogLevelFatal, "fatal", message, fields, err)
	if sl.metrics != nil {
		sl.metrics.RecordError("api", "fatal_error", "fatal")
	}
	os.Exit(1)
}

// log is the internal logging method
func (sl *StructuredLogger) log(ctx context.Context, level LogLevel, errorType, message string, fields map[string]interface{}, err error) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level.String(),
		Service:   "citystat-api",
		Message:   message,
		Fields:    fields,
	}
	
	// Extract context information
	if ctx != nil {
		if userID, ok := ctx.Value("user_id").(string); ok {
			entry.UserID = userID
		}
		if requestID, ok := ctx.Value("request_id").(string); ok {
			entry.RequestID = requestID
		}
		if duration, ok := ctx.Value("request_duration").(time.Duration); ok {
			entry.Duration = float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds
		}
	}
	
	if err != nil {
		entry.Error = err.Error()
	}
	
	// Marshal to JSON
	jsonBytes, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to simple logging if JSON marshaling fails
		sl.logger.Printf("[%s] %s: %s (JSON marshal error: %v)", level.String(), entry.Service, message, marshalErr)
		return
	}
	
	sl.logger.Println(string(jsonBytes))
}