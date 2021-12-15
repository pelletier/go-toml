package tracker

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

type keyKind uint8

const (
	invalidKind keyKind = iota
	valueKind
	tableKind
	arrayTableKind
)

func (k keyKind) String() string {
	switch k {
	case invalidKind:
		return "invalid"
	case valueKind:
		return "value"
	case tableKind:
		return "table"
	case arrayTableKind:
		return "array table"
	}
	panic("missing keyKind string mapping")
}

// SeenTracker tracks which keys have been seen with which TOML type to flag
// duplicates and mismatches according to the spec.
//
// Each node in the visited tree is represented by an entry. Each entry has an
// identifier, which is provided by a counter. Entries are stored in the array
// entries. As new nodes are discovered (referenced for the first time in the
// TOML document), entries are created and appended to the array. An entry
// points to its parent using its id.
//
// To find whether a given key (sequence of []byte) has already been visited,
// the entries are linearly searched, looking for one with the right name and
// parent id.
//
// Given that all keys appear in the document after their parent, it is
// guaranteed that all descendants of a node are stored after the node, this
// speeds up the search process.
//
// When encountering [[array tables]], the descendants of that node are removed
// to allow that branch of the tree to be "rediscovered". To maintain the
// invariant above, the deletion process needs to keep the order of entries.
// This results in more copies in that case.
type SeenTracker struct {
	entries    []entry
	currentIdx int
	lastIdx    int
}

var pool sync.Pool

func (s *SeenTracker) reset() {
	// Start unscoped, so idx is negative.
	s.currentIdx = -1
	s.lastIdx = -1
	s.entries = s.entries[:0]
}

type entry struct {
	parent   int
	name     []byte
	kind     keyKind
	explicit bool
}

// Remove all descendants of node at position idx.
func (s *SeenTracker) clear(idx int) {
	if idx >= len(s.entries) {
		return
	}
	for i := idx + 1; i < len(s.entries); i++ {
		if s.entries[i].parent == idx {
			s.entries[i].explicit = false
			s.entries[i].parent = -1
			s.entries[i].name = nil
			s.entries[i].kind = invalidKind
			s.clear(i)
		}
	}
}

func (s *SeenTracker) create(parentIdx int, name []byte, kind keyKind, explicit bool) int {
	idx := len(s.entries)
	s.entries = append(s.entries, entry{
		parent:   parentIdx,
		name:     name,
		kind:     kind,
		explicit: explicit,
	})
	s.lastIdx = idx
	return idx
}

// CheckExpression takes a top-level node and checks that it does not contain
// keys that have been seen in previous calls, and validates that types are
// consistent.
func (s *SeenTracker) CheckExpression(node *ast.Node) error {
	if s.entries == nil {
		s.reset()
	}
	switch node.Kind {
	case ast.KeyValue:
		return s.checkKeyValue(node)
	case ast.Table:
		return s.checkTable(node)
	case ast.ArrayTable:
		return s.checkArrayTable(node)
	default:
		panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
	}
}

func (s *SeenTracker) setExplicitFlag(parentIdx int) {
	offset := parentIdx + 1
	for idx, e := range s.entries[offset:] {
		if offset+idx > s.lastIdx {
			return
		}
		if e.parent == parentIdx {
			s.entries[offset+idx].explicit = true
			s.setExplicitFlag(offset + idx)
		}
	}
}

func (s *SeenTracker) checkTable(node *ast.Node) error {
	if s.currentIdx >= 0 {
		s.setExplicitFlag(s.currentIdx)
	}

	it := node.Key()

	parentIdx := -1

	// This code is duplicated in checkArrayTable. This is because factoring
	// it in a function requires to copy the iterator, or allocate it to the
	// heap, which is not cheap.
	for it.Next() {
		if it.IsLast() {
			break
		}

		k := it.Node().Data

		idx := s.find(parentIdx, k)

		if idx < 0 {
			idx = s.create(parentIdx, k, tableKind, false)
		} else {
			entry := s.entries[idx]
			if entry.kind == valueKind {
				return fmt.Errorf("toml: expected %s to be a table, not a %s", string(k), entry.kind)
			}
		}
		parentIdx = idx
	}

	k := it.Node().Data
	idx := s.find(parentIdx, k)

	if idx >= 0 {
		kind := s.entries[idx].kind
		if kind != tableKind {
			return fmt.Errorf("toml: key %s should be a table, not a %s", string(k), kind)
		}
		if s.entries[idx].explicit {
			return fmt.Errorf("toml: table %s already exists", string(k))
		}
		s.entries[idx].explicit = true
	} else {
		idx = s.create(parentIdx, k, tableKind, true)
	}

	s.currentIdx = idx
	s.lastIdx = idx

	return nil
}

