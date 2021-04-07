package toml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
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
	w io.Writer
}

type encoderCtx struct {
	// Current top-level key.
	parentKey []string

	// Key that should be used for a KV.
	key string
	// Extra flag to account for the empty string
	hasKey bool

	// Set to true to indicate that the encoder is inside a KV, so that all
	// tables need to be inlined.
	insideKv bool

	// Set to true to skip the first table header in an array table.
	skipTableHeader bool
}

func (ctx *encoderCtx) shiftKey() {
	if ctx.hasKey {
		ctx.parentKey = append(ctx.parentKey, ctx.key)
		ctx.clearKey()
	}
}

func (ctx *encoderCtx) setKey(k string) {
	ctx.key = k
	ctx.hasKey = true
}

func (ctx *encoderCtx) clearKey() {
	ctx.key = ""
	ctx.hasKey = false
}

// NewEncoder returns a new Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Encode writes a TOML representation of v to the stream.
//
// If v cannot be represented to TOML it returns an error.
//
// Encoding rules:
//
// 1. A top level slice containing only maps or structs is encoded as [[table
// array]].
//
// 2. All slices not matching rule 1 are encoded as [array]. As a result, any
// map or struct they contain is encoded as an {inline table}.
//
// 3. Nil interfaces and nil pointers are not supported.
//
// 4. Keys in key-values always have one part.
func (enc *Encoder) Encode(v interface{}) error {
	var b []byte
	var ctx encoderCtx
	b, err := enc.encode(b, ctx, reflect.ValueOf(v))
	if err != nil {
		return err
	}
	_, err = enc.w.Write(b)
	return err
}

func (enc *Encoder) encode(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	// containers
	switch v.Kind() {
	case reflect.Map:
		return enc.encodeMap(b, ctx, v)
	case reflect.Struct:
		return enc.encodeStruct(b, ctx, v)
	case reflect.Slice:
		return enc.encodeSlice(b, ctx, v)
	case reflect.Interface:
		if v.IsNil() {
			return nil, errNilInterface
		}
		return enc.encode(b, ctx, v.Elem())
	case reflect.Ptr:
		if v.IsNil() {
			return nil, errNilPointer
		}
		return enc.encode(b, ctx, v.Elem())
	}

	// values
	var err error
	switch v.Kind() {
	case reflect.String:
		b, err = enc.encodeString(b, v.String())
	case reflect.Float32:
		b = strconv.AppendFloat(b, v.Float(), 'e', -1, 32)
	case reflect.Float64:
		b = strconv.AppendFloat(b, v.Float(), 'e', -1, 64)
	case reflect.Bool:
		if v.Bool() {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		b = strconv.AppendUint(b, v.Uint(), 10)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		b = strconv.AppendInt(b, v.Int(), 10)
	default:
		err = fmt.Errorf("unsupported encode value kind: %s", v.Kind())
	}
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (enc *Encoder) encodeKv(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	var err error

	if !ctx.hasKey {
		panic("caller of encodeKv should have set the key in the context")
	}

	b, err = enc.encodeKey(b, ctx.key)
	if err != nil {
		return nil, err
	}

	b = append(b, " = "...)

	// create a copy of the context because the value of a KV shouldn't
	// modify the global context.
	subctx := ctx
	subctx.insideKv = true
	subctx.shiftKey()

	b, err = enc.encode(b, subctx, v)
	if err != nil {
		return nil, err
	}

	return b, nil
}

const literalQuote = '\''

func (enc *Encoder) encodeString(b []byte, v string) ([]byte, error) {
	if strings.ContainsRune(v, literalQuote) {
		panic("encoding strings with ' is not supported")
	} else {
		b = enc.encodeLiteralString(b, v)
	}
	return b, nil
}

// caller should have checked that the string does not contain new lines or '
func (enc *Encoder) encodeLiteralString(b []byte, v string) []byte {
	b = append(b, literalQuote)
	b = append(b, v...)
	b = append(b, literalQuote)
	return b
}

// called should have checked that the string is in A-Z / a-z / 0-9 / - / _
func (enc *Encoder) encodeUnquotedKey(b []byte, v string) []byte {
	return append(b, v...)
}

func (enc *Encoder) encodeTableHeader(b []byte, key []string) ([]byte, error) {
	if len(key) == 0 {
		return b, nil
	}

	b = append(b, '[')

	var err error
	b, err = enc.encodeKey(b, key[0])
	if err != nil {
		return nil, err
	}

	for _, k := range key[1:] {
		b = append(b, '.')
		b, err = enc.encodeKey(b, k)
		if err != nil {
			return nil, err
		}
	}

	b = append(b, "]\n"...)

	return b, nil
}

func (enc *Encoder) encodeKey(b []byte, k string) ([]byte, error) {
	needsQuotation := false
	cannotUseLiteral := false

	for _, c := range k {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		if c == '\n' {
			return nil, fmt.Errorf("TOML does not support multiline keys")
		}
		if c == literalQuote {
			cannotUseLiteral = true
		}
		needsQuotation = true
	}

	if cannotUseLiteral {
		// TODO: encode key using quotes and escaping
		panic("not implemented")
	} else if needsQuotation {
		b = enc.encodeLiteralString(b, k)
	} else {
		b = enc.encodeUnquotedKey(b, k)
	}

	return b, nil
}

func (enc *Encoder) encodeMap(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	if ctx.insideKv {
		// TODO
		panic("literal tables not supported yet")
	}

	type pair struct {
		k reflect.Value
		v reflect.Value
	}

	if v.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("type '%s' not supported as map key", v.Type().Key().Kind())
	}

	nonTablePairs := []pair{}
	tablePairs := []pair{}

	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		table, err := willConvertToTableOrArrayTable(v)
		if err != nil {
			return nil, err
		}

		kv := pair{
			k: k,
			v: v,
		}

		if table {
			tablePairs = append(tablePairs, kv)
		} else {
			nonTablePairs = append(nonTablePairs, kv)
		}
	}

	var err error

	ctx.shiftKey()

	if len(nonTablePairs) > 0 && !ctx.skipTableHeader {
		b, err = enc.encodeTableHeader(b, ctx.parentKey)
		if err != nil {
			return nil, err
		}
	}
	ctx.skipTableHeader = false

	for _, kv := range nonTablePairs {
		ctx.setKey(kv.k.String())
		b, err = enc.encodeKv(b, ctx, kv.v)
		if err != nil {
			return nil, err
		}

		b = append(b, '\n')
	}

	for _, kv := range tablePairs {
		ctx.setKey(kv.k.String())
		b, err = enc.encode(b, ctx, kv.v)
		if err != nil {
			return nil, err
		}

		b = append(b, '\n')
	}

	return b, nil
}

