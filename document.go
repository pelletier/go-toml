package toml

import "fmt"

type Document struct {
}

type docParser struct {
	document Document
}

func (d *docParser) ArrayTableBegin() {
	fmt.Println("ARRAY-TABLE[[")
}

func (d *docParser) ArrayTableEnd() {
	fmt.Println("ARRAY-TABLE]]")
}

func (d *docParser) StandardTableBegin() {
	fmt.Println("STD-TABLE[")
}

func (d *docParser) StandardTableEnd() {
	fmt.Println("STD-TABLE]")
}

func (d *docParser) InlineTableSeparator() {
	fmt.Println(", InlineTable SEPARATOR")
}

func (d *docParser) InlineTableBegin() {
	fmt.Println("{ InlineTable BEGIN")
}

func (d *docParser) InlineTableEnd() {
	fmt.Println("} InlineTable END")
}

func (d *docParser) ArraySeparator() {
	fmt.Println(", ARRAY SEPARATOR")
}

func (d *docParser) ArrayBegin() {
	fmt.Println("[ ARRAY BEGIN")
}

func (d *docParser) ArrayEnd() {
	fmt.Println("] ARRAY END")
}

func (d *docParser) Equal(b []byte) {
	s := string(b)
	fmt.Printf("EQUAL: '%s'\n", s)
}

func (d *docParser) Boolean(b []byte) {
	s := string(b)
	fmt.Printf("Boolean: '%s'\n", s)
}

func (d *docParser) Dot(b []byte) {
	s := string(b)
	fmt.Printf("DOT: '%s'\n", s)
}

func (d *docParser) BasicString(b []byte) {
	s := string(b)
	fmt.Printf("BasicString: '%s'\n", s)
}

func (d *docParser) LiteralString(b []byte) {
	s := string(b)
	fmt.Printf("LiteralString: '%s'\n", s)
}

func (d *docParser) UnquotedKey(b []byte) {
	s := string(b)
	fmt.Printf("UnquotedKey: '%s'\n", s)
}

func (d *docParser) Comment(b []byte) {
	s := string(b)
	fmt.Printf("Comment: '%s'\n", s)
}

func (d *docParser) Whitespace(b []byte) {
	s := string(b)
	fmt.Printf("Whitespace: '%s'\n", s)
}

func Parse(b []byte) (Document, error) {
	p := docParser{}
	l := lexer{parser: &p, data: b}
	err := l.run()
	if err != nil {
		return Document{}, err
	}
	return p.document, nil
}
