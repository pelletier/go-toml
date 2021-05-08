package tracker

import (
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

// SeenTracker tracks which keys have been seen with which TOML type to flag duplicates
// and mismatches according to the spec.
type SeenTracker struct {
	root    *info
	current *info
}

type info struct {
	parent   *info
	kind     keyKind
	children map[string]*info
	explicit bool
}

func (i *info) Clear() {
	i.children = nil
}

func (i *info) Has(k string) (*info, bool) {
	c, ok := i.children[k]
	return c, ok
}

func (i *info) SetKind(kind keyKind) {
	i.kind = kind
}

func (i *info) CreateTable(k string, explicit bool) *info {
	return i.createChild(k, tableKind, explicit)
}

func (i *info) CreateArrayTable(k string, explicit bool) *info {
	return i.createChild(k, arrayTableKind, explicit)
}

func (i *info) createChild(k string, kind keyKind, explicit bool) *info {
	if i.children == nil {
		i.children = make(map[string]*info, 1)
	}

	x := &info{
		parent:   i,
		kind:     kind,
		explicit: explicit,
	}
	i.children[k] = x
	return x
}

// CheckExpression takes a top-level node and checks that it does not contain keys
// that have been seen in previous calls, and validates that types are consistent.
func (s *SeenTracker) CheckExpression(node ast.Node) error {
	if s.root == nil {
		s.root = &info{
			kind: tableKind,
		}
		s.current = s.root
	}
	switch node.Kind {
	case ast.KeyValue:
		return s.checkKeyValue(s.current, node)
	case ast.Table:
		return s.checkTable(node)
	case ast.ArrayTable:
		return s.checkArrayTable(node)
	default:
		panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
	}

}
func (s *SeenTracker) checkTable(node ast.Node) error {
	s.current = s.root

	it := node.Key()
	// handle the first parts of the key, excluding the last one
	for it.Next() {
		if !it.Node().Next().Valid() {
			break
		}

		k := string(it.Node().Data)
		child, found := s.current.Has(k)
		if !found {
			child = s.current.CreateTable(k, false)
		}
		s.current = child
	}

	// handle the last part of the key
	k := string(it.Node().Data)

	i, found := s.current.Has(k)
	if found {
		if i.kind != tableKind {
			return fmt.Errorf("toml: key %s should be a table, not a %s", k, i.kind)
		}
		if i.explicit {
			return fmt.Errorf("toml: table %s already exists", k)
		}
		i.explicit = true
		s.current = i
	} else {
		s.current = s.current.CreateTable(k, true)
	}

	return nil
}

func (s *SeenTracker) checkArrayTable(node ast.Node) error {
	s.current = s.root

	it := node.Key()

	// handle the first parts of the key, excluding the last one
	for it.Next() {
		if !it.Node().Next().Valid() {
			break
		}

		k := string(it.Node().Data)
		child, found := s.current.Has(k)
		if !found {
			child = s.current.CreateTable(k, false)
		}
		s.current = child
	}

	// handle the last part of the key
	k := string(it.Node().Data)

	info, found := s.current.Has(k)
	if found {
		if info.kind != arrayTableKind {
			return fmt.Errorf("toml: key %s already exists as a %s,  but should be an array table", info.kind, k)
		}
		info.Clear()
	} else {
		info = s.current.CreateArrayTable(k, true)
	}

	s.current = info
	return nil
}

func (s *SeenTracker) checkKeyValue(context *info, node ast.Node) error {
	it := node.Key()

	// handle the first parts of the key, excluding the last one
	for it.Next() {
		k := string(it.Node().Data)
		child, found := context.Has(k)
		if found {
			if child.kind != tableKind {
				return fmt.Errorf("toml: expected %s to be a table, not a %s", k, child.kind)
			}
		} else {
			child = context.CreateTable(k, false)
		}
		context = child
	}

	if node.Value().Kind == ast.InlineTable {
		context.SetKind(tableKind)
	} else {
		context.SetKind(valueKind)
	}

	return nil
}
