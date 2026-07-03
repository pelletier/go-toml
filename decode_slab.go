package toml

// Generic decoding boxes every value of the document into an interface{}.
// Most of those boxes are unavoidable individual allocations, but the
// backing arrays of small []interface{} values (coordinate pairs and other
// short arrays) can be batched: slabAlloc cuts them from a chunk with plain
// subslicing. Chunks are never reused or pooled: once handed out, their
// memory belongs to the decoded document exactly as if each array had been
// allocated individually, and it is reclaimed by the GC when the document
// values become unreachable. The only cost is that a small array pins its
// chunk while alive, bounding waste to one chunk per set of live values.
//
// This file is intentionally plain Go: this codebase never uses unsafe.
type slabAlloc struct {
	any []interface{} // small array values
}

// anySlice returns a zeroed []interface{} of length and capacity n, cut from
// a chunk when small: documents made of many short arrays would otherwise
// pay one allocation per array.
func (s *slabAlloc) anySlice(n int) []interface{} {
	if n == 0 || n > 32 {
		return make([]interface{}, n)
	}
	if len(s.any) < n {
		s.any = make([]interface{}, 1024)
	}
	out := s.any[:n:n]
	s.any = s.any[n:]
	return out
}
