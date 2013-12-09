package toml

import (
	"fmt"
	"testing"
	"time"
)

func assertTree(t *testing.T, tree *TomlTree, err error, ref map[string]interface{}) {
	if err != nil {
		t.Error("Non-nil error:", err.Error())
		return
	}
	for k, v := range ref {
		if fmt.Sprintf("%v", tree.Get(k)) != fmt.Sprintf("%v", v) {
			t.Log("was expecting", v, "at", k, "but got", tree.Get(k))
			t.Error()
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
	tree, err := Load("a = 42")
	assertTree(t, tree, err, map[string]interface{}{
		"a": int64(42),
	})

	tree, _ = Load("a = 42\nb = 21")
	assertTree(t, tree, err, map[string]interface{}{
		"a": int64(42),
		"b": int64(21),
	})
}

func TestSimpleNumbers(t *testing.T) {
	tree, err := Load("a = +42\nb = -21\nc = +4.2\nd = -2.1")
	assertTree(t, tree, err, map[string]interface{}{
		"a": int64(42),
		"b": int64(-21),
		"c": float64(4.2),
		"d": float64(-2.1),
	})
}

func TestSimpleDate(t *testing.T) {
	tree, err := Load("a = 1979-05-27T07:32:00Z")
	assertTree(t, tree, err, map[string]interface{}{
		"a": time.Date(1979, time.May, 27, 7, 32, 0, 0, time.UTC),
	})
}

func TestSimpleString(t *testing.T) {
	tree, err := Load("a = \"hello world\"")
	assertTree(t, tree, err, map[string]interface{}{
		"a": "hello world",
	})
}

func TestStringEscapables(t *testing.T) {
	tree, err := Load("a = \"a \\n b\"")
	assertTree(t, tree, err, map[string]interface{}{
		"a": "a \n b",
	})

	tree, err = Load("a = \"a \\t b\"")
	assertTree(t, tree, err, map[string]interface{}{
		"a": "a \t b",
	})

	tree, err = Load("a = \"a \\r b\"")
	assertTree(t, tree, err, map[string]interface{}{
		"a": "a \r b",
	})

	tree, err = Load("a = \"a \\\\ b\"")
	assertTree(t, tree, err, map[string]interface{}{
		"a": "a \\ b",
	})
}

func TestBools(t *testing.T) {
	tree, err := Load("a = true\nb = false")
	assertTree(t, tree, err, map[string]interface{}{
		"a": true,
		"b": false,
	})
}

func TestNestedKeys(t *testing.T) {
	tree, err := Load("[a.b.c]\nd = 42")
	assertTree(t, tree, err, map[string]interface{}{
		"a.b.c.d": int64(42),
	})
}

func TestArrayOne(t *testing.T) {
	tree, err := Load("a = [1]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(1)},
	})
}

func TestArrayZero(t *testing.T) {
	tree, err := Load("a = []")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []interface{}{},
	})
}

func TestArraySimple(t *testing.T) {
	tree, err := Load("a = [42, 21, 10]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(42), int64(21), int64(10)},
	})

	tree, _ = Load("a = [42, 21, 10,]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(42), int64(21), int64(10)},
	})
}

func TestArrayMultiline(t *testing.T) {
	tree, err := Load("a = [42,\n21, 10,]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(42), int64(21), int64(10)},
	})
}

func TestArrayNested(t *testing.T) {
	tree, err := Load("a = [[42, 21], [10]]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": [][]int64{[]int64{int64(42), int64(21)}, []int64{int64(10)}},
	})
}

func TestArrayNestedStrings(t *testing.T) {
	tree, err := Load("data = [ [\"gamma\", \"delta\"], [\"Foo\"] ]")
	assertTree(t, tree, err, map[string]interface{}{
		"data": [][]string{[]string{"gamma", "delta"}, []string{"Foo"}},
	})
}

func TestMissingValue(t *testing.T) {
	_, err := Load("a = ")
	if err.Error() != "expecting a value" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestUnterminatedArray(t *testing.T) {
	_, err := Load("a = [1,")
	if err.Error() != "unterminated array" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestNewlinesInArrays(t *testing.T) {
	tree, err := Load("a = [1,\n2,\n3]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(1), int64(2), int64(3)},
	})
}

func TestArrayWithExtraComma(t *testing.T) {
	tree, err := Load("a = [1,\n2,\n3,\n]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(1), int64(2), int64(3)},
	})
}

func TestArrayWithExtraCommaComment(t *testing.T) {
	tree, err := Load("a = [1, # wow\n2, # such items\n3, # so array\n]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": []int64{int64(1), int64(2), int64(3)},
	})
}

func TestMissingFile(t *testing.T) {
	_, err := LoadFile("foo.toml")
	if err.Error() != "open foo.toml: no such file or directory" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestParseFile(t *testing.T) {
	tree, err := LoadFile("example.toml")

	assertTree(t, tree, err, map[string]interface{}{
		"title":                   "TOML Example",
		"owner.name":              "Tom Preston-Werner",
		"owner.organization":      "GitHub",
		"owner.bio":               "GitHub Cofounder & CEO\nLikes tater tots and beer.",
		"owner.dob":               time.Date(1979, time.May, 27, 7, 32, 0, 0, time.UTC),
		"database.server":         "192.168.1.1",
		"database.ports":          []int64{8001, 8001, 8002},
		"database.connection_max": 5000,
		"database.enabled":        true,
		"servers.alpha.ip":        "10.0.0.1",
		"servers.alpha.dc":        "eqdc10",
		"servers.beta.ip":         "10.0.0.2",
		"servers.beta.dc":         "eqdc10",
		"clients.data":            []interface{}{[]string{"gamma", "delta"}, []int64{1, 2}},
	})
}
