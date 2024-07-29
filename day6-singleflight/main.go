package main

import (
	"flag"
	"fmt"
	"gocache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":   "630",
	"Jack":  "589",
	"Sam":   "567",
	"Alice": "110",
}

// each group has a httpPool, which contains a hashRing
func createGroup() *gocache.Group {
	return gocache.NewGroup("students", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) {
			fmt.Printf("[SlowDB] search key %s\n", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s key not exist", key)
		}))
}

// register all nodes into the pool
func startCacheServer(addr string, addrs []string, group *gocache.Group) {
	pool := gocache.NewHTTPPool(addr)
	pool.Add(addrs...)
	group.RegisterNodes(pool)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], pool))
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
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
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
	startCacheServer(addrMap[port], []string(addrs), group)
}