func (enc *Encoder) encodeStruct(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	t := v.Type()

	nonTableFields := []int{}
	tableFields := []int{}

	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		table, err := willConvertToTableOrArrayTable(f)
		if err != nil {
			return nil, err
		}
		if table {
			tableFields = append(tableFields, i)
		} else {
			nonTableFields = append(nonTableFields, i)
		}
	}

	var err error

	ctx.shiftKey()

	if len(nonTableFields) > 0 && !ctx.skipTableHeader {
		b, err = enc.encodeTableHeader(b, ctx.parentKey)
		if err != nil {
			return nil, err
		}
	}
	ctx.skipTableHeader = false

	for _, i := range nonTableFields {
		k := t.Field(i).Name
		f := v.Field(i)

		ctx.setKey(k)
		b, err = enc.encodeKv(b, ctx, f)
		if err != nil {
			return nil, err
		}
		b = append(b, '\n')
	}

	for _, i := range tableFields {
		k := t.Field(i).Name
		f := v.Field(i)

		ctx.setKey(k)
		b, err = enc.encode(b, ctx, f)
		if err != nil {
			return nil, err
		}
		b = append(b, '\n')
	}

	return b, nil
}

var errNilInterface = errors.New("nil interface not supported")
var errNilPointer = errors.New("nil pointer not supported")

func willConvertToTable(v reflect.Value) (bool, error) {
	t := v.Type()
	switch t.Kind() {
	case reflect.Map, reflect.Struct:
		return true, nil
	case reflect.Interface:
		if v.IsNil() {
			return false, errNilInterface
		}
		return willConvertToTable(v.Elem())
	case reflect.Ptr:
		if v.IsNil() {
			return false, errNilPointer
		}
		return willConvertToTable(v.Elem())
	default:
		return false, nil
	}
}

func willConvertToTableOrArrayTable(v reflect.Value) (bool, error) {
	t := v.Type()
	if t.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			t, err := willConvertToTable(v.Index(i))
			if err != nil {
				return false, err
			}
			if !t {
				return false, nil
			}
		}
		return true, nil
	}

	return willConvertToTable(v)
}

func (enc *Encoder) encodeSlice(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	if v.Len() == 0 {
		b = append(b, "[]"...)
		return b, nil
	}

	allTables, err := willConvertToTableOrArrayTable(v)
	if err != nil {
		return nil, err
	}

	if allTables {
		return enc.encodeSliceAsArrayTable(b, ctx, v)
	}

	return enc.encodeSliceAsArray(b, ctx, v)
}

// caller should have checked that v is a slice that only contains values that
// encode into tables.
func (enc *Encoder) encodeSliceAsArrayTable(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	if v.Len() == 0 {
		return b, nil
	}

	ctx.shiftKey()

	var err error
	scratch := make([]byte, 0, 64)
	scratch = append(scratch, "[["...)
	for i, k := range ctx.parentKey {
		if i > 0 {
			scratch = append(scratch, '.')
		}
		scratch, err = enc.encodeKey(scratch, k)
		if err != nil {
			return nil, err
		}
	}
	scratch = append(scratch, "]]\n"...)
	ctx.skipTableHeader = true

	for i := 0; i < v.Len(); i++ {
		b = append(b, scratch...)
		b, err = enc.encode(b, ctx, v.Index(i))
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (enc *Encoder) encodeSliceAsArray(b []byte, ctx encoderCtx, v reflect.Value) ([]byte, error) {
	b = append(b, '[')

	var err error
	first := true
	for i := 0; i < v.Len(); i++ {
		if !first {
			b = append(b, ", "...)
		}
		first = false

		b, err = enc.encode(b, ctx, v.Index(i))
		if err != nil {
			return nil, err
		}
	}

	b = append(b, ']')
	return b, nil
}
