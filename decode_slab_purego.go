//go:build purego

package toml

// slabAlloc is the allocation-batching state used by generic decoding. The
// purego implementation keeps no state and performs the plain allocations;
// see decode_slab.go for the batched version and the rationale.
type slabAlloc struct{}

func (s *slabAlloc) slabString(b []byte) string {
	return string(b)
}

func (s *slabAlloc) anySlice(n int) []interface{} {
	return make([]interface{}, n)
}

func (s *slabAlloc) sliceAny(v []interface{}) interface{} {
	return v
}

func (s *slabAlloc) stringAny(v string) interface{} {
	return v
}

func (s *slabAlloc) float64Any(v float64) interface{} {
	return v
}

func (s *slabAlloc) int64Any(v int64) interface{} {
	return v
}
