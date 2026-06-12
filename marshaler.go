package toml

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Marshal serializes a Go value as a TOML document.
//
// It is a shortcut for Encoder.Encode() with the default options.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Encoder writes a TOML document to an output stream.
type Encoder struct {
	// output
	w io.Writer

	// global settings
	tablesInline       bool
	arraysMultiline    bool
	indentSymbol       string
	indentTables       bool
	marshalJSONNumbers bool
}

// NewEncoder returns a new Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:            w,
		indentSymbol: "  ",
	}
}

// SetTablesInline forces the encoder to emit all tables inline.
//
// This behavior can be controlled on an individual struct field basis with
// the inline tag:
//
//	MyField `toml:",inline"`
func (enc *Encoder) SetTablesInline(inline bool) *Encoder {
	enc.tablesInline = inline
	return enc
}

// SetArraysMultiline forces the encoder to emit all arrays with one element
// per line.
//
// This behavior can be controlled on an individual struct field basis with
// the multiline tag:
//
//	MyField `multiline:"true"`
func (enc *Encoder) SetArraysMultiline(multiline bool) *Encoder {
	enc.arraysMultiline = multiline
	return enc
}

// SetIndentSymbol defines the string that should be used for indentation. The
// provided string is repeated for each indentation level. Defaults to two
// spaces.
func (enc *Encoder) SetIndentSymbol(s string) *Encoder {
	enc.indentSymbol = s
	return enc
}

// SetIndentTables forces the encoder to intent tables and array tables.
func (enc *Encoder) SetIndentTables(indent bool) *Encoder {
	enc.indentTables = indent
	return enc
}

// SetMarshalJSONNumbers forces the encoder to serialize `json.Number` as a
// float or integer instead of relying on TextMarshaler to emit a string.
//
// *Unstable:* This method does not follow the compatibility guarantees of
// semver. It can be changed or removed without a new major version being
// issued.
func (enc *Encoder) SetMarshalJSONNumbers(indent bool) *Encoder {
	enc.marshalJSONNumbers = indent
	return enc
}

// Encode writes a TOML representation of v to the stream.
//
// If v cannot be represented to TOML it returns an error.
//
// # Encoding rules
//
// A top level slice containing only maps or structs is encoded as [[table
// array]].
//
// All slices not matching rule 1 are encoded as [array]. As a result, any map
// or struct they contain is encoded as an {inline table}.
//
// Nil interfaces and nil pointers are not supported.
//
// Keys in key-values always have one part.
//
// Intermediate tables are always printed.
//
// By default, strings are encoded as literal string, unless they contain
// either a newline character or a single quote. In that case they are emitted
// as quoted strings.
//
// Unsigned integers larger than math.MaxInt64 cannot be encoded. Doing so
// results in an error. This rule exists because the TOML specification only
// requires parsers to support at least the 64 bits integer range. Allowing
// larger numbers would create non-standard TOML documents, which may not be
// readable (at best) by other implementations. To encode such numbers, a
// solution is a custom type that implements encoding.TextMarshaler.
//
// When encoding structs, fields are encoded in order of definition, with
// their exact name.
//
// Tables and array tables are separated by empty lines. However, consecutive
// subtables definitions are not. For example:
//
//	[top1]
//
//	[top2]
//	[top2.child1]
//
//	[[array]]
//
//	[[array]]
//	[array.child2]
//
// # Struct tags
//
// The encoding of each public struct field can be customized by the format
// string in the "toml" key of the struct field's tag. This follows
// encoding/json's convention. The format string starts with the name of the
// field, optionally followed by a comma-separated list of options. The name
// may be empty in order to provide options without overriding the default
// name.
//
// The "multiline" option emits strings as quoted multi-line TOML strings. It
// has no effect on fields that would not be encoded as strings.
//
// The "inline" option turns fields that would be emitted as tables into
// inline tables instead. It has no effect on other fields.
//
// The "omitempty" option prevents empty values or groups from being emitted.
//
// The "omitzero" option prevents zero values or groups from being emitted.
//
// The "commented" option prefixes the value and all its children with a
// comment symbol.
//
// In addition to the "toml" tag struct tag, a "comment" tag can be used to
// emit a TOML comment before the value being annotated. Comments are ignored
// inside inline tables. For array tables, the comment is only present before
// the first element of the array.
func (enc *Encoder) Encode(v interface{}) error {
	e := encoderState{Encoder: enc}

	err := e.encodeRoot(v)
	if err != nil {
		return err
	}

	_, err = enc.w.Write(e.buf)
	if err != nil {
		return fmt.Errorf("toml: cannot write: %w", err)
	}
	return nil
}

