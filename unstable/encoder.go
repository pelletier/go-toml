package unstable

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2/internal/characters"
)

// Encoder writes a TOML representation of AST nodes to an output stream.
//
// Each call to Encode writes one top-level expression (Comment, Table,
// ArrayTable, or KeyValue) followed by a newline. Typical usage:
//
//	p := &unstable.Parser{KeepComments: true}
//	p.Reset(input)
//	enc := unstable.NewEncoder(w)
//	for p.NextExpression() {
//	    if err := enc.Encode(p.Expression()); err != nil { ... }
//	}
type Encoder struct {
	w   io.Writer
	buf []byte
}

// NewEncoder returns a new Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes the TOML representation of a single top-level expression node
// to the encoder's writer. The node must be of kind Comment, Table,
// ArrayTable, or KeyValue.
func (enc *Encoder) Encode(n *Node) error {
	enc.buf = enc.buf[:0]
	enc.encodeNode(n)
	_, err := enc.w.Write(enc.buf)
	return err
}

func (enc *Encoder) encodeNode(n *Node) {
	switch n.Kind {
	case Comment:
		enc.encodeStandaloneComment(n)
	case Table:
		enc.encodeTable(n)
	case ArrayTable:
		enc.encodeArrayTable(n)
	case KeyValue:
		enc.encodeKeyValue(n)
	case String:
		enc.encodeString(n.Data)
	case Bool, Integer, Float, LocalDate, LocalTime, LocalDateTime, DateTime:
		enc.buf = append(enc.buf, n.Data...)
	case Array:
		enc.encodeArray(n)
	case InlineTable:
		enc.encodeInlineTable(n)
	default:
		panic(fmt.Errorf("unstable.Encoder: unsupported node kind %s", n.Kind))
	}
}

func (enc *Encoder) encodeStandaloneComment(n *Node) {
	enc.buf = append(enc.buf, n.Data...)
	enc.buf = append(enc.buf, '\n')
}

func (enc *Encoder) encodeTable(n *Node) {
	enc.buf = append(enc.buf, '[')
	enc.encodeDottedKey(n.Key())
	enc.buf = append(enc.buf, ']')
	enc.encodeTrailingComment(n)
	enc.buf = append(enc.buf, '\n')
}

func (enc *Encoder) encodeArrayTable(n *Node) {
	enc.buf = append(enc.buf, '[', '[')
	enc.encodeDottedKey(n.Key())
	enc.buf = append(enc.buf, ']', ']')
	enc.encodeTrailingComment(n)
	enc.buf = append(enc.buf, '\n')
}

func (enc *Encoder) encodeKeyValue(n *Node) {
	enc.encodeDottedKey(n.Key())
	enc.buf = append(enc.buf, " = "...)
	enc.encodeNode(n.Value())
	enc.encodeTrailingComment(n)
	enc.buf = append(enc.buf, '\n')
}

func (enc *Encoder) encodeTrailingComment(n *Node) {
	c := n.Comment()
	if c != nil {
		enc.buf = append(enc.buf, ' ')
		enc.buf = append(enc.buf, c.Data...)
	}
}

func (enc *Encoder) encodeDottedKey(it Iterator) {
	first := true
	for it.Next() {
		if !first {
			enc.buf = append(enc.buf, '.')
		}
		enc.encodeKey(it.Node().Data)
		first = false
	}
}

func (enc *Encoder) encodeKey(data []byte) {
	k := string(data)

	if len(k) == 0 {
		enc.buf = append(enc.buf, "''"...)
		return
	}

	needsQuotation := false
	cannotUseLiteral := false

	for _, c := range k {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		if c == '\'' {
			cannotUseLiteral = true
		}
		needsQuotation = true
	}

	if needsQuotation && needsQuoting(k) {
		cannotUseLiteral = true
	}

	switch {
	case cannotUseLiteral:
		enc.encodeQuotedString(k)
	case needsQuotation:
		enc.encodeLiteralString(k)
	default:
		enc.buf = append(enc.buf, k...)
	}
}

func needsQuoting(v string) bool {
	for _, b := range []byte(v) {
		if b == '\'' || b == '\r' || b == '\n' || characters.InvalidASCII(b) {
			return true
		}
	}
	return false
}

func (enc *Encoder) encodeString(data []byte) {
	v := string(data)
	if needsQuoting(v) {
		enc.encodeQuotedString(v)
	} else {
		enc.encodeLiteralString(v)
	}
}

func (enc *Encoder) encodeLiteralString(v string) {
	enc.buf = append(enc.buf, '\'')
	enc.buf = append(enc.buf, v...)
	enc.buf = append(enc.buf, '\'')
}

const hextable = "0123456789ABCDEF"

