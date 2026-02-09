package cache

import (
	"sync"
	"time"
)

// CacheEntry represents a cached URL with expiration
type CacheEntry struct {
	URL        string
	ExpiryTime time.Time
}

// URLCache provides thread-safe URL caching
type URLCache struct {
	cache map[string]CacheEntry
	mutex sync.RWMutex
}

// NewURLCache creates a new URL cache instance
func NewURLCache() *URLCache {
	return &URLCache{
		cache: make(map[string]CacheEntry),
	}
}

// Get retrieves a URL from cache if not expired
func (c *URLCache) Get(key string) (string, bool) {
	c.mutex.RLock()
	entry, found := c.cache[key]
	c.mutex.RUnlock()

	if found && time.Now().Before(entry.ExpiryTime) {
		return entry.URL, true
	}

	return "", false
}

// Set stores a URL in cache with expiration time
func (c *URLCache) Set(key string, url string, expiry time.Time) {
	c.mutex.Lock()
	c.cache[key] = CacheEntry{
		URL:        url,
		ExpiryTime: expiry,
	}
	c.mutex.Unlock()
}

// Clear removes expired entries from cache
func (c *URLCache) Clear() {
	c.mutex.Lock()
	for key, entry := range c.cache {
		if time.Now().After(entry.ExpiryTime) {
			delete(c.cache, key)
		}
	}
	c.mutex.Unlock()
}
