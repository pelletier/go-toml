// Package toml is a TOML markup language parser.
//
// This version supports the specification as described in
// https://github.com/toml-lang/toml/blob/master/versions/toml-v0.2.0.md
package toml

import (
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type tomlValue struct {
	value    interface{}
	position Position
}

// TomlTree is the result of the parsing of a TOML file.
type TomlTree struct {
	values   map[string]interface{}
	position Position
}

func newTomlTree() *TomlTree {
	return &TomlTree{
		values:   make(map[string]interface{}),
		position: Position{0, 0},
	}
}

// Has returns a boolean indicating if the given key exists.
func (t *TomlTree) Has(key string) bool {
	if key == "" {
		return false
	}
	return t.HasPath(strings.Split(key, "."))
}

// HasPath returns true if the given path of keys exists, false otherwise.
func (t *TomlTree) HasPath(keys []string) bool {
	return t.GetPath(keys) != nil
}

// Keys returns the keys of the toplevel tree.
// Warning: this is a costly operation.
func (t *TomlTree) Keys() []string {
	var keys []string
	for k := range t.values {
		keys = append(keys, k)
	}
	return keys
}

// Get the value at key in the TomlTree.
// Key is a dot-separated path (e.g. a.b.c).
// Returns nil if the path does not exist in the tree.
// If keys is of length zero, the current tree is returned.
func (t *TomlTree) Get(key string) interface{} {
	if key == "" {
		return t
	}
	return t.GetPath(strings.Split(key, "."))
}

// GetPath returns the element in the tree indicated by 'keys'.
// If keys is of length zero, the current tree is returned.
func (t *TomlTree) GetPath(keys []string) interface{} {
	if len(keys) == 0 {
		return t
	}
	subtree := t
	for _, intermediateKey := range keys[:len(keys)-1] {
		value, exists := subtree.values[intermediateKey]
		if !exists {
			return nil
		}
		switch node := value.(type) {
		case *TomlTree:
			subtree = node
		case []*TomlTree:
			// go to most recent element
			if len(node) == 0 {
				return nil
			}
			subtree = node[len(node)-1]
		default:
			return nil // cannot naigate through other node types
		}
	}
	// branch based on final node type
	switch node := subtree.values[keys[len(keys)-1]].(type) {
	case *tomlValue:
		return node.value
	default:
		return node
	}
}

// GetPosition returns the position of the given key.
func (t *TomlTree) GetPosition(key string) Position {
	if key == "" {
		return Position{0, 0}
	}
	return t.GetPositionPath(strings.Split(key, "."))
}

// GetPositionPath returns the element in the tree indicated by 'keys'.
// If keys is of length zero, the current tree is returned.
func (t *TomlTree) GetPositionPath(keys []string) Position {
	if len(keys) == 0 {
		return t.position
	}
	subtree := t
	for _, intermediateKey := range keys[:len(keys)-1] {
		value, exists := subtree.values[intermediateKey]
		if !exists {
			return Position{0, 0}
		}
		switch node := value.(type) {
		case *TomlTree:
			subtree = node
		case []*TomlTree:
			// go to most recent element
			if len(node) == 0 {
				return Position{0, 0}
			}
			subtree = node[len(node)-1]
		default:
			return Position{0, 0}
		}
	}
	// branch based on final node type
	switch node := subtree.values[keys[len(keys)-1]].(type) {
	case *tomlValue:
		return node.position
	case *TomlTree:
		return node.position
	case []*TomlTree:
		// go to most recent element
		if len(node) == 0 {
			return Position{0, 0}
		}
		return node[len(node)-1].position
	default:
		return Position{0, 0}
	}
}

// GetDefault works like Get but with a default value
func (t *TomlTree) GetDefault(key string, def interface{}) interface{} {
	val := t.Get(key)
	if val == nil {
		return def
	}
	return val
}

// Set an element in the tree.
// Key is a dot-separated path (e.g. a.b.c).
// Creates all necessary intermediates trees, if needed.
func (t *TomlTree) Set(key string, value interface{}) {
	t.SetPath(strings.Split(key, "."), value)
}

// SetPath sets an element in the tree.
// Keys is an array of path elements (e.g. {"a","b","c"}).
// Creates all necessary intermediates trees, if needed.
func (t *TomlTree) SetPath(keys []string, value interface{}) {
	subtree := t
	for _, intermediateKey := range keys[:len(keys)-1] {
		nextTree, exists := subtree.values[intermediateKey]
		if !exists {
			nextTree = newTomlTree()
			subtree.values[intermediateKey] = &nextTree // add new element here
		}
		switch node := nextTree.(type) {
		case *TomlTree:
			subtree = node
		case []*TomlTree:
			// go to most recent element
			if len(node) == 0 {
				// create element if it does not exist
				subtree.values[intermediateKey] = append(node, newTomlTree())
			}
			subtree = node[len(node)-1]
		}
	}
	subtree.values[keys[len(keys)-1]] = value
}

// createSubTree takes a tree and a key and create the necessary intermediate
// subtrees to create a subtree at that point. In-place.
//
// e.g. passing a.b.c will create (assuming tree is empty) tree[a], tree[a][b]
// and tree[a][b][c]
//
// Returns nil on success, error object on failure
func (t *TomlTree) createSubTree(keys []string) error {
	subtree := t
	for _, intermediateKey := range keys {
		if intermediateKey == "" {
			return fmt.Errorf("empty intermediate table")
		}
		nextTree, exists := subtree.values[intermediateKey]
		if !exists {
			nextTree = newTomlTree()
			subtree.values[intermediateKey] = nextTree
		}

		switch node := nextTree.(type) {
		case []*TomlTree:
			subtree = node[len(node)-1]
		case *TomlTree:
			subtree = node
		default:
			return fmt.Errorf("unknown type for path %s (%s)",
				strings.Join(keys, "."), intermediateKey)
		}
	}
	return nil
}

// encodes a string to a TOML-compliant string value
func encodeTomlString(value string) string {
	result := ""
	for _, rr := range value {
		intRr := uint16(rr)
		switch rr {
		case '\b':
			result += "\\b"
		case '\t':
			result += "\\t"
		case '\n':
			result += "\\n"
		case '\f':
			result += "\\f"
		case '\r':
			result += "\\r"
		case '"':
			result += "\\\""
		case '\\':
			result += "\\\\"
		default:
			if intRr < 0x001F {
				result += fmt.Sprintf("\\u%0.4X", intRr)
			} else {
				result += string(rr)
			}
		}
	}
	return result
}

// Value print support function for ToString()
// Outputs the TOML compliant string representation of a value
func toTomlValue(item interface{}, indent int) string {
	tab := strings.Repeat(" ", indent)
	switch value := item.(type) {
	case int64:
		return tab + strconv.FormatInt(value, 10)
	case float64:
		return tab + strconv.FormatFloat(value, 'f', -1, 64)
	case string:
		return tab + "\"" + encodeTomlString(value) + "\""
	case bool:
		if value {
			return "true"
		}
		return "false"
	case time.Time:
		return tab + value.Format(time.RFC3339)
	case []interface{}:
		result := tab + "[\n"
		for _, item := range value {
			result += toTomlValue(item, indent+2) + ",\n"
		}
		return result + tab + "]"
	default:
		panic(fmt.Sprintf("unsupported value type: %v", value))
	}
}

// Recursive support function for ToString()
// Outputs a tree, using the provided keyspace to prefix group names
func (t *TomlTree) toToml(indent, keyspace string) string {
	result := ""
	for k, v := range t.values {
		// figure out the keyspace
		combinedKey := k
		if keyspace != "" {
			combinedKey = keyspace + "." + combinedKey
		}
		// output based on type
		switch node := v.(type) {
		case []*TomlTree:
			for _, item := range node {
				if len(item.Keys()) > 0 {
					result += fmt.Sprintf("\n%s[[%s]]\n", indent, combinedKey)
				}
				result += item.toToml(indent+"  ", combinedKey)
			}
		case *TomlTree:
			if len(node.Keys()) > 0 {
				result += fmt.Sprintf("\n%s[%s]\n", indent, combinedKey)
			}
			result += node.toToml(indent+"  ", combinedKey)
		case *tomlValue:
			result += fmt.Sprintf("%s%s = %s\n", indent, k, toTomlValue(node.value, 0))
		default:
			panic(fmt.Sprintf("unsupported node type: %v", node))
		}
	}
	return result
}

// ToString generates a human-readable representation of the current tree.
// Output spans multiple lines, and is suitable for ingest by a TOML parser
func (t *TomlTree) ToString() string {
	return t.toToml("", "")
}

// Load creates a TomlTree from a string.
func Load(content string) (tree *TomlTree, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = errors.New(r.(string))
		}
	}()
	_, flow := lex(content)
	tree = parse(flow)
	return
}

// LoadFile creates a TomlTree from a file.
func LoadFile(path string) (tree *TomlTree, err error) {
	buff, ferr := ioutil.ReadFile(path)
	if ferr != nil {
		err = ferr
	} else {
		s := string(buff)
		tree, err = Load(s)
	}
	return
}
