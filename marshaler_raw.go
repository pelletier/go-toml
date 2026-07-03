package toml

// This file contains the implementation of Encoder.EnableMarshalerInterface:
// the encode half of the unstable.Marshaler / unstable.RawMessage support. It
// is kept separate from marshaler.go so the feature's (opt-in, mostly cold)
// code does not sit between the default encode path's hot functions.

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"

	"github.com/pelletier/go-toml/v2/internal/parserbridge"
	"github.com/pelletier/go-toml/v2/unstable"
)

// rawShape classifies the bytes produced by an unstable.Marshaler.
type rawShape uint8

const (
	// shapeUnknown is the default: not an unstable.Marshaler, or the interface
	// is disabled.
	shapeUnknown rawShape = iota
	// shapeEmpty is whitespace-only content: it has no TOML representation and
	// is omitted from the output.
	shapeEmpty
	// shapeValue is a single TOML value, emitted inline as `key = <raw>`.
	shapeValue
	// shapeTable is one or more key-value lines, emitted as a `[key]` body.
	shapeTable
)

var marshalerType = reflect.TypeOf(new(unstable.Marshaler)).Elem()

// typeMarshalerCache caches, per type, how it implements unstable.Marshaler:
// 0 not at all, 1 the type itself, 2 its pointer. It is deliberately separate
// from typeEncPropsCache: growing typeEncProps changes the layout and register
// shape of every cached-props lookup on the default encode path, which is
// measurably slower. Only EnableMarshalerInterface paths consult this cache.
var typeMarshalerCache sync.Map // reflect.Type -> uint8

func marshalerPropsForType(t reflect.Type) uint8 {
	if p, ok := typeMarshalerCache.Load(t); ok {
		return p.(uint8)
	}
	var p uint8
	switch {
	case t.Implements(marshalerType):
		p = 1
	case reflect.PtrTo(t).Implements(marshalerType):
		p = 2
	}
	typeMarshalerCache.Store(t, p)
	return p
}

// encodeMarshalerRoot emits the bytes of a root-level unstable.Marshaler as
// the whole document: the encode counterpart of the decoder delivering the
// whole document to a root Unmarshaler. The bytes must form a TOML document
// (key-value lines and table headers); empty output produces an empty
// document.
func (e *encoderState) encodeMarshalerRoot(rv reflect.Value) error {
	raw, err := e.marshalerBytes(rv)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	if err := e.validateRawTableBody(rv.Type(), trimmed); err != nil {
		// A single TOML value has no meaning at the document root; report it
		// as such rather than as a syntax error.
		if shape, _ := e.classifyRaw(trimmed); shape == shapeValue {
			return fmt.Errorf("toml: cannot encode %s as a document root: MarshalTOML returned a single TOML value, not a document", rv.Type())
		}
		return err
	}
	e.buf = append(e.buf, trimmed...)
	e.buf = append(e.buf, '\n')
	e.lastWasHeader = false
	return nil
}

// isMarshalerArrayOfTables is the EnableMarshalerInterface variant of
// isArrayOfTables: a Marshaler element counts as a table only when its raw
// content is table shaped (key-value lines); one holding a single value makes
// the whole slice a plain array instead.
func (e *encoderState) isMarshalerArrayOfTables(v reflect.Value) bool {
	for i := 0; i < v.Len(); i++ {
		elem, ok := resolve(v.Index(i))
		if !ok {
			return false
		}
		if marshalerPropsForType(elem.Type()) != 0 {
			raw, err := e.marshalerBytes(elem)
			if err != nil {
				return false
			}
			if shape, _ := e.classifyRaw(raw); shape != shapeTable {
				return false
			}
			continue
		}
		if isValueKind(elem) {
			return false
		}
	}
	return true
}

// marshalerBytes returns the raw TOML produced by v's unstable.Marshaler
// implementation. The caller guarantees v implements the interface
// (marshalerPropsForType(v.Type()) != 0).
func (e *encoderState) marshalerBytes(v reflect.Value) ([]byte, error) {
	t := v.Type()
	var m unstable.Marshaler
	switch {
	case marshalerPropsForType(t) == 1:
		// The type itself implements Marshaler (e.g. a value receiver).
		m = v.Interface().(unstable.Marshaler)
	case v.CanAddr():
		// Only the pointer implements it, and v is addressable.
		m = v.Addr().Interface().(unstable.Marshaler)
	default:
		// Only the pointer implements it, but v is not addressable: take the
		// address of a copy.
		tmp := reflect.New(t)
		tmp.Elem().Set(v)
		m = tmp.Interface().(unstable.Marshaler)
	}
	b, err := m.MarshalTOML()
	if err != nil {
		return nil, fmt.Errorf("toml: error calling MarshalTOML for type %s: %w", t, err)
	}
	return b, nil
}

