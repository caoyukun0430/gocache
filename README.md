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

## Day 5 - Distributed Nodes

What we learnt?

Consistent Hash to select nodes      Yes                                              YES
    |----------> If remote node -------------> HTTP Client send req ------> succeed? -----> Cached in remote node and return value
                    |  NO                                                      ↓ NO 
                    |-----------------------------------------------> handled by local cache

1. Abstract the peerPicker interface and it should be implemented by each HTTP pool to select nodes/clients.

2. Implemented HTTP pool as both server which serverHTTP based on group name and key, as well as client, which contains a hash ring of nodes to be
selected based on keys.

3. The methods to retrieve cache are inside gocache, we need to extend it with peerPicker so that we can either get from
remote nodes or local source, depending on the node which the key belongs to.

## Day 6 - Cache Penetration

What we learnt?

To solve cache Penetration, i.e. there are a large number of get(key) requests in an instant,
and the key is not cached or not cached in the current node. If singleflight is not used, these
requests will be sent to remote nodes or read from the local database, which will cause a sharp increase
in pressure on remote nodes or local databases.

Therefore, we implement the singleflight Do() to make sure concurrent requests to the same key,
the following requests wait for the 1st to finish with c.wg.Wait(), and then reuse the response from 1st request.

## Day 7 - Proto Buffer

What we learnt?

To make HTTP requests more efficient, proto buffer is used to encode/decode request and responses.
The proto buffer request contains group and key element as the original one, and the response proto
buffer is an slice of bytes, similarly to ByteView. But the limitation is that we only use proto
buffer for encoding and decoding, but RPC is not used for communication.

## Day 7.2 - Proto Buffer Improvement

As we seen from Day 7, proto buffer is only used to serialize the resquest and response data
structure, but the communication is still HTTP based.

Now we make the actual GRPC communication
between the client and server. Therefore, we follow what http.go to formulate the GRPC client
and server. Both need to implement the service rpc Get(Request) returns (Response) defined
inside the proto file.
And the client should use grpc.Dial to establish the connection and use Get() method to send
the GRPC requests containing group and key. The grpc server listens on the address and reads
all the info from GRPC request and write the GRPC responses.

```go
func startCacheServerGrpc(addr string, addrs []string, group *gocache.Group) {
	pool := gocache.NewGrpcPool(addr)
	pool.Add(addrs...)
	group.RegisterNodes(pool)
	log.Println("gocache is running at", addr)
	pool.Run()
}

func startAPIServer(apiAddr string, group *gocache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// we just need to extract key from api addr as group httpPool has its parse /<basepath>/<groupname>/<key> required
			key := r.URL.Query().Get("key")
			view, err := group.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	// default arguments
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Gocache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: ":8001",
		8002: ":8002",
		8003: ":8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	// per port/server create a group, api server on port 8003 only
	group := createGroup()
	if api {
		go startAPIServer(apiAddr, group)
	}
	startCacheServerGrpc(addrMap[port], []string(addrs), group)
}
```