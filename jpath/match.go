package jpath

import (
  . "github.com/pelletier/go-toml"
)

type QueryPath []PathFn

type PathFn func(context interface{}, next QueryPath)

func (path QueryPath) Fn(context interface{}) {
  path[0](context, path[1:])
}

func treeValue(tree *TomlTree, key string) interface{} {
  return tree.GetPath([]string{key})
}

func matchKeyFn(name string) PathFn {
  return func(context interface{}, next QueryPath) {
    if tree, ok := context.(*TomlTree); ok {
      item := treeValue(tree, name)
      if item != nil {
        next.Fn(item)
      }
    }
  }
}

func matchIndexFn(idx int) PathFn {
  return func(context interface{}, next QueryPath) {
    if arr, ok := context.([]interface{}); ok {
      if idx < len(arr) && idx >= 0 {
        next.Fn(arr[idx])
      }
    }
  }
}

func matchSliceFn(start, end, step int) PathFn {
  return func(context interface{}, next QueryPath) {
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
        next.Fn(arr[idx])
      }
    }
  }
}

func matchAnyFn() PathFn {
  return func(context interface{}, next QueryPath) {
    if tree, ok := context.(*TomlTree); ok {
      for _, key := range tree.Keys() {
        item := treeValue(tree, key)
        next.Fn(item)
      }
    }
  }
}

func matchUnionFn(union QueryPath) PathFn {
  return func(context interface{}, next QueryPath) {
    for _, fn := range union {
      fn(context, next)
    }
  }
}

func matchRecurseFn() PathFn {
  return func(context interface{}, next QueryPath) {
    if tree, ok := context.(*TomlTree); ok {
      var visit func(tree *TomlTree)
      visit = func(tree *TomlTree) {
        for _, key := range tree.Keys() {
          item := treeValue(tree, key)
          next.Fn(item)
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
  }
}


func processPath(path QueryPath, context interface{}) []interface{} {
  // terminate the path with a collection funciton
  result := []interface{}{}
  newPath := append(path, func(context interface{}, next QueryPath) {
    result = append(result, context)
  })

  // execute the path
  newPath.Fn(context)
  return result
}
