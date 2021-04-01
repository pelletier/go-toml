package toml

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
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
	w   io.Writer
	ctx encCtx
}

type encCtx struct {
	key     []string
	flushed bool
}

func (enc *Encoder) push(k string) {
	enc.ctx.key = append(enc.ctx.key, k)
	enc.ctx.flushed = false
}

func (enc *Encoder) pop() {
	enc.ctx.key = enc.ctx.key[:len(enc.ctx.key)-1]
}

func (enc *Encoder) top() string {
	return enc.ctx.key[len(enc.ctx.key)-1]
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
func (enc *Encoder) Encode(v interface{}) error {
	var b []byte
	b, err := enc.encode(b, reflect.ValueOf(v))
	if err != nil {
		return err
	}
	_, err = enc.w.Write(b)
	return err
}

func (enc *Encoder) encode(b []byte, v reflect.Value) ([]byte, error) {
	// containers
	switch v.Kind() {
	case reflect.Map:
		return enc.encodeMap(b, v)
	case reflect.Slice:
		return enc.encodeSlice(b, v)
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return b, nil
		}
		return enc.encode(b, v.Elem())
	}

	// values
	var err error
	switch v.Kind() {
	case reflect.String:
		b, err = enc.encodeString(b, v.String())
	default:
		err = fmt.Errorf("unsupported encode value kind: %s", v.Kind())
	}
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (enc *Encoder) encodeKv(b []byte, v reflect.Value) ([]byte, error) {
	var err error
	if enc.hasContext() {
		b, err = enc.encodeTableHeaderFromContext(b)
		if err != nil {
			return nil, err
		}
	}

	b, err = enc.encodeKey(b, enc.top())
	if err != nil {
		return nil, err
	}

	b = append(b, " = "...)

	return enc.encode(b, v)
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

func (enc *Encoder) hasContext() bool {
	return len(enc.ctx.key) > 1
}

func (enc *Encoder) encodeTableHeaderFromContext(b []byte) ([]byte, error) {
	b = append(b, '[')

	var err error
	b, err = enc.encodeKey(b, enc.ctx.key[0])
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(enc.ctx.key)-1; i++ {
		b = append(b, '.')
		b, err = enc.encodeKey(b, enc.ctx.key[i])
		if err != nil {
			return nil, err
		}
	}

	b = append(b, "]\n"...)
	enc.ctx.flushed = true
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

func (enc *Encoder) encodeMap(b []byte, v reflect.Value) ([]byte, error) {
	var err error
	if v.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("type '%s' not supported as map key", v.Type().Key().Kind())
	}

	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		k := key.String()
		v := iter.Value()

		enc.push(k)

		if willConvertToTable(v.Type()) {
			b, err = enc.encode(b, v)
		} else {
			b, err = enc.encodeKv(b, v)
		}
		if err != nil {
			return nil, err
		}

		enc.pop()
		b = append(b, '\n')
	}

	return b, nil
}

func willConvertToTable(v reflect.Value) bool {
	t := v.Type()
	switch t.Kind() {
	case reflect.Map, reflect.Struct:
		return true
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return false
		}
		return willConvertToTable(v.Elem())
	default:
		return false
	}
}

func (enc *Encoder) encodeSlice(b []byte, v reflect.Value) ([]byte, error) {
	arrayTable := willConvertToTable(v.Type().Elem())

	if arrayTable {
		return enc.encodeSliceAsArrayTable(b, v)
	}

	return enc.encodeSliceAsArray(b, v)
}

func (enc *Encoder) encodeSliceAsArrayTable(b []byte, v reflect.Value) ([]byte, error) {
	panic("TODO")
}

func (enc *Encoder) encodeSliceAsArray(b []byte, v reflect.Value) ([]byte, error) {
	b = append(b, '[')

	var err error
	first := true
	for i := 0; i < v.Len(); i++ {
		if !first {
			b = append(b, ", "...)
		}
		first = false

		b, err = enc.encode(b, v.Index(i))
		if err != nil {
			return nil, err
		}
	}

	b = append(b, ']')
	return b, nil
}
