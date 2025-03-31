package gocache

import (
	"fmt"
	"gocache/lru"

	"sync"
)

// cache wrapped lru.cache with mutex lock for concurrent run
type cache struct {
	mu       sync.Mutex // mutual exclusive lock
	lru      *lru.Cache
	maxBytes int64 //maxbytes
}

// private func accessible within package
// add and get wrapped lru Add and Get
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Lazy initialization
	if c.lru == nil {
		c.lru = lru.New(c.maxBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		fmt.Printf("lru.Get %s\n", v)
		return v.(ByteView), ok
	}

	return
}
