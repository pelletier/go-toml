package ast

type Reference struct {
	idx int
	set bool
}

func (r Reference) Valid() bool {
	return r.set
}

type Builder struct {
	tree    Root
	lastIdx int
}

func (b *Builder) Tree() *Root {
	return &b.tree
}

func (b *Builder) NodeAt(ref Reference) Node {
	return b.tree.at(ref.idx)
}

func (b *Builder) Reset() {
	b.tree.nodes = b.tree.nodes[:0]
	b.lastIdx = 0
}

func (b *Builder) Push(n Node) Reference {
	n.root = &b.tree
	b.lastIdx = len(b.tree.nodes)
	b.tree.nodes = append(b.tree.nodes, n)
	return Reference{
		idx: b.lastIdx,
		set: true,
	}
}

func (b *Builder) PushAndChain(n Node) Reference {
	n.root = &b.tree
	newIdx := len(b.tree.nodes)
	b.tree.nodes = append(b.tree.nodes, n)
	if b.lastIdx >= 0 {
		b.tree.nodes[b.lastIdx].next = newIdx
	}
	b.lastIdx = newIdx
	return Reference{
		idx: b.lastIdx,
		set: true,
	}
}

func (b *Builder) AttachChild(parent Reference, child Reference) {
	b.tree.nodes[parent.idx].child = child.idx
}

func (b *Builder) Chain(from Reference, to Reference) {
	b.tree.nodes[from.idx].next = to.idx
}
