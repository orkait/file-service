package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkCacheGet measures read performance of the URL cache
func BenchmarkCacheGet(b *testing.B) {
	c := NewURLCache()
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("https://example.com/%d", i), time.Now().Add(10*time.Minute))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Get(fmt.Sprintf("key-%d", i%1000))
	}
}

// BenchmarkCacheSet measures write performance
func BenchmarkCacheSet(b *testing.B) {
	c := NewURLCache()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("https://example.com/%d", i), time.Now().Add(10*time.Minute))
	}
}

// BenchmarkCacheGetParallel measures concurrent read performance (contention)
func BenchmarkCacheGetParallel(b *testing.B) {
	c := NewURLCache()
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("https://example.com/%d", i), time.Now().Add(10*time.Minute))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(fmt.Sprintf("key-%d", i%1000))
			i++
		}
	})
}

// BenchmarkCacheMixedReadWrite simulates realistic mixed read/write workload
func BenchmarkCacheMixedReadWrite(b *testing.B) {
	c := NewURLCache()
	for i := 0; i < 500; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("https://example.com/%d", i), time.Now().Add(10*time.Minute))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%5 == 0 {
				c.Set(fmt.Sprintf("key-%d", i%1000), fmt.Sprintf("https://example.com/%d", i), time.Now().Add(10*time.Minute))
			} else {
				c.Get(fmt.Sprintf("key-%d", i%500))
			}
			i++
		}
	})
}

// BenchmarkCacheClear measures cache clear performance with many entries
func BenchmarkCacheClear(b *testing.B) {
	c := NewURLCache()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			c.Set(fmt.Sprintf("key-%d", j), fmt.Sprintf("url-%d", j), time.Now().Add(-1*time.Minute))
		}
		c.Clear()
	}
}

// BenchmarkCacheMemoryPressure simulates the cache growing under memory pressure
// Important for 1GB RAM environments - unbounded cache can OOM the process
func BenchmarkCacheMemoryPressure(b *testing.B) {
	c := NewURLCache()

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < b.N/10; i++ {
				key := fmt.Sprintf("worker-%d-key-%d", w, i)
				c.Set(key, fmt.Sprintf("https://cdn.example.com/very/long/url/path/%d/%d", w, i), time.Now().Add(5*time.Minute))
				c.Get(key)
			}
		}(worker)
	}
	wg.Wait()
}
