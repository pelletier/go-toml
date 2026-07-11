package toml

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

// tmText has a pointer-receiver TextMarshaler; tmRawVal/tmRawPtr implement
// unstable.Marshaler with value and pointer receivers.
type tmText struct{ s string }

func (t *tmText) MarshalText() ([]byte, error) { return []byte(t.s), nil }

type tmRawVal struct{}

func (tmRawVal) MarshalTOML() ([]byte, error) { return []byte("1"), nil }

type tmRawPtr struct{}

func (*tmRawPtr) MarshalTOML() ([]byte, error) { return []byte("2"), nil }

// TestEncPropsReceiverVariants covers the pointer-receiver classification
// branches of the encoder's type properties.
func TestEncPropsReceiverVariants(t *testing.T) {
	out, err := Marshal(map[string]tmText{"a": {s: "x"}})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), "'x'"), "got %q", out)

	var buf strings.Builder
	enc := NewEncoder(&buf)
	enc.EnableMarshalerInterface()
	assert.NoError(t, enc.Encode(map[string]interface{}{
		"v": tmRawVal{},
		"p": &tmRawPtr{},
	}))
	back := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(buf.String()), &back))
	assert.Equal(t, interface{}(int64(1)), back["v"])
	assert.Equal(t, interface{}(int64(2)), back["p"])
}

// TestMarshalLargeTypedMap covers the value-slab path of map entry
// collection (maps with at least eight entries).
func TestMarshalLargeTypedMap(t *testing.T) {
	m := map[string]int64{}
	for i := 0; i < 12; i++ {
		m[fmt.Sprintf("key%02d", i)] = int64(i)
	}
	out, err := Marshal(m)
	assert.NoError(t, err)
	back := map[string]int64{}
	assert.NoError(t, Unmarshal(out, &back))
	assert.Equal(t, m, back)
}

// TestMarshalGenericTree covers the reflection-free encoding of generic
// map[string]interface{} documents: every native scalar type, nested arrays
// and tables, multiline arrays, and fallback to the reflection encoder for
// non-generic values held in interfaces.
func TestMarshalGenericTree(t *testing.T) {
	doc := map[string]interface{}{
		"str":      "hello",
		"lit":      "with \"quotes\" and\ttab",
		"bool_t":   true,
		"bool_f":   false,
		"int64":    int64(42),
		"int":      7,
		"float":    1.5,
		"time":     time.Date(2021, 3, 30, 11, 21, 0, 0, time.UTC),
		"date":     LocalDate{2021, 3, 30},
		"ltime":    LocalTime{11, 21, 0, 0, 0},
		"ldt":      LocalDateTime{LocalDate{2021, 3, 30}, LocalTime{11, 21, 0, 0, 0}},
		"arr":      []interface{}{int64(1), "two", 3.0},
		"nested":   []interface{}{[]interface{}{int64(1)}, []interface{}{int64(2)}},
		"typed":    uint16(9),
		"tbl":      map[string]interface{}{"a": int64(1), "sub": map[string]interface{}{"b": "c"}},
		"arrtbl":   []interface{}{map[string]interface{}{"x": int64(1)}, map[string]interface{}{"x": int64(2)}},
		"mixedarr": []interface{}{map[string]interface{}{"x": int64(1)}, int64(2)},
	}

	out, err := Marshal(doc)
	assert.NoError(t, err)

	back := map[string]interface{}{}
	assert.NoError(t, Unmarshal(out, &back))
	assert.Equal(t, interface{}("hello"), back["str"])
	assert.Equal(t, interface{}(int64(42)), back["int64"])
	assert.Equal(t, interface{}(int64(7)), back["int"])
	assert.Equal(t, interface{}(int64(9)), back["typed"])
	assert.Equal(t, interface{}(1.5), back["float"])
	assert.Equal(t, 3, len(back["arr"].([]interface{})))
	assert.Equal(t, 2, len(back["arrtbl"].([]interface{})))
	assert.Equal(t, 2, len(back["mixedarr"].([]interface{})))
	assert.Equal(t, interface{}("c"),
		back["tbl"].(map[string]interface{})["sub"].(map[string]interface{})["b"])

	t.Run("multiline arrays and inline tables", func(t *testing.T) {
		var buf strings.Builder
		enc := NewEncoder(&buf)
		enc.SetArraysMultiline(true)
		enc.SetTablesInline(true)
		assert.NoError(t, enc.Encode(doc))
		back := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(buf.String()), &back))
		assert.Equal(t, 3, len(back["arr"].([]interface{})))
	})

	t.Run("nil in inline table errors", func(t *testing.T) {
		_, err := Marshal(map[string]interface{}{
			"a": []interface{}{nil},
		})
		assert.Error(t, err)
	})

	t.Run("indented tables", func(t *testing.T) {
		var buf strings.Builder
		enc := NewEncoder(&buf)
		enc.SetIndentTables(true)
		assert.NoError(t, enc.Encode(doc))
		back := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(buf.String()), &back))
		assert.Equal(t, interface{}(int64(1)),
			back["tbl"].(map[string]interface{})["a"])
	})
}

// TestMarshalerInterfaceGenericMap covers unstable.Marshaler values held in
// generic maps: the resolver materializes the interface value to classify it.
func TestMarshalerInterfaceGenericMap(t *testing.T) {
	doc := map[string]interface{}{
		"raw":   unstable.RawMessage("1"),
		"table": unstable.RawMessage("a = 1"),
		"plain": "hello",
		"num":   int64(3),
	}
	var buf strings.Builder
	enc := NewEncoder(&buf)
	enc.EnableMarshalerInterface()
	assert.NoError(t, enc.Encode(doc))

	back := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(buf.String()), &back))
	assert.Equal(t, interface{}(int64(1)), back["raw"])
	assert.Equal(t, interface{}("hello"), back["plain"])
	assert.Equal(t, interface{}(int64(1)), back["table"].(map[string]interface{})["a"])
}

