package cache

import (
	"sync"
	"time"
)

// PermissionCacheEntry represents a cached permission check result
type PermissionCacheEntry struct {
	Allowed    bool
	ExpiryTime time.Time
}

// PermissionCache provides thread-safe caching for RBAC permission checks
type PermissionCache struct {
	cache map[string]PermissionCacheEntry
	mutex sync.RWMutex
}

// NewPermissionCache creates a new permission cache instance
func NewPermissionCache() *PermissionCache {
	return &PermissionCache{
		cache: make(map[string]PermissionCacheEntry),
	}
}

// Get retrieves a permission check result from cache if not expired
func (c *PermissionCache) Get(key string) (bool, bool) {
	c.mutex.RLock()
	entry, found := c.cache[key]
	c.mutex.RUnlock()

	if found && time.Now().Before(entry.ExpiryTime) {
		return entry.Allowed, true
	}

	return false, false
}

// Set stores a permission check result in cache with expiration time
func (c *PermissionCache) Set(key string, allowed bool, expiry time.Time) {
	c.mutex.Lock()
	c.cache[key] = PermissionCacheEntry{
		Allowed:    allowed,
		ExpiryTime: expiry,
	}
	c.mutex.Unlock()
}

// Clear removes expired entries from cache
func (c *PermissionCache) Clear() {
	c.mutex.Lock()
	for key, entry := range c.cache {
		if time.Now().After(entry.ExpiryTime) {
			delete(c.cache, key)
		}
	}
	c.mutex.Unlock()
}

// BuildCacheKey creates a cache key from subject, resource, and action
func BuildCacheKey(subjectID string, resource string, action string) string {
	return subjectID + ":" + resource + ":" + action
}
