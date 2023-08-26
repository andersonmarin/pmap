# Partitioned Map

`pmap` is an open-source Go package that provides a concurrent map implementation with built-in key partitioning. This
allows for efficient concurrent access to the map while minimizing locking. The package is designed to handle
scenarios where multiple goroutines need to read from and write to a shared map concurrently.

## Features

- **Concurrent Access:** The `pmap` allows multiple goroutines to access and manipulate the map concurrently,
  with each partition being guarded by its own read-write mutex.

- **Key Partitioning:** The package utilizes a user-provided partitioning function to determine which partition a given
  key belongs to. This ensures that keys that are likely to be accessed concurrently are placed in the same partition,
  reducing contention.

- **Dynamic Sizing:** Each partitioned map can be initialized with a defined number of partitions and an estimated size
  for each partition. This allows for dynamic sizing and efficient memory usage.

## Usage

### Initialization

You can create a new `PartitionedMap` instance using the `NewPartitionedMap` function and use methods for reading and
writing key-value pairs:

```go
package main

import "github.com/andersonmarin/pmap"

func main() {
	partitions := 16
	partitionSize := 100
	partitionFinder := func(key string) int {
		// Define your partitioning logic here
		// Return the partition index for the given key
		return 0
	}

	// Create a new partitioned map
	m := pmap.NewPartitionedMap[string, string](partitions, partitionSize, partitionFinder)

	// Writing
	m.Set("key", "value")

	// Reading
	if value, exists := m.Get("key"); exists {
		// Do something with the value
	} else {
		// Key not found
	}
}

```

## Disclaimer

This repository and its contents are intended for educational and study purposes only. The code and documentation
provided are not intended for production use. Use at your own risk.