type tmRawErr struct{}

func (tmRawErr) MarshalTOML() ([]byte, error) { return nil, errors.New("nope") }

// TestMarshalerInterfaceMoreEdges covers the root single-value error, a
// failing MarshalTOML inside an array classification, and the fused scalar
// kind guard.
func TestMarshalerInterfaceMoreEdges(t *testing.T) {
	var buf strings.Builder
	enc := NewEncoder(&buf)
	enc.EnableMarshalerInterface()
	err := enc.Encode(unstable.RawMessage("42"))
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "document root"), "got %v", err)

	var buf2 strings.Builder
	enc2 := NewEncoder(&buf2)
	enc2.EnableMarshalerInterface()
	assert.Error(t, enc2.Encode(map[string]interface{}{
		"arr": []interface{}{tmRawErr{}, tmRawErr{}},
	}))

	d := getDecoder(false, false)
	defer putDecoder(d)
	_, err = d.fusedScalar(unstable.Comment, []byte("#"))
	assert.Error(t, err)
}

// TestMarshalGenericEdgeCoverage sweeps branches of the generic encoder that
// the reflection tests no longer reach: classification of exotic interface
// values, error propagation from nested tables, and nil-value skipping.
func TestMarshalGenericEdgeCoverage(t *testing.T) {
	t.Run("empty generic array", func(t *testing.T) {
		out, err := Marshal(map[string]interface{}{"a": []interface{}{}})
		assert.NoError(t, err)
		assert.Equal(t, "a = []\n", string(out))
	})

	t.Run("mixed table-like array elements", func(t *testing.T) {
		out, err := Marshal(map[string]interface{}{"a": []interface{}{
			map[string]interface{}{"x": int64(1)},
			map[string]int64{"y": 2},
		}})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "[[a]]"), "got %q", out)
	})

	t.Run("typed containers in interface", func(t *testing.T) {
		out, err := Marshal(map[string]interface{}{
			"arr": []map[string]int64{{"x": 1}},
			"tbl": map[string]int64{"y": 2},
		})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "[[arr]]"), "got %q", out)
		assert.True(t, strings.Contains(string(out), "[tbl]"), "got %q", out)
	})

	t.Run("bad key type in nested table", func(t *testing.T) {
		_, err := Marshal(map[string]interface{}{"t": map[complex128]int64{1: 2}})
		assert.Error(t, err)
	})

	t.Run("bad key type in array table element", func(t *testing.T) {
		_, err := Marshal(map[string]interface{}{"arr": []map[complex128]int64{{1: 2}}})
		assert.Error(t, err)
	})

	t.Run("nil values skipped in nested and typed maps", func(t *testing.T) {
		out, err := Marshal(map[string]interface{}{"t": map[string]interface{}{"a": nil, "b": int64(1)}})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "b = 1"), "got %q", out)
		assert.True(t, !strings.Contains(string(out), "a ="), "got %q", out)

		out, err = Marshal(map[string]error{"a": nil})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "[a]") == false, "got %q", out)
	})

	t.Run("named string key type", func(t *testing.T) {
		type K string
		out, err := Marshal(map[K]int64{"k": 1})
		assert.NoError(t, err)
		assert.Equal(t, "k = 1\n", string(out))
	})

	t.Run("multiline array element error", func(t *testing.T) {
		var buf strings.Builder
		enc := NewEncoder(&buf)
		enc.SetArraysMultiline(true)
		assert.Error(t, enc.Encode(map[string]interface{}{"a": []interface{}{nil}}))
	})

	t.Run("local date and datetime struct fields", func(t *testing.T) {
		var s struct {
			D  LocalDate     `toml:"d"`
			DT LocalDateTime `toml:"dt"`
		}
		s.D = LocalDate{Year: 2021, Month: 3, Day: 30}
		s.DT = LocalDateTime{LocalDate: s.D, LocalTime: LocalTime{Hour: 11}}
		out, err := Marshal(s)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "d = 2021-03-30"), "got %q", out)
		assert.True(t, strings.Contains(string(out), "dt = 2021-03-30T11:00:00"), "got %q", out)
	})
}

// flipFlopMarshaler returns a table body on its first call and empty bytes on
// the second, exercising the emit-time revalidation of table-shaped entries.
type flipFlopMarshaler struct{ n *int }

func (f flipFlopMarshaler) MarshalTOML() ([]byte, error) {
	*f.n++
	if *f.n == 1 {
		return []byte("x = 1"), nil
	}
	return nil, nil
}

// TestMarshalerInterfaceEmptyReemit covers a Marshaler that returns a table
// body during classification and empty bytes at emit time: the header is
// still written, with an empty body.
func TestMarshalerInterfaceEmptyReemit(t *testing.T) {
	var buf strings.Builder
	n := 0
	enc := NewEncoder(&buf)
	enc.EnableMarshalerInterface()
	assert.NoError(t, enc.Encode(map[string]interface{}{"a": flipFlopMarshaler{n: &n}}))
	assert.True(t, strings.Contains(buf.String(), "[a]"), "got %q", buf.String())
}

// TestMarshalerInterfaceOmitSuperTablesError covers MarshalTOML errors
// surfaced through the early entry resolution of SetOmitEmptySuperTables.
func TestMarshalerInterfaceOmitSuperTablesError(t *testing.T) {
	var buf strings.Builder
	enc := NewEncoder(&buf)
	enc.EnableMarshalerInterface()
	enc.SetOmitEmptySuperTables(true)
	assert.Error(t, enc.Encode(map[string]interface{}{"a": map[string]interface{}{"b": tmRawErr{}}}))
}
