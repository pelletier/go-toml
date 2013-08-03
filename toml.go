// TOML markup language parser.
//
// This version supports the specification as described in
// https://github.com/mojombo/toml/tree/e3656ad493400895f4460f1244a25f8f8e31a32a
package toml

import (
	"errors"
	"io/ioutil"
	"runtime"
	"strings"
)

// Definition of a TomlTree.
// This is the result of the parsing of a TOML file.
type TomlTree struct {
	Values   map[string]interface{}
	Comments map[string]Comment
}

// Definition for comment
type Comment struct {
	Multiline []string
	EndOfLine string
}

func (t *TomlTree) Init() {
	t.Values = make(map[string]interface{})
	t.Comments = make(map[string]Comment)
}

// Keys returns the keys of the toplevel tree.
// Warning: this is a costly operation.
func (t *TomlTree) Keys() []string {
	keys := make([]string, 0)
	//mp := (map[string]interface{})(*t)
	//for k, _ := range mp {
	for k, _ := range t.Values {
		keys = append(keys, k)
	}
	return keys
}

// Get the value at key in the TomlTree.
// Key is a dot-separated path (e.g. a.b.c).
// Returns nil if the path does not exist in the tree.
func (t *TomlTree) Get(key string) interface{} {
	//subtree := t
	subtree := t.Values
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		//_, exists := (*subtree)[intermediate_key]
		_, exists := subtree[intermediate_key]
		if !exists {
			return nil
		}
		//subtree = (*subtree)[intermediate_key].(*TomlTree)
		subtree = subtree[intermediate_key].(*TomlTree).Values
	}
	//return (*subtree)[keys[len(keys)-1]]
	return subtree[keys[len(keys)-1]]
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
	//subtree := t
	subtree := t.Values
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		//_, exists := (*subtree)[intermediate_key]
		_, exists := subtree[intermediate_key]
		if !exists {
			//var new_tree TomlTree = make(TomlTree)
			//(*subtree)[intermediate_key] = &new_tree
			var new_tree TomlTree = TomlTree{}
			new_tree.Init()
			subtree[intermediate_key] = &new_tree
		}
		//subtree = (*subtree)[intermediate_key].(*TomlTree)
		subtree = subtree[intermediate_key].(*TomlTree).Values
	}
	//(*subtree)[keys[len(keys)-1]] = value
	subtree[keys[len(keys)-1]] = value
}

// Set Multi-line comment by key
// Key is a dot-separated path (e.g. a.b.c).
func (t *TomlTree) SetComments(key string, multiline ...string) {
	c, exists := t.Comments[key]
	if !exists {
		c = Comment{}
	}
	c.Multiline = multiline
	t.Comments[key] = c
}

// Set End-Of-Line comment by key
// Key is a dot-separated path (e.g. a.b.c).
func (t *TomlTree) SetComment(key string, endofline string) {
	c, exists := t.Comments[key]
	if !exists {
		c = Comment{}
	}
	c.EndOfLine = endofline
	t.Comments[key] = c
}

// Get Comment by key
// Key is a dot-separated path (e.g. a.b.c).
func (t *TomlTree) GetComment(key string) Comment {
	c, exists := t.Comments[key]
	if !exists {
		return Comment{}
	}
	return c
}

// createSubTree takes a tree and a key and create the necessary intermediate
// subtrees to create a subtree at that point. In-place.
//
// e.g. passing a.b.c will create (assuming tree is empty) tree[a], tree[a][b]
// and tree[a][b][c]
func (t *TomlTree) createSubTree(key string) {
	//subtree := t
	subtree := t.Values
	for _, intermediate_key := range strings.Split(key, ".") {
		//_, exists := (*subtree)[intermediate_key]
		_, exists := subtree[intermediate_key]
		if !exists {
			//var new_tree TomlTree = make(TomlTree)
			//(*subtree)[intermediate_key] = &new_tree
			var new_tree TomlTree = TomlTree{}
			new_tree.Init()
			subtree[intermediate_key] = &new_tree
		}
		//subtree = (*subtree)[intermediate_key].(*TomlTree)
		subtree = subtree[intermediate_key].(*TomlTree).Values
	}
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
