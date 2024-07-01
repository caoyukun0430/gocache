package lru

import "container/list"

// Cache is a LRU cache linked hashmap. It is not safe for concurrent access.
// we regulate front is most recently used, end is least recently used
type Cache struct {
	maxBytes  int64
	usedBytes int64
	// doubly linked list
	dLL *list.List
	// key string, val is pointer to element/nodes in DLL
	cache map[string]*list.Element
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value)
}

// entry is data type of DLL node
type Entry struct {
	key   string
	value Value
}

// Value is quite generice, use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		dLL:       list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	if e, ok := c.cache[key]; ok {
		c.dLL.MoveToFront(e)
		// before get e.val we need to cast/make sure type is entry
		// e.Value is list.list.Element.Value .(*entry) is type assertation
		kvPair := e.Value.(*Entry)
		return kvPair.value, true
	}
	return nil, false
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	// check if element exists, if so, update and movetofront, else pushfront
	// map should be updated, and usedByte
	if e, ok := c.cache[key]; ok {
		c.dLL.MoveToFront(e)
		// update value of e
		kvPair := e.Value.(*Entry)
		c.usedBytes += int64(value.Len()) - int64(kvPair.value.Len())
		kvPair.value = value
	} else {
		e := c.dLL.PushFront(&Entry{key, value})
		c.cache[key] = e
		c.usedBytes += int64(len(key)) + int64(value.Len())
	}
	// if exceed maxByte, we remoe LRU
	// align with groupcache, if c.maxBytes == 0 means no limit
	for c.maxBytes != 0 && c.usedBytes > c.maxBytes {
		c.removeLRU()
	}
}

// RemoveOldest removes the LRU item
func (c *Cache) removeLRU() {
	// we need to remove from DLL, delete from map, reduce usedBytes with key+val length
	// execute onEvicted if needed
	if e := c.dLL.Back(); e != nil {
		c.dLL.Remove(e)
		kvPair := e.Value.(*Entry)
		delete(c.cache, kvPair.key)
		c.usedBytes -= int64(len(kvPair.key)) + int64(kvPair.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kvPair.key, kvPair.value)
		}
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.dLL.Len()
}