type encoderState struct {
	*Encoder

	buf []byte

	// lastWasHeader is true when the last line written was a table header,
	// used to avoid empty lines between consecutive table definitions.
	lastWasHeader bool
}

// valueOptions are the encoding options attached to one entry of a table.
type valueOptions struct {
	multiline bool
	inline    bool
	omitempty bool
	omitzero  bool
	commented bool
	comment   string
}

// entry is a deferred key-value of a table being encoded.
type entry struct {
	key     string
	value   reflect.Value
	options valueOptions
}

func (e *encoderState) encodeRoot(v interface{}) error {
	if v == nil {
		return fmt.Errorf("toml: cannot encode a nil interface")
	}

	rv := reflect.ValueOf(v)
	rv, ok := resolve(rv)
	if !ok {
		return fmt.Errorf("toml: cannot encode a nil pointer")
	}

	switch rv.Kind() {
	case reflect.Map, reflect.Struct:
		if isValueKind(rv) {
			return fmt.Errorf("toml: cannot encode a %s as a document root", rv.Type())
		}
		return e.encodeTable(nil, rv, false, 0)
	default:
		return fmt.Errorf("toml: cannot encode a %s as a document root", rv.Type())
	}
}

// resolve unwraps pointers and interfaces until a concrete value is found.
// Returns false if it resolves to nil.
func resolve(v reflect.Value) (reflect.Value, bool) {
	for {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				return v, false
			}
			v = v.Elem()
		case reflect.Interface:
			if v.IsNil() {
				return v, false
			}
			v = v.Elem()
		default:
			return v, true
		}
	}
}

// isValueKind returns true when the resolved value is encoded as a TOML
// value (as opposed to a table).
func isValueKind(v reflect.Value) bool {
	t := v.Type()
	switch t {
	case timeType, localDateType, localTimeType, localDateTimeType:
		return true
	}
	if t.Implements(textMarshalerType) || reflect.PtrTo(t).Implements(textMarshalerType) {
		return true
	}
	switch v.Kind() {
	case reflect.Map, reflect.Struct:
		return false
	default:
		return true
	}
}

// isTableLike returns true when the value should be encoded as a table (or
// an array of tables for slices).
func (e *encoderState) isTableLike(v reflect.Value) bool {
	v, ok := resolve(v)
	if !ok {
		// nil pointers to maps and structs are encoded as empty tables;
		// other nils are values (or errors) handled later.
		t := v.Type()
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct || t.Kind() == reflect.Map {
			z := reflect.New(t).Elem()
			return !isValueKind(z)
		}
		return false
	}
	return !isValueKind(v)
}

// isArrayOfTables returns true when the value is a non-empty slice or array
// containing only table-like values.
func (e *encoderState) isArrayOfTables(v reflect.Value) bool {
	v, ok := resolve(v)
	if !ok {
		return false
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return false
	}
	if v.Len() == 0 {
		return false
	}
	for i := 0; i < v.Len(); i++ {
		elem, ok := resolve(v.Index(i))
		if !ok || isValueKind(elem) {
			return false
		}
	}
	return true
}

