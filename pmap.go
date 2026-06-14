// Package pmap provides a generic concurrent map with key-based sharding.
// Instead of protecting a single map with one global mutex, pmap divides entries
// into independent partitions each guarded by its own RWMutex. This reduces lock
// contention for workloads with many concurrent writers.
package pmap

import "sync"

// PartitionFinder maps a key to a partition index. The index may be any integer;
// pmap takes it modulo the partition count to select the actual partition.
type PartitionFinder[K comparable] func(key K) int

// PartitionedMap is a generic concurrent map sharded into fixed partitions.
// The zero value is not usable; create one with NewPartitionedMap.
type PartitionedMap[K comparable, V any] struct {
	mx []sync.RWMutex
	mp []map[K]V
	pf PartitionFinder[K]
}

// NewPartitionedMap creates a PartitionedMap with the given number of partitions.
// size is the initial capacity hint per partition (0 is valid).
// partitionFinder determines which partition owns a key; pmap takes its return
// value modulo the partition count, so any integer is a valid result.
func NewPartitionedMap[K comparable, V any](partitions, size int, partitionFinder func(key K) int) *PartitionedMap[K, V] {
	m := PartitionedMap[K, V]{
		mx: make([]sync.RWMutex, partitions),
		mp: make([]map[K]V, partitions),
		pf: partitionFinder,
	}

	for i := 0; i < partitions; i++ {
		m.mp[i] = make(map[K]V, size)
	}

	return &m
}

// Get returns the value stored under key and true. If key is absent it returns
// the zero value and false. Safe for concurrent use.
func (m *PartitionedMap[K, V]) Get(key K) (V, bool) {
	p := m.pf(key) % len(m.mp)

	m.mx[p].RLock()
	defer m.mx[p].RUnlock()

	value, ok := m.mp[p][key]
	return value, ok
}

// Set stores value under key, overwriting any existing value.
// Safe for concurrent use.
func (m *PartitionedMap[K, V]) Set(key K, value V) {
	p := m.pf(key) % len(m.mp)

	m.mx[p].Lock()
	defer m.mx[p].Unlock()

	m.mp[p][key] = value
}

// Delete removes key from the map and reports whether the key was present.
// Safe for concurrent use.
func (m *PartitionedMap[K, V]) Delete(key K) bool {
	p := m.pf(key) % len(m.mp)

	m.mx[p].Lock()
	defer m.mx[p].Unlock()

	_, ok := m.mp[p][key]
	delete(m.mp[p], key)
	return ok
}

// Len returns the total number of entries across all partitions. Because
// partitions are locked individually, the result is a non-atomic snapshot
// and may be stale by the time it is used.
func (m *PartitionedMap[K, V]) Len() int {
	var total int
	for i := range m.mp {
		m.mx[i].RLock()
		total += len(m.mp[i])
		m.mx[i].RUnlock()
	}
	return total
}

// Range calls fn sequentially for each key-value pair present in the map.
// If fn returns false, iteration stops early. The map is locked one partition
// at a time, so fn must not call Set or Delete on the same map instance —
// that would deadlock. Calling Get from fn is safe.
func (m *PartitionedMap[K, V]) Range(fn func(key K, value V) bool) {
	for i := range m.mp {
		m.mx[i].RLock()
		cont := true
		for k, v := range m.mp[i] {
			if !fn(k, v) {
				cont = false
				break
			}
		}
		m.mx[i].RUnlock()
		if !cont {
			return
		}
	}
}
