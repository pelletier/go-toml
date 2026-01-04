package unstable

// root contains a full AST.
//
// It is immutable once constructed with Builder.
type root struct {
	first *Node
}

// Iterator over the top level nodes.
func (r *root) Iterator() Iterator {
	return Iterator{node: r.first}
}

type reference struct {
	*Node
}

var invalidReference = reference{}

func (r reference) Valid() bool {
	return r.Node != nil
}

type builder struct {
	// chunks of nodes. Pointers to nodes are stable because we only append
	// to the last chunk, and chunks are allocated with fixed capacity.
	chunks [][]Node
	// current chunk index
	chunkIdx int

	// root node of the tree
	root root

	// last pushed node (for chaining)
	last *Node
}

const initialChunkSize = 16
const maxChunkSize = 2048

func (b *builder) Tree() *root {
	return &b.root
}

func (b *builder) NodeAt(ref reference) *Node {
	return ref.Node
}

func (b *builder) Reset() {
	b.chunkIdx = 0
	for i := range b.chunks {
		b.chunks[i] = b.chunks[i][:0]
	}
	b.root.first = nil
	b.last = nil
}

func (b *builder) ensureCapacity() {
	if b.chunkIdx >= len(b.chunks) {
		size := initialChunkSize
		if len(b.chunks) > 0 {
			lastCap := cap(b.chunks[len(b.chunks)-1])
			size = lastCap * 2
			if size > maxChunkSize {
				size = maxChunkSize
			}
		}
		b.chunks = append(b.chunks, make([]Node, 0, size))
	}
	if len(b.chunks[b.chunkIdx]) == cap(b.chunks[b.chunkIdx]) {
		b.chunkIdx++
		if b.chunkIdx >= len(b.chunks) {
			size := initialChunkSize
			if len(b.chunks) > 0 {
				lastCap := cap(b.chunks[len(b.chunks)-1])
				size = lastCap * 2
				if size > maxChunkSize {
					size = maxChunkSize
				}
			}
			b.chunks = append(b.chunks, make([]Node, 0, size))
		}
	}
}

func (b *builder) push(n Node) *Node {
	b.ensureCapacity()
	chunk := &b.chunks[b.chunkIdx]
	*chunk = append(*chunk, n)
	return &(*chunk)[len(*chunk)-1]
}

func (b *builder) Push(n Node) reference {
	ptr := b.push(n)
	if b.root.first == nil {
		b.root.first = ptr
	}
	b.last = ptr
	return reference{ptr}
}

func (b *builder) PushAndChain(n Node) reference {
	ptr := b.push(n)
	if b.root.first == nil {
		b.root.first = ptr
	}
	if b.last != nil {
		b.last.next = ptr
	}
	b.last = ptr
	return reference{ptr}
}

func (b *builder) AttachChild(parent reference, child reference) {
	parent.child = child.Node
}

func (b *builder) Chain(from reference, to reference) {
	from.next = to.Node
}
