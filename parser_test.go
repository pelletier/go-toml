package toml

import (
	"fmt"
	"testing"
	"time"
)

func assertTree(t *testing.T, tree *TomlTree, ref map[string]interface{}) {
	for k, v := range ref {
		if fmt.Sprintf("%v", tree.Get(k)) != fmt.Sprintf("%v", v) {
			t.Log("was expecting", v, "at", k, "but got", tree.Get(k))
			t.Fail()
		}
	}
}

func TestCreateSubTree(t *testing.T) {
	tree := make(TomlTree)
	tree.createSubTree("a.b.c")
	tree.Set("a.b.c", 42)
	if tree.Get("a.b.c") != 42 {
		t.Fail()
	}
}

func TestSimpleKV(t *testing.T) {
	tree, _ := Load("a = 42")
	assertTree(t, tree, map[string]interface{}{
		"a": int64(42),
	})

	tree, _ = Load("a = 42\nb = 21")
	assertTree(t, tree, map[string]interface{}{
		"a": int64(42),
		"b": int64(21),
	})
}

func TestSimpleNumbers(t *testing.T) {
	tree, _ := Load("a = +42\nb = -21\nc = +4.2\nd = -2.1")
	assertTree(t, tree, map[string]interface{}{
		"a": int64(42),
		"b": int64(-21),
		"c": float64(4.2),
		"d": float64(-2.1),
	})
}

func TestSimpleDate(t *testing.T) {
	tree, _ := Load("a = 1979-05-27T07:32:00Z")
	assertTree(t, tree, map[string]interface{}{
		"a": time.Date(1979, time.May, 27, 7, 32, 0, 0, time.UTC),
	})
}

func TestSimpleString(t *testing.T) {
	tree, _ := Load("a = \"hello world\"")
	assertTree(t, tree, map[string]interface{}{
		"a": "hello world",
	})
}

func TestBools(t *testing.T) {
	tree, _ := Load("a = true\nb = false")
	assertTree(t, tree, map[string]interface{}{
		"a": true,
		"b": false,
	})
}

func TestNestedKeys(t *testing.T) {
	tree, _ := Load("[a.b.c]\nd = 42")
	assertTree(t, tree, map[string]interface{}{
		"a.b.c.d": int64(42),
	})
}

func TestArrayOne(t *testing.T) {
	tree, _ := Load("a = [1]")
	assertTree(t, tree, map[string]interface{}{
		"a": []int64{int64(1)},
	})
}

func TestArrayZero(t *testing.T) {
	tree, _ := Load("a = []")
	assertTree(t, tree, map[string]interface{}{
		"a": []interface{}{},
	})
}

func TestArraySimple(t *testing.T) {
	tree, _ := Load("a = [42, 21, 10]")
	assertTree(t, tree, map[string]interface{}{
		"a": []int64{int64(42), int64(21), int64(10)},
	})

	tree, _ = Load("a = [42, 21, 10,]")
	assertTree(t, tree, map[string]interface{}{
		"a": []int64{int64(42), int64(21), int64(10)},
	})
}

func TestArrayMultiline(t *testing.T) {
	tree, _ := Load("a = [42,\n21, 10,]")
	assertTree(t, tree, map[string]interface{}{
		"a": []int64{int64(42), int64(21), int64(10)},
	})
}

func TestArrayNested(t *testing.T) {
	tree, _ := Load("a = [[42, 21], [10]]")
	assertTree(t, tree, map[string]interface{}{
		"a": [][]int64{[]int64{int64(42), int64(21)}, []int64{int64(10)}},
	})
}

func TestMissingValue(t *testing.T) {
	_, err := Load("a = ")
	if (err.Error() != "expecting a value") {
		t.Error("Bad error message:", err.Error())
	}
}

func TestUnterminatedArray(t *testing.T) {
	_, err := Load("a = [1,")
	if (err.Error() != "unterminated array") {
		t.Error("Bad error message:", err.Error())
	}
}

func TestMissingFile(t *testing.T) {
	_, err := LoadFile("foo.toml")
	if (err.Error() != "open foo.toml: no such file or directory") {
		t.Error("Bad error message:", err.Error())
	}
}

func TestParseFile(t *testing.T) {
	tree, err := LoadFile("example.toml")
	if (err != nil) {
		t.Fatal("Non-nil error:", err.Error())
	}
	assertTree(t, tree, map[string]interface{}{
		"a": [][]int64{[]int64{int64(42), int64(21)}, []int64{int64(10)}},
	})
}