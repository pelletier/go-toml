package jpath

import (
	. "github.com/pelletier/go-toml"
)

// result set for storage of results
type pathResult struct {
  Values []interface{}
}

func newPathResult() *pathResult {
  return &pathResult {
    Values: []interface{}{},
  }
}

func (r *pathResult) Append(value interface{}) {
  r.Values = append(r.Values, value)
}

// generic path functor interface
type PathFn interface{
  SetNext(next PathFn)
  Call(context interface{}, results *pathResult)
}

// contains a functor chain
type QueryPath struct {
  root PathFn
  tail PathFn
}

func newQueryPath() *QueryPath {
  return &QueryPath {
    root: nil,
    tail: nil,
  }
}

func (path *QueryPath) Append(next PathFn) {
  if path.root == nil {
    path.root = next
  } else {
    path.tail.SetNext(next)
  }
  path.tail = next
  next.SetNext(newTerminatingFn()) // init the next functor
}

func (path *QueryPath) Call(context interface{}) []interface{} {
  results := newPathResult()
  if path.root == nil {
    results.Append(context)  // identity query for no predicates
  } else {
    path.root.Call(context, results)
  }
  return results.Values
}

// base match
type matchBase struct {
  next PathFn
}

func (f *matchBase) SetNext(next PathFn) {
  f.next = next
}

// terminating functor - gathers results
type terminatingFn struct {
  // empty
}

func newTerminatingFn() *terminatingFn {
  return &terminatingFn{}
}

func (f *terminatingFn) SetNext(next PathFn) {
  // do nothing
}

func (f *terminatingFn) Call(context interface{}, results *pathResult) {
  results.Append(context)
}

// shim to ease functor writing
func treeValue(tree *TomlTree, key string) interface{} {
	return tree.GetPath([]string{key})
}

// match single key
type matchKeyFn struct {
  matchBase
  Name string
}

func newMatchKeyFn(name string) *matchKeyFn {
  return &matchKeyFn{ Name: name }
}

func (f *matchKeyFn) Call(context interface{}, results *pathResult) {
  if tree, ok := context.(*TomlTree); ok {
    item := treeValue(tree, f.Name)
    if item != nil {
      f.next.Call(item, results)
    }
  }
}

// match single index
type matchIndexFn struct {
  matchBase
  Idx int
}

func newMatchIndexFn(idx int) *matchIndexFn {
  return &matchIndexFn{ Idx: idx }
}

func (f *matchIndexFn) Call(context interface{}, results *pathResult) {
  if arr, ok := context.([]interface{}); ok {
    if f.Idx < len(arr) && f.Idx >= 0 {
      f.next.Call(arr[f.Idx], results)
    }
  }
}

// filter by slicing
type matchSliceFn struct {
  matchBase
  Start, End, Step int
}

func newMatchSliceFn(start, end, step int) *matchSliceFn {
  return &matchSliceFn{ Start: start, End: end, Step: step }
}

func (f *matchSliceFn) Call(context interface{}, results *pathResult) {
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
      f.next.Call(arr[idx], results)
    }
  }
}

// match anything
type matchAnyFn struct {
  matchBase
  // empty
}

func newMatchAnyFn() *matchAnyFn {
  return &matchAnyFn{}
}

func (f *matchAnyFn) Call(context interface{}, results *pathResult) {
  if tree, ok := context.(*TomlTree); ok {
    for _, key := range tree.Keys() {
      item := treeValue(tree, key)
      f.next.Call(item, results)
    }
  }
}

// filter through union
type matchUnionFn struct {
  Union []PathFn
}

func (f *matchUnionFn) SetNext(next PathFn) {
  for _, fn := range f.Union {
    fn.SetNext(next)
  }
}

func (f *matchUnionFn) Call(context interface{}, results *pathResult) {
  for _, fn := range f.Union {
    fn.Call(context, results)
  }
}

// match every single last node in the tree
type matchRecursiveFn struct {
  matchBase
}

func newMatchRecursiveFn() *matchRecursiveFn{
  return &matchRecursiveFn{}
}

func (f *matchRecursiveFn) Call(context interface{}, results *pathResult) {
  if tree, ok := context.(*TomlTree); ok {
    var visit func(tree *TomlTree)
    visit = func(tree *TomlTree) {
      for _, key := range tree.Keys() {
        item := treeValue(tree, key)
        f.next.Call(item, results)
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
