package ast

import (
	"fmt"
)

// Iterator starts uninitialized, you need to call Next() first.
//
// For example:
//
// it := n.Children()
// for it.Next() {
// 		it.Node()
// }
type Iterator struct {
	started bool
	node    Node
}

// Next moves the iterator forward and returns true if points to a node, false
// otherwise.
func (c *Iterator) Next() bool {
	if !c.started {
		c.started = true
	} else if c.node.Valid() {
		c.node = c.node.Next()
	}
	return c.node.Valid()
}

// Node returns a copy of the node pointed at by the iterator.
func (c *Iterator) Node() Node {
	return c.node
}

// Root contains a full AST.
//
// It is immutable once constructed with Builder.
type Root struct {
	nodes []Node
}

// Iterator over the top level nodes.
func (r *Root) Iterator() Iterator {
	it := Iterator{}
	if len(r.nodes) > 0 {
		it.node = r.nodes[0]
	}
	return it
}

func (r *Root) at(idx int) Node {
	// TODO: unsafe to point to the node directly
	return r.nodes[idx]
}

// Arrays have one child per element in the array.
// InlineTables have one child per key-value pair in the table.
// KeyValues have at least two children. The first one is the value. The
// rest make a potentially dotted key.
// Table and Array table have one child per element of the key they
// represent (same as KeyValue, but without the last node being the value).
// children []Node
type Node struct {
	Kind Kind
	Data []byte // Raw bytes from the input

	// next idx (in the root array). 0 if last of the collection.
	next int
	// child idx (in the root array). 0 if no child.
	child int
	// pointer to the root array
	root *Root
}

// Next returns a copy of the next node, or an invalid Node if there is no
// next node.
func (n Node) Next() Node {
	if n.next <= 0 {
		return noNode
	}
	return n.root.at(n.next)
}

// Child returns a copy of the first child node of this node. Other children
// can be accessed calling Next on the first child.
// Returns an invalid Node if there is none.
func (n Node) Child() Node {
	if n.child <= 0 {
		return noNode
	}
	return n.root.at(n.child)
}

// Valid returns true if the node's kind is set (not to Invalid).
func (n Node) Valid() bool {
	return n.Kind != Invalid
}

var noNode = Node{}

// Key returns the child nodes making the Key on a supported node. Panics
// otherwise.
// They are guaranteed to be all be of the Kind Key. A simple key would return
// just one element.
func (n *Node) Key() Iterator {
	switch n.Kind {
	case KeyValue:
		value := n.Child()
		if !value.Valid() {
			panic(fmt.Errorf("KeyValue should have at least two children"))
		}
		return Iterator{node: value.Next()}
	case Table, ArrayTable:
		return Iterator{node: n.Child()}
	default:
		panic(fmt.Errorf("Key() is not supported on a %s", n.Kind))
	}
}

// Value returns a pointer to the value node of a KeyValue.
// Guaranteed to be non-nil.
// Panics if not called on a KeyValue node, or if the Children are malformed.
func (n Node) Value() Node {
	assertKind(KeyValue, n)
	return n.Child()
}

// Children returns an iterator over a node's children.
func (n Node) Children() Iterator {
	return Iterator{node: n.Child()}
}

func assertKind(k Kind, n Node) {
	if n.Kind != k {
		panic(fmt.Errorf("method was expecting a %s, not a %s", k, n.Kind))
	}
}
