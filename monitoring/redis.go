
// monitoring/redis.go
package monitoring

import (
	"context"
	"time"
	
	"github.com/go-redis/redis/v8"
)

// RedisMetricsWrapper wraps Redis operations with metrics
type RedisMetricsWrapper struct {
	client  *redis.Client
	metrics *Metrics
}

// NewRedisMetricsWrapper creates a new Redis metrics wrapper
func NewRedisMetricsWrapper(client *redis.Client, metrics *Metrics) *RedisMetricsWrapper {
	wrapper := &RedisMetricsWrapper{
		client:  client,
		metrics: metrics,
	}
	
	// Start connection metrics collection
	go wrapper.collectConnectionMetrics()
	
	return wrapper
}

// Ping wraps Redis ping with metrics
func (rmw *RedisMetricsWrapper) Ping(ctx context.Context) error {
	start := time.Now()
	err := rmw.client.Ping(ctx).Err()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("ping", duration, err)
	return err
}

// Get wraps Redis GET with metrics
func (rmw *RedisMetricsWrapper) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	result, err := rmw.client.Get(ctx, key).Result()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("get", duration, err)
	
	// Record cache hit/miss
	if err == redis.Nil {
		rmw.metrics.RedisCacheHits.WithLabelValues("get", "miss").Inc()
	} else if err == nil {
		rmw.metrics.RedisCacheHits.WithLabelValues("get", "hit").Inc()
	}
	
	return result, err
}

// Set wraps Redis SET with metrics
func (rmw *RedisMetricsWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	err := rmw.client.Set(ctx, key, value, expiration).Err()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("set", duration, err)
	rmw.metrics.RedisCacheHits.WithLabelValues("set", "success").Inc()
	
	return err
}

// Del wraps Redis DEL with metrics
func (rmw *RedisMetricsWrapper) Del(ctx context.Context, keys ...string) error {
	start := time.Now()
	err := rmw.client.Del(ctx, keys...).Err()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("del", duration, err)
	rmw.metrics.RedisCacheHits.WithLabelValues("delete", "success").Inc()
	
	return err
}

// ZAdd wraps Redis ZADD with metrics (for rate limiting)
func (rmw *RedisMetricsWrapper) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	start := time.Now()
	err := rmw.client.ZAdd(ctx, key, members...).Err()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("zadd", duration, err)
	return err
}

// ZRemRangeByScore wraps Redis ZREMRANGEBYSCORE with metrics
func (rmw *RedisMetricsWrapper) ZRemRangeByScore(ctx context.Context, key, min, max string) error {
	start := time.Now()
	err := rmw.client.ZRemRangeByScore(ctx, key, min, max).Err()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("zremrangebyscore", duration, err)
	return err
}

// ZCard wraps Redis ZCARD with metrics
func (rmw *RedisMetricsWrapper) ZCard(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	result, err := rmw.client.ZCard(ctx, key).Result()
	duration := time.Since(start)
	
	rmw.metrics.RecordRedisOperation("zcard", duration, err)
	return result, err
}

// collectConnectionMetrics collects Redis connection metrics
func (rmw *RedisMetricsWrapper) collectConnectionMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		stats := rmw.client.PoolStats()
		rmw.metrics.RedisConnectionsTotal.Set(float64(stats.TotalConns))
	}
}