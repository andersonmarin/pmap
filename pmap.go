package pmap

import "sync"

type PartitionFinder[K comparable] func(key K) int

type PartitionedMap[K comparable, V any] struct {
	mx []sync.RWMutex
	mp []map[K]V
	pf PartitionFinder[K]
}

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

func (m *PartitionedMap[K, V]) Get(key K) (V, bool) {
	p := m.pf(key) % len(m.mp)

	m.mx[p].RLock()
	defer m.mx[p].RUnlock()

	value, ok := m.mp[p][key]
	return value, ok
}

func (m *PartitionedMap[K, V]) Set(key K, value V) {
	p := m.pf(key) % len(m.mp)

	m.mx[p].Lock()
	defer m.mx[p].Unlock()

	m.mp[p][key] = value
}
