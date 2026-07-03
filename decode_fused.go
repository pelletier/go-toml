package toml

import (
	"errors"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2/internal/parserbridge"
	"github.com/pelletier/go-toml/v2/unstable"
)

// unmarshalFused decodes a whole document into a native map[string]interface{}
// tree with no reflection on the document structure and no AST at all: table
// headers, keys, scalars, arrays, and inline tables are all scanned directly
// into their native representation. Scratch stacks make the containers
// exact-size, and the keys declared inside a container value are logged and
// replayed through the seen-tracker once the expression has fully parsed (see
// fusedOp), so validation and error precedence match the AST path.
//
// It is used when the target is a fully generic value (interface{} or
// map[string]interface{}) and the unmarshaler interface is disabled. The
// seen-tracker validates the document (duplicate keys, type consistency), so
// the builder creates and merges containers without revalidating. Strict mode
// never applies to a generic target (a map has no "unknown fields"), and
// captures never apply (a generic value implements no Unmarshaler).
func (d *decoder) unmarshalFused(root reflect.Value, data []byte) error {
	var m map[string]interface{}
	if !root.IsNil() {
		// Decode into (merge with) an existing generic map when present.
		if em, ok := root.Interface().(map[string]interface{}); ok {
			m = em
		}
	}
	if m == nil {
		m = map[string]interface{}{}
	}

	if err := d.fusedDocument(m, data); err != nil {
		// An error may have unwound partially-built containers: drop the
		// references still held by the scratch stacks so the pooled decoder
		// does not retain user data. (On success both stacks are empty and
		// already zeroed by their copy-outs.)
		clear(d.anyStack[:cap(d.anyStack)])
		d.anyStack = d.anyStack[:0]
		clear(d.kvStack[:cap(d.kvStack)])
		d.kvStack = d.kvStack[:0]
		return d.wrapFusedError(data, err)
	}

	if root.CanSet() {
		root.Set(reflect.ValueOf(m))
	}
	return nil
}

// fusedDocument runs the top-level expression loop, mirroring
// Parser.NextExpression but storing values directly into native maps.
func (d *decoder) fusedDocument(m map[string]interface{}, b []byte) error {
	cur := m
	for {
		b = fusedSkipWS(b)
		if len(b) == 0 {
			return nil
		}
		switch b[0] {
		case '\n':
			b = b[1:]
		case '\r':
			if len(b) > 1 && b[1] == '\n' {
				b = b[2:]
				continue
			}
			return unstable.NewParserError(b[:1], "expected newline but got %#U", b[0])
		case '#':
			_, rest, err := parserbridge.ScanComment(b)
			if err != nil {
				return err
			}
			rest, err = fusedConsumeEOL(rest)
			if err != nil {
				return err
			}
			b = rest
		case '[':
			rest, err := d.fusedTable(b, m, &cur)
			if err != nil {
				return err
			}
			b = rest
		default:
			rest, err := d.fusedKeyVal(b, cur)
			if err != nil {
				return err
			}
			b = rest
		}
	}
}

// fusedTable handles a [table] or [[array table]] header. b starts at '['. It
// updates *cur to the table the following key-values belong to.
func (d *decoder) fusedTable(b []byte, root map[string]interface{}, cur *map[string]interface{}) ([]byte, error) {
	arrayTable := len(b) > 1 && b[1] == '['

	var start []byte
	if arrayTable {
		start = fusedSkipWS(b[2:])
	} else {
		start = fusedSkipWS(b[1:])
	}

	var err error
	var rawKey []byte
	d.keyParts, rawKey, b, err = parserbridge.ScanKey(&d.p, start, d.keyParts[:0])
	if err != nil {
		return nil, err
	}

	if arrayTable {
		if len(b) < 2 || b[0] != ']' || b[1] != ']' {
			return nil, unstable.NewParserError(fusedHL1(b), "expected ']]' to close array table name")
		}
		b = b[2:]
	} else {
		if len(b) == 0 || b[0] != ']' {
			return nil, unstable.NewParserError(fusedHL1(b), "expected ']' to close table name")
		}
		b = b[1:]
	}

	// The whole expression (including its line termination) is parsed before
	// it is validated, to keep error precedence identical to the AST path.
	b, err = d.fusedFinishLine(b)
	if err != nil {
		return nil, err
	}

	if arrayTable {
		first, err := d.seen.CheckArrayTable(d.keyParts)
		if err != nil {
			return nil, d.fusedSeenError(rawKey, d.keyParts, err)
		}
		*cur = d.anyArrayTableParts(root, d.keyParts, first)
	} else {
		if _, err := d.seen.CheckTable(d.keyParts); err != nil {
			return nil, d.fusedSeenError(rawKey, d.keyParts, err)
		}
		*cur = d.anyTableParts(root, d.keyParts)
	}
	return b, nil
}

