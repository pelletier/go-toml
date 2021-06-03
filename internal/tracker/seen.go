package tracker

import (
	"bytes"
	"fmt"

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
	nextID     int
}

type entry struct {
	id       int
	parent   int
	name     []byte
	kind     keyKind
	explicit bool
}

// Remove all descendent of node at position idx.
func (s *SeenTracker) clear(idx int) {
	p := s.entries[idx].id
	rest := clear(p, s.entries[idx+1:])
	s.entries = s.entries[:idx+1+len(rest)]
}

func clear(parentID int, entries []entry) []entry {
	for i := 0; i < len(entries); {
		if entries[i].parent == parentID {
			id := entries[i].id
			copy(entries[i:], entries[i+1:])
			entries = entries[:len(entries)-1]
			rest := clear(id, entries[i:])
			entries = entries[:i+len(rest)]
		} else {
			i++
		}
	}
	return entries
}

func (s *SeenTracker) create(parentIdx int, name []byte, kind keyKind, explicit bool) int {
	parentID := s.id(parentIdx)

	idx := len(s.entries)
	s.entries = append(s.entries, entry{
		id:       s.nextID,
		parent:   parentID,
		name:     name,
		kind:     kind,
		explicit: explicit,
	})
	s.nextID++
	return idx
}

// CheckExpression takes a top-level node and checks that it does not contain keys
// that have been seen in previous calls, and validates that types are consistent.
func (s *SeenTracker) CheckExpression(node *ast.Node) error {
	if s.entries == nil {
		// s.entries = make([]entry, 0, 8)
		// Skip ID = 0 to remove the confusion between nodes whose parent has
		// id 0 and root nodes (parent id is 0 because it's the zero value).
		s.nextID = 1
		// Start unscoped, so idx is negative.
		s.currentIdx = -1
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

func (s *SeenTracker) checkTable(node *ast.Node) error {
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

	return nil
}

func (s *SeenTracker) checkArrayTable(node *ast.Node) error {
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

	return nil
}

func (s *SeenTracker) checkKeyValue(node *ast.Node) error {
	it := node.Key()

	parentIdx := s.currentIdx

	for it.Next() {
		k := it.Node().Data

		idx := s.find(parentIdx, k)

		if idx >= 0 {
			if s.entries[idx].kind != tableKind {
				return fmt.Errorf("toml: expected %s to be a table, not a %s", string(k), s.entries[idx].kind)
			}
		} else {
			idx = s.create(parentIdx, k, tableKind, false)
		}
		parentIdx = idx
	}

	kind := valueKind

	if node.Value().Kind == ast.InlineTable {
		kind = tableKind
	}
	s.entries[parentIdx].kind = kind

	return nil
}

func (s *SeenTracker) id(idx int) int {
	if idx >= 0 {
		return s.entries[idx].id
	}
	return 0
}

func (s *SeenTracker) find(parentIdx int, k []byte) int {
	parentID := s.id(parentIdx)

	for i := parentIdx + 1; i < len(s.entries); i++ {
		if s.entries[i].parent == parentID && bytes.Equal(s.entries[i].name, k) {
			return i
		}
	}

	return -1
}
