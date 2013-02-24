// TOML Parser.

package toml

import (
	"strings"
)

// Given a tree and a key, create the necessary intermediate subtrees to create
// a subtree at that point. In-place.
//
// e.g. passing a.b.c will create (assuming tree is empty) tree[a], tree[a][b]
// and tree[a][b][c]
func createSubTree(tree *tomlTree, key string) {
	subtree := tree
	for _, intermediate_key := range strings.Split(key, ".") {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			(*subtree)[intermediate_key] = make(tomlTree)
		}
		subtree = (*subtree)[intermediate_key].(*tomlTree)
	}
}


func parse(chan token) *tomlTree {
	result := make(tomlTree)
	return &result
}
