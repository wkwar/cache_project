package groupcache

import (
	lru "groupcache/cache"
	"sync"
	"time"
)

// 缓存处理 lru规则 + 读写锁(多并发)
type cache struct {
	mu         sync.RWMutex
	cacheBytes int64
	lruCache   lru.Cache
}

func (c *cache) lruCacheLazyLoadIfNeed() {
	if c.lruCache == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.lruCache == nil {
			c.lruCache = lru.NewLRUCache(c.cacheBytes)
		}
	}
}

func (c *cache) add(key string, value ByteView) {
	c.lruCacheLazyLoadIfNeed()
	c.lruCache.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lruCache == nil {
		return
	}
	vi, ok := c.lruCache.Get(key)
	if !ok {
		return
	}
	return vi.(ByteView), ok
}


func (c *cache) addWithExpiration(key string, value ByteView, expirationTime time.Time) {
	c.lruCacheLazyLoadIfNeed()
	c.lruCache.AddWithExpiration(key, value, expirationTime)

}

func (c *cache) delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lruCache == nil {
		return true
	}
	return c.lruCache.Delete(key)
}