func (s *SeenTracker) checkArrayTable(node *ast.Node) error {
	if s.currentIdx >= 0 {
		s.setExplicitFlag(s.currentIdx)
	}

	it := node.Key()

	parentIdx := -1

	for it.Next() {
		if it.IsLast() {
			break
		}

		k := it.Node().Data

		idx := s.find(parentIdx, k)

		if idx < 0 {
			idx = s.create(parentIdx, k, tableKind, false)
		} else {
			entry := s.entries[idx]
			if entry.kind == valueKind {
				return fmt.Errorf("toml: expected %s to be a table, not a %s", string(k), entry.kind)
			}
		}

		parentIdx = idx
	}

	k := it.Node().Data
	idx := s.find(parentIdx, k)

	if idx >= 0 {
		kind := s.entries[idx].kind
		if kind != arrayTableKind {
			return fmt.Errorf("toml: key %s already exists as a %s,  but should be an array table", kind, string(k))
		}
		s.clear(idx)
	} else {
		idx = s.create(parentIdx, k, arrayTableKind, true)
	}

	s.currentIdx = idx
	s.lastIdx = idx

	return nil
}

func (s *SeenTracker) checkKeyValue(node *ast.Node) error {
	parentIdx := s.currentIdx
	it := node.Key()

	for it.Next() {
		k := it.Node().Data

		idx := s.find(parentIdx, k)

		if idx < 0 {
			idx = s.create(parentIdx, k, tableKind, false)
		} else {
			entry := s.entries[idx]
			if it.IsLast() {
				return fmt.Errorf("toml: key %s is already defined", string(k))
			} else if entry.kind != tableKind {
				return fmt.Errorf("toml: expected %s to be a table, not a %s", string(k), entry.kind)
			} else if entry.explicit {
				return fmt.Errorf("toml: cannot redefine table %s that has already been explicitly defined", string(k))
			}
		}

		parentIdx = idx
	}

	s.entries[parentIdx].kind = valueKind

	value := node.Value()

	switch value.Kind {
	case ast.InlineTable:
		return s.checkInlineTable(value)
	case ast.Array:
		return s.checkArray(value)
	}

	return nil
}

func (s *SeenTracker) checkArray(node *ast.Node) error {
	it := node.Children()
	for it.Next() {
		n := it.Node()
		switch n.Kind {
		case ast.InlineTable:
			err := s.checkInlineTable(n)
			if err != nil {
				return err
			}
		case ast.Array:
			err := s.checkArray(n)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SeenTracker) checkInlineTable(node *ast.Node) error {
	if pool.New == nil {
		pool.New = func() interface{} {
			return &SeenTracker{}
		}
	}

	s = pool.Get().(*SeenTracker)
	s.reset()

	it := node.Children()
	for it.Next() {
		n := it.Node()
		err := s.checkKeyValue(n)
		if err != nil {
			return err
		}
	}

	// As inline tables are self-contained, the tracker does not
	// need to retain the details of what they contain. The
	// keyValue element that creates the inline table is kept to
	// mark the presence of the inline table and prevent
	// redefinition of its keys: check* functions cannot walk into
	// a value.
	pool.Put(s)
	return nil
}

func (s *SeenTracker) find(parentIdx int, k []byte) int {
	for i := parentIdx + 1; i < len(s.entries); i++ {
		if s.entries[i].parent == parentIdx && bytes.Equal(s.entries[i].name, k) {
			return i
		}
	}

	return -1
}
