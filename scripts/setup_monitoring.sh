# #!/bin/bash

# # Setup monitoring and alerts for citystat-api

# set -e

# echo "🚀 Setting up monitoring for citystat-api..."

# # Check if Uptrace CLI is installed
# if ! command -v uptrace &> /dev/null; then
#     echo "Installing Uptrace CLI..."
#     go install github.com/uptrace/uptrace/cmd/uptrace@latest
# fi

# # Load environment variables
# if [ -f .env ]; then
#     export $(cat .env | xargs)
# fi

# # Validate required environment variables
# if [ -z "$UPTRACE_DSN" ]; then
#     echo "❌ UPTRACE_DSN environment variable is required"
#     exit 1
# fi

# echo "📊 Setting up alerts..."

# # Apply alert rules (this is pseudo-code - actual implementation depends on your monitoring setup)
# if [ -f "monitoring/alerts.yaml" ]; then
#     echo "Applying alert configuration..."
#     # For Uptrace, you would typically configure alerts through their web interface
#     # or API. This is a placeholder for the actual implementation.
#     echo "✅ Alert rules configured (please manually configure in Uptrace UI)"
# fi

# echo "🔧 Setting up dashboards..."
# if [ -d "monitoring/dashboards" ]; then
#     echo "Dashboard templates available in monitoring/dashboards/"
#     echo "Please import them manually into your Uptrace project"
# fi

# echo "✅ Monitoring setup complete!"
# echo ""
# echo "Next steps:"
# echo "1. Go to your Uptrace dashboard: https://app.uptrace.dev"
# echo "2. Import dashboard templates from monitoring/dashboards/"
# echo "3. Configure alert notifications (email, Slack, PagerDuty)"
# echo "4. Test alerts with: go run scripts/test_telemetry.go"

#!/bin/bash

# Setup monitoring and alerts for citystat-api

set -e

echo "🚀 Setting up monitoring for citystat-api..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

