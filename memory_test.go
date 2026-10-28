package randomizer

import (
	"runtime"
	"testing"
)

var retainedWeights [16]*Weighted

func BenchmarkWeightedRetained(b *testing.B) {
	weights := make([]uint64, 4096)
	for i := range weights {
		weights[i] = uint64(i%7 + 1)
	}
	var next int
	b.ReportAllocs()
	for b.Loop() {
		table, err := NewWeighted(weights)
		if err != nil {
			b.Fatal(err)
		}
		retainedWeights[next%len(retainedWeights)] = table
		next++
	}
	runtime.GC()
}
