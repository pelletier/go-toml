package edit

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// splice is a pending replacement of a byte range of the document. An empty
// range is an insertion; an empty repl is a deletion.
type splice struct {
	span
	repl []byte
}

// Set sets the value at the given key path, creating the tables leading to
// it as needed.
//
// If the path designates an existing key-value, only the bytes of its value
// are replaced: comments and layout around it are preserved. Otherwise a new
// `key = value` line is added at the end of the section of the closest
// existing parent table, under new [table] headers or dotted keys as
// appropriate. The value is rendered like toml.Marshal would, in inline
// (single-line) form.
//
// Setting a path that designates or crosses an existing table, array of
// tables, or non-table value is an error: Set never silently discards parts
// of the document. Delete first to replace a whole table.
func (d *Document) Set(key []string, value interface{}) error {
	if len(key) == 0 {
		return errors.New("toml/edit: key must have at least one element")
	}

	t := d.root
	i := 0
	for ; i < len(key); i++ {
		it := t.items[key[i]]
		if it == nil {
			break
		}
		last := i == len(key)-1
		switch {
		case it.leaf != nil:
			if !last {
				return fmt.Errorf("toml/edit: cannot set %q: %q is not a table", pathString(key), pathString(key[:i+1]))
			}
			raw, err := d.renderValue(value)
			if err != nil {
				return err
			}
			return d.apply(splice{it.leaf.value, raw})
		case it.arr != nil:
			return fmt.Errorf("toml/edit: cannot set %q: %q is an array of tables", pathString(key), pathString(key[:i+1]))
		default:
			if last {
				return fmt.Errorf("toml/edit: cannot set %q: it is a table; delete it first or set its keys individually", pathString(key))
			}
			t = it.tbl
		}
	}

	// key[i:] does not exist and t is the deepest existing table on the path.
	suffix := key[i:]
	if len(suffix) == 0 {
		return fmt.Errorf("toml/edit: internal error: no keys left to create for %q", pathString(key))
	}

	if t.dotted {
		// t was defined by a dotted key in the section of t.host: extend it
		// the same way, relative to the host table.
		rel := make([]string, 0, len(t.path)-len(t.host.path)+len(suffix))
		rel = append(rel, t.path[len(t.host.path):]...)
		rel = append(rel, suffix...)
		line, err := d.renderDottedKV(rel, value)
		if err != nil {
			return err
		}
		return d.apply(d.insertion(t.host.insertAt, line, false))
	}

	if (t.explicit || t.root) && len(suffix) == 1 {
		line, err := d.renderKV(suffix[0], value)
		if err != nil {
			return err
		}
		return d.apply(d.insertion(t.insertAt, line, false))
	}

	// The remaining tables need a section of their own, placed right after
	// everything that belongs to t. A single header with the full dotted
	// path creates all of them. t may be an implicit table (only defined
	// through the headers of its sub-tables): declaring its header afterwards
	// is valid TOML.
	headerPath := make([]string, 0, len(t.path)+len(suffix)-1)
	headerPath = append(headerPath, t.path...)
	headerPath = append(headerPath, suffix[:len(suffix)-1]...)
	headerKey, err := d.renderKeyPath(headerPath)
	if err != nil {
		return err
	}
	kv, err := d.renderKV(suffix[len(suffix)-1], value)
	if err != nil {
		return err
	}
	text := append([]byte("["+headerKey+"]"), d.eol()...)
	text = append(text, kv...)
	return d.apply(d.insertion(subtreeEnd(t), text, true))
}

// Delete removes the key-value, table, or array of tables at the given key
// path, along with the comments attached to it. It returns true if something
// was deleted. Deleting a table removes all its content, including sections
// defined elsewhere in the document.
func (d *Document) Delete(key []string) bool {
	if len(key) == 0 {
		return false
	}
	t := d.root
	for _, name := range key[:len(key)-1] {
		it := t.items[name]
		if it == nil || it.tbl == nil {
			return false
		}
		t = it.tbl
	}
	it := t.items[key[len(key)-1]]
	if it == nil {
		return false
	}

	spans := mergeSpans(collectSpans(it, nil))
	splices := make([]splice, len(spans))
	for i, s := range spans {
		splices[i] = splice{span: d.absorbTrailingBlanks(s)}
	}
	return d.apply(splices...) == nil
}

// collectSpans accumulates the byte ranges expressing an item: key-value
// lines and whole sections. Ranges may overlap (a section contains the lines
// of its key-values): mergeSpans resolves that.
func collectSpans(it *item, spans []span) []span {
	switch {
	case it.leaf != nil:
		spans = append(spans, it.leaf.line)
	case it.tbl != nil:
		spans = collectTableSpans(it.tbl, spans)
	default:
		for _, elem := range it.arr {
			spans = collectTableSpans(elem, spans)
		}
	}
	return spans
}

func collectTableSpans(t *table, spans []span) []span {
	if t.explicit {
		spans = append(spans, t.section)
	}
	for _, child := range t.items {
		spans = collectSpans(child, spans)
	}
	return spans
}

func mergeSpans(spans []span) []span {
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	out := spans[:0]
	for _, s := range spans {
		if n := len(out); n > 0 && s.start <= out[n-1].end {
			if s.end > out[n-1].end {
				out[n-1].end = s.end
			}
			continue
		}
		out = append(out, s)
	}
	return out
}

