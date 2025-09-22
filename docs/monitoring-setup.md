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
