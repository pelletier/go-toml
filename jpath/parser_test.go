package jpath

import (
	"fmt"
	. "github.com/pelletier/go-toml"
	"testing"
  "sort"
  "strings"
)

func valueString(root interface{}) string {
  result := "" //fmt.Sprintf("%T:", root)
	switch node := root.(type) {
	case []interface{}:
    items := []string{}
    for _, v := range node {
      items = append(items, valueString(v))
    }
    sort.Strings(items)
    result = "[" + strings.Join(items, ", ") + "]"
	case *TomlTree:
    // workaround for unreliable map key ordering
    items := []string{}
    for _, k := range node.Keys() {
      v := node.GetPath([]string{k})
      items = append(items, k + ":" + valueString(v))
    }
    sort.Strings(items)
    result = "{" + strings.Join(items, ", ") + "}"
	case map[string]interface{}:
    // workaround for unreliable map key ordering
    items := []string{}
    for k, v := range node {
      items = append(items, k + ":" + valueString(v))
    }
    sort.Strings(items)
    result = "{" + strings.Join(items, ", ") + "}"
	case int64:
    result += fmt.Sprintf("%d", node)
  case string:
    result += "'" + node + "'"
  }
  return result
}

func assertValue(t *testing.T, result, ref interface{}) {
  pathStr := valueString(result)
  refStr := valueString(ref)
  if pathStr != refStr {
    t.Errorf("values do not match")
		t.Log("test:", pathStr)
		t.Log("ref: ", refStr)
  }
}

func assertQuery(t *testing.T, toml, query string, ref []interface{}) {
	tree, err := Load(toml)
	if err != nil {
		t.Errorf("Non-nil toml parse error: %v", err)
		return
	}
	results := Compile(query).Execute(tree)
	assertValue(t, results.Values(), ref)
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

func TestQueryKey(t *testing.T) {
	assertQuery(t,
		"[foo]\na = 42",
		"$.foo.a",
		[]interface{}{
			int64(42),
		})
}

func TestQueryKeyString(t *testing.T) {
	assertQuery(t,
		"[foo]\na = 42",
		"$.foo['a']",
		[]interface{}{
			int64(42),
		})
}

func TestQueryIndex(t *testing.T) {
	assertQuery(t,
		"[foo]\na = [1,2,3,4,5,6,7,8,9,0]",
		"$.foo.a[0]",
		[]interface{}{
			int64(1),
		})
}

func TestQuerySliceRange(t *testing.T) {
	assertQuery(t,
		"[foo]\na = [1,2,3,4,5,6,7,8,9,0]",
		"$.foo.a[0:5]",
		[]interface{}{
			int64(1),
			int64(2),
			int64(3),
			int64(4),
			int64(5),
		})
}

func TestQuerySliceStep(t *testing.T) {
	assertQuery(t,
		"[foo]\na = [1,2,3,4,5,6,7,8,9,0]",
		"$.foo.a[0:5:2]",
		[]interface{}{
			int64(1),
			int64(3),
			int64(5),
		})
}

func TestQueryAny(t *testing.T) {
	assertQuery(t,
		"[foo.bar]\na=1\nb=2\n[foo.baz]\na=3\nb=4",
		"$.foo.*",
		[]interface{}{
			map[string]interface{}{
				"a": int64(1),
				"b": int64(2),
			},
			map[string]interface{}{
				"a": int64(3),
				"b": int64(4),
			},
		})
}
func TestQueryUnionSimple(t *testing.T) {
	assertQuery(t,
		"[foo.bar]\na=1\nb=2\n[baz.foo]\na=3\nb=4\n[gorf.foo]\na=5\nb=6",
		"$.*[bar,foo]",
		[]interface{}{
			map[string]interface{}{
				"a": int64(1),
				"b": int64(2),
			},
			map[string]interface{}{
				"a": int64(3),
				"b": int64(4),
			},
			map[string]interface{}{
				"a": int64(5),
				"b": int64(6),
			},
		})
}

func TestQueryRecursionAll(t *testing.T) {
	assertQuery(t,
		"[foo.bar]\na=1\nb=2\n[baz.foo]\na=3\nb=4\n[gorf.foo]\na=5\nb=6",
		"$..*",
		[]interface{}{
			map[string]interface{}{
				"bar": map[string]interface{}{
					"a": int64(1),
					"b": int64(2),
				},
			},
			map[string]interface{}{
				"a": int64(1),
				"b": int64(2),
			},
			int64(1),
			int64(2),
			map[string]interface{}{
				"foo": map[string]interface{}{
					"a": int64(3),
					"b": int64(4),
				},
			},
			map[string]interface{}{
				"a": int64(3),
				"b": int64(4),
			},
			int64(3),
			int64(4),
			map[string]interface{}{
				"foo": map[string]interface{}{
					"a": int64(5),
					"b": int64(6),
				},
			},
			map[string]interface{}{
				"a": int64(5),
				"b": int64(6),
			},
			int64(5),
			int64(6),
		})
}

func TestQueryRecursionUnionSimple(t *testing.T) {
	assertQuery(t,
		"[foo.bar]\na=1\nb=2\n[baz.foo]\na=3\nb=4\n[gorf.foo]\na=5\nb=6",
		"$..['foo','bar']",
		[]interface{}{
			map[string]interface{}{
				"a": int64(1),
				"b": int64(2),
			},
			map[string]interface{}{
				"a": int64(3),
				"b": int64(4),
			},
			map[string]interface{}{
				"a": int64(5),
				"b": int64(6),
			},
		})
}

func TestQueryScriptFnLast(t *testing.T) {
	assertQuery(t,
		"[foo]\na = [0,1,2,3,4,5,6,7,8,9]",
		"$.foo.a[(last)]",
		[]interface{}{
			int64(9),
		})
}

func TestQueryFilterFnOdd(t *testing.T) {
	assertQuery(t,
		"[foo]\na = [0,1,2,3,4,5,6,7,8,9]",
		"$.foo.a[?(odd)]",
		[]interface{}{
			int64(1),
			int64(3),
			int64(5),
			int64(7),
			int64(9),
		})
}

func TestQueryFilterFnEven(t *testing.T) {
	assertQuery(t,
		"[foo]\na = [0,1,2,3,4,5,6,7,8,9]",
		"$.foo.a[?(even)]",
		[]interface{}{
			int64(0),
			int64(2),
			int64(4),
			int64(6),
			int64(8),
		})
}