// fusedKeyVal handles a `key = value` expression relative to the current table
// cur. b starts at the first character of the key.
func (d *decoder) fusedKeyVal(b []byte, cur map[string]interface{}) ([]byte, error) {
	var err error
	var rawKey []byte
	d.keyParts, rawKey, b, err = parserbridge.ScanKey(&d.p, b, d.keyParts[:0])
	if err != nil {
		return nil, err
	}
	if len(b) == 0 || b[0] != '=' {
		return nil, unstable.NewParserError(fusedHL1(b), "expected '=' after key")
	}
	b = fusedSkipWS(b[1:])
	if len(b) == 0 {
		return nil, unstable.NewParserError(b, "expected value, not end of input")
	}

	if c := b[0]; c == '[' || c == '{' {
		// Container value: build the native value directly, recording the keys
		// declared inside it. They are validated only after the whole line has
		// parsed and the outer key has been checked, so that error precedence
		// is identical to the AST path.
		d.fusedParts = d.fusedParts[:0]
		d.fusedOps = d.fusedOps[:0]
		av, rest, err := d.fusedContainerValue(b)
		if err != nil {
			return nil, err
		}
		rest, err = d.fusedFinishLine(rest)
		if err != nil {
			return nil, err
		}
		leafID, err := d.seen.CheckKeyValue(d.keyParts)
		if err != nil {
			return nil, d.fusedSeenError(rawKey, d.keyParts, err)
		}
		if err := d.replayFusedOps(leafID); err != nil {
			return nil, d.fusedSeenError(rawKey, d.keyParts, err)
		}
		d.setFusedLeaf(cur, d.keyParts, av)
		return rest, nil
	}

	// Scalar value: scan it without building a node, then validate and convert
	// it natively.
	k, _, value, rest, err := parserbridge.ScanScalar(&d.p, b)
	if err != nil {
		return nil, err
	}
	kind := unstable.Kind(k)
	rest, err = d.fusedFinishLine(rest)
	if err != nil {
		return nil, err
	}
	if _, err := d.seen.CheckKeyValue(d.keyParts); err != nil {
		return nil, d.fusedSeenError(rawKey, d.keyParts, err)
	}
	av, err := d.fusedScalar(kind, value)
	if err != nil {
		return nil, err
	}
	d.setFusedLeaf(cur, d.keyParts, av)
	return rest, nil
}

// fusedOp is one step of the deferred key validation of a container value.
// The keys declared inside an array or inline table cannot be checked while
// the value parses (a syntax error later on the line must win, as must a
// duplicate of the outer key), so parsing logs the declarations and
// replayFusedOps runs them through the seen-tracker afterwards.
type fusedOp struct {
	lo, hi int32 // fusedOpKey: the parts d.fusedParts[lo:hi] of the key
	op     uint8
}

const (
	// fusedOpKey declares a (possibly dotted) key of an inline table; the
	// scopes of the value it introduces follow until the matching fusedOpPop.
	fusedOpKey = iota
	// fusedOpAnon enters an inline table stored in an array, which is its own
	// anonymous key scope, until the matching fusedOpPop.
	fusedOpAnon
	// fusedOpPop leaves the scope entered by fusedOpKey or fusedOpAnon.
	fusedOpPop
)

