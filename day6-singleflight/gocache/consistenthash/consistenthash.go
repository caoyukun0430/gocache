package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Ring constains all hashed keys
type HashRing struct {
	hash     Hash           // custom Hash func
	replicas int            // virtual and phyical node scale
	keys     []int          // Sorted virtual node hash values
	hashMap  map[int]string // vitual node hash value to physical node name
}

// New creates a Map instance
func New(replicas int, hash Hash) *HashRing {
	ring := &HashRing{
		hash:     hash,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	// default Hash func
	if ring.hash == nil {
		ring.hash = crc32.ChecksumIEEE
	}
	return ring
}

// add vitual nodes to ring based on physical node/key
// each node corresponds to these virtiual nodes strconv.Itoa(i) + node
func (ring *HashRing) Add(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < ring.replicas; i++ {
			vHash := int(ring.hash([]byte(strconv.Itoa(i) + node)))
			ring.hashMap[vHash] = node
			ring.keys = append(ring.keys, vHash)
		}
	}
	// sort keys
	sort.Ints(ring.keys)
}

// get physical node of the key
// binary search find the node which hash value first >= key hash value
func (ring *HashRing) Get(key string) string {
	if len(ring.keys) == 0 {
		return ""
	}
	keyHash := int(ring.hash([]byte(key)))
	// binary search
	nodeIdx := sort.Search(len(ring.keys), func(i int) bool {
		return ring.keys[i] >= keyHash
	})
	// in case nodeIdx == len(keys) take mod
	return ring.hashMap[ring.keys[nodeIdx%len(ring.keys)]]
}

// remove physical node, we dont need to sort again as it's sorted in Add()
func (ring *HashRing) Remove(key string) {
	for i := 0; i < ring.replicas; i++ {
		virtualHash := int(ring.hash([]byte(strconv.Itoa(i) + key)))
		// remove it from keys, first find the idx of the key
		virtualIdx := sort.SearchInts(ring.keys, virtualHash)
		ring.keys = append(ring.keys[:virtualIdx], ring.keys[virtualIdx+1:]...)
		// remove from hashmap
		delete(ring.hashMap, virtualHash)
	}
}
