package tracker

import (
	"bytes"
	"fmt"
	"hash/maphash"

	"github.com/pelletier/go-toml/v2/unstable"
)

type keyKind uint8

const (
	invalidKind keyKind = iota
	// valueKind is a regular value (scalar, array, or inline table). It
	// cannot be extended.
	valueKind
	// kvTableKind is a table created implicitly by a dotted key. It can only
	// be extended by other dotted keys.
	kvTableKind
	// tableKind is a table created by a [header]. The explicit flag tells
	// whether the table was created by its own header (true) or as an
	// intermediate step of a longer key (false).
	tableKind
	// arrayTableKind is an array of tables created by [[header]].
	arrayTableKind
	// anonymousKind is an entry that cannot be looked up by name. It serves
	// as the parent of the content of inline tables stored inside arrays.
	anonymousKind
)

func (k keyKind) String() string {
	switch k {
	case invalidKind:
		return "invalid"
	case valueKind:
		return "value"
	case kvTableKind:
		return "kv-table"
	case tableKind:
		return "table"
	case arrayTableKind:
		return "array-table"
	case anonymousKind:
		return "anonymous"
	}
	panic("missing keyKind string mapping")
}

// spillThreshold is the number of children past which a parent's children are
// moved from its sibling chain to the shared hash index. Chains keep small
// tables (the common case) free of any hashing cost; the index keeps huge
// tables O(1) per key.
const spillThreshold = 32

// entry represents a node that has been seen in the document. Its size has a
// direct impact on the performance of unmarshaling documents: keep it as
// small as possible.
type entry struct {
	name       []byte
	parent     int32
	firstChild int32  // head of the child chain, -1 when empty
	next       int32  // next sibling in the parent's chain, -1 at the tail
	childCount uint16 // saturating count of named children
	kind       keyKind
	explicit   bool
}

// spilled reports whether the children of an entry live in the hash index
// instead of its sibling chain. Spilling happens exactly when the child count
// crosses spillThreshold, so the count doubles as the flag.
func (e *entry) spilled() bool {
	return e.childCount > spillThreshold
}

// SeenTracker tracks which keys have been seen with which TOML type to flag
// duplicates and mismatches according to the spec.
//
// Each node in the visited tree is represented by an entry. Each entry has
// an identifier, which is provided by a counter. Entries are stored in the
// array entries. As new nodes are discovered (referenced for the first time
// in the TOML document), entries are created and appended to the array. An
// entry points to its parent using its id. The array is append-only: ids are
// stable for the duration of a document.
//
// The named children of an entry are linked in a sibling chain, so looking a
// key up costs at most the number of keys in its table. Parents whose child
// count crosses spillThreshold have their children moved ("spilled") to a
// shared open-addressing hash index keyed by (parent, name), which keeps
// pathological tables with thousands of keys O(1) per lookup.
//
// When encountering [[array tables]], the keys seen in the previous element
// must not shadow the keys of the new element. Instead of deleting the
// previous element's entries, the array table entry is replaced by a fresh
// entry (with a fresh id): the previous descendants still exist but hang off
// the old id, which no future lookup uses.
type SeenTracker struct {
	entries      []entry
	currentTable int32

	// index is the open-addressing hash table of entry ids (stored as id+1,
	// 0 meaning empty) for the children of spilled parents, keyed by
	// hash(parent, name). Its size is always a power of two. inserted counts
	// used slots for load-factor purposes.
	index    []int32
	inserted int
	seed     maphash.Seed
	seeded   bool
}

// Reset brings the tracker to its initial state, with just a root table, so
// that it can be reused across documents.
func (s *SeenTracker) Reset() {
	s.reset()
}

// reset brings the tracker to its initial state, with just a root table.
func (s *SeenTracker) reset() {
	s.entries = append(s.entries[:0], entry{
		parent:     -1,
		firstChild: -1,
		next:       -1,
		kind:       tableKind,
	})
	s.currentTable = 0
	if s.inserted > 0 {
		clear(s.index)
		s.inserted = 0
	}
}

