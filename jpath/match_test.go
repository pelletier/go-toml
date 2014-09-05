package jpath

import (
  "fmt"
  "math"
	"testing"
)

func pathString(path QueryPath) string {
  result := "["
  for _, v := range path {
    result += fmt.Sprintf("%T:%v, ", v, v)
  }
  return result + "]"
}

func assertPathMatch(t *testing.T, path, ref QueryPath) bool {
  if len(path) != len(ref) {
    t.Errorf("lengths do not match: %v vs %v",
      pathString(path), pathString(ref))
    return false
  } else {
    for i, v := range ref {
      pass := false
      node := path[i]
      // compare by value
      switch refNode := v.(type) {
      case *matchKeyFn:
        castNode, ok := node.(*matchKeyFn)
        pass = ok && (*refNode == *castNode)
      case *matchIndexFn:
        castNode, ok := node.(*matchIndexFn)
        pass = ok && (*refNode == *castNode)
      case *matchSliceFn:
        castNode, ok := node.(*matchSliceFn)
        pass = ok && (*refNode == *castNode)
      case *matchAnyFn:
        castNode, ok := node.(*matchAnyFn)
        pass = ok && (*refNode == *castNode)
      case *matchUnionFn:
        castNode, ok := node.(*matchUnionFn)
        // special case - comapre all contents
        pass = ok && assertPathMatch(t, castNode.Union, refNode.Union)
      case *matchRecursiveFn:
        castNode, ok := node.(*matchRecursiveFn)
        pass = ok && (*refNode == *castNode)
      }
      if !pass {
        t.Errorf("paths do not match at index %d: %v vs %v",
          i, pathString(path), pathString(ref))
        return false
      }
    }
  }
  return true
}

func assertPath(t *testing.T, query string, ref QueryPath) {
	_, flow := lex(query)
	path := parse(flow)
  assertPathMatch(t, path, ref)
}

func TestPathRoot(t *testing.T) {
	assertPath(t,
		"$",
		QueryPath{
      // empty
    })
}

func TestPathKey(t *testing.T) {
	assertPath(t,
		"$.foo",
		QueryPath{
      &matchKeyFn{ "foo" },
    })
}

func TestPathBracketKey(t *testing.T) {
	assertPath(t,
		"$[foo]",
		QueryPath{
      &matchKeyFn{ "foo" },
    })
}

func TestPathBracketStringKey(t *testing.T) {
	assertPath(t,
		"$['foo']",
		QueryPath{
      &matchKeyFn{ "foo" },
    })
}

func TestPathIndex(t *testing.T) {
	assertPath(t,
		"$[123]",
		QueryPath{
      &matchIndexFn{ 123 },
    })
}

func TestPathSliceStart(t *testing.T) {
	assertPath(t,
		"$[123:]",
		QueryPath{
      &matchSliceFn{ 123, math.MaxInt64, 1 },
    })
}

func TestPathSliceStartEnd(t *testing.T) {
	assertPath(t,
		"$[123:456]",
		QueryPath{
      &matchSliceFn{ 123, 456, 1 },
    })
}

func TestPathSliceStartEndColon(t *testing.T) {
	assertPath(t,
		"$[123:456:]",
		QueryPath{
      &matchSliceFn{ 123, 456, 1 },
    })
}

func TestPathSliceStartStep(t *testing.T) {
	assertPath(t,
		"$[123::7]",
		QueryPath{
      &matchSliceFn{ 123, math.MaxInt64, 7 },
    })
}

func TestPathSliceEndStep(t *testing.T) {
	assertPath(t,
		"$[:456:7]",
		QueryPath{
      &matchSliceFn{ 0, 456, 7 },
    })
}

func TestPathSliceStep(t *testing.T) {
	assertPath(t,
		"$[::7]",
		QueryPath{
      &matchSliceFn{ 0, math.MaxInt64, 7 },
    })
}

func TestPathSliceAll(t *testing.T) {
	assertPath(t,
		"$[123:456:7]",
		QueryPath{
      &matchSliceFn{ 123, 456, 7 },
    })
}

func TestPathAny(t *testing.T) {
	assertPath(t,
		"$.*",
		QueryPath{
      &matchAnyFn{},
    })
}

func TestPathUnion(t *testing.T) {
	assertPath(t,
		"$[foo, bar, baz]",
		QueryPath{
      &matchUnionFn{ []PathFn {
        &matchKeyFn{ "foo" },
        &matchKeyFn{ "bar" },
        &matchKeyFn{ "baz" },
      }},
    })
}

func TestPathRecurse(t *testing.T) {
	assertPath(t,
		"$..*",
		QueryPath{
      &matchRecursiveFn{},
    })
}
