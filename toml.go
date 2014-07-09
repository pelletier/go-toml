// TOML markup language parser.
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

// Definition of a TomlTree.
// This is the result of the parsing of a TOML file.
type TomlTree map[string]interface{}

// Has returns a boolean indicating if the given key exists.
func (t *TomlTree) Has(key string) bool {
	if key == "" {
		return false
	}
	return t.HasPath(strings.Split(key, "."))
}

// Returns true if the given path of keys exists, false otherwise.
func (t *TomlTree) HasPath(keys []string) bool {
	if len(keys) == 0 {
		return false
	}
	subtree := t
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			return false
		}
		switch node := (*subtree)[intermediate_key].(type) {
		case *TomlTree:
			subtree = node
		case []*TomlTree:
			// go to most recent element
			if len(node) == 0 {
				return false
			}
			subtree = node[len(node)-1]
		}
	}
	return true
}

// Keys returns the keys of the toplevel tree.
// Warning: this is a costly operation.
func (t *TomlTree) Keys() []string {
	keys := make([]string, 0)
	mp := (map[string]interface{})(*t)
	for k, _ := range mp {
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

// Returns the element in the tree indicated by 'keys'.
// If keys is of length zero, the current tree is returned.
func (t *TomlTree) GetPath(keys []string) interface{} {
	if len(keys) == 0 {
		return t
	}
	subtree := t
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			return nil
		}
		switch node := (*subtree)[intermediate_key].(type) {
		case *TomlTree:
			subtree = node
		case []*TomlTree:
			// go to most recent element
			if len(node) == 0 {
				return nil //(*subtree)[intermediate_key] = append(node, &TomlTree{})
			}
			subtree = node[len(node)-1]
		}
	}
	return (*subtree)[keys[len(keys)-1]]
}

// Same as Get but with a default value
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

func (t *TomlTree) SetPath(keys []string, value interface{}) {
	subtree := t
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			var new_tree TomlTree = make(TomlTree)
			(*subtree)[intermediate_key] = &new_tree
		}
		switch node := (*subtree)[intermediate_key].(type) {
		case *TomlTree:
			subtree = node
		case []*TomlTree:
			// go to most recent element
			if len(node) == 0 {
				(*subtree)[intermediate_key] = append(node, &TomlTree{})
			}
			subtree = node[len(node)-1]
		}
	}
	(*subtree)[keys[len(keys)-1]] = value
}

// createSubTree takes a tree and a key and create the necessary intermediate
// subtrees to create a subtree at that point. In-place.
//
// e.g. passing a.b.c will create (assuming tree is empty) tree[a], tree[a][b]
// and tree[a][b][c]
func (t *TomlTree) createSubTree(key string) {
	subtree := t
	for _, intermediate_key := range strings.Split(key, ".") {
		if intermediate_key == "" {
			panic("empty intermediate table")
		}
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			var new_tree TomlTree = make(TomlTree)
			(*subtree)[intermediate_key] = &new_tree
		}
		subtree = ((*subtree)[intermediate_key]).(*TomlTree)
	}
}

// encodes a string to a TOML-compliant string value
func encodeTomlString(value string) string {
	result := ""
	for _, rr := range value {
		int_rr := uint16(rr)
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
			if int_rr < 0x001F {
				result += fmt.Sprintf("\\u%0.4X", int_rr)
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
		} else {
			return "false"
		}
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
func (t *TomlTree) toToml(keyspace string) string {
	result := ""
	for k, v := range (map[string]interface{})(*t) {
		// figure out the keyspace
		combined_key := k
		if keyspace != "" {
			combined_key = keyspace + "." + combined_key
		}
		// output based on type
		switch node := v.(type) {
		case []*TomlTree:
			for _, item := range node {
				if len(item.Keys()) > 0 {
					result += fmt.Sprintf("\n[[%s]]\n", combined_key)
				}
				result += item.toToml(combined_key)
			}
		case *TomlTree:
			if len(node.Keys()) > 0 {
				result += fmt.Sprintf("\n[%s]\n", combined_key)
			}
			result += node.toToml(combined_key)
		default:
			result += fmt.Sprintf("%s = %s\n", k, toTomlValue(node, 0))
		}
	}
	return result
}

// Generates a human-readable representation of the current tree.
// Output spans multiple lines, and is suitable for ingest by a TOML parser
func (t *TomlTree) ToString() string {
	return t.toToml("")
}

// Create a TomlTree from a string.
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

// Create a TomlTree from a file.
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
