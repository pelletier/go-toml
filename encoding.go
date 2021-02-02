package toml

type unmarshaler struct {
}

func (u unmarshaler) Whitespace(b []byte) {}
func (u unmarshaler) Comment(b []byte)    {}

func (u unmarshaler) UnquotedKey(b []byte) {
	panic("implement me")
}

func (u unmarshaler) LiteralString(b []byte) {
	panic("implement me")
}

func (u unmarshaler) BasicString(b []byte) {
	panic("implement me")
}

func (u unmarshaler) Dot(b []byte) {
	panic("implement me")
}

func (u unmarshaler) Boolean(b []byte) {
	panic("implement me")
}

func (u unmarshaler) Equal(b []byte) {
	panic("implement me")
}

func (u unmarshaler) ArrayBegin() {
	panic("implement me")
}

func (u unmarshaler) ArrayEnd() {
	panic("implement me")
}

func (u unmarshaler) ArraySeparator() {
	panic("implement me")
}

func (u unmarshaler) InlineTableBegin() {
	panic("implement me")
}

func (u unmarshaler) InlineTableEnd() {
	panic("implement me")
}

func (u unmarshaler) InlineTableSeparator() {
	panic("implement me")
}

func (u unmarshaler) StandardTableBegin() {
	panic("implement me")
}

func (u unmarshaler) StandardTableEnd() {
	panic("implement me")
}

func (u unmarshaler) ArrayTableBegin() {
	panic("implement me")
}

func (u unmarshaler) ArrayTableEnd() {
	panic("implement me")
}

func Unmarshal(data []byte, v interface{}) error {
	p := unmarshaler{}
	l := lexer{parser: &p, data: data}
	return l.run()
}

func Marshal(v interface{}) ([]byte, error) {
	// TODO
	return nil, nil
}
