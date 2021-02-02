package toml

type parser interface {
	Whitespace(b []byte)
	Comment(b []byte)
	UnquotedKey(b []byte)
	LiteralString(b []byte)
	BasicString(b []byte)
	Dot(b []byte)
	Boolean(b []byte)
	Equal(b []byte)
	ArrayBegin()
	ArrayEnd()
	ArraySeparator()
	InlineTableBegin()
	InlineTableEnd()
	InlineTableSeparator()
	StandardTableBegin()
	StandardTableEnd()
	ArrayTableBegin()
	ArrayTableEnd()
}