// replayFusedOps validates the keys recorded by the last container value
// against the seen-tracker, under the entry of the key-value that holds it.
// It mirrors SeenTracker.checkValue, driven by the log instead of an AST.
func (d *decoder) replayFusedOps(rootID int32) error {
	if len(d.fusedOps) == 0 {
		return nil
	}
	d.idStack = d.idStack[:0]
	top := rootID
	for _, op := range d.fusedOps {
		switch op.op {
		case fusedOpKey:
			id, err := d.seen.CheckKeyValueUnder(top, d.fusedParts[op.lo:op.hi])
			if err != nil {
				return err
			}
			d.idStack = append(d.idStack, top)
			top = id
		case fusedOpAnon:
			d.idStack = append(d.idStack, top)
			top = d.seen.CreateAnonymous(top)
		default: // fusedOpPop
			top = d.idStack[len(d.idStack)-1]
			d.idStack = d.idStack[:len(d.idStack)-1]
		}
	}
	return nil
}

// fusedContainerValue parses an array or inline table (b starts at '[' or
// '{') directly into its native generic representation, without building an
// AST. The keys declared inside are appended to the d.fusedOps log for later
// validation.
func (d *decoder) fusedContainerValue(b []byte) (interface{}, []byte, error) {
	if b[0] == '[' {
		return d.fusedArray(b)
	}
	return d.fusedInlineTable(b)
}

// fusedValue parses any TOML value nested in a container. b is not empty.
func (d *decoder) fusedValue(b []byte) (interface{}, []byte, error) {
	switch b[0] {
	case '[', '{':
		return d.fusedContainerValue(b)
	default:
		k, _, value, rest, err := parserbridge.ScanScalar(&d.p, b)
		if err != nil {
			return nil, nil, err
		}
		av, err := d.fusedScalar(unstable.Kind(k), value)
		if err != nil {
			return nil, nil, err
		}
		return av, rest, nil
	}
}

// fusedArray parses an array value natively. b starts at '['. It mirrors the
// grammar (and error messages) of Parser.parseValArray. Elements accumulate
// on d.anyStack and are copied out to an exact-size slice on ']'.
func (d *decoder) fusedArray(b []byte) (interface{}, []byte, error) {
	b = b[1:]
	base := len(d.anyStack)
	afterValue := false
	for {
		b = fusedSkipWS(b)
		if len(b) == 0 {
			return nil, nil, unstable.NewParserError(b, "array is incomplete")
		}
		switch b[0] {
		case ']':
			elems := d.anyStack[base:]
			out := make([]interface{}, len(elems))
			copy(out, elems)
			clear(elems)
			d.anyStack = d.anyStack[:base]
			return out, b[1:], nil
		case '\n':
			b = b[1:]
		case '\r':
			if len(b) > 1 && b[1] == '\n' {
				b = b[2:]
				continue
			}
			return nil, nil, unstable.NewParserError(b[:1], "expected newline but got %#U", b[0])
		case '#':
			_, rest, err := parserbridge.ScanComment(b)
			if err != nil {
				return nil, nil, err
			}
			b = rest
		case ',':
			if !afterValue {
				return nil, nil, unstable.NewParserError(b[:1], "expected value but got %#U", b[0])
			}
			afterValue = false
			b = b[1:]
		default:
			if afterValue {
				return nil, nil, unstable.NewParserError(b[:1], "expected ',' or ']' after array value")
			}
			var (
				v   interface{}
				err error
			)
			if b[0] == '{' {
				// An inline table in an array is its own key scope.
				d.fusedOps = append(d.fusedOps, fusedOp{op: fusedOpAnon})
				v, b, err = d.fusedInlineTable(b)
				if err == nil {
					d.fusedOps = append(d.fusedOps, fusedOp{op: fusedOpPop})
				}
			} else {
				v, b, err = d.fusedValue(b)
			}
			if err != nil {
				return nil, nil, err
			}
			d.anyStack = append(d.anyStack, v)
			afterValue = true
		}
	}
}

