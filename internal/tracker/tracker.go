package tracker

import (
	"fmt"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

type keyKind uint8

const (
	invalid keyKind = iota // also used for the root key
	value
	table
	arrayTable
)

type key string

type builder struct {
	prefix [][]byte
	local  [][]byte
}

func (b *builder) Reset(prefix [][]byte) {
	b.prefix = prefix
	b.local = b.local[:0]
}

// Computes the number of bytes required to store the full key.
func (b *builder) size() int {
	size := len(b.prefix) + len(b.local) - 1
	for _, p := range b.prefix {
		size += len(p)
	}
	for _, p := range b.local {
		size += len(p)
	}
	return size
}

func (b *builder) copy(firstJoin bool, from [][]byte, to []byte) int {
	offset := 0
	for i, p := range from {
		if i > 0 || firstJoin {
			to[offset] = 0x1E
			offset++
		}
		copy(to[offset:], p)
		offset += len(p)
	}
	return offset
}

func (b *builder) MakeKey() key {
	k := make([]byte, b.size())
	b.copy(false, b.prefix, k)
	b.copy(len(b.prefix) > 0, b.local, k)
	return key(k)
}

func (b *builder) Append(k []byte) {
	b.local = append(b.local, k)
}

// Tracks which keys have been seen with which TOML type to flag duplicates
// and mismatches according to the spec.
type Seen struct {
	keys map[key]keyKind

	// scoping from the previous CheckExpression call.
	current [][]byte

	// key builder
	builder builder
}

// CheckExpression takes a top-level node and checks that it does not contain keys
// that have been seen in previous calls, and validates that types are consistent.
func (s *Seen) CheckExpression(node ast.Node) error {
	s.builder.Reset(s.current)
	switch node.Kind {
	case ast.KeyValue:
		return s.checkKeyValue(node)
	case ast.Table:
	case ast.ArrayTable:
	default:
		panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
	}
	return nil
}

func (s *Seen) checkKeyValue(node ast.Node) error {
	it := node.Key()
	for it.Next() {
		s.builder.Append(it.Node().Data)
	}
}
