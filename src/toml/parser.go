// TOML Parser.

package toml

import (
	"strings"
)

// createSubTree takes a tree and a key andcreate the necessary intermediate
// subtrees to create a subtree at that point. In-place.
//
// e.g. passing a.b.c will create (assuming tree is empty) tree[a], tree[a][b]
// and tree[a][b][c]
func createSubTree(tree *TomlTree, key string) {
	subtree := tree
	for _, intermediate_key := range strings.Split(key, ".") {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			(*subtree)[intermediate_key] = make(TomlTree)
		}
		subtree = (*subtree)[intermediate_key].(*TomlTree)
	}
}


func parse(chan token) *TomlTree {
	result := make(TomlTree)
	return &result
}
