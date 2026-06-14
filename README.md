# pmap

[![CI](https://github.com/andersonmarin/pmap/actions/workflows/ci.yml/badge.svg)](https://github.com/andersonmarin/pmap/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/andersonmarin/pmap.svg)](https://pkg.go.dev/github.com/andersonmarin/pmap)

`pmap` is a zero-dependency Go library providing a generic concurrent map with automatic key sharding.

Instead of protecting a single map with one global mutex — which becomes a bottleneck under high concurrency — pmap divides entries into independent partitions, each guarded by its own `sync.RWMutex`. This reduces lock contention significantly for write-heavy workloads.

## When to use pmap

| Scenario | Recommended |
|---|---|
| High read-to-write ratio, stable key set | `sync.Map` |
| Write-heavy or balanced read/write concurrency | **pmap** |
| Single goroutine | `map[K]V` |

## Installation

```bash
go get github.com/andersonmarin/pmap
```

Requires Go 1.21+.

## Usage

### Create a map

```go
import (
    "runtime"
    "github.com/andersonmarin/pmap"
)

m := pmap.NewPartitionedMap[string, int](
    runtime.NumCPU(), // number of partitions
    100,              // initial capacity hint per partition
    func(key string) int {
        // Distribute keys across partitions.
        // Any integer is valid; pmap takes it modulo the partition count.
        var sum int
        for i, c := range key {
            sum += int(c) * (i + 1)
        }
        return sum
    },
)
```

### Operations

```go
// Write
m.Set("hello", 42)

// Read
if value, ok := m.Get("hello"); ok {
    fmt.Println(value) // 42
}

// Delete — reports whether the key existed
existed := m.Delete("hello") // true

// Count entries (non-atomic snapshot across partitions)
fmt.Println(m.Len())

// Iterate — return false to stop early
m.Range(func(key string, value int) bool {
    fmt.Printf("%s → %d\n", key, value)
    return true
})
```

> **Note:** Inside a `Range` callback, calling `Get` is safe. Calling `Set` or `Delete` on the same map will deadlock, because the partition lock is already held.

## Choosing a partition count

A good starting point is `runtime.NumCPU()`. For very write-heavy workloads, try `runtime.NumCPU() * 4`. More partitions lower contention but allocate more memory. Profile with `go test -bench=. -benchmem` to find the sweet spot for your workload.

## Benchmarks

Run on an 8-core machine (`go test -bench=. -count=5 -benchmem`):

```
BenchmarkMapSet/standard_map     1912 ns/op
BenchmarkMapSet/sync.Map          —
BenchmarkMapSet/pmap               678 ns/op   (~3× faster than map+RWMutex)

BenchmarkMapGet/standard_map      320 ns/op
BenchmarkMapGet/sync.Map          551 ns/op
BenchmarkMapGet/pmap              757 ns/op

BenchmarkMapGetSet/standard_map  4330 ns/op
BenchmarkMapGetSet/sync.Map      3200 ns/op
BenchmarkMapGetSet/pmap          2983 ns/op   (fastest for mixed workloads)
```

Reproduce locally:

```bash
go test -bench=. -benchmem ./...
```

## License

MIT — see [LICENSE](LICENSE).
