package unmarshaler

import (
	"fmt"
	"reflect"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

func FromAst(tree ast.Root, target interface{}) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("need to target a pointer, not %s", v.Kind())
	}
	if v.IsNil() {
		return fmt.Errorf("target pointer must be non-nil")
	}

	x := valueTarget(v.Elem())

	for _, node := range tree {
		err := unmarshalTopLevelNode(x, &node)
		if err != nil {
			return err
		}
	}

	return nil
}

func unmarshalTopLevelNode(x target, node *ast.Node) error {
	switch node.Kind {
	case ast.Table:
		panic("TODO")
	case ast.ArrayTable:
		panic("TODO")
	case ast.KeyValue:
		return unmarshalKeyValue(x, node)
	default:
		panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
	}
}

func unmarshalKeyValue(x target, node *ast.Node) error {
	assertNode(ast.KeyValue, node)

	key := node.Key()

	var err error
	for _, n := range key {
		x, err = scopeTarget(x, string(n.Data))
		if err != nil {
			return err
		}
	}

	return unmarshalValue(x, node.Value())
}

func unmarshalValue(x target, node *ast.Node) error {
	switch node.Kind {
	case ast.String:
		return unmarshalString(x, node)
	case ast.Array:
		return unmarshalArray(x, node)
	default:
		panic(fmt.Errorf("unhandled unmarshalValue kind %s", node.Kind))
	}
}

func unmarshalString(x target, node *ast.Node) error {
	assertNode(ast.String, node)

	return x.setString(string(node.Data))
}

func unmarshalArray(x target, node *ast.Node) error {
	assertNode(ast.Array, node)

	x.ensure()

	for _, n := range node.Children {
		v, err := x.pushNew()
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
