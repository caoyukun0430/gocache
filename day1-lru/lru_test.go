package lru

import (
	"reflect"
	"testing"
)

// As Entry.value it must implement Len() method, so we need to define String type
type String string

func (d String) Len() int {
	return len(d)
}

func TestGet(t *testing.T) {
	// maxByte 0 no limit
	lru := New(int64(0), nil)
	lru.Add("key1", String("1234"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + v1 + k2 + v2)
	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	// key1 should be removed
	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
	if _, ok := lru.Get("k3"); !ok {
		t.Fatalf("k3 is not added")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	removedKeys := func(key string, value Value) {
		keys = append(keys, key)
	}
	// max cap 10, when k2 added, k1 removed...
	lru := New(int64(10), removedKeys)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("v2"))
	lru.Add("k3", String("v3"))
	lru.Add("k4", String("v4"))

	expect := []string{"key1", "k2"}
	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %+v", expect)
	}
}
