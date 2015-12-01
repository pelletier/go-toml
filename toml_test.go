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

func TestTomlGetGroup(t *testing.T) {
	tree, err := Load("[[foo]]\nbar=1\n[[foo]]\nbar=2")
	if err != nil {
		t.Error(err)
		return
	}
	bar := tree.Get("foo.0.bar")
	if bar == nil {
		t.Error("Expected index 0 to be non-nil")
		return
	}
	if bar, ok := bar.(int64); !ok {
		t.Error("Expected index 0 to be int64, but wasn't")
		return
	} else if bar != 1 {
		t.Errorf("Expected index 0 to be 1, but got %d", bar)
		return
	}
	bar = tree.Get("foo.1.bar")
	if bar == nil {
		t.Error("Expected index 1 to be non-nil")
		return
	}
	if bar, ok := bar.(int64); !ok {
		t.Error("Expected index 1 to be int64, but wasn't")
		return
	} else if bar != 2 {
		t.Errorf("Expected index 1 to be 2, but got %d", bar)
		return
	}
	bar = tree.Get("foo.2.bar")
	if bar != nil {
		t.Error("Expected index 1 to be nil, but wasn't")
		return
	}
}

func TestTomlQuery(t *testing.T) {
	tree, err := Load("[foo.bar]\na=1\nb=2\n[baz.foo]\na=3\nb=4\n[gorf.foo]\na=5\nb=6")
	if err != nil {
		t.Error(err)
		return
	}
	result, err := tree.Query("$.foo.bar")
	if err != nil {
		t.Error(err)
		return
	}
	values := result.Values()
	if len(values) != 1 {
		t.Errorf("Expected resultset of 1, got %d instead: %v", len(values), values)
	}

	if tt, ok := values[0].(*TomlTree); !ok {
		t.Errorf("Expected type of TomlTree: %T Tv", values[0], values[0])
	} else if tt.Get("a") != int64(1) {
		t.Errorf("Expected 'a' with a value 1: %v", tt.Get("a"))
	} else if tt.Get("b") != int64(2) {
		t.Errorf("Expected 'b' with a value 2: %v", tt.Get("b"))
	}
}
