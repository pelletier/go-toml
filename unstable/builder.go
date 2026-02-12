package unstable

// root contains a full AST.
//
// It is immutable once constructed with Builder.
type root struct {
	nodes []Node
}

func (r *root) at(idx Reference) *Node {
	return &r.nodes[idx]
}

type Reference int

const InvalidReference Reference = -1

func (r Reference) Valid() bool {
	return r != InvalidReference
}

type Builder struct {
	tree    root
	lastIdx int
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) NodeAt(ref Reference) *Node {
	n := b.tree.at(ref)
	n.nodes = &b.tree.nodes
	return n
}

func (b *Builder) Reset() {
	b.tree.nodes = b.tree.nodes[:0]
	b.lastIdx = 0
}

func (b *Builder) Push(n Node) Reference {
	b.lastIdx = len(b.tree.nodes)
	n.next = -1
	n.child = -1
	n.comment = -1
	b.tree.nodes = append(b.tree.nodes, n)
	return Reference(b.lastIdx)
}

func (b *Builder) PushAndChain(n Node) Reference {
	newIdx := len(b.tree.nodes)
	n.next = -1
	n.child = -1
	n.comment = -1
	b.tree.nodes = append(b.tree.nodes, n)
	if b.lastIdx >= 0 {
		b.tree.nodes[b.lastIdx].next = int32(newIdx) //nolint:gosec // TOML ASTs are small
	}
	b.lastIdx = newIdx
	return Reference(b.lastIdx)
}

func (b *Builder) AttachChild(parent Reference, child Reference) {
	b.tree.nodes[parent].child = int32(child) //nolint:gosec // TOML ASTs are small
}

func (b *Builder) Chain(from Reference, to Reference) {
	b.tree.nodes[from].next = int32(to) //nolint:gosec // TOML ASTs are small
}

func (b *Builder) AttachComment(node Reference, comment Reference) {
	b.tree.nodes[node].comment = int32(comment) //nolint:gosec // TOML ASTs are small
}
