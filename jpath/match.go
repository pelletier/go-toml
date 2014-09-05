package jpath

import (
	. "github.com/pelletier/go-toml"
)

type PathFn interface{
  Call(context interface{}, next QueryPath)
}

type QueryPath []PathFn

func (path QueryPath) Fn(context interface{}) {
	path[0].Call(context, path[1:])
}

// shim to ease functor writing
func treeValue(tree *TomlTree, key string) interface{} {
	return tree.GetPath([]string{key})
}

// match single key
type matchKeyFn struct {
  Name string
}

func (f *matchKeyFn) Call(context interface{}, next QueryPath) {
  if tree, ok := context.(*TomlTree); ok {
    item := treeValue(tree, f.Name)
    if item != nil {
      next.Fn(item)
    }
  }
}

// match single index
type matchIndexFn struct {
  Idx int
}

func (f *matchIndexFn) Call(context interface{}, next QueryPath) {
  if arr, ok := context.([]interface{}); ok {
    if f.Idx < len(arr) && f.Idx >= 0 {
      next.Fn(arr[f.Idx])
    }
  }
}

// filter by slicing
type matchSliceFn struct {
  Start, End, Step int
}

func (f *matchSliceFn) Call(context interface{}, next QueryPath) {
  if arr, ok := context.([]interface{}); ok {
    // adjust indexes for negative values, reverse ordering
    realStart, realEnd := f.Start, f.End
    if realStart < 0 {
      realStart = len(arr) + realStart
    }
    if realEnd < 0 {
      realEnd = len(arr) + realEnd
    }
    if realEnd < realStart {
      realEnd, realStart = realStart, realEnd // swap
    }
    // loop and gather
    for idx := realStart; idx < realEnd; idx += f.Step {
      next.Fn(arr[idx])
    }
  }
}

// match anything
type matchAnyFn struct {
  // empty
}

func (f *matchAnyFn) Call(context interface{}, next QueryPath) {
  if tree, ok := context.(*TomlTree); ok {
    for _, key := range tree.Keys() {
      item := treeValue(tree, key)
      next.Fn(item)
    }
  }
}

// filter through union
type matchUnionFn struct {
  Union QueryPath
}

func (f *matchUnionFn) Call(context interface{}, next QueryPath) {
  for _, fn := range f.Union {
    fn.Call(context, next)
  }
}

// match every single last node in the tree
type matchRecursiveFn struct {
  // empty
}

func (f *matchRecursiveFn) Call(context interface{}, next QueryPath) {
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

// terminating expression
type matchEndFn struct {
	Result []interface{}
}

func (f *matchEndFn) Call(context interface{}, next QueryPath) {
	f.Result = append(f.Result, context)
}

func processPath(path QueryPath, context interface{}) []interface{} {
	// terminate the path with a collection funciton
  endFn := &matchEndFn{ []interface{}{} }
	newPath := append(path, endFn)

	// execute the path
	newPath.Fn(context)
	return endFn.Result
}
