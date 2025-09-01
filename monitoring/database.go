// monitoring/database.go
package monitoring

import (
	"time"
)

// DatabaseMetricsWrapper wraps database operations with metrics
type DatabaseMetricsWrapper struct {
	metrics *Metrics
	db      interface{} // Could be *sql.DB or your Prisma client
}

// NewDatabaseMetricsWrapper creates a new database metrics wrapper
func NewDatabaseMetricsWrapper(metrics *Metrics, db interface{}) *DatabaseMetricsWrapper {
	return &DatabaseMetricsWrapper{
		metrics: metrics,
		db:      db,
	}
}

// WrapPrismaOperation wraps Prisma operations with metrics
func (dmw *DatabaseMetricsWrapper) WrapPrismaOperation(operation, table string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	dmw.metrics.RecordDBQuery(operation, table, duration, err)
	return err
}

// UpdateConnectionMetrics updates database connection metrics
func (dmw *DatabaseMetricsWrapper) UpdateConnectionMetrics(total, idle int) {
	dmw.metrics.DBConnectionsTotal.Set(float64(total))
	dmw.metrics.DBConnectionsIdle.Set(float64(idle))
}