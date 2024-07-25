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
	picker    PeerPicker
}

// global vars
var (
	mu sync.RWMutex
	// string-group map stores all groups
	groups = make(map[string]*Group)
)

// RegisterNodes registers a PeerPicker for choosing remote nodes
// httpPool is a picker as we register all nodes inside and implemented PickPeer method
func (g *Group) RegisterNodes(picker PeerPicker) {
	if g.picker != nil {
		panic("RegisterNodes called more than once")
	}
	g.picker = picker
}

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

// MOST IMPORTANT Get value for a key from the group
// when it called for remote node, the remote node also has to call its Get and cache the value into remote cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// no hit, retrieve from remote peer OR local source with callback Getter
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	if g.picker != nil {
		// we register peers, we see if the node is remote or not.
		// if remote, we ask remote to send GET request
		if remote, ok := g.picker.PickPeer(key); ok {
			if value, err := g.getFromRemote(remote, key); err == nil {
				return value, nil
			}
		}
	}
	// if no picker registered/no remote node/ remote is myself, we get locally
	return g.getLocal(key)
}

// FOR DISTRIBUTED CASE
// the core idea is that we dont cache remote value, otherwise each node will cache same value redundantly
func (g *Group) getFromRemote(node PeerClient, key string) (ByteView, error) {
	bytes, err := node.Request(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// we call the defined Getter Get() to get value from local source and store in cache
func (g *Group) getLocal(key string) (ByteView, error) {
	fmt.Printf("getLocal key %s\n", key)
	bytes, err := g.getter.Get(key)

	if err != nil {
		return ByteView{}, err
	}
	// copy of bytes
	value := ByteView{b: cloneBytes(bytes)}
	g.addCache(key, value)
	return value, nil
}

// add the retrieved pair to group cache
func (g *Group) addCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