// encodeTable writes the content of a table at the given key path.
func (e *encoderState) encodeTable(key []string, v reflect.Value, commented bool, indent int) error {
	entries, err := e.collectEntries(v)
	if err != nil {
		return err
	}

	var tables []entry

	// First pass: emit all key-values, and collect the tables.
	for _, ent := range entries {
		if !e.tablesInline && !ent.options.inline && (e.isTableLike(ent.value) || e.isArrayOfTables(ent.value)) {
			tables = append(tables, ent)
			continue
		}

		err := e.encodeKeyValue(ent, commented, indent)
		if err != nil {
			return err
		}
	}

	// Second pass: emit the sub-tables.
	for _, ent := range tables {
		entCommented := commented || ent.options.commented
		subKey := append(key, ent.key) //nolint:gocritic

		if e.isArrayOfTables(ent.value) {
			err := e.encodeArrayTable(subKey, ent, entCommented, indent)
			if err != nil {
				return err
			}
			continue
		}

		tv, ok := resolve(ent.value)
		if !ok {
			// nil pointer to a table: encode an empty table.
			t := ent.value.Type()
			for t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			tv = reflect.New(t).Elem()
		}

		e.writeTableHeader(subKey, ent.options.comment, entCommented, false, indent)

		err := e.encodeTable(subKey, tv, entCommented, indent+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeArrayTable writes all the elements of an array of tables.
func (e *encoderState) encodeArrayTable(key []string, ent entry, commented bool, indent int) error {
	v, _ := resolve(ent.value)
	comment := ent.options.comment
	for i := 0; i < v.Len(); i++ {
		elem, ok := resolve(v.Index(i))
		if !ok {
			return fmt.Errorf("toml: cannot encode a nil element in an array of tables")
		}

		e.writeTableHeader(key, comment, commented, true, indent)
		// The comment is only present before the first element.
		comment = ""

		err := e.encodeTable(key, elem, commented, indent+1)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeTableHeader emits a [table] or [[array table]] header line, preceded
// by an empty line and comments as needed.
func (e *encoderState) writeTableHeader(key []string, comment string, commented bool, array bool, indent int) {
	if len(e.buf) > 0 && !e.lastWasHeader {
		e.buf = append(e.buf, '\n')
	}

	headerIndent := indent

	e.writeComment(comment, headerIndent)

	e.writeIndent(headerIndent)
	if commented {
		e.buf = append(e.buf, "# "...)
	}
	e.buf = append(e.buf, '[')
	if array {
		e.buf = append(e.buf, '[')
	}
	for i, part := range key {
		if i > 0 {
			e.buf = append(e.buf, '.')
		}
		e.buf = e.appendKey(e.buf, part)
	}
	e.buf = append(e.buf, ']')
	if array {
		e.buf = append(e.buf, ']')
	}
	e.buf = append(e.buf, '\n')
	e.lastWasHeader = true
}

func (e *encoderState) writeIndent(indent int) {
	if !e.indentTables {
		return
	}
	for i := 0; i < indent; i++ {
		e.buf = append(e.buf, e.indentSymbol...)
	}
}

// writeComment emits the comment lines attached to an entry.
func (e *encoderState) writeComment(comment string, indent int) {
	if comment == "" {
		return
	}
	for _, line := range strings.Split(comment, "\n") {
		e.writeIndent(indent)
		e.buf = append(e.buf, "# "...)
		e.buf = append(e.buf, line...)
		e.buf = append(e.buf, '\n')
	}
}

// encodeKeyValue writes one `key = value` line of a table.
func (e *encoderState) encodeKeyValue(ent entry, commented bool, indent int) error {
	commented = commented || ent.options.commented

	e.writeComment(ent.options.comment, indent)

	e.writeIndent(indent)
	if commented {
		e.buf = append(e.buf, "# "...)
	}
	e.buf = e.appendKey(e.buf, ent.key)
	e.buf = append(e.buf, " = "...)

	var err error
	e.buf, err = e.appendValue(e.buf, ent.value, ent.options, indent)
	if err != nil {
		return err
	}
	e.buf = append(e.buf, '\n')
	e.lastWasHeader = false
	return nil
}

// collectEntries builds the ordered list of the entries of a table,
// applying tags and omission rules.
func (e *encoderState) collectEntries(v reflect.Value) ([]entry, error) {
	switch v.Kind() {
	case reflect.Map:
		return e.collectMapEntries(v)
	case reflect.Struct:
		var entries []entry
		err := e.collectStructEntries(&entries, v)
		if err != nil {
			return nil, err
		}
		return dedupEntries(entries), nil
	default:
		return nil, fmt.Errorf("toml: cannot encode a %s as a table", v.Type())
	}
}

func (e *encoderState) collectMapEntries(v reflect.Value) ([]entry, error) {
	entries := make([]entry, 0, v.Len())

	iter := v.MapRange()
	for iter.Next() {
		key, err := mapKeyString(iter.Key())
		if err != nil {
			return nil, err
		}
		value := iter.Value()
		if value.Kind() == reflect.Interface && value.IsNil() {
			// nil interface values are skipped
			continue
		}
		if value.Kind() == reflect.Ptr && value.IsNil() {
			// nil pointers in maps are encoded as their zero value
			value = reflect.New(value.Type().Elem()).Elem()
		}
		entries = append(entries, entry{key: key, value: value})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	return entries, nil
}

// mapKeyString converts a map key to its string representation.
func mapKeyString(k reflect.Value) (string, error) {
	kr, ok := resolve(k)
	if !ok {
		return "", fmt.Errorf("toml: cannot encode a nil map key")
	}
	if kr.Type().Implements(textMarshalerType) {
		b, err := kr.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return "", fmt.Errorf("toml: cannot marshal map key: %w", err)
		}
		return string(b), nil
	}
	if kr.CanAddr() && reflect.PtrTo(kr.Type()).Implements(textMarshalerType) {
		b, err := kr.Addr().Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return "", fmt.Errorf("toml: cannot marshal map key: %w", err)
		}
		return string(b), nil
	}
	if !kr.CanAddr() && reflect.PtrTo(kr.Type()).Implements(textMarshalerType) {
		tmp := reflect.New(kr.Type())
		tmp.Elem().Set(kr)
		b, err := tmp.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return "", fmt.Errorf("toml: cannot marshal map key: %w", err)
		}
		return string(b), nil
	}

	switch kr.Kind() {
	case reflect.String:
		return kr.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(kr.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(kr.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(kr.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(kr.Float(), 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("toml: cannot encode a map with key type %s", k.Type())
	}
}

// collectStructEntries appends the entries of a struct, flattening embedded
// structs in place.
func (e *encoderState) collectStructEntries(entries *[]entry, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := v.Field(i)

		tag, tagged := f.Tag.Lookup("toml")
		if tag == "-" {
			continue
		}

		name := f.Name
		var opts valueOptions
		if tagged {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, opt := range parts[1:] {
				switch opt {
				case "multiline":
					opts.multiline = true
				case "inline":
					opts.inline = true
				case "omitempty":
					opts.omitempty = true
				case "omitzero":
					opts.omitzero = true
				case "commented":
					opts.commented = true
				}
			}
		}
		// Standalone boolean tags.
		if f.Tag.Get("multiline") == "true" {
			opts.multiline = true
		}
		if f.Tag.Get("inline") == "true" {
			opts.inline = true
		}
		if f.Tag.Get("commented") == "true" {
			opts.commented = true
		}
		opts.comment = f.Tag.Get("comment")

		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && (!tagged || tagName(tag) == "") {
				sub, ok := resolve(fv)
				if !ok {
					// nil embedded pointer: skipped
					continue
				}
				err := e.collectStructEntries(entries, sub)
				if err != nil {
					return err
				}
				continue
			}
			if ft.Kind() == reflect.Interface && fv.IsNil() {
				continue
			}
			if f.PkgPath != "" {
				continue
			}
		} else if f.PkgPath != "" {
			// unexported
			continue
		}

		// nil values in struct fields are skipped
		if (fv.Kind() == reflect.Interface || fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Map) && fv.IsNil() {
			continue
		}

		if opts.omitempty && isEmptyValue(fv) {
			continue
		}
		if opts.omitzero && isZeroValue(fv) {
			continue
		}

		*entries = append(*entries, entry{key: name, value: fv, options: opts})
	}
	return nil
}

func tagName(tag string) string {
	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		return tag[:idx]
	}
	return tag
}

// dedupEntries removes the entries shadowed by another one with the same
// name, keeping the order of first appearance.
func dedupEntries(entries []entry) []entry {
	seen := make(map[string]int, len(entries))
	for _, ent := range entries {
		seen[ent.key]++
	}
	if len(seen) == len(entries) {
		return entries
	}
	out := entries[:0]
	for _, ent := range entries {
		if seen[ent.key] > 1 {
			// Multiple fields with the same name: this can only come from
			// embedding, where the shallowest field wins. Entries from the
			// outer struct are collected before the embedded ones, so the
			// first one wins.
			if seen[ent.key+"\x00done"] > 0 {
				continue
			}
			seen[ent.key+"\x00done"] = 1
		}
		out = append(out, ent)
	}
	return out
}

// isEmptyValue implements the omitempty rules.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Map, reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		return v.IsZero()
	}
	return false
}

// isZeroValue implements the omitzero rules: the type's own IsZero() when
// implemented, the reflect zero value otherwise.
func isZeroValue(v reflect.Value) bool {
	if v.Type().Implements(isZeroerType) {
		return v.Interface().(isZeroer).IsZero()
	}
	if v.CanAddr() && reflect.PtrTo(v.Type()).Implements(isZeroerType) {
		return v.Addr().Interface().(isZeroer).IsZero()
	}
	if !v.CanAddr() && reflect.PtrTo(v.Type()).Implements(isZeroerType) {
		tmp := reflect.New(v.Type())
		tmp.Elem().Set(v)
		return tmp.Interface().(isZeroer).IsZero()
	}
	return v.IsZero()
}

// appendKey emits a key, quoted only if necessary.
func (e *encoderState) appendKey(b []byte, key string) []byte {
	if isBareKey(key) {
		return append(b, key...)
	}
	return e.appendString(b, key)
}

func isBareKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range []byte(key) {
		if !isUnquotedKeyByte(c) {
			return false
		}
	}
	return true
}

func isUnquotedKeyByte(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

// appendValue emits a TOML value.
func (e *encoderState) appendValue(b []byte, v reflect.Value, opts valueOptions, indent int) ([]byte, error) {
	t := v.Type()

	// Special types take precedence over their kind.
	switch t {
	case timeType:
		return v.Interface().(time.Time).AppendFormat(b, "2006-01-02T15:04:05.999999999Z07:00"), nil
	case localDateType:
		return append(b, v.Interface().(LocalDate).String()...), nil
	case localTimeType:
		return append(b, v.Interface().(LocalTime).String()...), nil
	case localDateTimeType:
		return append(b, v.Interface().(LocalDateTime).String()...), nil
	case jsonNumberType:
		if e.marshalJSONNumbers {
			return appendJSONNumber(b, v.Interface().(json.Number))
		}
	}

	if t.Implements(textMarshalerType) && t.Kind() != reflect.String {
		return e.appendTextMarshaler(b, v.Interface().(encoding.TextMarshaler))
	}
	if reflect.PtrTo(t).Implements(textMarshalerType) {
		if v.CanAddr() {
			return e.appendTextMarshaler(b, v.Addr().Interface().(encoding.TextMarshaler))
		}
		tmp := reflect.New(t)
		tmp.Elem().Set(v)
		return e.appendTextMarshaler(b, tmp.Interface().(encoding.TextMarshaler))
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			// nil pointers are encoded as the zero value of their element
			// type.
			return e.appendValue(b, reflect.Zero(t.Elem()), opts, indent)
		}
		return e.appendValue(b, v.Elem(), opts, indent)
	case reflect.Interface:
		if v.IsNil() {
			return nil, fmt.Errorf("toml: cannot encode a nil interface")
		}
		return e.appendValue(b, v.Elem(), opts, indent)
	case reflect.String:
		if opts.multiline {
			return e.appendMultilineString(b, v.String()), nil
		}
		return e.appendString(b, v.String()), nil
	case reflect.Bool:
		if v.Bool() {
			return append(b, "true"...), nil
		}
		return append(b, "false"...), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(b, v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := v.Uint()
		if u > math.MaxInt64 {
			return nil, fmt.Errorf("toml: cannot encode an unsigned integer above math.MaxInt64: %d", u)
		}
		return strconv.AppendUint(b, u, 10), nil
	case reflect.Float32:
		return appendFloat(b, v.Float(), 32), nil
	case reflect.Float64:
		return appendFloat(b, v.Float(), 64), nil
	case reflect.Slice, reflect.Array:
		return e.appendArray(b, v, opts, indent)
	case reflect.Map:
		return e.appendInlineTable(b, v, indent)
	case reflect.Struct:
		return e.appendInlineTable(b, v, indent)
	default:
		return nil, fmt.Errorf("toml: cannot encode value of type %s", v.Type())
	}
}

var jsonNumberType = reflect.TypeOf(json.Number(""))

func appendJSONNumber(b []byte, n json.Number) ([]byte, error) {
	if n == "" {
		return append(b, '0'), nil
	}
	if i, err := n.Int64(); err == nil {
		return strconv.AppendInt(b, i, 10), nil
	}
	f, err := n.Float64()
	if err != nil {
		return nil, fmt.Errorf("toml: cannot encode json.Number %q: %w", string(n), err)
	}
	return appendFloat(b, f, 64), nil
}

func appendFloat(b []byte, f float64, bitSize int) []byte {
	switch {
	case math.IsNaN(f):
		return append(b, "nan"...)
	case math.IsInf(f, 1):
		return append(b, "inf"...)
	case math.IsInf(f, -1):
		return append(b, "-inf"...)
	}
	start := len(b)
	b = strconv.AppendFloat(b, f, 'f', -1, bitSize)
	// TOML floats must have a fractional part or an exponent.
	if bytes.IndexAny(b[start:], ".eE") < 0 {
		b = append(b, ".0"...)
	}
	return b
}

func (e *encoderState) appendTextMarshaler(b []byte, m encoding.TextMarshaler) ([]byte, error) {
	text, err := m.MarshalText()
	if err != nil {
		return nil, fmt.Errorf("toml: error calling MarshalText: %w", err)
	}
	return e.appendString(b, string(text)), nil
}

// appendArray encodes a slice or array value.
func (e *encoderState) appendArray(b []byte, v reflect.Value, opts valueOptions, indent int) ([]byte, error) {
	multiline := opts.multiline || e.arraysMultiline

	b = append(b, '[')
	if multiline && v.Len() > 0 {
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b = append(b, ',')
			}
			b = append(b, '\n')
			for j := 0; j <= indent; j++ {
				b = append(b, e.indentSymbol...)
			}
			var err error
			b, err = e.appendValue(b, v.Index(i), valueOptions{}, indent+1)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, '\n')
		for j := 0; j < indent; j++ {
			b = append(b, e.indentSymbol...)
		}
	} else {
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b = append(b, ", "...)
			}
			var err error
			b, err = e.appendValue(b, v.Index(i), valueOptions{}, indent)
			if err != nil {
				return nil, err
			}
		}
	}
	return append(b, ']'), nil
}

