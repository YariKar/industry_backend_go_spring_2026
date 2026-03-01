package main

type Cache[K comparable, V any] struct {
	data map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	var m map[K]V
	if capacity > 0 {
		m = make(map[K]V, capacity)
	}
	return &Cache[K, V]{data: m}
}

func (c *Cache[K, V]) Set(key K, value V) {
	if c.data == nil {
		return
	}
	c.data[key] = value
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	if c.data == nil {
		var zero V
		return zero, false
	}
	val, ok := c.data[key]
	return val, ok
}
