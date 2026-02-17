package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// BenchmarkPingHandler measures raw handler performance for the ping endpoint
func BenchmarkPingHandler(b *testing.B) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = ping(c)
	}
}

// BenchmarkPingHandlerParallel measures concurrent ping performance
func BenchmarkPingHandlerParallel(b *testing.B) {
	e := echo.New()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			_ = ping(c)
		}
	})
}

// BenchmarkJSONSerialization measures the overhead of JSON response creation
func BenchmarkJSONSerialization(b *testing.B) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = c.JSON(http.StatusOK, map[string]string{"message": "pong"})
	}
}

// BenchmarkBatchDownloadRequestParsing measures JSON body parsing overhead
func BenchmarkBatchDownloadRequestParsing(b *testing.B) {
	e := echo.New()

	paths := make([]string, 50)
	for i := range paths {
		paths[i] = "folder/file-" + string(rune('a'+i%26)) + ".txt"
	}
	payload, _ := json.Marshal(map[string][]string{"paths": paths})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/batch-download", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var body struct {
			Paths []string
		}
		_ = c.Bind(&body)
	}
}
