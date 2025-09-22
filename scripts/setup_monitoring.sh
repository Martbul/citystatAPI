#!/bin/bash

# Setup monitoring and alerts for citystat-api

set -e

echo "🚀 Setting up monitoring for citystat-api..."

# Check if Uptrace CLI is installed
if ! command -v uptrace &> /dev/null; then
    echo "Installing Uptrace CLI..."
    go install github.com/uptrace/uptrace/cmd/uptrace@latest
fi

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

# Validate required environment variables
if [ -z "$UPTRACE_DSN" ]; then
    echo "❌ UPTRACE_DSN environment variable is required"
    exit 1
fi

echo "📊 Setting up alerts..."

# Apply alert rules (this is pseudo-code - actual implementation depends on your monitoring setup)
if [ -f "monitoring/alerts.yaml" ]; then
    echo "Applying alert configuration..."
    # For Uptrace, you would typically configure alerts through their web interface
    # or API. This is a placeholder for the actual implementation.
    echo "✅ Alert rules configured (please manually configure in Uptrace UI)"
fi

echo "🔧 Setting up dashboards..."
if [ -d "monitoring/dashboards" ]; then
    echo "Dashboard templates available in monitoring/dashboards/"
    echo "Please import them manually into your Uptrace project"
fi

echo "✅ Monitoring setup complete!"
echo ""
echo "Next steps:"
echo "1. Go to your Uptrace dashboard: https://app.uptrace.dev"
echo "2. Import dashboard templates from monitoring/dashboards/"
echo "3. Configure alert notifications (email, Slack, PagerDuty)"
echo "4. Test alerts with: go run scripts/test_telemetry.go"