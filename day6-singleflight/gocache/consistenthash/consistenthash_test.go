package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	// simple hash func so we know the hash value
	hash := New(3, func(key []byte) uint32 {
		// str to int
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	// Given the above hash function, this will give replicas with "hashes":
	// each node has three replcias 02 12 22; 04 14 24; 06 16 26
	hash.Add("2", "4", "6")

	// map between key and physical node it belongs
	testCases := map[string]string{
		"2":  "2",
		"11": "2", // 12
		"23": "4", // 24
		"26": "6", // 26
		"28": "2", // 02
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// Adds 8, 18, 28
	hash.Add("8")

	// 28 should now map to 8.
	testCases["28"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// remove node 8, 27 belongs to 2 again
	hash.Remove("8")
	delete(testCases, "28")
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

}
