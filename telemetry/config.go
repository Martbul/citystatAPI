package telemetry

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/uptrace/uptrace-go/uptrace"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    // "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type TelemetryConfig struct {
    ServiceName    string
    ServiceVersion string
    Environment    string
    UptraceDSN     string
    EnableMetrics  bool
    EnableTracing  bool
}

type TelemetryShutdown func(context.Context) error


func InitTelemetry(cfg TelemetryConfig) (TelemetryShutdown, error) {
    var shutdownFuncs []func(context.Context) error

    // Create resource with service information
    res, err := resource.New(context.Background(),
        resource.WithAttributes(
            semconv.ServiceName(cfg.ServiceName),
            semconv.ServiceVersion(cfg.ServiceVersion),
            semconv.DeploymentEnvironment(cfg.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    if cfg.UptraceDSN != "" {
        // ✅ Use Uptrace — handles traces + metrics
        uptrace.ConfigureOpentelemetry(
            uptrace.WithDSN(cfg.UptraceDSN),
            uptrace.WithServiceName(cfg.ServiceName),
            uptrace.WithServiceVersion(cfg.ServiceVersion),
            uptrace.WithDeploymentEnvironment(cfg.Environment),
            uptrace.WithMetricsEnabled(cfg.EnableMetrics),
        )
        log.Println("✅ Uptrace initialized (tracing + metrics)")

        return func(ctx context.Context) error {
            // Flush before shutdown
            uptrace.Shutdown(ctx)
            return nil
        }, nil
    }

    // Fallback: manual exporters (only if not using Uptrace)
    if cfg.EnableTracing {
        traceShutdown, err := initTracing(res)
        if err != nil {
            return nil, fmt.Errorf("failed to initialize tracing: %w", err)
        }
        shutdownFuncs = append(shutdownFuncs, traceShutdown)
        log.Println("✅ Tracing initialized (manual)")
    }

    if cfg.EnableMetrics {
        metricShutdown, err := initMetrics(res)
        if err != nil {
            return nil, fmt.Errorf("failed to initialize metrics: %w", err)
        }
        shutdownFuncs = append(shutdownFuncs, metricShutdown)
        log.Println("✅ Metrics initialized (manual)")
    }

    return func(ctx context.Context) error {
        var err error
        for _, fn := range shutdownFuncs {
            if shutdownErr := fn(ctx); shutdownErr != nil {
                err = fmt.Errorf("shutdown error: %v; %w", shutdownErr, err)
            }
        }
        return err
    }, nil
}

// InitTelemetry initializes OpenTelemetry with Uptrace
// func InitTelemetry(cfg TelemetryConfig) (TelemetryShutdown, error) {
//     var shutdownFuncs []func(context.Context) error

//     // Create resource with service information
//     res, err := resource.New(context.Background(),
//         resource.WithAttributes(
//             semconv.ServiceName(cfg.ServiceName),
//             semconv.ServiceVersion(cfg.ServiceVersion),
//             semconv.DeploymentEnvironment(cfg.Environment),
//         ),
//     )
//     if err != nil {
//         return nil, fmt.Errorf("failed to create resource: %w", err)
//     }

//     // Initialize Uptrace
//     if cfg.UptraceDSN != "" {
//         uptrace.ConfigureOpentelemetry(
//             uptrace.WithDSN(cfg.UptraceDSN),
//             uptrace.WithServiceName(cfg.ServiceName),
//             uptrace.WithServiceVersion(cfg.ServiceVersion),
//         )
//         log.Println("✅ Uptrace initialized")
//     }

//     // Set up trace propagation
//     otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
//         propagation.TraceContext{},
//         propagation.Baggage{},
//     ))

//     // Initialize tracing
//     if cfg.EnableTracing {
//         traceShutdown, err := initTracing(res)
//         if err != nil {
//             return nil, fmt.Errorf("failed to initialize tracing: %w", err)
//         }
//         shutdownFuncs = append(shutdownFuncs, traceShutdown)
//         log.Println("✅ Tracing initialized")
//     }

//     // Initialize metrics
//     if cfg.EnableMetrics {
//         metricShutdown, err := initMetrics(res)
//         if err != nil {
//             return nil, fmt.Errorf("failed to initialize metrics: %w", err)
//         }
//         shutdownFuncs = append(shutdownFuncs, metricShutdown)
//         log.Println("✅ Metrics initialized")
//     }

//     // Return combined shutdown function
//     return func(ctx context.Context) error {
//         var err error
//         for _, fn := range shutdownFuncs {
//             if shutdownErr := fn(ctx); shutdownErr != nil {
//                 err = fmt.Errorf("shutdown error: %v; %w", shutdownErr, err)
//             }
//         }
//         return err
//     }, nil
// }

func initTracing(res *resource.Resource) (func(context.Context) error, error) {
    traceExporter, err := otlptracehttp.New(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to create trace exporter: %w", err)
    }

    traceProvider := trace.NewTracerProvider(
        trace.WithBatcher(traceExporter),
        trace.WithResource(res),
        trace.WithSampler(trace.AlwaysSample()),
    )

    otel.SetTracerProvider(traceProvider)

    return traceProvider.Shutdown, nil
}

func initMetrics(res *resource.Resource) (func(context.Context) error, error) {
    metricExporter, err := otlpmetrichttp.New(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to create metric exporter: %w", err)
    }

    meterProvider := metric.NewMeterProvider(
        metric.WithReader(metric.NewPeriodicReader(metricExporter,
            metric.WithInterval(30*time.Second))),
        metric.WithResource(res),
    )

    otel.SetMeterProvider(meterProvider)

    return meterProvider.Shutdown, nil
}

// GetTelemetryConfigFromEnv loads configuration from environment variables
func GetTelemetryConfigFromEnv() TelemetryConfig {
    return TelemetryConfig{
        ServiceName:    getEnvOrDefault("OTEL_SERVICE_NAME", "citystat-api"),
        ServiceVersion: getEnvOrDefault("OTEL_SERVICE_VERSION", "1.0.0"),
        Environment:    getEnvOrDefault("ENVIRONMENT", "development"),
        UptraceDSN:     os.Getenv("UPTRACE_DSN"),
        EnableMetrics:  getEnvOrDefault("ENABLE_METRICS", "true") == "true",
        EnableTracing:  getEnvOrDefault("ENABLE_TRACING", "true") == "true",
    }
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}