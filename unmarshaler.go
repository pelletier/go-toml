package toml

import (
	"fmt"
	"reflect"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

func Unmarshal(data []byte, v interface{}) error {
	p := parser{}
	err := p.parse(data)
	if err != nil {
		return err
	}
	return fromAst(p.tree, v)
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
	var root target = valueTarget(r.Elem())
	current := root
	for _, node := range tree {
		current, err = unmarshalTopLevelNode(root, current, &node)
		if err != nil {
			return err
		}
	}

	return nil
}

// The target return value is the target for the next top-level node. Mostly
// unchanged, except by table and array table.
func unmarshalTopLevelNode(root target, x target, node *ast.Node) (target, error) {
	switch node.Kind {
	case ast.KeyValue:
		return x, unmarshalKeyValue(x, node)
	case ast.Table:
		return scopeWithKey(root, node.Key())
	case ast.ArrayTable:
		return scopeWithArrayTable(root, node.Key())
	default:
		panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
	}
}

// scopeWithKey performs target scoping when unmarshaling an ast.KeyValue node.
//
// The goal is to hop from target to target recursively using the names in key.
// Parts of the key should be used to resolve field names for structs, and as
// keys when targeting maps.
//
// When encountering slices, it should always use its last element, and error
// if the slice does not have any.
func scopeWithKey(x target, key []ast.Node) (target, error) {
	var err error
	for _, n := range key {
		x, err = scopeTableTarget(false, x, string(n.Data))
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

// scopeWithArrayTable performs target scoping when unmarshaling an
// ast.ArrayTable node.
//
// It is the same as scopeWithKey, but when scoping the last part of the key
// it creates a new element in the array instead of using the last one.
func scopeWithArrayTable(x target, key []ast.Node) (target, error) {
	var err error
	if len(key) > 1 {
		for _, n := range key[:len(key)-1] {
			x, err = scopeTableTarget(false, x, string(n.Data))
			if err != nil {
				return nil, err
			}
		}
	}
	x, err = scopeTableTarget(true, x, string(key[len(key)-1].Data))
	if err != nil {
		return x, err
	}

	v := x.get()

	if v.Kind() == reflect.Interface {
		x, err = scopeInterface(true, x)
		if err != nil {
			return x, err
		}
		v = x.get()
	}

	if v.Kind() == reflect.Slice {
		return scopeSlice(true, x)
	}

	return x, err
}

func unmarshalKeyValue(x target, node *ast.Node) error {
	assertNode(ast.KeyValue, node)

	x, err := scopeWithKey(x, node.Key())
	if err != nil {
		return err
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
	default:
		panic(fmt.Errorf("unhandled unmarshalValue kind %s", node.Kind))
	}
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
	v, err := node.DecodeInteger()
	if err != nil {
		return err
	}
	return setInt64(x, v)
}

func unmarshalFloat(x target, node *ast.Node) error {
	assertNode(ast.Float, node)
	v, err := node.DecodeFloat()
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
