package pmap

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func TestPartitionedMap(t *testing.T) {
	const (
		partitions = 2
		size       = 7
	)

	m := NewPartitionedMap[string, string](partitions, size, func(key string) int {
		return len(key)
	})

	if got := len(m.mx); got != partitions {
		t.Errorf("len(mx) = %d, want %d", got, partitions)
	}

	if got := len(m.mp); got != partitions {
		t.Errorf("len(mp) = %d, want %d", got, partitions)
	}

	if got := len(m.mp[0]); got != 0 {
		t.Errorf("len(mp[0]) = %d, want empty", got)
	}

	if got := len(m.mp[1]); got != 0 {
		t.Errorf("len(mp[1]) = %d, want empty", got)
	}

	m.Set("ab", "hello")
	if got := m.mp[0]; !reflect.DeepEqual(got, map[string]string{"ab": "hello"}) {
		t.Errorf("unexpected mp[0] = %v", got)
	}

	m.Set("abc", "world")
	if got := m.mp[1]; !reflect.DeepEqual(got, map[string]string{"abc": "world"}) {
		t.Errorf("unexpected mp[1] = %v", got)
	}

	if got, ok := m.Get("ab"); !ok || got != "hello" {
		t.Errorf("unexpected Get(ab) = %v, %v", got, ok)
	}

	if got, ok := m.Get("abc"); !ok || got != "world" {
		t.Errorf("unexpected Get(abc) = %v, %v", got, ok)
	}

	if _, ok := m.Get("not"); ok {
		t.Errorf("unexpected Get(abc) = _, %v", ok)
	}
}

func BenchmarkMapSet(b *testing.B) {
	b.Run("benchmark standard map set", func(b *testing.B) {
		var (
			wg sync.WaitGroup
			mx sync.RWMutex
		)
		m := make(map[string]int)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				mx.Lock()
				defer mx.Unlock()
				m[fmt.Sprintf("%d", index)] = index
			}(i)
		}
		wg.Wait()
	})

	b.Run("benchmark sync map set", func(b *testing.B) {
		var (
			wg sync.WaitGroup
			m  sync.Map
		)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				m.Store(fmt.Sprintf("%d", index), index)
			}(i)
		}
		wg.Wait()
	})

	b.Run("benchmark partitioned map set", func(b *testing.B) {
		var wg sync.WaitGroup
		m := NewPartitionedMap[string, int](runtime.NumCPU(), 0, func(key string) int {
			var sum int
			for i, s := range key {
				sum += int(s) * (i + 1)
			}
			return sum
		})
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				m.Set(fmt.Sprintf("%d", index), index)
			}(i)
		}
		wg.Wait()
	})
}
