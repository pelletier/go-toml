package tracker

import "github.com/pelletier/go-toml/v2/unstable"

// KeyTracker is a tracker that keeps track of the current Key as the AST is
// walked. It holds the raw bytes of each part (valid for the lifetime of the
// document being parsed) and only materializes strings when Key is called,
// so tracking is allocation-free for documents that produce no error.
type KeyTracker struct {
	k [][]byte
}

// UpdateTable sets the state of the tracker with the AST table node.
func (t *KeyTracker) UpdateTable(node *unstable.Node) {
	t.Reset()
	t.Push(node)
}

// UpdateArrayTable sets the state of the tracker with the AST array table
// node.
func (t *KeyTracker) UpdateArrayTable(node *unstable.Node) {
	t.Reset()
	t.Push(node)
}

// Push the given key on the stack.
func (t *KeyTracker) Push(node *unstable.Node) {
	it := node.Key()
	for it.Next() {
		t.k = append(t.k, it.Node().Data)
	}
}

// Pop key from stack.
func (t *KeyTracker) Pop(node *unstable.Node) {
	it := node.Key()
	for it.Next() {
		t.k = t.k[:len(t.k)-1]
	}
}

// PushParts pushes already-decoded key parts on the stack, for callers that
// track keys without an AST.
func (t *KeyTracker) PushParts(parts [][]byte) {
	t.k = append(t.k, parts...)
}

// PopN pops n parts from the stack.
func (t *KeyTracker) PopN(n int) {
	t.k = t.k[:len(t.k)-n]
}

// Key returns the current key.
func (t *KeyTracker) Key() []string {
	k := make([]string, len(t.k))
	for i, b := range t.k {
		k[i] = string(b)
	}
	return k
}

// Reset empties the tracker, keeping the allocated stack for reuse.
func (t *KeyTracker) Reset() {
	t.k = t.k[:0]
}
