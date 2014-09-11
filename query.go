package toml

type nodeFilterFn func(node interface{}) bool
type nodeFn func(node interface{}) interface{}

type QueryResult struct {
	items []interface{}
  positions []Position
}

func (r *QueryResult) appendResult(node interface{}, pos Position) {
  r.items = append(r.items, node)
  r.positions = append(r.positions, pos)
}

func (r *QueryResult) Values() []interface{} {
  return r.items
}

func (r *QueryResult) Positions() []Position {
  return r.positions
}

// runtime context for executing query paths
type queryContext struct {
  result *QueryResult
	filters *map[string]nodeFilterFn
	scripts *map[string]nodeFn
  lastPosition Position
}

// generic path functor interface
type PathFn interface {
	SetNext(next PathFn)
	Call(node interface{}, ctx *queryContext)
}

// encapsulates a query functor chain and script callbacks
type Query struct {
	root    PathFn
	tail    PathFn
	filters *map[string]nodeFilterFn
	scripts *map[string]nodeFn
}

func newQuery() *Query {
	return &Query{
		root:    nil,
		tail:    nil,
		filters: &defaultFilterFunctions,
		scripts: &defaultScriptFunctions,
	}
}

func (q *Query) appendPath(next PathFn) {
	if q.root == nil {
		q.root = next
	} else {
		q.tail.SetNext(next)
	}
	q.tail = next
	next.SetNext(newTerminatingFn()) // init the next functor
}

// TODO: return (err,query) instead
func Compile(path string) (*Query, error) {
	return parseQuery(lexQuery(path))
}

func (q *Query) Execute(tree *TomlTree) *QueryResult {
  result := &QueryResult {
    items: []interface{}{},
    positions: []Position{},
  }
	if q.root == nil {
    result.appendResult(tree, tree.GetPosition(""))
	} else {
    ctx := &queryContext{
      result: result,
      filters: q.filters,
      scripts: q.scripts,
    }
    q.root.Call(tree, ctx)
  }
	return result
}

func (q *Query) SetFilter(name string, fn nodeFilterFn) {
	if q.filters == &defaultFilterFunctions {
		// clone the static table
		q.filters = &map[string]nodeFilterFn{}
		for k, v := range defaultFilterFunctions {
			(*q.filters)[k] = v
		}
	}
	(*q.filters)[name] = fn
}

func (q *Query) SetScript(name string, fn nodeFn) {
	if q.scripts == &defaultScriptFunctions {
		// clone the static table
		q.scripts = &map[string]nodeFn{}
		for k, v := range defaultScriptFunctions {
			(*q.scripts)[k] = v
		}
	}
	(*q.scripts)[name] = fn
}

var defaultFilterFunctions = map[string]nodeFilterFn{
	"odd": func(node interface{}) bool {
		if ii, ok := node.(int64); ok {
			return (ii & 1) == 1
		}
		return false
	},
	"even": func(node interface{}) bool {
		if ii, ok := node.(int64); ok {
			return (ii & 1) == 0
		}
		return false
	},
}

var defaultScriptFunctions = map[string]nodeFn{
	"last": func(node interface{}) interface{} {
		if arr, ok := node.([]interface{}); ok {
			return len(arr) - 1
		}
		return nil
	},
}