// hash computes the index hash of a (parent, name) pair. The seeded name hash
// keeps probe sequences unpredictable for untrusted documents.
func (s *SeenTracker) hash(parent int32, name []byte) uint64 {
	h := maphash.Bytes(s.seed, name)
	return h ^ uint64(uint32(parent))*0x9E3779B97F4A7C15
}

// find returns the id of the entry with the given parent and name, or -1.
// Anonymous entries are never returned (they are not linked nor indexed).
func (s *SeenTracker) find(parent int32, name []byte) int32 {
	if len(s.entries) == 0 {
		s.reset()
	}
	p := &s.entries[parent]
	if p.spilled() {
		return s.indexFind(parent, name)
	}
	for id := p.firstChild; id >= 0; id = s.entries[id].next {
		if bytes.Equal(s.entries[id].name, name) {
			return id
		}
	}
	return -1
}

// create appends a new entry and returns its id. Anonymous entries cannot be
// found; every other entry must not already exist under the same parent.
func (s *SeenTracker) create(parent int32, name []byte, kind keyKind, explicit bool) int32 {
	id := int32(len(s.entries)) //nolint:gosec // entry counts are bounded by document size
	s.entries = append(s.entries, entry{
		parent:     parent,
		firstChild: -1,
		next:       -1,
		name:       name,
		kind:       kind,
		explicit:   explicit,
	})
	if kind != anonymousKind {
		s.link(parent, name, id)
	}
	return id
}

// link attaches a new named child to its parent: to the sibling chain for
// regular parents, or to the hash index for spilled ones. Children are
// prepended to the chain, so no tail pointer is needed.
func (s *SeenTracker) link(parent int32, name []byte, id int32) {
	p := &s.entries[parent]
	if p.childCount < 0xFFFF {
		p.childCount++
	}
	if p.spilled() {
		if p.childCount == spillThreshold+1 {
			// The parent just crossed the threshold: move its chain to the
			// hash index. The chain links are left dangling; they are never
			// read again.
			s.spill(parent)
		}
		s.indexInsert(parent, name, id)
		return
	}
	s.entries[id].next = p.firstChild
	p.firstChild = id
}

// spill moves the children of a parent from its sibling chain to the hash
// index.
func (s *SeenTracker) spill(parent int32) {
	if !s.seeded {
		s.seed = maphash.MakeSeed()
		s.seeded = true
	}
	if s.index == nil {
		s.index = make([]int32, 128)
	}
	p := &s.entries[parent]
	for id := p.firstChild; id >= 0; id = s.entries[id].next {
		s.indexInsert(parent, s.entries[id].name, id)
	}
}

// indexFind probes the hash index for the child of a spilled parent.
func (s *SeenTracker) indexFind(parent int32, name []byte) int32 {
	mask := uint64(len(s.index) - 1)
	for i := s.hash(parent, name) & mask; ; i = (i + 1) & mask {
		v := s.index[i]
		if v == 0 {
			return -1
		}
		e := &s.entries[v-1]
		if e.parent == parent && bytes.Equal(e.name, name) {
			return v - 1
		}
	}
}

// indexInsert adds an id to the index under (parent, name), growing the table
// when its load factor reaches 3/4.
func (s *SeenTracker) indexInsert(parent int32, name []byte, id int32) {
	if (s.inserted+1)*4 > len(s.index)*3 {
		s.grow()
	}
	mask := uint64(len(s.index) - 1)
	i := s.hash(parent, name) & mask
	for s.index[i] != 0 {
		i = (i + 1) & mask
	}
	s.index[i] = id + 1
	s.inserted++
}

// indexReplace overwrites the index slot of (parent, name) with a new id. The
// slot must exist.
func (s *SeenTracker) indexReplace(parent int32, name []byte, id int32) {
	mask := uint64(len(s.index) - 1)
	for i := s.hash(parent, name) & mask; ; i = (i + 1) & mask {
		v := s.index[i]
		if v == 0 {
			panic("toml: internal error: indexReplace on missing entry")
		}
		e := &s.entries[v-1]
		if e.parent == parent && bytes.Equal(e.name, name) {
			s.index[i] = id + 1
			return
		}
	}
}

