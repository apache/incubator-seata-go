package uuid

import (
	"fmt"
	"testing"
)

func BenchmarkNextID(b *testing.B) {
	NextID()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := NextID()
			fmt.Println(id)
		}
	})
}
