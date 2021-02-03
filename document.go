package toml

import "fmt"

type tokenKind int

const (
	whitespace tokenKind = iota
	arrayTableBegin
	arrayTableEnd
	standardTableBegin
	standardTableEnd
	inlineTableSeparator
	inlineTableBegin
	inlineTableEnd
	arraySeparator
	arrayBegin
	arrayEnd
	equal
	boolean
	dot
	basicString
	literalString
	unquotedKey
	comment
)

type token struct {
	data []byte
	kind tokenKind
}

type Document struct {
	tokens []token
}

func (d *Document) appendToken(kind tokenKind, data []byte) {
	d.tokens = append(d.tokens, token{data: data, kind: kind})
}

type docParser struct {
	document Document
}

func (d *docParser) ArrayTableBegin() {
	fmt.Println("ARRAY-TABLE[[")
	d.document.appendToken(arrayTableBegin, nil)
}

func (d *docParser) ArrayTableEnd() {
	fmt.Println("ARRAY-TABLE]]")
	d.document.appendToken(arrayTableEnd, nil)
}

func (d *docParser) StandardTableBegin() {
	fmt.Println("STD-TABLE[")
	d.document.appendToken(standardTableBegin, nil)
}

func (d *docParser) StandardTableEnd() {
	fmt.Println("STD-TABLE]")
	d.document.appendToken(standardTableEnd, nil)
}

func (d *docParser) InlineTableSeparator() {
	fmt.Println(", InlineTable SEPARATOR")
	d.document.appendToken(inlineTableSeparator, nil)
}

func (d *docParser) InlineTableBegin() {
	fmt.Println("{ InlineTable BEGIN")
	d.document.appendToken(inlineTableBegin, nil)
}

func (d *docParser) InlineTableEnd() {
	fmt.Println("} InlineTable END")
	d.document.appendToken(inlineTableEnd, nil)
}

func (d *docParser) ArraySeparator() {
	fmt.Println(", ARRAY SEPARATOR")
	d.document.appendToken(arraySeparator, nil)
}

func (d *docParser) ArrayBegin() {
	fmt.Println("[ ARRAY BEGIN")
	d.document.appendToken(arrayBegin, nil)
}

func (d *docParser) ArrayEnd() {
	fmt.Println("] ARRAY END")
	d.document.appendToken(arrayEnd, nil)
}

func (d *docParser) Equal(b []byte) {
	s := string(b)
	fmt.Printf("EQUAL: '%s'\n", s)
	d.document.appendToken(equal, b)
}

func (d *docParser) Boolean(b []byte) {
	s := string(b)
	fmt.Printf("Boolean: '%s'\n", s)
	d.document.appendToken(boolean, b)
}

func (d *docParser) Dot(b []byte) {
	s := string(b)
	fmt.Printf("DOT: '%s'\n", s)
	d.document.appendToken(dot, b)
}

func (d *docParser) BasicString(b []byte) {
	s := string(b)
	fmt.Printf("BasicString: '%s'\n", s)
	d.document.appendToken(basicString, b)
}

func (d *docParser) LiteralString(b []byte) {
	s := string(b)
	fmt.Printf("LiteralString: '%s'\n", s)
	d.document.appendToken(literalString, b)
}

func (d *docParser) UnquotedKey(b []byte) {
	s := string(b)
	fmt.Printf("UnquotedKey: '%s'\n", s)
	d.document.appendToken(unquotedKey, b)
}

func (d *docParser) Comment(b []byte) {
	s := string(b)
	fmt.Printf("Comment: '%s'\n", s)
	d.document.appendToken(comment, b)
}

func (d *docParser) Whitespace(b []byte) {
	s := string(b)
	fmt.Printf("Whitespace: '%s'\n", s)
	d.document.appendToken(whitespace, b)
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