// fusedInlineTable parses an inline table value natively. b starts at '{'.
// It mirrors the grammar (and error messages) of Parser.parseInlineTable.
// Key-values accumulate on d.kvStack so that the map is created with its
// exact size on '}'. Key declarations are logged for deferred validation:
// conflicting keys may temporarily build garbage into the result, but the
// replay rejects the document before the value is used.
func (d *decoder) fusedInlineTable(b []byte) (interface{}, []byte, error) {
	b = b[1:]
	base := len(d.kvStack)
	afterValue := false
	for {
		b = fusedSkipWS(b)
		if len(b) == 0 {
			return nil, nil, unstable.NewParserError(b, "inline table is incomplete")
		}
		switch b[0] {
		case '}':
			pairs := d.kvStack[base:]
			m := make(map[string]interface{}, len(pairs))
			for i := range pairs {
				kv := &pairs[i]
				parts := d.fusedParts[kv.lo:kv.hi]
				cur := m
				for j := 0; j < len(parts)-1; j++ {
					cur = d.anyChildTable(cur, d.intern(parts[j]))
				}
				cur[d.intern(parts[len(parts)-1])] = kv.v
			}
			clear(pairs)
			d.kvStack = d.kvStack[:base]
			return m, b[1:], nil
		case '\n':
			b = b[1:]
		case '\r':
			if len(b) > 1 && b[1] == '\n' {
				b = b[2:]
				continue
			}
			return nil, nil, unstable.NewParserError(b[:1], "expected newline but got %#U", b[0])
		case '#':
			_, rest, err := parserbridge.ScanComment(b)
			if err != nil {
				return nil, nil, err
			}
			b = rest
		case ',':
			if !afterValue {
				return nil, nil, unstable.NewParserError(b[:1], "unexpected comma in inline table")
			}
			afterValue = false
			b = b[1:]
		default:
			if afterValue {
				return nil, nil, unstable.NewParserError(b[:1], "expected ',' or '}' after inline table key-value")
			}
			var err error
			b, err = d.fusedInlineKeyval(b)
			if err != nil {
				return nil, nil, err
			}
			afterValue = true
		}
	}
}

// fusedInlineKeyval parses one `key = value` pair of an inline table onto
// d.kvStack. b starts at the first character of the key.
func (d *decoder) fusedInlineKeyval(b []byte) ([]byte, error) {
	save := len(d.fusedParts)
	var err error
	d.fusedParts, _, b, err = parserbridge.ScanKey(&d.p, b, d.fusedParts)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 || b[0] != '=' {
		return nil, unstable.NewParserError(fusedHL1(b), "expected '=' after key")
	}
	b = fusedSkipWS(b[1:])
	if len(b) == 0 {
		return nil, unstable.NewParserError(b, "expected value, not end of input")
	}

	// The parts range must be pinned before parsing the value: the keys of a
	// nested inline table extend d.fusedParts.
	lo := int32(save)              //nolint:gosec // part counts are bounded by document size
	hi := int32(len(d.fusedParts)) //nolint:gosec // part counts are bounded by document size
	d.fusedOps = append(d.fusedOps, fusedOp{op: fusedOpKey, lo: lo, hi: hi})
	v, b, err := d.fusedValue(b)
	if err != nil {
		return nil, err
	}
	d.fusedOps = append(d.fusedOps, fusedOp{op: fusedOpPop})

	d.kvStack = append(d.kvStack, fusedKV{v: v, lo: lo, hi: hi})
	return b, nil
}

// fusedSeenError turns a bare error returned by a SeenTracker parts-method
// into a ParserError carrying the position (the raw key span) and key path of
// the offending expression, so that it is reported as a DecodeError with
// context. It mirrors decoder.wrapSeenError for the fused (AST-less) path.
func (d *decoder) fusedSeenError(rawKey []byte, parts [][]byte, err error) error {
	key := make(Key, len(parts))
	for i, p := range parts {
		key[i] = string(p)
	}
	return &unstable.ParserError{
		Highlight: rawKey,
		Message:   strings.TrimPrefix(err.Error(), "toml: "),
		Key:       key,
	}
}

