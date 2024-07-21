# 7 Days Go Distributed Cache from Scratch

Gocache studies and simplifies the [groupcache](https://github.com/golang/groupcache) implementation, but overall keeps most of the features:

* LRU (least recently used) cache mechanism


## Day 1 - LRU cache

What we learnt?

1. Construct the LRU cache data structure, which is a linked hash map. It's implemented using Golang doubly linked list, which stores
the nodes and the key-node pairs are stored also inside the hashmap.

2. maxByte is defined for LRU deletion.

## Day 2 - Single Node Concurrent Cache

What we learnt?

1. Use sync.Mutex mutual exclusive lock to protect cache Add and Get operation from concurrent go routines, because both changes the structure of linked list.

2. Introduce the read-only ByteView []byte struct for gocache value, ByteView itself uses value receiver to make sure method operates on a copy of the ByteView instance.
And cloneBytes is also used when retrieving cache from local source.

3. Introduce the Getter interface and GetterFunc interface function (similar to http handle and handlerFunc) to simulate local data retrieve. Wrapped all inside
the Group struct and use Group to interace with users and controlls cache retrieve and storage.

```go
func TestGet(t *testing.T) {
	hitCount := make(map[string]int, len(db))
	students := NewGroup("students", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			fmt.Printf("[SlowDB] search key %s\n", key)
			if v, ok := db[key]; ok {
				if _, ok := hitCount[key]; !ok {
					// create entry
					hitCount[key] = 0
				}
				hitCount[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s key not exist", key)
		}))

	for k, v := range db {
		if view, err := students.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		}
		// this get should hit cache
		if _, err := students.Get(k); err != nil || hitCount[k] > 1 {
			t.Fatalf("cache %s exists but not used", k)
		}
	}
	if vBytes, err := students.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be empty, but %s got", vBytes)
	}
}
```

## Day 3 - HTTP Server peer

What we learnt?

1. Construct an HTTP server pool to simulate one peer for cache. Base on rule /<basepath>/<groupname>/<key>,
the cache can be retrieved from a running HTTP server connects to an backend DB/cache.

```go
func main() {
	// create a students group
	gocache.NewGroup("students", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) {
			fmt.Printf("[SlowDB] search key %s\n", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s key not exist", key)
		}))
	addr := "localhost:9999"
	peers := gocache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
```


## Day 4 - Consistent Hash

What we learnt?

1. To save the issue of request key not deterministically lay onto the same node, we need
hash func. So that the same request key always hashed to land on the same node, where
the data requested from the source is cached previously. So that we avoid time consuming source
requests as well as redadunt data.
2. The reason to introduce consistent hash is that when the number of node changes, all the key
hash needs to be recalculated and the cache is invalid, leads to Cache Avalanche! Therefore,
consistent hash will only relocate a small part of data which are affect by new nodes!
3. data skew issue, when the number of nodes is small, there can be the case that some nodes
are heavily cached and no cache lands on the other nodes, leading to unbalance. Therefore, we
scale physical nodes to solve it.
