package pmap

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func partitionFinder(key string) int {
	var sum int
	for i, c := range key {
		sum += int(c) * (i + 1)
	}
	return sum
}

func newTestMap(partitions, size int) *PartitionedMap[string, string] {
	return NewPartitionedMap[string, string](partitions, size, func(key string) int {
		return len(key)
	})
}

func TestNewPartitionedMap(t *testing.T) {
	const (
		partitions = 2
		size       = 7
	)

	m := newTestMap(partitions, size)

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
}

func TestSetAndGet(t *testing.T) {
	m := newTestMap(2, 7)

	m.Set("ab", "hello")
	if got := m.mp[0]; !reflect.DeepEqual(got, map[string]string{"ab": "hello"}) {
		t.Errorf("unexpected mp[0] = %v", got)
	}

	m.Set("abc", "world")
	if got := m.mp[1]; !reflect.DeepEqual(got, map[string]string{"abc": "world"}) {
		t.Errorf("unexpected mp[1] = %v", got)
	}

	if got, ok := m.Get("ab"); !ok || got != "hello" {
		t.Errorf("Get(ab) = %v, %v; want hello, true", got, ok)
	}

	if got, ok := m.Get("abc"); !ok || got != "world" {
		t.Errorf("Get(abc) = %v, %v; want world, true", got, ok)
	}

	if _, ok := m.Get("not"); ok {
		t.Errorf("Get(not) = _, true; want _, false")
	}
}

func TestDelete(t *testing.T) {
	m := newTestMap(2, 0)

	m.Set("ab", "hello")
	m.Set("abc", "world")

	if existed := m.Delete("ab"); !existed {
		t.Errorf("Delete(ab) = false; want true")
	}

	if _, ok := m.Get("ab"); ok {
		t.Errorf("Get(ab) after Delete = _, true; want _, false")
	}

	// Deleting a key that does not exist must return false.
	if existed := m.Delete("ab"); existed {
		t.Errorf("Delete(ab) second time = true; want false")
	}

	// Other keys must be unaffected.
	if got, ok := m.Get("abc"); !ok || got != "world" {
		t.Errorf("Get(abc) after unrelated Delete = %v, %v; want world, true", got, ok)
	}
}

func TestLen(t *testing.T) {
	m := newTestMap(4, 0)

	if got := m.Len(); got != 0 {
		t.Errorf("Len() = %d; want 0", got)
	}

	keys := []string{"a", "bb", "ccc", "dddd", "eeeee"}
	for _, k := range keys {
		m.Set(k, k)
	}

	if got, want := m.Len(), len(keys); got != want {
		t.Errorf("Len() = %d; want %d", got, want)
	}

	m.Delete("a")

	if got, want := m.Len(), len(keys)-1; got != want {
		t.Errorf("Len() after Delete = %d; want %d", got, want)
	}
}

func TestRange(t *testing.T) {
	m := newTestMap(4, 0)
	input := map[string]string{
		"a":    "1",
		"bb":   "2",
		"ccc":  "3",
		"dddd": "4",
	}
	for k, v := range input {
		m.Set(k, v)
	}

	collected := make(map[string]string)
	m.Range(func(k, v string) bool {
		collected[k] = v
		return true
	})

	if !reflect.DeepEqual(collected, input) {
		t.Errorf("Range collected %v; want %v", collected, input)
	}
}

func TestRangeEarlyStop(t *testing.T) {
	m := newTestMap(4, 0)
	for i := 0; i < 10; i++ {
		m.Set(fmt.Sprintf("%d", i), fmt.Sprintf("v%d", i))
	}

	count := 0
	m.Range(func(k, v string) bool {
		count++
		return count < 3
	})

	if count != 3 {
		t.Errorf("Range visited %d entries after early stop; want 3", count)
	}
}

