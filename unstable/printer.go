package unstable

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2/internal/characters"
)

// Printer writes a TOML representation of AST nodes to an output stream.
//
// Each call to Format writes one top-level expression (Comment, Table,
// ArrayTable, or KeyValue) followed by a newline. Typical usage:
//
//	p := &unstable.Parser{KeepComments: true}
//	p.Reset(input)
//	pr := unstable.NewPrinter(w)
//	for p.NextExpression() {
//	    if err := pr.Format(p.Expression()); err != nil { ... }
//	}
type Printer struct {
	w   io.Writer
	buf []byte
}

// NewPrinter returns a new Printer that writes to w.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// Format writes the TOML representation of a single top-level expression node
// to the printer's writer. The node must be of kind Comment, Table,
// ArrayTable, or KeyValue.
func (p *Printer) Format(n *Node) error {
	p.buf = p.buf[:0]
	p.printNode(n)
	_, err := p.w.Write(p.buf)
	return err
}

func (p *Printer) printNode(n *Node) {
	switch n.Kind {
	case Comment:
		p.printStandaloneComment(n)
	case Table:
		p.printTable(n)
	case ArrayTable:
		p.printArrayTable(n)
	case KeyValue:
		p.printKeyValue(n)
	case String:
		p.printString(n.Data)
	case Bool, Integer, Float, LocalDate, LocalTime, LocalDateTime, DateTime:
		p.buf = append(p.buf, n.Data...)
	case Array:
		p.printArray(n)
	case InlineTable:
		p.printInlineTable(n)
	default:
		panic(fmt.Errorf("unstable.Printer: unsupported node kind %s", n.Kind))
	}
}

func (p *Printer) printStandaloneComment(n *Node) {
	p.buf = append(p.buf, n.Data...)
	p.buf = append(p.buf, '\n')
}

func (p *Printer) printTable(n *Node) {
	p.buf = append(p.buf, '[')
	p.printDottedKey(n.Key())
	p.buf = append(p.buf, ']')
	p.printTrailingComment(n)
	p.buf = append(p.buf, '\n')
}

func (p *Printer) printArrayTable(n *Node) {
	p.buf = append(p.buf, '[', '[')
	p.printDottedKey(n.Key())
	p.buf = append(p.buf, ']', ']')
	p.printTrailingComment(n)
	p.buf = append(p.buf, '\n')
}

func (p *Printer) printKeyValue(n *Node) {
	p.printDottedKey(n.Key())
	p.buf = append(p.buf, " = "...)
	p.printNode(n.Value())
	p.printTrailingComment(n)
	p.buf = append(p.buf, '\n')
}

func (p *Printer) printTrailingComment(n *Node) {
	c := n.Comment()
	if c != nil {
		p.buf = append(p.buf, ' ')
		p.buf = append(p.buf, c.Data...)
	}
}

func (p *Printer) printDottedKey(it Iterator) {
	first := true
	for it.Next() {
		if !first {
			p.buf = append(p.buf, '.')
		}
		p.printKey(it.Node().Data)
		first = false
	}
}

func (p *Printer) printKey(data []byte) {
	k := string(data)

	if len(k) == 0 {
		p.buf = append(p.buf, "''"...)
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
		p.printQuotedString(k)
	case needsQuotation:
		p.printLiteralString(k)
	default:
		p.buf = append(p.buf, k...)
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

func (p *Printer) printString(data []byte) {
	v := string(data)
	if needsQuoting(v) {
		p.printQuotedString(v)
	} else {
		p.printLiteralString(v)
	}
}

func (p *Printer) printLiteralString(v string) {
	p.buf = append(p.buf, '\'')
	p.buf = append(p.buf, v...)
	p.buf = append(p.buf, '\'')
}

const hextable = "0123456789ABCDEF"

func (p *Printer) printQuotedString(v string) {
	p.buf = append(p.buf, '"')

	for i := 0; i < len(v); i++ {
		r := v[i]
		switch r {
		case '\\':
			p.buf = append(p.buf, `\\`...)
		case '"':
			p.buf = append(p.buf, `\"`...)
		case '\b':
			p.buf = append(p.buf, `\b`...)
		case '\f':
			p.buf = append(p.buf, `\f`...)
		case '\n':
			p.buf = append(p.buf, `\n`...)
		case '\r':
			p.buf = append(p.buf, `\r`...)
		case '\t':
			p.buf = append(p.buf, `\t`...)
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
				p.buf = append(p.buf, `\u00`...)
				p.buf = append(p.buf, hextable[r>>4])
				p.buf = append(p.buf, hextable[r&0x0f])
			default:
				p.buf = append(p.buf, r)
			}
		}
	}

	p.buf = append(p.buf, '"')
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

func (p *Printer) printArray(n *Node) {
	if arrayHasComments(n) {
		p.printMultilineArray(n)
		return
	}

	p.buf = append(p.buf, '[')
	it := n.Children()
	first := true
	for it.Next() {
		if !first {
			p.buf = append(p.buf, ", "...)
		}
		p.printNode(it.Node())
		first = false
	}
	p.buf = append(p.buf, ']')
}

func (p *Printer) printMultilineArray(n *Node) {
	p.buf = append(p.buf, '[')

	if c := n.Comment(); c != nil {
		p.buf = append(p.buf, ' ')
		p.buf = append(p.buf, c.Data...)
	}
	p.buf = append(p.buf, '\n')

	it := n.Children()
	for it.Next() {
		child := it.Node()
		if child.Kind == Comment {
			p.printCommentGroup(child)
		} else {
			p.buf = append(p.buf, "  "...)
			p.printNode(child)
			p.buf = append(p.buf, ',')
			p.printTrailingComment(child)
			p.buf = append(p.buf, '\n')
		}
	}

	p.buf = append(p.buf, ']')
}

// printCommentGroup writes a comment node and all its chained children
// (representing consecutive comment lines from parseOptionalWhitespaceCommentNewline).
func (p *Printer) printCommentGroup(n *Node) {
	p.buf = append(p.buf, "  "...)
	p.buf = append(p.buf, n.Data...)
	p.buf = append(p.buf, '\n')

	child := n.Child()
	for child != nil {
		p.buf = append(p.buf, "  "...)
		p.buf = append(p.buf, child.Data...)
		p.buf = append(p.buf, '\n')
		child = child.Next()
	}
}

func (p *Printer) printInlineTable(n *Node) {
	if inlineTableHasComments(n) {
		p.printMultilineInlineTable(n)
		return
	}

	p.buf = append(p.buf, '{')
	it := n.Children()
	first := true
	for it.Next() {
		if !first {
			p.buf = append(p.buf, ", "...)
		}
		child := it.Node()
		p.printDottedKey(child.Key())
		p.buf = append(p.buf, " = "...)
		p.printNode(child.Value())
		first = false
	}
	p.buf = append(p.buf, '}')
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

func (p *Printer) printMultilineInlineTable(n *Node) {
	p.buf = append(p.buf, '{')

	if c := n.Comment(); c != nil {
		p.buf = append(p.buf, ' ')
		p.buf = append(p.buf, c.Data...)
	}
	p.buf = append(p.buf, '\n')

	it := n.Children()
	for it.Next() {
		child := it.Node()
		if child.Kind == Comment {
			p.printCommentGroup(child)
		} else {
			p.buf = append(p.buf, "  "...)
			p.printDottedKey(child.Key())
			p.buf = append(p.buf, " = "...)
			p.printNode(child.Value())
			p.buf = append(p.buf, ',')
			p.printTrailingComment(child)
			p.buf = append(p.buf, '\n')
		}
	}

	p.buf = append(p.buf, '}')
}
