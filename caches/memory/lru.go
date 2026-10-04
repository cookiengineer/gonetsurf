// Package memory provides a bounded in-memory LRU cache.
package memory

import (
	"container/list"
	"sync"
)

// Cache is a concurrency-safe least-recently-used cache.
type Cache[Key comparable, Value any] struct {
	capacity int

	mutex    sync.Mutex
	elements map[Key]*list.Element
	order    *list.List
}

type entry[Key comparable, Value any] struct {
	key   Key
	value Value
}

// New returns a cache holding at most capacity entries. A capacity of zero
// disables storage.
func New[Key comparable, Value any](capacity int) *Cache[Key, Value] {
	return &Cache[Key, Value]{
		capacity: capacity,
		elements: make(map[Key]*list.Element),
		order:    list.New(),
	}
}

// Get returns the value for key and marks it as recently used.
func (cache *Cache[Key, Value]) Get(key Key) (Value, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	element, ok := cache.elements[key]
	if !ok {
		var zero Value
		return zero, false
	}
	cache.order.MoveToFront(element)
	return element.Value.(*entry[Key, Value]).value, true
}

// Put stores a value, evicting the least recently used entry when full.
func (cache *Cache[Key, Value]) Put(key Key, value Value) {
	if cache.capacity <= 0 {
		return
	}
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if element, ok := cache.elements[key]; ok {
		element.Value.(*entry[Key, Value]).value = value
		cache.order.MoveToFront(element)
		return
	}
	element := cache.order.PushFront(&entry[Key, Value]{key: key, value: value})
	cache.elements[key] = element
	for cache.order.Len() > cache.capacity {
		oldest := cache.order.Back()
		if oldest == nil {
			break
		}
		cache.order.Remove(oldest)
		delete(cache.elements, oldest.Value.(*entry[Key, Value]).key)
	}
}

// Remove deletes an entry.
func (cache *Cache[Key, Value]) Remove(key Key) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if element, ok := cache.elements[key]; ok {
		cache.order.Remove(element)
		delete(cache.elements, key)
	}
}

// Len returns the number of stored entries.
func (cache *Cache[Key, Value]) Len() int {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	return cache.order.Len()
}
