# 7 Days Go Distributed Cache from Scratch

Gocache studies and simplifies the [gocache](https://github.com/golang/groupcache) implementation, but overall keeps most of the features:

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