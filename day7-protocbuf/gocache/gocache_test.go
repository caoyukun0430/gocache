package gocache

import (
	"fmt"
	"reflect"
	"testing"
)

// simulated db
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

func TestGetGroup(t *testing.T) {
	groupName := "scores"
	NewGroup(groupName, 2<<10, GetterFunc(
		func(key string) (bytes []byte, err error) { return }))
	if group := GetGroup(groupName); group == nil || group.name != groupName {
		t.Fatalf("group %s not exist", groupName)
	}

	if group := GetGroup(groupName + "111"); group != nil {
		t.Fatalf("expect nil, but %s got", group.name)
	}
}

// define GetterFunc to simulate retrieving from db
// test 1. if cache is empty, retrieve from db triggered 2. if cache exists, db should not be hit
// by restricting number of hits <= 1
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
