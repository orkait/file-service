package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
)

// Metrics holds all performance counters for the application.
// Thread-safe via atomics and mutex.
type Metrics struct {
	TotalRequests     int64            `json:"total_requests"`
	ActiveRequests    int64            `json:"active_requests"`
	TotalErrors       int64            `json:"total_errors"`
	TotalLatencyMs    int64            `json:"total_latency_ms"`
	MaxLatencyMs      int64            `json:"max_latency_ms"`
	StartTime         time.Time        `json:"start_time"`
	EndpointCounts    map[string]int64 `json:"endpoint_counts"`
	EndpointLatencies map[string]int64 `json:"endpoint_latencies"` // total ms per endpoint
	StatusCodes       map[int]int64    `json:"status_codes"`
	mu                sync.Mutex
}

var globalMetrics *Metrics
var once sync.Once

// GetMetrics returns the singleton metrics instance
func GetMetrics() *Metrics {
	once.Do(func() {
		globalMetrics = &Metrics{
			StartTime:         time.Now(),
			EndpointCounts:    make(map[string]int64),
			EndpointLatencies: make(map[string]int64),
			StatusCodes:       make(map[int]int64),
		}
	})
	return globalMetrics
}

// MetricsMiddleware tracks request count, latency, active connections, and error rates
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			m := GetMetrics()

			atomic.AddInt64(&m.ActiveRequests, 1)
			start := time.Now()

			err := next(c)

			latencyMs := time.Since(start).Milliseconds()
			atomic.AddInt64(&m.ActiveRequests, -1)
			atomic.AddInt64(&m.TotalRequests, 1)
			atomic.AddInt64(&m.TotalLatencyMs, latencyMs)

			// Update max latency (lock-free CAS loop)
			for {
				current := atomic.LoadInt64(&m.MaxLatencyMs)
				if latencyMs <= current {
					break
				}
				if atomic.CompareAndSwapInt64(&m.MaxLatencyMs, current, latencyMs) {
					break
				}
			}

			// Track per-endpoint and status code stats (needs mutex)
			statusCode := c.Response().Status
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}
			endpoint := fmt.Sprintf("%s %s", c.Request().Method, path)

			m.mu.Lock()
			m.EndpointCounts[endpoint]++
			m.EndpointLatencies[endpoint] += latencyMs
			m.StatusCodes[statusCode]++
			if statusCode >= 400 {
				atomic.AddInt64(&m.TotalErrors, 1)
			}
			m.mu.Unlock()

			return err
		}
	}
}

// MetricsSnapshot is a point-in-time snapshot of performance data
type MetricsSnapshot struct {
	TotalRequests  int64            `json:"total_requests"`
	ActiveRequests int64            `json:"active_requests"`
	TotalErrors    int64            `json:"total_errors"`
	ErrorRate      float64          `json:"error_rate_pct"`
	AvgLatencyMs   float64          `json:"avg_latency_ms"`
	MaxLatencyMs   int64            `json:"max_latency_ms"`
	RequestsPerSec float64          `json:"requests_per_sec"`
	UptimeSeconds  float64          `json:"uptime_seconds"`
	EndpointCounts map[string]int64 `json:"endpoint_counts"`
	EndpointAvgMs  map[string]int64 `json:"endpoint_avg_latency_ms"`
	StatusCodes    map[int]int64    `json:"status_codes"`
}

// RegisterMetricsRoute adds /metrics/requests endpoint
func RegisterMetricsRoute(e *echo.Echo) {
	e.GET("/metrics/requests", func(c echo.Context) error {
		m := GetMetrics()
		total := atomic.LoadInt64(&m.TotalRequests)
		errors := atomic.LoadInt64(&m.TotalErrors)
		totalLatency := atomic.LoadInt64(&m.TotalLatencyMs)
		uptime := time.Since(m.StartTime).Seconds()

		var avgLatency float64
		if total > 0 {
			avgLatency = float64(totalLatency) / float64(total)
		}

		var errorRate float64
		if total > 0 {
			errorRate = float64(errors) / float64(total) * 100
		}

		m.mu.Lock()
		endpointCounts := make(map[string]int64, len(m.EndpointCounts))
		endpointAvg := make(map[string]int64, len(m.EndpointLatencies))
		for k, v := range m.EndpointCounts {
			endpointCounts[k] = v
			if v > 0 {
				endpointAvg[k] = m.EndpointLatencies[k] / v
			}
		}
		statusCodes := make(map[int]int64, len(m.StatusCodes))
		for k, v := range m.StatusCodes {
			statusCodes[k] = v
		}
		m.mu.Unlock()

		snapshot := MetricsSnapshot{
			TotalRequests:  total,
			ActiveRequests: atomic.LoadInt64(&m.ActiveRequests),
			TotalErrors:    errors,
			ErrorRate:      errorRate,
			AvgLatencyMs:   avgLatency,
			MaxLatencyMs:   atomic.LoadInt64(&m.MaxLatencyMs),
			RequestsPerSec: float64(total) / uptime,
			UptimeSeconds:  uptime,
			EndpointCounts: endpointCounts,
			EndpointAvgMs:  endpointAvg,
			StatusCodes:    statusCodes,
		}

		return c.JSON(http.StatusOK, snapshot)
	})

	// Reset metrics endpoint (useful between test runs)
	e.POST("/metrics/reset", func(c echo.Context) error {
		m := GetMetrics()
		atomic.StoreInt64(&m.TotalRequests, 0)
		atomic.StoreInt64(&m.ActiveRequests, 0)
		atomic.StoreInt64(&m.TotalErrors, 0)
		atomic.StoreInt64(&m.TotalLatencyMs, 0)
		atomic.StoreInt64(&m.MaxLatencyMs, 0)
		m.mu.Lock()
		m.EndpointCounts = make(map[string]int64)
		m.EndpointLatencies = make(map[string]int64)
		m.StatusCodes = make(map[int]int64)
		m.StartTime = time.Now()
		m.mu.Unlock()
		return c.JSON(http.StatusOK, map[string]string{"status": "metrics_reset"})
	})
}
