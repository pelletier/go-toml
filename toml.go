// TOML markup language parser.
//
// This version supports the specification as described in
// https://github.com/mojombo/toml/tree/e3656ad493400895f4460f1244a25f8f8e31a32a
package toml

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// Definition of a TomlTree.
// This is the result of the parsing of a TOML file.
type TomlTree struct {
	FileName string
	Values   map[string]interface{}
	Comments map[string]Comment
	keyorder []string
}

// Definition for comment
type Comment struct {
	Multiline []string
	EndOfLine string
}

func (t *TomlTree) Init() {
	t.Values = make(map[string]interface{})
	t.Comments = make(map[string]Comment)
	t.keyorder = []string{}
}

// Keys returns the keys of the toplevel tree.
// Warning: this is a costly operation.
func (t *TomlTree) Keys() []string {
	keys := make([]string, 0)
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
		_, exists := subtree[intermediate_key]
		if !exists {
			return nil
		}
		subtree = subtree[intermediate_key].(*TomlTree).Values
	}
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
	parent := t
	subtree := t.Values
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := subtree[intermediate_key]
		if !exists {
			parent.keyorder = append(parent.keyorder, intermediate_key)
			new_tree := new(TomlTree)
			new_tree.Init()
			subtree[intermediate_key] = new_tree
		}
		parent = subtree[intermediate_key].(*TomlTree)
		subtree = parent.Values
	}
	_, exists := subtree[keys[len(keys)-1]]
	if !exists {
		parent.keyorder = append(parent.keyorder, keys[len(keys)-1])
	}
	subtree[keys[len(keys)-1]] = value
}

// Set Multi-line comment by key
// Key is a dot-separated path (e.g. a.b.c).
func (t *TomlTree) SetComments(key string, multiline ...string) {
	parent := t
	subtree := t.Values
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := subtree[intermediate_key]
		if !exists {
			new_tree := new(TomlTree)
			new_tree.Init()
			subtree[intermediate_key] = new_tree
		}
		parent = subtree[intermediate_key].(*TomlTree)
		subtree = parent.Values
	}
	key = keys[len(keys)-1]
	c, exists := parent.Comments[key]
	if !exists {
		c = Comment{}
	}
	c.Multiline = make([]string, len(multiline))
	for i, s := range multiline {
		c.Multiline[i] = strings.TrimSpace(s)
	}
	parent.Comments[key] = c
}

// Set End-Of-Line comment by key
// Key is a dot-separated path (e.g. a.b.c).
func (t *TomlTree) SetComment(key string, endofline string) {
	parent := t
	subtree := t.Values
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := subtree[intermediate_key]
		if !exists {
			new_tree := new(TomlTree)
			new_tree.Init()
			subtree[intermediate_key] = new_tree
		}
		parent = subtree[intermediate_key].(*TomlTree)
		subtree = parent.Values
	}
	key = keys[len(keys)-1]
	c, exists := parent.Comments[key]
	if !exists {
		c = Comment{}
	}
	c.EndOfLine = strings.TrimSpace(endofline)
	parent.Comments[key] = c
}

// Get Comment by key
// Key is a dot-separated path (e.g. a.b.c).
func (t *TomlTree) GetComment(key string) Comment {
	parent := t
	subtree := t.Values
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := subtree[intermediate_key]
		if !exists {
			return Comment{}
		}
		parent = subtree[intermediate_key].(*TomlTree)
		subtree = parent.Values
	}
	key = keys[len(keys)-1]
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
	parent := t
	subtree := t.Values
	for _, intermediate_key := range strings.Split(key, ".") {
		_, exists := subtree[intermediate_key]
		if !exists {
			parent.keyorder = append(parent.keyorder, intermediate_key)
			new_tree := new(TomlTree)
			new_tree.Init()
			subtree[intermediate_key] = new_tree
		}
		parent = subtree[intermediate_key].(*TomlTree)
		subtree = parent.Values
	}
}

// SaveToFile writes Toml document to local file system
func (t *TomlTree) SaveToFile() error {
	w, err := os.Create(t.FileName)
	if err != nil {
		return err
	}
	defer func() { w.Close() }()
	t.string(func(indent int, ss ...string) {
		if err != nil {
			return
		}
		for _, s := range ss {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			_, err = w.WriteString(strings.Repeat("\t", indent) + s + "\n")
		}
	}, 0, "")
	return err
}

// String reassembles the TomlTree into a valid TOML document string.
func (t *TomlTree) String() string {
	lines := ""
	t.string(func(indent int, ss ...string) {
		for _, s := range ss {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			lines += strings.Repeat("\t", indent) + s + "\n"
		}
	}, 0, "")
	return lines
}

func (t *TomlTree) string(write func(int, ...string), indent int, prefix string) {
	for _, key := range t.keyorder {
		i := t.Get(key)
		if i == nil {
			continue
		}
		tree, ok := i.(*TomlTree)
		if !ok {
			comment := t.GetComment(key)
			write(indent, comment.Multiline...)
			if comment.EndOfLine == "" {
				write(indent, key+" = "+toString(i))
			} else {
				write(indent, key+" = "+toString(i)+" "+comment.EndOfLine)
			}
			continue
		}

		comment := t.GetComment(key)
		write(indent, comment.Multiline...)
		if comment.EndOfLine == "" {
			write(indent, "["+prefix+key+"]")
		} else {
			write(indent, "["+prefix+key+"] "+comment.EndOfLine)
		}
		tree.string(write, indent+1, prefix+key+".")
	}
}

func toString(i interface{}) string {
	date, ok := i.(time.Time)
	if ok {
		return date.Format(time.RFC3339)
	}
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Slice {
		return fmt.Sprintf("%#v", i)
	}
	s := "["
	comma := ""
	for j := 0; j < v.Len(); j++ {
		s += comma + toString(v.Index(j).Interface())
		if comma == "" {
			comma = ", "
		}
	}
	s += "]"
	return s
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
	if err == nil {
		tree.FileName = path
	}
	return
}
