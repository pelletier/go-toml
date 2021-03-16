package ast

import (
	"fmt"
	"strings"
)

type Kind int

const (
	// meta
	Comment Kind = iota
	Key

	// top level structures
	Table
	ArrayTable
	KeyValue

	// containers values
	Array
	InlineTable

	// values
	String
	Bool
	Float
	Integer
	LocalDate
	LocalDateTime
	DateTime
	Time
)

func (k Kind) String() string {
	switch k {
	case Comment:
		return "Comment"
	case Key:
		return "Key"
	case Table:
		return "Table"
	case ArrayTable:
		return "ArrayTable"
	case KeyValue:
		return "KeyValue"
	case Array:
		return "Array"
	case InlineTable:
		return "InlineTable"
	case String:
		return "String"
	case Bool:
		return "Bool"
	case Float:
		return "Float"
	case Integer:
		return "Integer"
	case LocalDate:
		return "LocalDate"
	case LocalDateTime:
		return "LocalDateTime"
	case DateTime:
		return "DateTime"
	case Time:
		return "Time"
	}
	panic(fmt.Errorf("Kind.String() not implemented for '%d'", k))
}

type Root []Node

// Dot returns a dot representation of the AST for debugging.
func (r Root) Sdot() string {
	type edge struct {
		from int
		to   int
	}

	var nodes []string
	var edges []edge // indexes into nodes

	nodes = append(nodes, "root")

	labelForNode := func(node *Node) string {
		return fmt.Sprintf("{%s}", node.Kind)
	}

	var processNode func(int, *Node)
	processNode = func(parentIdx int, node *Node) {
		idx := len(nodes)
		label := labelForNode(node)
		nodes = append(nodes, label)
		edges = append(edges, edge{from: parentIdx, to: idx})

		for _, c := range node.Children {
			processNode(idx, &c)
		}
	}

	for _, n := range r {
		processNode(0, &n)
	}

	var b strings.Builder

	b.WriteString("digraph tree {\n")

	for i, label := range nodes {
		_, _ = fmt.Fprintf(&b, "\tnode%d [label=\"%s\"];\n", i, label)
	}

	b.WriteString("\n")

	for _, e := range edges {
		_, _ = fmt.Fprintf(&b, "\tnode%d -> node%d;\n", e.from, e.to)
	}

	b.WriteString("}")

	return b.String()
}

type Node struct {
	Kind Kind
	Data []byte // Raw bytes from the input

	// Arrays have one child per element in the array.
	// InlineTables have one child per key-value pair in the table.
	// KeyValues have at least two children. The last one is the value. The
	// rest make a potentially dotted key.
	// Table and Array table have one child per element of the key they
	// represent (same as KeyValue, but without the last node being the value).
	Children []Node
}

var NoNode = Node{}

// Key returns the child nodes making the Key on a supported node. Panics
// otherwise.
// They are guaranteed to be all be of the Kind Key. A simple key would return
// just one element.
func (n *Node) Key() []Node {
	switch n.Kind {
	case KeyValue:
		if len(n.Children) < 2 {
			panic(fmt.Errorf("KeyValue should have at least two children, not %d", len(n.Children)))
		}
		return n.Children[:len(n.Children)-1]
	case Table, ArrayTable:
		return n.Children
	default:
		panic(fmt.Errorf("Key() is not supported on a %s", n.Kind))
	}
}

// Value returns a pointer to the value node of a KeyValue.
// Guaranteed to be non-nil.
// Panics if not called on a KeyValue node, or if the Children are malformed.
func (n *Node) Value() *Node {
	assertKind(KeyValue, n)
	if len(n.Children) < 2 {
		panic(fmt.Errorf("KeyValue should have at least two children, not %d", len(n.Children)))
	}
	return &n.Children[len(n.Children)-1]
}

// DecodeInteger parse the data of an Integer node and returns the represented
// int64, or an error.
// Panics if not called on an Integer node.
func (n *Node) DecodeInteger() (int64, error) {
	assertKind(Integer, n)
	if len(n.Data) > 2 && n.Data[0] == '0' {
		switch n.Data[1] {
		case 'x':
			return parseIntHex(n.Data)
		case 'b':
			return parseIntBin(n.Data)
		case 'o':
			return parseIntOct(n.Data)
		default:
			return 0, fmt.Errorf("invalid base: '%c'", n.Data[1])
		}
	}
	return parseIntDec(n.Data)
}

// DecodeFloat parse the data of a Float node and returns the represented
// float64, or an error.
// Panics if not called on an Float node.
func (n *Node) DecodeFloat() (float64, error) {
	assertKind(Float, n)
	return parseFloat(n.Data)
}

func assertKind(k Kind, n *Node) {
	if n.Kind != k {
		panic(fmt.Errorf("method was expecting a %s, not a %s", k, n.Kind))
	}
}