// grow doubles the index and rehashes every used slot.
func (s *SeenTracker) grow() {
	old := s.index
	n := len(old) * 2
	if n == 0 {
		n = 128
	}
	s.index = make([]int32, n)
	mask := uint64(len(s.index) - 1)
	for _, v := range old {
		if v == 0 {
			continue
		}
		e := &s.entries[v-1]
		i := s.hash(e.parent, e.name) & mask
		for s.index[i] != 0 {
			i = (i + 1) & mask
		}
		s.index[i] = v
	}
}

// refreshArrayTable replaces the array table entry id with a fresh entry (and
// id), so that the descendants recorded by the previous element no longer
// shadow the keys of the new element. Returns the new id.
func (s *SeenTracker) refreshArrayTable(id int32) int32 {
	old := s.entries[id]
	nid := int32(len(s.entries)) //nolint:gosec // entry counts are bounded by document size
	s.entries = append(s.entries, entry{
		parent:     old.parent,
		firstChild: -1,
		next:       old.next,
		name:       old.name,
		kind:       arrayTableKind,
		explicit:   true,
	})
	p := &s.entries[old.parent]
	if p.spilled() {
		s.indexReplace(old.parent, old.name, nid)
		return nid
	}
	// Swap the id for nid in the parent's chain.
	if p.firstChild == id {
		p.firstChild = nid
	} else {
		prev := p.firstChild
		for s.entries[prev].next != id {
			prev = s.entries[prev].next
		}
		s.entries[prev].next = nid
	}
	return nid
}

// CheckExpression takes a top-level node and checks that it does not contain
// keys that have been seen in previous calls, and validates that types are
// consistent. It returns true if it is the first time this node's key is
// seen. Useful to clear array tables on first use.
func (s *SeenTracker) CheckExpression(node *unstable.Node) (bool, error) {
	if len(s.entries) == 0 {
		s.reset()
	}
	switch node.Kind {
	case unstable.KeyValue:
		return false, s.checkKeyValue(s.currentTable, node)
	case unstable.Table:
		return s.checkTable(node)
	case unstable.ArrayTable:
		return s.checkArrayTable(node)
	default:
		return false, fmt.Errorf("toml: unexpected expression kind %s", node.Kind)
	}
}

// CheckTable validates a [table] header given the decoded parts of its key.
// It mirrors checkTable but is driven directly from the key parts instead of
// an AST, for callers that decode without building one. It returns whether the
// table is seen for the first time.
func (s *SeenTracker) CheckTable(parts [][]byte) (bool, error) {
	parent := int32(0)
	for k := 0; k < len(parts); k++ {
		name := parts[k]
		if k == len(parts)-1 {
			// Final part of the key.
			i := s.find(parent, name)
			if i < 0 {
				i = s.create(parent, name, tableKind, true)
				s.currentTable = i
				return true, nil
			}
			e := &s.entries[i]
			switch e.kind {
			case tableKind:
				if e.explicit {
					return false, fmt.Errorf("toml: table %s already exists", name)
				}
				e.explicit = true
				s.currentTable = i
				return false, nil
			case kvTableKind:
				return false, fmt.Errorf("toml: table %s already exists as defined by a dotted key", name)
			case arrayTableKind:
				return false, fmt.Errorf("toml: table %s already exists as an array of tables", name)
			default:
				return false, fmt.Errorf("toml: key %s should be a table, not a %s", name, e.kind)
			}
		}

		i := s.find(parent, name)
		if i < 0 {
			i = s.create(parent, name, tableKind, false)
		} else {
			switch s.entries[i].kind {
			case tableKind, arrayTableKind, kvTableKind:
				// Tables created by dotted keys can receive new sub-tables,
				// but cannot be redefined (handled by the last-part case).
			default:
				return false, fmt.Errorf("toml: key %s already exists as a value", name)
			}
		}
		parent = i
	}
	panic("unreachable: table expression without key")
}

