package jpath

import (
	"fmt"
	. "github.com/pelletier/go-toml"
	"testing"
)

func assertQuery(t *testing.T, toml, query string, ref []interface{}) {
	tree, err := Load(toml)
	if err != nil {
		t.Errorf("Non-nil toml parse error: %v", err)
		return
	}
  results := Compile(query).Execute(tree)
	assertValue(t, results, ref, "((" + query + ")) -> ")
}

func assertValue(t *testing.T, result, ref interface{}, location string) {
	switch node := ref.(type) {
	case []interface{}:
		if resultNode, ok := result.([]interface{}); !ok {
			t.Errorf("{%s} result value not of type %T: %T",
				location, node, resultNode)
		} else {
      if len(node) != len(resultNode) {
        t.Errorf("{%s} lengths do not match: %v vs %v",
          location, node, resultNode)
      } else {
        for i, v := range node {
          assertValue(t, resultNode[i], v, fmt.Sprintf("%s[%d]", location, i))
        }
      }
		}
	case map[string]interface{}:
		if resultNode, ok := result.(*TomlTree); !ok {
			t.Errorf("{%s} result value not of type %T: %T",
				location, node, resultNode)
		} else {
			for k, v := range node {
				assertValue(t, resultNode.GetPath([]string{k}), v, location+"."+k)
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
