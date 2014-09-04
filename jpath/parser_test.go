package jpath

import (
	"fmt"
	"testing"
  . "github.com/pelletier/go-toml"
)

func assertQuery(t *testing.T, toml, query string, ref []interface{}) {
	tree, err := Load(toml)
	if err != nil {
		t.Errorf("Non-nil toml parse error: %v", err)
    return
	}
	_, flow := lex(query)
	if err != nil {
		t.Errorf("Non-nil query lex error: %v", err)
		return
	}
	path := parse(flow)
  result := processPath(path, tree)
  assertValue(t, result, ref, "")
}

func assertValue(t *testing.T, result, ref interface{}, location string) {
  switch node := ref.(type) {
  case []interface{}:
    if resultNode, ok := result.([]interface{}); !ok {
      t.Errorf("{%s} result value not of type %T: %T",
        location, node, resultNode)
    } else {
      for i, v := range node {
        assertValue(t, resultNode[i], v, fmt.Sprintf("%s[%d]", location, i))
      }
    }
  case map[string]interface{}:
    if resultNode, ok := result.(*TomlTree); !ok {
      t.Errorf("{%s} result value not of type %T: %T",
        location, node, resultNode)
    } else {
      for k, v := range node {
        assertValue(t, resultNode.GetPath([]string{k}), v, location + "." + k)
      }
    }
  case int64:
    if resultNode, ok := result.(int64); !ok {
      t.Errorf("{%s} result value not of type %T: %T",
        location, node, resultNode)
    } else {
      if node != resultNode {
        t.Errorf("{%s} result value does not match", location)
      }
    }
  case string:
    if resultNode, ok := result.(string); !ok {
      t.Errorf("{%s} result value not of type %T: %T",
        location, node, resultNode)
    } else {
      if node != resultNode {
        t.Errorf("{%s} result value does not match", location)
      }
    }
  default:
    if fmt.Sprintf("%v", node) != fmt.Sprintf("%v", ref) {
      t.Errorf("{%s} result value does not match: %v != %v",
        location, node, ref)
    }
  }
}

func TestQueryRoot(t *testing.T) {
  assertQuery(t,
    "a = 42",
    "$",
	  []interface{}{
      map[string]interface{}{
		    "a": int64(42),
	    },
    })
}