// CheckArrayTable validates a [[array table]] header given the decoded parts
// of its key. It mirrors checkArrayTable but is driven directly from the key
// parts. It returns whether the array table is seen for the first time.
func (s *SeenTracker) CheckArrayTable(parts [][]byte) (bool, error) {
	parent := int32(0)
	for k := 0; k < len(parts); k++ {
		name := parts[k]
		if k == len(parts)-1 {
			i := s.find(parent, name)
			if i < 0 {
				i = s.create(parent, name, arrayTableKind, true)
				s.currentTable = i
				return true, nil
			}
			if s.entries[i].kind != arrayTableKind {
				return false, fmt.Errorf("toml: key %s already exists as a %s, but should be an array table", name, s.entries[i].kind)
			}
			// Make the descendants of this array table re-discoverable for
			// the new element.
			s.currentTable = s.refreshArrayTable(i)
			return false, nil
		}

		i := s.find(parent, name)
		if i < 0 {
			i = s.create(parent, name, tableKind, false)
		} else {
			switch s.entries[i].kind {
			case tableKind, arrayTableKind, kvTableKind:
				// Tables created by dotted keys can receive new sub-tables,
				// but cannot be redefined (handled by the last-part case).
			default:
				return false, fmt.Errorf("toml: key %s already exists as a value", name)
			}
		}
		parent = i
	}
	panic("unreachable: array table expression without key")
}

// CheckKeyValue validates the (possibly dotted) key of a key-value under the
// current table, WITHOUT validating its value. It returns the id of the leaf
// entry, so the caller can validate a container value with CheckValueUnder.
func (s *SeenTracker) CheckKeyValue(parts [][]byte) (int32, error) {
	return s.CheckKeyValueUnder(s.currentTable, parts)
}

// CreateAnonymous creates an anonymous entry under parent and returns its id.
// It gives each inline table stored in an array its own key scope, so that
// identical keys in sibling tables do not collide.
func (s *SeenTracker) CreateAnonymous(parent int32) int32 {
	return s.create(parent, nil, anonymousKind, false)
}

// CheckKeyValueUnder validates the (possibly dotted) key of a key-value under
// the given parent entry, WITHOUT validating its value. It mirrors
// checkKeyValue but is driven directly from the key parts, for callers that
// decode without building an AST.
func (s *SeenTracker) CheckKeyValueUnder(parent int32, parts [][]byte) (int32, error) {
	for k := 0; k < len(parts); k++ {
		name := parts[k]
		if k == len(parts)-1 {
			if i := s.find(parent, name); i >= 0 {
				return -1, fmt.Errorf("toml: key %s is already defined", name)
			}
			return s.create(parent, name, valueKind, false), nil
		}

		i := s.find(parent, name)
		if i < 0 {
			i = s.create(parent, name, kvTableKind, false)
		} else if s.entries[i].kind != kvTableKind {
			return -1, fmt.Errorf("toml: key %s is already defined", name)
		}
		parent = i
	}
	panic("unreachable: key-value expression without key")
}

// CheckValueUnder validates the content of a value stored under the given
// entry (typically the leaf returned by CheckKeyValue): inline tables cannot
// contain duplicate keys, including in the inline tables and arrays they
// contain.
func (s *SeenTracker) CheckValueUnder(parent int32, value *unstable.Node) error {
	return s.checkValue(parent, value)
}