// apply replaces the given byte ranges, then validates and re-indexes the
// document. If the result is not a valid TOML document, no change is made.
func (d *Document) apply(splices ...splice) error {
	sort.Slice(splices, func(i, j int) bool { return splices[i].start < splices[j].start })
	var buf bytes.Buffer
	prev := 0
	for _, s := range splices {
		if s.start < prev {
			return errors.New("toml/edit: internal error: overlapping edits")
		}
		buf.Write(d.data[prev:s.start])
		buf.Write(s.repl)
		prev = s.end
	}
	buf.Write(d.data[prev:])
	data := buf.Bytes()

	var v interface{}
	if err := toml.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("toml/edit: edit would produce an invalid TOML document: %w", err)
	}

	oldData, oldRoot := d.data, d.root
	d.data = data
	if err := d.reindex(); err != nil {
		d.data, d.root = oldData, oldRoot
		return fmt.Errorf("toml/edit: edit would produce an invalid TOML document: %w", err)
	}
	return nil
}

// insertion builds the splice inserting text (one or more full lines, ending
// with a newline) at pos, which is always a line boundary or the end of the
// document. Sections get a blank line separating them from the surrounding
// expressions.
func (d *Document) insertion(pos int, text []byte, section bool) splice {
	eol := d.eol()
	var b []byte
	freshLine := pos == 0 || d.data[pos-1] == '\n'
	if !freshLine {
		// Only possible at the end of a document with no final newline.
		b = append(b, eol...)
	}
	if section && pos > 0 && (!freshLine || !d.blankLineBefore(pos)) {
		b = append(b, eol...)
	}
	b = append(b, text...)
	if section && pos < len(d.data) {
		b = append(b, eol...)
	}
	return splice{span{pos, pos}, b}
}

// absorbTrailingBlanks extends a deletion reaching the end of the document
// over the blank lines that separated it from the content above, so that the
// document does not end with stray blank lines.
func (d *Document) absorbTrailingBlanks(s span) span {
	if s.end != len(d.data) {
		return s
	}
	for s.start > 0 && d.data[s.start-1] == '\n' && d.blankLineBefore(s.start) {
		i := s.start - 1
		if i > 0 && d.data[i-1] == '\r' {
			i--
		}
		s.start = i
	}
	return s
}

// blankLineBefore reports whether the line ending at pos (excluded) is
// empty. Only meaningful when pos is a line boundary.
func (d *Document) blankLineBefore(pos int) bool {
	i := pos - 1 // d.data[i] == '\n'
	if i > 0 && d.data[i-1] == '\r' {
		i--
	}
	return i == 0 || d.data[i-1] == '\n'
}

// eol returns the line ending to use for inserted lines: CRLF if the
// document already uses it, LF otherwise.
func (d *Document) eol() []byte {
	if bytes.Contains(d.data, []byte("\r\n")) {
		return []byte("\r\n")
	}
	return []byte("\n")
}

// renderKVLine renders a complete `key = value` line (terminated by '\n')
// with the toml encoder, so that key quoting and value formatting follow
// Marshal conventions exactly. Values render in inline form; a value that
// cannot render to a single line is an error.
func renderKVLine(name string, value interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetTablesInline(true)
	if err := enc.Encode(map[string]interface{}{name: value}); err != nil {
		return nil, fmt.Errorf("toml/edit: cannot render value: %w", err)
	}
	line := buf.Bytes()
	// A value the encoder omits instead of erroring on (e.g. a nil map
	// value) renders to nothing.
	if len(line) == 0 || line[len(line)-1] != '\n' || bytes.IndexByte(line, '\n') != len(line)-1 {
		return nil, fmt.Errorf("toml/edit: cannot render value of type %T as a single key-value line", value)
	}
	return line, nil
}

// renderKV renders a `key = value` line ending with the document's line
// ending.
func (d *Document) renderKV(name string, value interface{}) ([]byte, error) {
	line, err := renderKVLine(name, value)
	if err != nil {
		return nil, err
	}
	return d.withEOL(line), nil
}

// renderValue renders the bare TOML representation of a value.
func (d *Document) renderValue(value interface{}) ([]byte, error) {
	line, err := renderKVLine("v", value)
	if err != nil {
		return nil, err
	}
	prefix := []byte("v = ")
	if !bytes.HasPrefix(line, prefix) {
		return nil, errors.New("toml/edit: internal error: unexpected encoder output for value")
	}
	return line[len(prefix) : len(line)-1], nil
}

// renderDottedKV renders a `dotted.key = value` line ending with the
// document's line ending.
func (d *Document) renderDottedKV(parts []string, value interface{}) ([]byte, error) {
	kp, err := d.renderKeyPath(parts)
	if err != nil {
		return nil, err
	}
	v, err := d.renderValue(value)
	if err != nil {
		return nil, err
	}
	line := make([]byte, 0, len(kp)+len(v)+5)
	line = append(line, kp...)
	line = append(line, " = "...)
	line = append(line, v...)
	return append(line, d.eol()...), nil
}

// renderKeyPath renders a dotted key path, quoting the parts that need it
// the same way the toml encoder does.
func (d *Document) renderKeyPath(parts []string) (string, error) {
	rendered := make([]string, len(parts))
	for i, part := range parts {
		line, err := renderKVLine(part, 0)
		if err != nil {
			return "", err
		}
		suffix := []byte(" = 0\n")
		if !bytes.HasSuffix(line, suffix) {
			return "", errors.New("toml/edit: internal error: unexpected encoder output for key")
		}
		rendered[i] = string(line[:len(line)-len(suffix)])
	}
	return strings.Join(rendered, "."), nil
}

func (d *Document) withEOL(line []byte) []byte {
	// line ends with '\n'; rewrite it as CRLF if the document uses CRLF.
	eol := d.eol()
	if len(eol) == 2 {
		line = append(line[:len(line)-1], eol...)
	}
	return line
}

func pathString(key []string) string {
	return strings.Join(key, ".")
}
