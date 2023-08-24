package pmap

import (
	"reflect"
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
