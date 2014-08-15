// Testing support for go-toml

package toml

import (
	"testing"
)

func TestTomlHas(t *testing.T) {
	tree, _ := Load(`
		[test]
		key = "value"
	`)

	if !tree.Has("test.key") {
		t.Errorf("Has - expected test.key to exists")
	}
}

func TestTomlHasPath(t *testing.T) {
	tree, _ := Load(`
		[test]
		key = "value"
	`)

	if !tree.HasPath([]string{"test", "key"}) {
		t.Errorf("HasPath - expected test.key to exists")
	}
}

func TestTomlGetPath(t *testing.T) {
	node := newTomlTree()
	//TODO: set other node data

	for idx, item := range []struct {
		Path     []string
		Expected *TomlTree
	}{
		{ // empty path test
			[]string{},
			node,
		},
	} {
		result := node.GetPath(item.Path)
		if result != item.Expected {
			t.Errorf("GetPath[%d] %v - expected %v, got %v instead.", idx, item.Path, item.Expected, result)
		}
	}
}
