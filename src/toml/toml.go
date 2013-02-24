// TOML interface.

package toml

import (
	"strings"
)

// Definition of a tomlTree
type tomlTree map[string]interface{}

// Retrieve an element from the tree.
//
// If the path described by the key does not exist, nil is returned.
func (t *tomlTree) Get(key string) interface{} {
	subtree := t
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			return nil
		}
		subtree = (*subtree)[intermediate_key].(*tomlTree)
	}
	return (*subtree)[keys[len(keys) - 1]]
}

// Set an element in the tree.
//
// Creates all necessary intermediates trees, if needed.
func (t *tomlTree) Set(key string, value interface{}) {
	subtree := t
	keys := strings.Split(key, ".")
	for _, intermediate_key := range keys[:len(keys)-1] {
		_, exists := (*subtree)[intermediate_key]
		if !exists {
			(*subtree)[intermediate_key] = make(tomlTree)
		}
		subtree = (*subtree)[intermediate_key].(*tomlTree)
	}
	(*subtree)[keys[len(key) - 1]] = value
}


func Load() tomlTree {
	result := make(tomlTree)
	return result
}
