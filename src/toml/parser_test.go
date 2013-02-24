package toml

import "testing"


func testCreateSubTree(t *testing.T) {
	tree := make(TomlTree)
	createSubTree(&tree, "a.b.c")
	tree.Set("a.b.c", 42)
	if tree.Get("a.b.c") != 42 {
		t.Fail()
	}
}