func (enc *Encoder) encodeQuotedString(v string) {
	enc.buf = append(enc.buf, '"')

	for i := 0; i < len(v); i++ {
		r := v[i]
		switch r {
		case '\\':
			enc.buf = append(enc.buf, `\\`...)
		case '"':
			enc.buf = append(enc.buf, `\"`...)
		case '\b':
			enc.buf = append(enc.buf, `\b`...)
		case '\f':
			enc.buf = append(enc.buf, `\f`...)
		case '\n':
			enc.buf = append(enc.buf, `\n`...)
		case '\r':
			enc.buf = append(enc.buf, `\r`...)
		case '\t':
			enc.buf = append(enc.buf, `\t`...)
		default:
			const (
				nul = 0x0
				bs  = 0x8
				lf  = 0xa
				us  = 0x1f
				del = 0x7f
			)
			switch {
			case r >= nul && r <= bs, r >= lf && r <= us, r == del:
				enc.buf = append(enc.buf, `\u00`...)
				enc.buf = append(enc.buf, hextable[r>>4])
				enc.buf = append(enc.buf, hextable[r&0x0f])
			default:
				enc.buf = append(enc.buf, r)
			}
		}
	}

	enc.buf = append(enc.buf, '"')
}

// arrayHasComments returns true if the array contains any comment nodes
// among its children or has a trailing comment on the opening bracket.
func arrayHasComments(n *Node) bool {
	if n.Comment() != nil {
		return true
	}
	it := n.Children()
	for it.Next() {
		child := it.Node()
		if child.Kind == Comment {
			return true
		}
		if child.Comment() != nil {
			return true
		}
	}
	return false
}

func (enc *Encoder) encodeArray(n *Node) {
	if arrayHasComments(n) {
		enc.encodeMultilineArray(n)
		return
	}

	enc.buf = append(enc.buf, '[')
	it := n.Children()
	first := true
	for it.Next() {
		if !first {
			enc.buf = append(enc.buf, ", "...)
		}
		enc.encodeNode(it.Node())
		first = false
	}
	enc.buf = append(enc.buf, ']')
}

func (enc *Encoder) encodeMultilineArray(n *Node) {
	enc.buf = append(enc.buf, '[')

	if c := n.Comment(); c != nil {
		enc.buf = append(enc.buf, ' ')
		enc.buf = append(enc.buf, c.Data...)
	}
	enc.buf = append(enc.buf, '\n')

	it := n.Children()
	for it.Next() {
		child := it.Node()
		if child.Kind == Comment {
			enc.encodeCommentGroup(child)
		} else {
			enc.buf = append(enc.buf, "  "...)
			enc.encodeNode(child)
			enc.buf = append(enc.buf, ',')
			enc.encodeTrailingComment(child)
			enc.buf = append(enc.buf, '\n')
		}
	}

	enc.buf = append(enc.buf, ']')
}

// encodeCommentGroup writes a comment node and all its chained children
// (representing consecutive comment lines from parseOptionalWhitespaceCommentNewline).
func (enc *Encoder) encodeCommentGroup(n *Node) {
	enc.buf = append(enc.buf, "  "...)
	enc.buf = append(enc.buf, n.Data...)
	enc.buf = append(enc.buf, '\n')

	child := n.Child()
	for child != nil {
		enc.buf = append(enc.buf, "  "...)
		enc.buf = append(enc.buf, child.Data...)
		enc.buf = append(enc.buf, '\n')
		child = child.Next()
	}
}

func (enc *Encoder) encodeInlineTable(n *Node) {
	if inlineTableHasComments(n) {
		enc.encodeMultilineInlineTable(n)
		return
	}

	enc.buf = append(enc.buf, '{')
	it := n.Children()
	first := true
	for it.Next() {
		if !first {
			enc.buf = append(enc.buf, ", "...)
		}
		child := it.Node()
		enc.encodeDottedKey(child.Key())
		enc.buf = append(enc.buf, " = "...)
		enc.encodeNode(child.Value())
		first = false
	}
	enc.buf = append(enc.buf, '}')
}

func inlineTableHasComments(n *Node) bool {
	if n.Comment() != nil {
		return true
	}
	it := n.Children()
	for it.Next() {
		child := it.Node()
		if child.Kind == Comment {
			return true
		}
		if child.Comment() != nil {
			return true
		}
	}
	return false
}

func (enc *Encoder) encodeMultilineInlineTable(n *Node) {
	enc.buf = append(enc.buf, '{')

	if c := n.Comment(); c != nil {
		enc.buf = append(enc.buf, ' ')
		enc.buf = append(enc.buf, c.Data...)
	}
	enc.buf = append(enc.buf, '\n')

	it := n.Children()
	for it.Next() {
		child := it.Node()
		if child.Kind == Comment {
			enc.encodeCommentGroup(child)
		} else {
			enc.buf = append(enc.buf, "  "...)
			enc.encodeDottedKey(child.Key())
			enc.buf = append(enc.buf, " = "...)
			enc.encodeNode(child.Value())
			enc.buf = append(enc.buf, ',')
			enc.encodeTrailingComment(child)
			enc.buf = append(enc.buf, '\n')
		}
	}

	enc.buf = append(enc.buf, '}')
}
