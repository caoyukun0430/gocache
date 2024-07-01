# 7 Days Go Distributed Cache from Scratch

Gocache studies and simplifies the [gocache](https://github.com/golang/groupcache) implementation, but overall keeps most of the features:

* LRU (least recently used) cache mechanism


## Day 1 - LRU cache

What we learnt?

1. Construct the LRU cache data structure, which is a linked hash map. It's implemented using Golang doubly linked list, which stores
the nodes and the key-node pairs are stored also inside the hashmap.

2. maxByte is defined for LRU deletion.