func TestConcurrentSetDelete(t *testing.T) {
	m := NewPartitionedMap[string, int](runtime.NumCPU(), 0, func(key string) int {
		return partitionFinder(key)
	})

	var wg sync.WaitGroup
	const n = 1000

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)
			m.Set(key, idx)
			m.Delete(key)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentLen(t *testing.T) {
	m := NewPartitionedMap[string, int](runtime.NumCPU(), 0, func(key string) int {
		return partitionFinder(key)
	})

	var wg sync.WaitGroup
	const n = 500

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m.Set(fmt.Sprintf("key-%d", idx), idx)
			_ = m.Len()
		}(i)
	}

	wg.Wait()
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
				m[fmt.Sprintf("%d", index)] = index
				mx.Unlock()
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
		m := NewPartitionedMap[string, int](runtime.NumCPU(), 0, partitionFinder)
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

func BenchmarkMapGet(b *testing.B) {
	b.Run("benchmark standard map get", func(b *testing.B) {
		var (
			wg sync.WaitGroup
			mx sync.RWMutex
		)
		m := make(map[string]int)
		for i := 0; i < b.N; i++ {
			m[fmt.Sprintf("%d", i)] = i
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				mx.RLock()
				v, ok := m[fmt.Sprintf("%d", index)]
				mx.RUnlock()
				if !ok && v == 0 {
					b.Fail()
				}
			}(i)
		}

		wg.Wait()
	})

	b.Run("benchmark sync map get", func(b *testing.B) {
		var (
			wg sync.WaitGroup
			m  sync.Map
		)
		for i := 0; i < b.N; i++ {
			m.Store(fmt.Sprintf("%d", i), i)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				v, ok := m.Load(fmt.Sprintf("%d", index))
				if !ok && v == 0 {
					b.Fail()
				}
			}(i)
		}

		wg.Wait()
	})

	b.Run("benchmark partitioned map get", func(b *testing.B) {
		var wg sync.WaitGroup
		m := NewPartitionedMap[string, int](runtime.NumCPU(), 0, partitionFinder)
		for i := 0; i < b.N; i++ {
			m.Set(fmt.Sprintf("%d", i), i)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				v, ok := m.Get(fmt.Sprintf("%d", index))
				if !ok && v == 0 {
					b.Fail()
				}
			}(i)
		}

		wg.Wait()
	})
}

func BenchmarkMapGetSet(b *testing.B) {
	b.Run("benchmark standard map get set", func(b *testing.B) {
		var mx sync.RWMutex
		m := make(map[string]int)
		c := make(chan int, 0xff)

		b.ReportAllocs()
		b.ResetTimer()

		go func() {
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					mx.Lock()
					m[fmt.Sprintf("%d", index)] = index
					mx.Unlock()
					c <- index
				}(i)
			}
			wg.Wait()
			close(c)
		}()

		var wg sync.WaitGroup
		for i := range c {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				mx.RLock()
				v, ok := m[fmt.Sprintf("%d", index)]
				mx.RUnlock()
				if !ok && v == 0 {
					b.Fail()
				}
			}(i)
		}
		wg.Wait()
	})

	b.Run("benchmark sync map get set", func(b *testing.B) {
		var m sync.Map
		c := make(chan int, 0xff)

		b.ReportAllocs()
		b.ResetTimer()

		go func() {
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					m.Store(fmt.Sprintf("%d", index), index)
					c <- index
				}(i)
			}
			wg.Wait()
			close(c)
		}()

		var wg sync.WaitGroup
		for i := range c {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				v, ok := m.Load(fmt.Sprintf("%d", index))
				if !ok && v == 0 {
					b.Fail()
				}
			}(i)
		}
		wg.Wait()
	})

	b.Run("benchmark partitioned map get set", func(b *testing.B) {
		m := NewPartitionedMap[string, int](runtime.NumCPU(), 0, partitionFinder)
		c := make(chan int, 0xff)

		b.ReportAllocs()
		b.ResetTimer()

		go func() {
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					m.Set(fmt.Sprintf("%d", index), index)
					c <- index
				}(i)
			}
			wg.Wait()
			close(c)
		}()

		var wg sync.WaitGroup
		for i := range c {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				v, ok := m.Get(fmt.Sprintf("%d", index))
				if !ok && v == 0 {
					b.Fail()
				}
			}(i)
		}
		wg.Wait()
	})
}