// validateRawTableBody checks that trimmed — table-shaped Marshaler output
// about to be spliced verbatim — is syntactically valid TOML, so a Marshaler
// cannot silently corrupt the document. It reuses e.parser, like classifyRaw.
func (e *encoderState) validateRawTableBody(t reflect.Type, trimmed []byte) error {
	e.parser.Reset(trimmed)
	for e.parser.NextExpression() {
	}
	if err := e.parser.Error(); err != nil {
		return fmt.Errorf("toml: error calling MarshalTOML for type %s: invalid TOML: %w", t, err)
	}
	return nil
}

// classifyRaw decides whether b (the output of an unstable.Marshaler) is empty,
// a single TOML value, or a table body. It returns the trimmed bytes that
// should be spliced into the document. It reuses e.parser, so it is not safe
// for concurrent use (encoderState is not shared).
func (e *encoderState) classifyRaw(b []byte) (rawShape, []byte) {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 {
		return shapeEmpty, trimmed
	}
	e.parser.Reset(trimmed)
	_, rest, err := parserbridge.ParseValue(&e.parser, trimmed)
	if err == nil && len(bytes.TrimSpace(rest)) == 0 {
		return shapeValue, trimmed
	}
	return shapeTable, trimmed
}

// resolveMarshalerEntries classifies every entry whose value implements
// unstable.Marshaler, recording the shape on the entry so the two table passes
// can route it without re-classifying. Any error from MarshalTOML is surfaced
// eagerly. The marshaled bytes themselves are produced again at emit time by
// marshalerValue; that keeps entry small on the encoder's hot path, and only
// runs when the (opt-in) interface is enabled.
func (e *encoderState) resolveMarshalerEntries(entries []entry) error {
	for i := range entries {
		ent := &entries[i]
		if ent.options.rawShape != shapeUnknown {
			// Already classified: SetOmitEmptySuperTables resolves the
			// entries early to route them by shape.
			continue
		}
		v, ok := resolve(ent.value)
		if !ok || marshalerPropsForType(v.Type()) == 0 {
			continue
		}
		raw, err := e.marshalerBytes(v)
		if err != nil {
			return err
		}
		ent.options.rawShape, _ = e.classifyRaw(raw)
	}
	return nil
}

// marshalerValue returns the trimmed bytes to splice for an entry already known
// to be an unstable.Marshaler (ent.options.rawShape != shapeUnknown).
func (e *encoderState) marshalerValue(ent *entry) ([]byte, error) {
	v, _ := resolve(ent.value)
	raw, err := e.marshalerBytes(v)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(raw), nil
}

// appendMarshalerInline emits an unstable.Marshaler value in a value position
// (a `key = ` line, an array element or an inline-table member), where only a
// single TOML value is valid. It classifies the bytes it just fetched, so a
// Marshaler returning different content across calls cannot splice non-value
// bytes into a value position. The caller guarantees v implements the
// interface; appendValue's guard branch returns into this function without
// rejoining, so the common encode path keeps no values live across a call.
func (e *encoderState) appendMarshalerInline(b []byte, v reflect.Value, t reflect.Type) ([]byte, error) {
	raw, err := e.marshalerBytes(v)
	if err != nil {
		return nil, err
	}
	shape, trimmed := e.classifyRaw(raw)
	switch shape {
	case shapeValue:
		return append(b, trimmed...), nil
	case shapeEmpty:
		return nil, fmt.Errorf("toml: cannot encode an empty %s as an inline value", t)
	default:
		return nil, fmt.Errorf("toml: cannot encode %s as an inline value: %q is not a single TOML value", t, trimmed)
	}
}

