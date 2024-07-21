package gocache

import (
	"fmt"
	"log"
	"sync"
)

// gocache is the main process

// A Getter loads data for a key. It's used as a callback
// once the cache is not hit and need to retrieve from source
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace with unique name, e.g. scores, name
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

// global vars
var (
	mu sync.RWMutex
	// string-group map stores all groups
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, maxBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	// each group has a cache
	group := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{maxBytes: maxBytes},
	}
	groups[name] = group
	return group
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	//  Multiple goroutines can acquire a read lock simultaneously, as long as no goroutine has acquired the write lock
	mu.RLock()
	group := groups[name]

	mu.RUnlock()
	return group
}

// MOST IMPORTANT Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// no hit, retrieve from local source with callback Getter
	return g.getLocal(key)
}

// FOR DISTRIBUTED CASE, we need another wrapper over getLocal as we also can getFromPeer
// we call the defined Getter Get() to get value from local source and store in cache
func (g *Group) getLocal(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)

	if err != nil {
		return ByteView{}, err
	}
	// copy of bytes
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