func (s *SeenTracker) checkTable(node *unstable.Node) (bool, error) {
	parent := int32(0)

	it := node.Key()
	// Handle the intermediate parts of the key.
	for it.Next() {
		part := it.Node()
		name := part.Data
		if it.IsLast() {
			// Final part of the key.
			i := s.find(parent, name)
			if i < 0 {
				i = s.create(parent, name, tableKind, true)
				s.currentTable = i
				return true, nil
			}
			e := &s.entries[i]
			switch e.kind {
			case tableKind:
				if e.explicit {
					return false, fmt.Errorf("toml: table %s already exists", name)
				}
				e.explicit = true
				s.currentTable = i
				return false, nil
			case kvTableKind:
				return false, fmt.Errorf("toml: table %s already exists as defined by a dotted key", name)
			case arrayTableKind:
				return false, fmt.Errorf("toml: table %s already exists as an array of tables", name)
			default:
				return false, fmt.Errorf("toml: key %s should be a table, not a %s", name, e.kind)
			}
		}

		i := s.find(parent, name)
		if i < 0 {
			i = s.create(parent, name, tableKind, false)
		} else {
			switch s.entries[i].kind {
			case tableKind, arrayTableKind, kvTableKind:
				// Tables created by dotted keys can receive new sub-tables,
				// but cannot be redefined (handled by the last-part case).
			default:
				return false, fmt.Errorf("toml: key %s already exists as a value", name)
			}
		}
		parent = i
	}
	panic("unreachable: table expression without key")
}

func (s *SeenTracker) checkArrayTable(node *unstable.Node) (bool, error) {
	parent := int32(0)

	it := node.Key()
	for it.Next() {
		part := it.Node()
		name := part.Data
		if it.IsLast() {
			i := s.find(parent, name)
			if i < 0 {
				i = s.create(parent, name, arrayTableKind, true)
				s.currentTable = i
				return true, nil
			}
			if s.entries[i].kind != arrayTableKind {
				return false, fmt.Errorf("toml: key %s already exists as a %s, but should be an array table", name, s.entries[i].kind)
			}
			// Make the descendants of this array table re-discoverable for
			// the new element.
			s.currentTable = s.refreshArrayTable(i)
			return false, nil
		}

		i := s.find(parent, name)
		if i < 0 {
			i = s.create(parent, name, tableKind, false)
		} else {
			switch s.entries[i].kind {
			case tableKind, arrayTableKind, kvTableKind:
				// Tables created by dotted keys can receive new sub-tables,
				// but cannot be redefined (handled by the last-part case).
			default:
				return false, fmt.Errorf("toml: key %s already exists as a value", name)
			}
		}
		parent = i
	}
	panic("unreachable: array table expression without key")
}

func (s *SeenTracker) checkKeyValue(parent int32, node *unstable.Node) error {
	it := node.Key()
	for it.Next() {
		part := it.Node()
		name := part.Data
		if it.IsLast() {
			if i := s.find(parent, name); i >= 0 {
				return fmt.Errorf("toml: key %s is already defined", name)
			}
			id := s.create(parent, name, valueKind, false)
			return s.checkValue(id, node.Value())
		}

		i := s.find(parent, name)
		if i < 0 {
			i = s.create(parent, name, kvTableKind, false)
		} else if s.entries[i].kind != kvTableKind {
			return fmt.Errorf("toml: key %s is already defined", name)
		}
		parent = i
	}
	panic("unreachable: key-value expression without key")
}

// checkValue verifies the content of a value: inline tables cannot contain
// duplicate keys, including in the inline tables and arrays they contain.
func (s *SeenTracker) checkValue(id int32, value *unstable.Node) error {
	switch value.Kind {
	case unstable.InlineTable:
		it := value.Children()
		for it.Next() {
			if err := s.checkKeyValue(id, it.Node()); err != nil {
				return err
			}
		}
	case unstable.Array:
		it := value.Children()
		for it.Next() {
			elem := it.Node()
			switch elem.Kind {
			case unstable.InlineTable:
				// Each inline table is its own key scope: it needs a fresh
				// anonymous parent so that identical keys in sibling tables
				// do not collide.
				elemID := s.create(id, nil, anonymousKind, false)
				if err := s.checkValue(elemID, elem); err != nil {
					return err
				}
			case unstable.Array:
				// Arrays declare no keys themselves: pass through without
				// creating an entry.
				if err := s.checkValue(id, elem); err != nil {
					return err
				}
			}
		}
	default:
	}
	return nil
}