// encodeTableWithMarshalers is the EnableMarshalerInterface variant of
// encodeTable's two passes: entries whose value implements unstable.Marshaler
// are classified up front and routed by shape; everything else follows the
// baseline logic.
func (e *encoderState) encodeTableWithMarshalers(entries []entry, commented bool, indent int) error {
	// Classify Marshaler entries once so the passes can route them by shape
	// and surface MarshalTOML errors eagerly.
	if err := e.resolveMarshalerEntries(entries); err != nil {
		return err
	}

	// First pass: emit all key-values; tables are handled by the second
	// pass.
	for i := range entries {
		ent := &entries[i]
		switch ent.options.rawShape {
		case shapeUnknown:
			// Not a Marshaler: handled by the baseline logic below the
			// switch.
		case shapeEmpty:
			// No TOML representation: omit the key.
			continue
		case shapeValue:
			if err := e.encodeKeyValue(*ent, commented, indent); err != nil {
				return err
			}
			continue
		case shapeTable:
			// A table body is emitted in the second pass, unless it is
			// forced inline (SetTablesInline / inline tag), which has no
			// valid inline form and is reported as an error by the value
			// path.
			if e.tablesInline || ent.options.inline {
				if err := e.encodeKeyValue(*ent, commented, indent); err != nil {
					return err
				}
			}
			continue
		}
		if e.entryIsTable(ent) {
			continue
		}
		if err := e.encodeKeyValue(*ent, commented, indent); err != nil {
			return err
		}
	}

	// Second pass: emit the sub-tables, extending the shared key stack.
	for i := range entries {
		ent := entries[i]
		switch ent.options.rawShape {
		case shapeUnknown:
			// Not a Marshaler: handled by the baseline logic below the
			// switch.
		case shapeValue, shapeEmpty:
			// Not a table: already handled (or omitted) in the first pass.
			continue
		case shapeTable:
			// Emit the raw body verbatim under the freshly pushed header.
			// (The forced-inline case already errored in the first pass.)
			entCommented := commented || ent.options.commented
			e.keyStack = append(e.keyStack, ent.key)
			if err := e.encodeMarshalerTable(&ent, entCommented, indent); err != nil {
				return err
			}
			e.keyStack = e.keyStack[:len(e.keyStack)-1]
			continue
		}
		if !e.entryIsTable(&ent) {
			continue
		}
		entCommented := commented || ent.options.commented
		e.keyStack = append(e.keyStack, ent.key)

		if e.isArrayOfTables(ent.value) {
			if err := e.encodeArrayTable(ent, entCommented, indent); err != nil {
				return err
			}
			e.keyStack = e.keyStack[:len(e.keyStack)-1]
			continue
		}

		// The value is resolvable: entryIsTable already resolved it.
		tv, _ := resolve(ent.value)

		subEntries, err := e.collectEntries(tv)
		if err != nil {
			return err
		}

		subIndent := indent + 1
		if e.omitEmptySuperTables && ent.options.comment == "" {
			// onlySubTables routes entries by their Marshaler shape, so
			// classification cannot wait for the recursive call below.
			// resolveMarshalerEntries skips already-classified entries,
			// making the second classification there a no-op.
			if err := e.resolveMarshalerEntries(subEntries); err != nil {
				return err
			}
			if e.onlySubTables(subEntries) {
				// The table has no key-value of its own and at least one
				// sub-table: emitting the sub-tables implicitly defines it,
				// so its header can be omitted. Its children take its place
				// in the indentation hierarchy.
				subIndent = indent
			} else {
				e.writeTableHeader(ent.options.comment, entCommented, false, indent)
			}
		} else {
			e.writeTableHeader(ent.options.comment, entCommented, false, indent)
		}

		if err := e.encodeTableEntries(subEntries, entCommented, subIndent); err != nil {
			return err
		}
		e.keyStack = e.keyStack[:len(e.keyStack)-1]
	}

	e.putEntries(entries)
	return nil
}

// encodeMarshalerTable emits a table-shaped unstable.Marshaler entry: the
// header for the key currently on the stack, then the raw body verbatim.
func (e *encoderState) encodeMarshalerTable(ent *entry, commented bool, indent int) error {
	raw, err := e.marshalerValue(ent)
	if err != nil {
		return err
	}
	// Validate the bytes actually being spliced: MarshalTOML is called again
	// for the emit, so this both rejects invalid TOML and guards against an
	// implementation that returned different content than during
	// classification.
	if err := e.validateRawTableBody(ent.value.Type(), raw); err != nil {
		return err
	}
	e.writeTableHeader(ent.options.comment, commented, false, indent)
	e.spliceRawTableBody(raw, commented)
	return nil
}

// spliceRawTableBody appends a raw table body verbatim after its header. When
// the table is commented, every physical line is prefixed with the comment
// marker so the body does not leak into the document as live keys.
func (e *encoderState) spliceRawTableBody(raw []byte, commented bool) {
	if len(raw) == 0 {
		return
	}
	if commented {
		e.buf = append(e.buf, "# "...)
		e.buf = append(e.buf, bytes.ReplaceAll(raw, []byte("\n"), []byte("\n# "))...)
	} else {
		e.buf = append(e.buf, raw...)
	}
	e.buf = append(e.buf, '\n')
	e.lastWasHeader = false
}

// encodeArrayTableWithMarshalers is the EnableMarshalerInterface variant of
// encodeArrayTable: an unstable.Marshaler element splices its raw table body
// verbatim instead of being encoded structurally.
func (e *encoderState) encodeArrayTableWithMarshalers(ent entry, commented bool, indent int) error {
	v, _ := resolve(ent.value)
	comment := ent.options.comment
	for i := 0; i < v.Len(); i++ {
		// Elements are resolvable: isArrayOfTables already resolved them.
		elem, _ := resolve(v.Index(i))

		e.writeTableHeader(comment, commented, true, indent)
		// The comment is only present before the first element.
		comment = ""

		// The shape was checked by isArrayOfTables, but MarshalTOML is called
		// again for the emit, so the spliced bytes are validated here.
		if marshalerPropsForType(elem.Type()) != 0 {
			raw, err := e.marshalerBytes(elem)
			if err != nil {
				return err
			}
			trimmed := bytes.TrimSpace(raw)
			if err := e.validateRawTableBody(elem.Type(), trimmed); err != nil {
				return err
			}
			e.spliceRawTableBody(trimmed, commented)
			continue
		}

		err := e.encodeTable(elem, commented, indent+1)
		if err != nil {
			return err
		}
	}
	return nil
}