// appendInlineTable encodes a map or a struct as an inline table.
func (e *encoderState) appendInlineTable(b []byte, v reflect.Value, indent int) ([]byte, error) {
	entries, err := e.collectEntries(v)
	if err != nil {
		return nil, err
	}

	b = append(b, '{')
	for i, ent := range entries {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = e.appendKey(b, ent.key)
		b = append(b, " = "...)
		// multiline strings are not allowed inside inline tables: they
		// would break the single-line requirement.
		opts := ent.options
		opts.multiline = false
		b, err = e.appendValue(b, ent.value, opts, indent)
		if err != nil {
			return nil, err
		}
	}
	return append(b, '}'), nil
}

// appendString encodes a string, using a literal string when possible and a
// basic string otherwise.
func (e *encoderState) appendString(b []byte, s string) []byte {
	if canBeLiteral(s) {
		b = append(b, '\'')
		b = append(b, s...)
		return append(b, '\'')
	}
	return appendBasicString(b, s)
}

// canBeLiteral returns true when the string can be represented as a TOML
// literal string: no control characters, no single quote, no newline.
func canBeLiteral(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' || c == 0x7f || c < 0x20 {
			return false
		}
	}
	return utf8.ValidString(s)
}

// appendBasicString encodes a string as a TOML basic (double-quoted) string.
func appendBasicString(b []byte, s string) []byte {
	b = append(b, '"')
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"':
			b = append(b, '\\', '"')
			i++
		case c == '\\':
			b = append(b, '\\', '\\')
			i++
		case c == '\b':
			b = append(b, '\\', 'b')
			i++
		case c == '\f':
			b = append(b, '\\', 'f')
			i++
		case c == '\n':
			b = append(b, '\\', 'n')
			i++
		case c == '\r':
			b = append(b, '\\', 'r')
			i++
		case c == '\t':
			b = append(b, '\\', 't')
			i++
		case c < 0x20 || c == 0x7f:
			b = append(b, fmt.Sprintf("\\u%04X", c)...)
			i++
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				// Replace invalid bytes by the replacement character.
				b = append(b, fmt.Sprintf("\\u%04X", c)...)
				i++
				continue
			}
			b = append(b, s[i:i+size]...)
			i += size
		}
	}
	return append(b, '"')
}

