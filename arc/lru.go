package arc

import (
	"container/heap"
)

type lruNode struct {
	key     string
	value   *Page
	lastUse int
	index   int
}

// a wrapper for the node so that the pointer does not invalidate after Swap()
type nodeWrap struct {
	nodeptr *lruNode
}

// heap interface implementation: https://pkg.go.dev/container/heap
type nodeHeap [](*nodeWrap)

func (h nodeHeap) Len() int {
	return len(h)
}

// sort with respect to the timer value
func (h nodeHeap) Less(i, j int) bool {
	return h[i].nodeptr.lastUse < h[j].nodeptr.lastUse
}

func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].nodeptr.index = i
	h[j].nodeptr.index = j
}

func (h *nodeHeap) Push(elem interface{}) {
	n := len(*h)
	pushNode := elem.(*nodeWrap)
	pushNode.nodeptr.index = n
	*h = append(*h, pushNode)
}

func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	retNode := old[n-1]
	old[n-1] = nil
	retNode.nodeptr.index = -1
	*h = old[0 : n-1]
	return retNode
}

// An LRU is a fixed-size in-memory cache with least-recently-used eviction
type LRU struct {
	hasKey map[string](*lruNode)
	heap   nodeHeap
	timer  int

	maxStorage int
	remStorage int
	length     int
	hits       int
	miss       int
}

// NewLRU returns a pointer to a new LRU with a capacity to store limit bytes
func NewLru(limit int) *LRU {
	lru := &LRU{
		make(map[string](*lruNode)),
		make([](*nodeWrap), 0),
		0,
		limit,
		limit,
		0,
		0,
		0,
	}

	heap.Init(&lru.heap)
	return lru
}

// MaxStorage returns the maximum number of bytes this LRU can store
func (lru *LRU) MaxStorage() int {
	return lru.maxStorage
}

// RemainingStorage returns the number of unused bytes available in this LRU
func (lru *LRU) RemainingStorage() int {
	return lru.remStorage
}

// Get returns the value associated with the given key, if it exists.
// This operation counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (lru *LRU) Get(key string) (value *Page, ok bool) {
	addr, ok := lru.hasKey[key]
	// miss
	if !ok {
		lru.miss++
		value = nil
		return
	}
	// hit
	lru.hits++
	value = addr.value

	// update lastUse and fix the heap
	addr.lastUse = lru.timer
	lru.timer++
	heap.Fix(&lru.heap, addr.index)
	return
}

// Remove removes and returns the value associated with the given key, if it exists.
// ok is true if a value was found and false otherwise
func (lru *LRU) Remove(key string) (value *Page, ok bool) {
	addr, ok := lru.hasKey[key]

	// miss
	if !ok {
		value = nil
		return
	}
	// hit
	heap.Remove(&lru.heap, addr.index)
	delete(lru.hasKey, key)
	lru.remStorage += 1
	lru.length--
	value = addr.value
	return
}

// Set associates the given value with the given key, possibly evicting values
// to make room. Returns true if the binding was added successfully, else false.
func (lru *LRU) Set(key string, value *Page) bool {
	addr, ok := lru.hasKey[key]
	if ok {
		addr.value = value
		addr.lastUse = lru.timer
		lru.timer++
		heap.Fix(&lru.heap, addr.index)
		return true
	}

	newNode := &nodeWrap{&lruNode{key, value, lru.timer, -1}}
	lru.timer++
	if lru.remStorage == 0 {
		lru.evict()
	}
	lru.remStorage--
	lru.length++
	heap.Push(&lru.heap, newNode)
	lru.hasKey[key] = newNode.nodeptr
	return true
}

// Len returns the number of bindings in the LRU.
func (lru *LRU) Len() int {
	return lru.length
}

// Stats returns statistics about how many search hits and misses have occurred.
func (lru *LRU) Stats() *Stats {
	return &Stats{lru.hits, lru.miss}
}

func (lru *LRU) evict() (string, *Page) {
	popNode := heap.Pop(&lru.heap).(*nodeWrap).nodeptr
	delete(lru.hasKey, popNode.key)
	lru.remStorage++
	lru.length--
	return popNode.key, popNode.value
}
