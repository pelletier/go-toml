package toml

import "testing"


func assertTree(t *testing.T, tree *TomlTree, ref map[string]interface{}) {
	for k, v := range ref {
		if tree.Get(k) != v {
			t.Log("was expecting", v, "at", k, "but got", tree.Get(k))
			t.Fail()
		}
	}
}

func testCreateSubTree(t *testing.T) {
	tree := make(TomlTree)
	tree.createSubTree("a.b.c")
	tree.Set("a.b.c", 42)
	if tree.Get("a.b.c") != 42 {
		t.Fail()
	}
}


func testSimpleKV(t *testing.T) {
	tree := Load("a = 42")
	assertTree(t, tree, map[string]interface{}{
		"a": 42,
	})

	tree = Load("a = 42\nb = 21")
	assertTree(t, tree, map[string]interface{}{
		"a": 42,
		"b": 21,
	})
}

func testSimpleIntegers(t *testing.T) {
	tree := Load("a = +42\nb = -21\nc = +4.2\nd = -2.1")
	assertTree(t, tree, map[string]interface{}{
		"a": 42,
		"b": -21,
		"c": 4.2,
		"d": -4.2,
	})
}