print_error() {
    echo -e "${RED}❌${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ️${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "main.go" ]; then
    print_error "Please run this script from your project root directory (where main.go is located)"
    exit 1
fi

# Load environment variables if .env exists
if [ -f .env ]; then
    print_info "Loading environment variables from .env file..."
    export $(cat .env | grep -v '^#' | xargs)
    print_status "Environment variables loaded"
else
    print_warning "No .env file found. Creating template..."
    cat > .env.example << 'EOF'
# Database
DATABASE_URL=your_database_url

# Clerk Authentication
CLERK_SECRET_KEY=your_clerk_secret_key

# Telemetry Configuration
UPTRACE_DSN=https://your-project-key@api.uptrace.dev/your-project-id
OTEL_SERVICE_NAME=citystat-api
OTEL_SERVICE_VERSION=1.0.0
ENVIRONMENT=development
ENABLE_METRICS=true
ENABLE_TRACING=true

# Optional: Custom OpenTelemetry endpoints
# OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
# OTEL_EXPORTER_OTLP_HEADERS=uptrace-dsn=https://your-project-key@api.uptrace.dev/your-project-id
EOF
    print_info "Created .env.example template. Please copy it to .env and fill in your values."
fi

# Validate required environment variables for telemetry
if [ -z "$UPTRACE_DSN" ] && [ -z "$OTEL_EXPORTER_OTLP_ENDPOINT" ]; then
    print_warning "Neither UPTRACE_DSN nor OTEL_EXPORTER_OTLP_ENDPOINT is set"
    print_info "You can still run the API, but telemetry data won't be exported"
    print_info "To set up Uptrace:"
    print_info "1. Go to https://uptrace.dev"
    print_info "2. Create an account and project"
    print_info "3. Copy your DSN and add it to your .env file"
fi

echo ""
print_info "Checking Go dependencies..."

# Check if required Go modules are already installed
if ! go mod tidy >/dev/null 2>&1; then
    print_error "Failed to tidy Go modules. Please check your go.mod file."
    exit 1
fi

print_status "Go modules are clean"

echo ""
print_info "Installing required OpenTelemetry dependencies..."

# Install required dependencies
DEPS=(
    "go.opentelemetry.io/otel@latest"
    "go.opentelemetry.io/otel/trace@latest"
    "go.opentelemetry.io/otel/metric@latest"
    "go.opentelemetry.io/otel/propagation@latest"
    "go.opentelemetry.io/otel/sdk@latest"
    "go.opentelemetry.io/otel/sdk/trace@latest"
    "go.opentelemetry.io/otel/sdk/metric@latest"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@latest"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp@latest"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux@latest"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@latest"
    "github.com/uptrace/uptrace-go@latest"
    "github.com/prometheus/client_golang@latest"
)

for dep in "${DEPS[@]}"; do
    print_info "Installing $dep..."
    if go get "$dep" >/dev/null 2>&1; then
        print_status "Installed $dep"
    else
        print_warning "Failed to install $dep, but continuing..."
    fi
done

echo ""
print_info "Creating monitoring directory structure..."

# Create directory structure
mkdir -p monitoring/{alert-rules,dashboards}
mkdir -p scripts
mkdir -p docs

print_status "Directories created"

echo ""
print_info "Creating monitoring configuration files..."

# Create alerts.yaml if it doesn't exist
if [ ! -f "monitoring/alerts.yaml" ]; then
    cat > monitoring/alerts.yaml << 'EOF'
# Uptrace Alert Configuration
# Configure these alerts in your Uptrace dashboard

alerts:
  - name: "High Error Rate"
    description: "API error rate is above 5% for 5 minutes"
    query: |
      rate(http_requests_total{status_class="4xx" or status_class="5xx"}[5m]) / 
      rate(http_requests_total[5m]) > 0.05
    severity: "critical"
    for: "5m"
    
  - name: "Slow Response Time"
    description: "95th percentile response time is above 2 seconds"
    query: |
      histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
    severity: "warning"
    for: "10m"
    
  - name: "High Memory Usage"
    description: "Memory usage is above 1GB"
    query: |
      memory_usage_bytes > 1073741824
    severity: "warning"
    for: "15m"
    
  - name: "Service Down"
    description: "Service is not responding to health checks"
    query: |
      up{job="citystat-api"} == 0
    severity: "critical"
    for: "1m"
EOF
    print_status "Created monitoring/alerts.yaml"
else
    print_info "monitoring/alerts.yaml already exists, skipping"
fi

# Create basic dashboard template
if [ ! -f "monitoring/dashboards/api-overview.json" ]; then
    cat > monitoring/dashboards/api-overview.json << 'EOF'
{
  "dashboard": {
    "title": "CitystatAPI Overview",
    "description": "Main dashboard for CitystatAPI monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "query": "rate(http_requests_total[5m])",
        "description": "Requests per second"
      },
      {
        "title": "Error Rate",
        "type": "graph", 
        "query": "rate(http_requests_total{status_class=~\"4xx|5xx\"}[5m]) / rate(http_requests_total[5m])",
        "description": "Percentage of failed requests"
      },
      {
        "title": "Response Time (P95)",
        "type": "graph",
        "query": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
        "description": "95th percentile response time"
      },
      {
        "title": "Memory Usage", 
        "type": "graph",
        "query": "memory_usage_bytes",
        "description": "Memory usage in bytes"
      }
    ]
  }
}
EOF
    print_status "Created monitoring/dashboards/api-overview.json"
else
    print_info "Dashboard template already exists, skipping"
fi

echo ""
print_info "Creating test and utility scripts..."

# Create a simple test script without external dependencies
if [ ! -f "scripts/test_telemetry.go" ]; then
    cat > scripts/test_telemetry.go << 'EOF'
package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
)

func main() {
    fmt.Println("🧪 Testing telemetry endpoints...")
    
    baseURL := "http://localhost:3333"
    
    // Test health endpoint
    testEndpoint(baseURL + "/health")
    
    // Test metrics endpoint
    testEndpoint(baseURL + "/metrics")
    
    // Generate some traffic
    generateTraffic(baseURL, 20)
    
    fmt.Println("✅ Test completed! Check your monitoring dashboards.")
}