// fusedScalar converts a scanned scalar value into the native Go value used
// for generic targets. It mirrors the scalar cases of decodeAny.
func (d *decoder) fusedScalar(kind unstable.Kind, value []byte) (interface{}, error) {
	switch kind {
	case unstable.String:
		return string(value), nil
	case unstable.Integer:
		i, err := parseInteger(value)
		return i, err
	case unstable.Float:
		f, err := parseFloat(value)
		return f, err
	case unstable.Bool:
		return value[0] == 't', nil
	case unstable.DateTime:
		t, err := parseDateTime(value)
		return t, err
	case unstable.LocalDateTime:
		dt, rest, err := parseLocalDateTime(value)
		if err != nil {
			return nil, err
		}
		if len(rest) > 0 {
			return nil, unstable.NewParserError(rest, "extra characters at the end of a local date time")
		}
		return dt, nil
	case unstable.LocalDate:
		date, err := parseLocalDate(value)
		return date, err
	case unstable.LocalTime:
		t, rest, err := parseLocalTime(value)
		if err != nil {
			return nil, err
		}
		if len(rest) > 0 {
			return nil, unstable.NewParserError(rest, "extra characters at the end of a local time")
		}
		return t, nil
	default:
		return nil, unstable.NewParserError(value, "unsupported value kind %s", kind)
	}
}

// anyTableParts navigates a [table] header (given its key parts) to the map it
// designates, creating intermediate tables as needed.
func (d *decoder) anyTableParts(m map[string]interface{}, parts [][]byte) map[string]interface{} {
	cur := m
	for _, p := range parts {
		cur = d.anyChildTable(cur, d.intern(p))
	}
	return cur
}

// anyArrayTableParts navigates a [[array table]] header (given its key parts),
// appends a fresh element to the designated array, and returns it. first is
// true the first time this header is seen, in which case any pre-existing array
// (from a reused target) is reset.
func (d *decoder) anyArrayTableParts(m map[string]interface{}, parts [][]byte, first bool) map[string]interface{} {
	cur := m
	name := d.intern(parts[0])
	for i := 1; i < len(parts); i++ {
		cur = d.anyChildTable(cur, name)
		name = d.intern(parts[i])
	}
	s, _ := cur[name].([]interface{})
	if first {
		s = s[:0]
	}
	elem := map[string]interface{}{}
	cur[name] = append(s, elem)
	return elem
}

// setFusedLeaf assigns av at the (possibly dotted) key parts within cur,
// creating intermediate maps as needed.
func (d *decoder) setFusedLeaf(cur map[string]interface{}, parts [][]byte, av interface{}) {
	for i := 0; i < len(parts)-1; i++ {
		cur = d.anyChildTable(cur, d.intern(parts[i]))
	}
	cur[d.intern(parts[len(parts)-1])] = av
}

// wrapFusedError gives document context to errors produced by the fused
// decoder.
func (d *decoder) wrapFusedError(data []byte, err error) error {
	var perr *unstable.ParserError
	if errors.As(err, &perr) && len(perr.Highlight) == 0 {
		// Mirror NextExpression: give end-of-input errors a usable position by
		// extending the empty highlight to the last byte of the document.
		if offset := cap(data) - cap(perr.Highlight); offset > 0 && offset == len(data) {
			perr.Highlight = data[offset-1 : offset]
		}
	}
	return d.wrapError(data, err)
}

func fusedSkipWS(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	return b
}

func fusedConsumeEOL(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return b, nil
	}
	switch b[0] {
	case '\n':
		return b[1:], nil
	case '\r':
		if len(b) > 1 && b[1] == '\n' {
			return b[2:], nil
		}
	}
	return nil, unstable.NewParserError(b[:1], "expected newline but got %#U", b[0])
}

// fusedFinishLine consumes `ws [comment] (newline|eof)` after an expression.
func (d *decoder) fusedFinishLine(b []byte) ([]byte, error) {
	b = fusedSkipWS(b)
	if len(b) > 0 && b[0] == '#' {
		_, rest, err := parserbridge.ScanComment(b)
		if err != nil {
			return nil, err
		}
		b = rest
	}
	return fusedConsumeEOL(b)
}

func fusedHL1(b []byte) []byte {
	if len(b) > 0 {
		return b[:1]
	}
	return b
}
