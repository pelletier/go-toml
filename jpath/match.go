package jpath

import (
  . "github.com/pelletier/go-toml"
)

type PathFn func(context interface{}) []interface{}

func treeValue(tree *TomlTree, key string) interface{} {
  return tree.GetPath([]string{key})
}

func matchKeyFn(name string) PathFn {
  return func(context interface{}) []interface{} {
    if tree, ok := context.(*TomlTree); ok {
      item := treeValue(tree, name)
      if item != nil {
        return []interface{}{ item }
      }
    }
    return []interface{}{}
  }
}

func matchIndexFn(idx int) PathFn {
  return func(context interface{}) []interface{} {
    if arr, ok := context.([]interface{}); ok {
      if idx < len(arr) && idx >= 0 {
        return arr[idx:idx+1]
      }
    }
    return []interface{}{}
  }
}

func matchSliceFn(start, end, step int) PathFn {
  return func(context interface{}) []interface{} {
    result := []interface{}{}
    if arr, ok := context.([]interface{}); ok {
      // adjust indexes for negative values, reverse ordering
      realStart, realEnd := start, end
      if realStart < 0 {
        realStart = len(arr) + realStart
      }
      if realEnd < 0 {
        realEnd = len(arr) + realEnd
      }
      if realEnd < realStart {
        realEnd, realStart = realStart, realEnd  // swap
      }
      // loop and gather
      for idx := realStart; idx < realEnd; idx += step {
         result = append(result, arr[idx])
      }
    }
    return result
  }
}

func matchAnyFn() PathFn {
  return func(context interface{}) []interface{} {
    result := []interface{}{}
    if tree, ok := context.(*TomlTree); ok {
      for _, key := range tree.Keys() {
        item := treeValue(tree, key)
        result = append(result, item)
      }
    }
    return result
  }
}

func matchUnionFn(union []PathFn) PathFn {
  return func(context interface{}) []interface{} {
    result := []interface{}{}
    for _, fn := range union {
      result = append(result, fn(context)...)
    }
    return result
  }
}

func matchRecurseFn() PathFn {
  return func(context interface{}) []interface{} {
    result := []interface{}{ context }

    if tree, ok := context.(*TomlTree); ok {
      var visit func(tree *TomlTree)
      visit = func(tree *TomlTree) {
        for _, key := range tree.Keys() {
          item := treeValue(tree, key)
          result = append(result, item)
          switch node := item.(type) {
          case *TomlTree:
            visit(node)
          case []*TomlTree:
            for _, subtree := range node {
              visit(subtree)
            }
          }
        }
      }
      visit(tree)
    }
    return result
  }
}

func processPath(path []PathFn, context interface{}) []interface{} {
  result := []interface{}{ context }  // start with the root
  for _, fn := range path {
    next := []interface{}{}
    for _, ctx := range result {
      next = append(next, fn(ctx)...)
    }
    if len(next) == 0 {
      return next // exit if there is nothing more to search
    }
    result = next // prep the next iteration
  }
  return result
}
