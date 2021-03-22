package toml

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

func Unmarshal(data []byte, v interface{}) error {
	p := parser{}
	err := p.parse(data)
	if err != nil {
		return err
	}

	// TODO: remove me; sanity check
	allValidOrDump(p.tree, p.tree)

	return fromAst(p.tree, v)
}

func allValidOrDump(tree ast.Root, nodes []ast.Node) bool {
	for i, n := range nodes {
		if n.Kind == ast.Invalid {
			fmt.Printf("AST contains invalid node! idx=%d\n", i)
			fmt.Fprintf(os.Stderr, "%s\n", tree.Sdot())
			return false
		}
		ok := allValidOrDump(tree, n.Children)
		if !ok {
			return ok
		}
	}
	return true
}

func fromAst(tree ast.Root, v interface{}) error {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Ptr {
		return fmt.Errorf("need to target a pointer, not %s", r.Kind())
	}
	if r.IsNil() {
		return fmt.Errorf("target pointer must be non-nil")
	}

	var err error
	var skipUntilTable bool
	var root target = valueTarget(r.Elem())
	current := root
	for _, node := range tree {
		var found bool
		switch node.Kind {
		case ast.KeyValue:
			if skipUntilTable {
				continue
			}
			err = unmarshalKeyValue(current, &node)
			found = true
		case ast.Table:
			current, found, err = scopeWithKey(root, node.Key())
		case ast.ArrayTable:
			current, found, err = scopeWithArrayTable(root, node.Key())
		default:
			panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
		}

		if err != nil {
			return err
		}

		if !found {
			skipUntilTable = true
		}
	}

	return nil
}

// scopeWithKey performs target scoping when unmarshaling an ast.KeyValue node.
//
// The goal is to hop from target to target recursively using the names in key.
// Parts of the key should be used to resolve field names for structs, and as
// keys when targeting maps.
//
// When encountering slices, it should always use its last element, and error
// if the slice does not have any.
func scopeWithKey(x target, key []ast.Node) (target, bool, error) {
	var err error
	found := true
	for _, n := range key {
		x, found, err = scopeTableTarget(false, x, string(n.Data))
		if err != nil || !found {
			return nil, found, err
		}
	}
	return x, true, nil
}

// scopeWithArrayTable performs target scoping when unmarshaling an
// ast.ArrayTable node.
//
// It is the same as scopeWithKey, but when scoping the last part of the key
// it creates a new element in the array instead of using the last one.
func scopeWithArrayTable(x target, key []ast.Node) (target, bool, error) {
	var err error
	found := true
	if len(key) > 1 {
		for _, n := range key[:len(key)-1] {
			x, found, err = scopeTableTarget(false, x, string(n.Data))
			if err != nil || !found {
				return nil, found, err
			}
		}
	}
	x, found, err = scopeTableTarget(false, x, string(key[len(key)-1].Data))
	if err != nil || !found {
		return x, found, err
	}

	v := x.get()

	if v.Kind() == reflect.Interface {
		x, err = scopeInterface(true, x)
		if err != nil {
			return x, found, err
		}
		v = x.get()
	}

	if v.Kind() == reflect.Slice {
		x, err = scopeSlice(true, x)
	}

	return x, found, err
}

func unmarshalKeyValue(x target, node *ast.Node) error {
	assertNode(ast.KeyValue, node)

	x, found, err := scopeWithKey(x, node.Key())
	if err != nil {
		return err
	}

	// A struct in the path was not found. Skip this value.
	if !found {
		return nil
	}

	return unmarshalValue(x, node.Value())
}

func unmarshalValue(x target, node *ast.Node) error {
	switch node.Kind {
	case ast.String:
		return unmarshalString(x, node)
	case ast.Bool:
		return unmarshalBool(x, node)
	case ast.Integer:
		return unmarshalInteger(x, node)
	case ast.Float:
		return unmarshalFloat(x, node)
	case ast.Array:
		return unmarshalArray(x, node)
	case ast.InlineTable:
		return unmarshalInlineTable(x, node)
	case ast.LocalDateTime:
		return unmarshalLocalDateTime(x, node)
	case ast.DateTime:
		return unmarshalDateTime(x, node)
	default:
		panic(fmt.Errorf("unhandled unmarshalValue kind %s", node.Kind))
	}
}

func unmarshalLocalDateTime(x target, node *ast.Node) error {
	assertNode(ast.LocalDateTime, node)
	v, rest, err := parseLocalDateTime(node.Data)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("extra characters at the end of a local date time")
	}
	return setLocalDateTime(x, v)
}

func unmarshalDateTime(x target, node *ast.Node) error {
	assertNode(ast.DateTime, node)
	v, err := parseDateTime(node.Data)
	if err != nil {
		return err
	}
	return setDateTime(x, v)
}

func setLocalDateTime(x target, v LocalDateTime) error {
	return x.set(reflect.ValueOf(v))
}

func setDateTime(x target, v time.Time) error {
	return x.set(reflect.ValueOf(v))
}

func unmarshalString(x target, node *ast.Node) error {
	assertNode(ast.String, node)
	return setString(x, string(node.Data))
}

func unmarshalBool(x target, node *ast.Node) error {
	assertNode(ast.Bool, node)
	v := node.Data[0] == 't'
	return setBool(x, v)
}

func unmarshalInteger(x target, node *ast.Node) error {
	assertNode(ast.Integer, node)
	v, err := parseInteger(node.Data)
	if err != nil {
		return err
	}
	return setInt64(x, v)
}

func unmarshalFloat(x target, node *ast.Node) error {
	assertNode(ast.Float, node)
	v, err := parseFloat(node.Data)
	if err != nil {
		return err
	}
	return setFloat64(x, v)
}

func unmarshalInlineTable(x target, node *ast.Node) error {
	assertNode(ast.InlineTable, node)

	for _, kv := range node.Children {
		err := unmarshalKeyValue(x, &kv)
		if err != nil {
			return err
		}
	}
	return nil
}

func unmarshalArray(x target, node *ast.Node) error {
	assertNode(ast.Array, node)

	err := ensureSlice(x)
	if err != nil {
		return err
	}

	for _, n := range node.Children {
		v, err := pushNew(x)
		if err != nil {
			return err
		}
		err = unmarshalValue(v, &n)
		if err != nil {
			return err
		}
	}
	return nil
}

func assertNode(expected ast.Kind, node *ast.Node) {
	if node.Kind != expected {
		panic(fmt.Errorf("expected node of kind %s, not %s", expected, node.Kind))
	}
}