func testEndpoint(url string) {
    fmt.Printf("Testing %s... ", url)
    
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        fmt.Printf("❌ Error: %v\n", err)
        return
    }
    defer resp.Body.Close()
    
    fmt.Printf("✅ Status: %d\n", resp.StatusCode)
}

func generateTraffic(baseURL string, count int) {
    fmt.Printf("🚀 Generating %d test requests...\n", count)
    
    var wg sync.WaitGroup
    client := &http.Client{Timeout: 10 * time.Second}
    
    endpoints := []string{"/health", "/api/user", "/metrics"}
    
    for i := 0; i < count; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            endpoint := endpoints[i%len(endpoints)]
            url := baseURL + endpoint
            
            req, _ := http.NewRequest("GET", url, nil)
            if endpoint != "/health" && endpoint != "/metrics" {
                req.Header.Set("Authorization", "Bearer test-token")
            }
            
            resp, err := client.Do(req)
            if err != nil {
                return
            }
            resp.Body.Close()
            
            if i%5 == 0 {
                fmt.Printf("📊 Completed %d/%d requests\n", i+1, count)
            }
        }(i)
        
        time.Sleep(100 * time.Millisecond)
    }
    
    wg.Wait()
    fmt.Println("✅ Traffic generation completed")
}
EOF
    print_status "Created scripts/test_telemetry.go"
else
    print_info "Test script already exists, skipping"
fi

echo ""
print_info "Creating documentation..."

# Create monitoring documentation
if [ ! -f "docs/monitoring-setup.md" ]; then
    cat > docs/monitoring-setup.md << 'EOF'
# Monitoring Setup

## Quick Start

1. **Set up Uptrace account**:
   - Go to https://uptrace.dev
   - Create account and project
   - Copy your DSN

2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env and add your UPTRACE_DSN
   ```

3. **Run your application**:
   ```bash
   go run main.go
   ```

4. **Test telemetry**:
   ```bash
   cd scripts
   go run test_telemetry.go
   ```

5. **View metrics**:
   - Health: http://localhost:3333/health  
   - Metrics: http://localhost:3333/metrics
   - Uptrace Dashboard: https://app.uptrace.dev

## Alert Configuration

Import the alert rules from `monitoring/alerts.yaml` into your Uptrace dashboard.

## Dashboards

Import dashboard templates from `monitoring/dashboards/` into Uptrace or Grafana.

## Troubleshooting

- **No metrics in Uptrace**: Check your UPTRACE_DSN in .env
- **High memory usage**: Monitor goroutine count and database connections
- **Slow responses**: Check database query performance in traces
EOF
    print_status "Created docs/monitoring-setup.md"
else
    print_info "Monitoring documentation already exists, skipping"
fi

echo ""
print_status "Monitoring setup completed!"
echo ""
print_info "Next steps:"
echo "1. 📝 Configure your UPTRACE_DSN in .env file"
echo "2. 🚀 Start your API: go run main.go"
echo "3. 🧪 Test telemetry: cd scripts && go run test_telemetry.go"
echo "4. 📊 View metrics at http://localhost:3333/metrics"
echo "5. 📈 Check Uptrace dashboard at https://app.uptrace.dev"
echo ""
print_info "Configuration files created:"
echo "  - monitoring/alerts.yaml (alert rules)"
echo "  - monitoring/dashboards/api-overview.json (dashboard template)"
echo "  - scripts/test_telemetry.go (testing script)"
echo "  - docs/monitoring-setup.md (documentation)"
echo ""
print_warning "Manual steps required:"
echo "  1. Sign up for Uptrace at https://uptrace.dev"
echo "  2. Create a project and copy the DSN"
echo "  3. Add UPTRACE_DSN to your .env file"
echo "  4. Import alert rules and dashboards into Uptrace"
echo ""
print_status "Monitoring setup script completed successfully! 🎉"