/*
// NOTE: from the BurntSushi test suite
// NOTE: this test is pure evil due to the embedded '.'
func TestSpecialKV(t *testing.T) {
	tree, err := Load("~!@#$^&*()_+-`1234567890[]\\|/?><.,;: = 1")
	assertTree(t, tree, err, map[string]interface{}{
		"~!@#$^&*()_+-`1234567890[]\\|/?><.,;:": int64(1),
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
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": int64(42),
				},
			},
		},
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

func TestNestedEmptyArrays(t *testing.T) {
	tree, err := Load("a = [[[]]]")
	assertTree(t, tree, err, map[string]interface{}{
		"a": [][][]interface{}{[][]interface{}{[]interface{}{}}},
	})
}

func TestArrayMixedTypes(t *testing.T) {
	_, err := Load("a = [42, 16.0]")
	if err.Error() != "(1, 10): mixed types in array" {
		t.Error("Bad error message:", err.Error())
	}

	_, err = Load("a = [42, \"hello\"]")
	if err.Error() != "(1, 11): mixed types in array" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestArrayNestedStrings(t *testing.T) {
	tree, err := Load("data = [ [\"gamma\", \"delta\"], [\"Foo\"] ]")
	assertTree(t, tree, err, map[string]interface{}{
		"data": [][]string{[]string{"gamma", "delta"}, []string{"Foo"}},
	})
}

func TestMissingValue(t *testing.T) {
	_, err := Load("a = ")
	if err.Error() != "(1, 4): expecting a value" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestUnterminatedArray(t *testing.T) {
	_, err := Load("a = [1,")
	if err.Error() != "(1, 8): unterminated array" {
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

func TestDuplicateGroups(t *testing.T) {
	_, err := Load("[foo]\na=2\n[foo]b=3")
	if err.Error() != "(3, 2): duplicated tables" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestDuplicateKeys(t *testing.T) {
	_, err := Load("foo = 2\nfoo = 3")
	if err.Error() != "(2, 1): The following key was defined twice: foo" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestEmptyIntermediateTable(t *testing.T) {
	_, err := Load("[foo..bar]")
	if err.Error() != "(1, 2): empty intermediate table" {
		t.Error("Bad error message:", err.Error())
	}
}

func TestImplicitDeclarationBefore(t *testing.T) {
	tree, err := Load("[a.b.c]\nanswer = 42\n[a]\nbetter = 43")
	assertTree(t, tree, err, map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"answer": int64(42),
				},
			},
			"better": int64(43),
		},
	})
}

func TestFloatsWithoutLeadingZeros(t *testing.T) {
	_, err := Load("a = .42")
	if err.Error() != "(1, 4): cannot start float with a dot" {
		t.Error("Bad error message:", err.Error())
	}

	_, err = Load("a = -.42")
	if err.Error() != "(1, 5): cannot start float with a dot" {
		t.Error("Bad error message:", err.Error())
	}
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
		"title": "TOML Example",
		"owner": map[string]interface{}{
			"name":         "Tom Preston-Werner",
			"organization": "GitHub",
			"bio":          "GitHub Cofounder & CEO\nLikes tater tots and beer.",
			"dob":          time.Date(1979, time.May, 27, 7, 32, 0, 0, time.UTC),
		},
		"database": map[string]interface{}{
			"server":         "192.168.1.1",
			"ports":          []int64{8001, 8001, 8002},
			"connection_max": 5000,
			"enabled":        true,
		},
		"servers": map[string]interface{}{
			"alpha": map[string]interface{}{
				"ip": "10.0.0.1",
				"dc": "eqdc10",
			},
			"beta": map[string]interface{}{
				"ip": "10.0.0.2",
				"dc": "eqdc10",
			},
		},
		"clients": map[string]interface{}{
			"data": []interface{}{
				[]string{"gamma", "delta"},
				[]int64{1, 2},
			},
		},
	})
}

func TestParseKeyGroupArray(t *testing.T) {
	tree, err := Load("[[foo.bar]] a = 42\n[[foo.bar]] a = 69")
	assertTree(t, tree, err, map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": []map[string]interface{}{
				{"a": int64(42)},
				{"a": int64(69)},
			},
		},
	})
}

func TestParseKeyGroupArraySpec(t *testing.T) {
	tree, err := Load("[[fruit]]\n name=\"apple\"\n [fruit.physical]\n color=\"red\"\n shape=\"round\"\n [[fruit]]\n name=\"banana\"")
	assertTree(t, tree, err, map[string]interface{}{
		"fruit": []map[string]interface{}{
			{"name": "apple", "physical": map[string]interface{}{"color": "red", "shape": "round"}},
			{"name": "banana"},
		},
	})
}

func TestToTomlValue(t *testing.T) {
	for idx, item := range []struct {
		Value  interface{}
		Expect string
	}{
		{int64(12345), "12345"},
		{float64(123.45), "123.45"},
		{bool(true), "true"},
		{"hello world", "\"hello world\""},
		{"\b\t\n\f\r\"\\", "\"\\b\\t\\n\\f\\r\\\"\\\\\""},
		{"\x05", "\"\\u0005\""},
		{time.Date(1979, time.May, 27, 7, 32, 0, 0, time.UTC),
			"1979-05-27T07:32:00Z"},
		{[]interface{}{"gamma", "delta"},
			"[\n  \"gamma\",\n  \"delta\",\n]"},
	} {
		result := toTomlValue(item.Value, 0)
		if result != item.Expect {
			t.Errorf("Test %d - got '%s', expected '%s'", idx, result, item.Expect)
		}
	}
}

func TestToString(t *testing.T) {
	tree, err := Load("[foo]\n\n[[foo.bar]]\na = 42\n\n[[foo.bar]]\na = 69\n")
	if err != nil {
		t.Errorf("Test failed to parse: %v", err)
		return
	}
	result := tree.ToString()
	expected := "\n[foo]\n\n  [[foo.bar]]\n    a = 42\n\n  [[foo.bar]]\n    a = 69\n"
	if result != expected {
		t.Errorf("Expected got '%s', expected '%s'", result, expected)
	}
}

func assertPosition(t *testing.T, text string, ref map[string]Position) {
	tree, err := Load(text)
	if err != nil {
		t.Errorf("Error loading document text: `%v`", text)
		t.Errorf("Error: %v", err)
	}
	for path, pos := range ref {
		testPos := tree.GetPosition(path)
		if testPos.Invalid() {
			t.Errorf("Failed to query tree path: %s", path)
		} else if pos != testPos {
			t.Errorf("Expected position %v, got %v instead", pos, testPos)
		}
	}
}

func TestDocumentPositions(t *testing.T) {
	assertPosition(t,
		"[foo]\nbar=42\nbaz=69",
		map[string]Position{
			"foo":     Position{1, 1},
			"foo.bar": Position{2, 1},
			"foo.baz": Position{3, 1},
		})
}

func TestDocumentPositionsWithSpaces(t *testing.T) {
	assertPosition(t,
		"  [foo]\n  bar=42\n  baz=69",
		map[string]Position{
			"foo":     Position{1, 3},
			"foo.bar": Position{2, 3},
			"foo.baz": Position{3, 3},
		})
}

func TestDocumentPositionsWithGroupArray(t *testing.T) {
	assertPosition(t,
		"[[foo]]\nbar=42\nbaz=69",
		map[string]Position{
			"foo":     Position{1, 1},
			"foo.bar": Position{2, 1},
			"foo.baz": Position{3, 1},
		})
}

func TestDocumentPositionsEmptyPath(t *testing.T) {
	text := "[foo]\nbar=42\nbaz=69"
	tree, err := Load(text)
	if err != nil {
		t.Errorf("Error loading document text: `%v`", text)
		t.Errorf("Error: %v", err)
	}
	if pos := tree.GetPosition(""); !pos.Invalid() {
		t.Errorf("Valid position was returned for empty path")
	}
}
*/
