package profiling

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// RegisterPprofRoutes adds Go pprof profiling endpoints under /debug/pprof/
// These are essential for CPU, memory, goroutine, and mutex profiling during stress tests.
func RegisterPprofRoutes(e *echo.Echo) {
	g := e.Group("/debug/pprof")
	g.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	g.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	g.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	g.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	g.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	g.GET("/allocs", echo.WrapHandler(pprof.Handler("allocs")))
	g.GET("/block", echo.WrapHandler(pprof.Handler("block")))
	g.GET("/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
	g.GET("/heap", echo.WrapHandler(pprof.Handler("heap")))
	g.GET("/mutex", echo.WrapHandler(pprof.Handler("mutex")))
	g.GET("/threadcreate", echo.WrapHandler(pprof.Handler("threadcreate")))
}

// MemoryStats returns current memory usage of the application
type MemoryStats struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
	Goroutines   int     `json:"goroutines"`
	HeapObjects  uint64  `json:"heap_objects"`
	HeapInUseMB  float64 `json:"heap_in_use_mb"`
	StackInUseMB float64 `json:"stack_in_use_mb"`
	Timestamp    string  `json:"timestamp"`
}

func GetMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemoryStats{
		AllocMB:      float64(m.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(m.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(m.Sys) / 1024 / 1024,
		NumGC:        m.NumGC,
		Goroutines:   runtime.NumGoroutine(),
		HeapObjects:  m.HeapObjects,
		HeapInUseMB:  float64(m.HeapInuse) / 1024 / 1024,
		StackInUseMB: float64(m.StackInuse) / 1024 / 1024,
		Timestamp:    time.Now().Format(time.RFC3339),
	}
}

// RegisterHealthRoutes adds /health and /metrics/memory endpoints
func RegisterHealthRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"memory": GetMemoryStats(),
		})
	})

	e.GET("/metrics/memory", func(c echo.Context) error {
		return c.JSON(http.StatusOK, GetMemoryStats())
	})

	// Force GC endpoint — useful during stress testing to observe memory behavior
	e.POST("/debug/gc", func(c echo.Context) error {
		runtime.GC()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "gc_triggered",
			"memory": GetMemoryStats(),
		})
	})
}

// IsProfilingEnabled checks if profiling is enabled via ENABLE_PROFILING env var
func IsProfilingEnabled() bool {
	val := strings.ToLower(os.Getenv("ENABLE_PROFILING"))
	return val == "true" || val == "1" || val == "yes"
}

// SetMemoryLimit configures GOMEMLIMIT for 1GB RAM environments.
// Reserves ~200MB for OS/other processes, gives Go ~800MB.
func SetMemoryLimit() {
	// GOMEMLIMIT can also be set via env var directly
	if os.Getenv("GOMEMLIMIT") == "" {
		// For 1GB RAM: leave ~200MB for OS, set Go limit to ~800MB
		limit := int64(800 * 1024 * 1024) // 800 MiB
		// Note: debug.SetMemoryLimit is available in Go 1.19+
		// If using Go 1.20+, it's automatically respected
		fmt.Printf("Setting GOMEMLIMIT to %d bytes (~%d MB)\n", limit, limit/1024/1024)
		os.Setenv("GOMEMLIMIT", fmt.Sprintf("%d", limit))
	}

	// Set aggressive GC for low-memory environments
	if os.Getenv("GOGC") == "" {
		os.Setenv("GOGC", "50") // More frequent GC to keep memory low
		fmt.Println("Setting GOGC=50 for low-memory environment")
	}
}
