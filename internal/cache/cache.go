package cache

import (
	"encoding/json"
	"sync"
	"time"
)

const ttl = 60 * time.Minute
const Key = "doc_%s_%d"

type CacheItem struct {
	Data      []byte
	Mime      string
	ExpiresAt time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]CacheItem),
	}
	go cache.cleanup(cleanupInterval)
	return cache
}

func (c *MemoryCache) SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{
		Data:      data,
		Mime:      "application/json",
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *MemoryCache) GetJSON(key string, dest interface{}) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		return false, nil
	}
	err := json.Unmarshal(item.Data, dest)
	if err != nil {
		return false, err
	}
	return true, nil
}
func (c *MemoryCache) SetFile(key string, data []byte, mime string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{
		Data:      data,
		Mime:      mime,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) GetFile(key string) ([]byte, string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		return nil, "", false
	}
	return item.Data, item.Mime, true
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *MemoryCache) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
