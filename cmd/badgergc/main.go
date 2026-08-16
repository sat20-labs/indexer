//go:build badger

package main

import (
	"flag"
	"fmt"
)

// Badger is not an active indexer backend. Keep this tagged command as an
// explicit no-op so historical build scripts continue to compile without
// implying that production Pebble databases were compacted.
func main() {
	root := flag.String("path", "", "ignored: Badger GC is disabled")
	cacheMB := flag.Int("cache-mb", 64, "ignored: Badger GC is disabled")
	flag.Parse()

	fmt.Printf("GC_NOOP backend=badger path=%s cache_mb=%d reason=backend_disabled\n", *root, *cacheMB)
}
