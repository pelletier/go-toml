package ast

type Builder struct {
	nodes   []Node
	lastIdx int
}

type Reference struct {
	idx int
	set bool
}

func (r Reference) Valid() bool {
	return r.set
}

func (b *Builder) Finish() *Root {
	r := &Root{
		nodes: b.nodes,
	}
	b.nodes = nil

	for i := range r.nodes {
		r.nodes[i].root = r
	}

	return r
}

func (b *Builder) Push(n Node) Reference {
	b.lastIdx = len(b.nodes)
	b.nodes = append(b.nodes, n)
	return Reference{
		idx: b.lastIdx,
		set: true,
	}
}

func (b *Builder) PushAndChain(n Node) Reference {
	newIdx := len(b.nodes)
	b.nodes = append(b.nodes, n)
	if b.lastIdx >= 0 {
		b.nodes[b.lastIdx].next = newIdx
	}
	b.lastIdx = newIdx
	return Reference{
		idx: b.lastIdx,
		set: true,
	}
}

func (b *Builder) AttachChild(parent Reference, child Reference) {
	b.nodes[parent.idx].child = child.idx
}

func (b *Builder) Chain(from Reference, to Reference) {
	b.nodes[from.idx].next = to.idx
}
