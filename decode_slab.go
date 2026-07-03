//go:build !purego

package toml

import (
	"unsafe"
)

// Generic decoding boxes every scalar of the document into an interface{},
// which costs one small heap allocation per value (two for strings: the
// bytes and the header). Those objects are the dominant share of the
// allocations of a generic decode, and their number drives the GC pressure.
//
// The slab allocator below batches them: values are placed into chunked
// arrays and the interfaces are assembled to point into the chunks. Chunks
// are never reused or pooled: once handed out, their memory belongs to the
// decoded document exactly as if each value had been allocated individually,
// and it is reclaimed by the GC when the document values become unreachable.
//
// The unsafe usage is limited to two well-understood constructions, gated
// behind the purego build tag (which selects the plain implementation):
//   - unsafe.String over bytes copied into a private chunk, the same
//     guarantee strings.Clone provides;
//   - assembling an interface{} from the static type word of a template
//     interface and a pointer into a chunk (interior pointers keep their
//     whole chunk alive, so the values are always valid).

// slabAlloc is the per-decoder slab state. It intentionally has no cleanup:
// partially used chunks simply carry over to the next document, and the
// unused tail is wasted at most once per chunk size.
type slabAlloc struct {
	str  []byte          // string bytes
	hdr  []string        // string headers
	f64  []float64       // float64 values
	i64  []int64         // int64 values
	any  []interface{}   // small array values
	shdr [][]interface{} // slice headers
}

// anySlice returns a zeroed []interface{} of length and capacity n, cut from
// a chunk when small: documents made of many short arrays (coordinate pairs)
// otherwise pay one allocation per array. This is plain subslicing; it lives
// with the unsafe slabs because it shares their chunk-pinning trade-off.
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

const (
	slabStrChunk = 4096
	// Strings longer than this get their own allocation, so that one huge
	// string does not waste most of a chunk, and so that a small string kept
	// alive by the caller pins at most slabStrChunk bytes.
	slabStrMax   = 512
	slabValChunk = 256
)

// eface mirrors the runtime layout of an empty interface. The type word of a
// non-nil interface{} holding a T is the same static pointer for every T
// value, so it can be copied from a template.
type eface struct {
	typ unsafe.Pointer
	dat unsafe.Pointer
}

func efaceTypeOf(v interface{}) unsafe.Pointer {
	return (*eface)(unsafe.Pointer(&v)).typ //nolint:gosec // reading the type word of a live interface
}

var (
	efaceStringType  = efaceTypeOf(string(""))
	efaceFloat64Type = efaceTypeOf(float64(0))
	efaceInt64Type   = efaceTypeOf(int64(0))
	efaceSliceType   = efaceTypeOf([]interface{}(nil))
)

// sliceAny boxes a []interface{} into an interface{} without allocating a
// fresh slice header (the conversion otherwise costs one heap object per
// array in the document).
func (s *slabAlloc) sliceAny(v []interface{}) (out interface{}) {
	if len(s.shdr) == 0 {
		s.shdr = make([][]interface{}, slabValChunk)
	}
	s.shdr[0] = v
	e := (*eface)(unsafe.Pointer(&out)) //nolint:gosec // assembling a slice interface
	e.typ = efaceSliceType
	e.dat = unsafe.Pointer(&s.shdr[0])
	s.shdr = s.shdr[1:]
	return out
}

// slabString returns a string with the same contents as b, backed by the
// string chunk when small enough.
func (s *slabAlloc) slabString(b []byte) string {
	n := len(b)
	if n == 0 {
		return ""
	}
	if n > slabStrMax {
		return string(b)
	}
	if len(s.str) < n {
		s.str = make([]byte, slabStrChunk)
	}
	copy(s.str, b)
	out := unsafe.String(&s.str[0], n)
	s.str = s.str[n:]
	return out
}

// stringAny boxes a string already backed by stable memory (an interned key,
// a slabString result, or a plain allocation) into an interface{} without
// allocating a fresh string header.
func (s *slabAlloc) stringAny(v string) (out interface{}) {
	if len(s.hdr) == 0 {
		s.hdr = make([]string, slabValChunk)
	}
	s.hdr[0] = v
	e := (*eface)(unsafe.Pointer(&out)) //nolint:gosec // assembling a string interface
	e.typ = efaceStringType
	e.dat = unsafe.Pointer(&s.hdr[0])
	s.hdr = s.hdr[1:]
	return out
}

// float64Any boxes a float64 into an interface{} from the float chunk.
func (s *slabAlloc) float64Any(v float64) (out interface{}) {
	if len(s.f64) == 0 {
		s.f64 = make([]float64, slabValChunk)
	}
	s.f64[0] = v
	e := (*eface)(unsafe.Pointer(&out)) //nolint:gosec // assembling a float64 interface
	e.typ = efaceFloat64Type
	e.dat = unsafe.Pointer(&s.f64[0])
	s.f64 = s.f64[1:]
	return out
}

// int64Any boxes an int64 into an interface{} from the int chunk. Small
// values keep using the runtime conversion, which is allocation-free for
// 0..255.
func (s *slabAlloc) int64Any(v int64) (out interface{}) {
	if v >= 0 && v < 256 {
		return v
	}
	if len(s.i64) == 0 {
		s.i64 = make([]int64, slabValChunk)
	}
	s.i64[0] = v
	e := (*eface)(unsafe.Pointer(&out)) //nolint:gosec // assembling an int64 interface
	e.typ = efaceInt64Type
	e.dat = unsafe.Pointer(&s.i64[0])
	s.i64 = s.i64[1:]
	return out
}