// appendMultilineString encodes a string as a TOML multi-line basic string.
func appendMultilineString(b []byte, s string) []byte {
	b = append(b, `"""`...)
	b = append(b, '\n')
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"':
			// Runs of three or more quotes must be escaped.
			j := i
			for j < len(s) && s[j] == '"' {
				j++
			}
			if j-i >= 3 {
				for ; i < j; i++ {
					b = append(b, '\\', '"')
				}
			} else {
				b = append(b, s[i:j]...)
				i = j
			}
		case c == '\\':
			b = append(b, '\\', '\\')
			i++
		case c == '\n':
			b = append(b, '\n')
			i++
		case c == '\b':
			b = append(b, '\\', 'b')
			i++
		case c == '\f':
			b = append(b, '\\', 'f')
			i++
		case c == '\r':
			b = append(b, '\\', 'r')
			i++
		case c == '\t':
			b = append(b, '\t')
			i++
		case c < 0x20 || c == 0x7f:
			b = append(b, fmt.Sprintf("\\u%04X", c)...)
			i++
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				b = append(b, fmt.Sprintf("\\u%04X", c)...)
				i++
				continue
			}
			b = append(b, s[i:i+size]...)
			i += size
		}
	}
	return append(b, `"""`...)
}

func (e *encoderState) appendMultilineString(b []byte, s string) []byte {
	return appendMultilineString(b, s)
}
