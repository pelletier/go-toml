package toml

import (
	"testing"
)

func TestTomlGetPath(t *testing.T) {
	node := make(TomlTree)
	//TODO: set other node data

	for idx, item := range []struct {
		Path     []string
		Expected interface{}
	}{
		{ // empty path test
			[]string{},
			&node,
		},
	} {
		result := node.GetPath(item.Path)
		if result != item.Expected {
			t.Errorf("GetPath[%d] %v - expected %v, got %v instead.", idx, item.Path, item.Expected, result)
		}
	}
}
