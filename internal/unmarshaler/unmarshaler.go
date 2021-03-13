package unmarshaler

import (
	"fmt"
	"reflect"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

func FromAst(tree ast.Root, target interface{}) error {
	x := reflect.ValueOf(target)
	if x.Kind() != reflect.Ptr {
		return fmt.Errorf("need to target a pointer, not %s", x.Kind())
	}
	if x.IsNil() {
		return fmt.Errorf("target pointer must be non-nil")
	}

	for _, node := range tree {
		err := topLevelNode(x, &node)
		if err != nil {
			return err
		}
	}

	return nil
}

func topLevelNode(x reflect.Value, node *ast.Node) error {
	if x.Kind() != reflect.Ptr {
		panic("topLevelNode should receive target, which should be a pointer")
	}
	if x.IsNil() {
		panic("topLevelNode should receive target, which should not be a nil pointer")
	}

	switch node.Kind {
	case ast.Table:
		panic("TODO")
	case ast.ArrayTable:
		panic("TODO")
	case ast.KeyValue:
		return keyValue(x, node)
	default:
		panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
	}
}

func keyValue(x reflect.Value, node *ast.Node) error {
	assertNode(ast.KeyValue, node)
	assertPtr(x)

	key := node.Key()
	key = key
	// TODO
	return nil
}

func assertNode(expected ast.Kind, node *ast.Node) {
	if node.Kind != expected {
		panic(fmt.Errorf("expected node of kind %s, not %s", expected, node.Kind))
	}
}

func assertPtr(x reflect.Value) {
	if x.Kind() != reflect.Ptr {
		panic(fmt.Errorf("should be a pointer, not a %s", x.Kind()))
	}
}
