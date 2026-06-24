package toml_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

// encodeRaw marshals v with the unstable.Marshaler interface enabled.
func encodeRaw(tb testing.TB, v interface{}) (string, error) {
	tb.Helper()
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).EnableMarshalerInterface().Encode(v)
	return buf.String(), err
}

// rawValueConfig and rawTableConfig are tiny reusable targets.
type rawValueConfig struct {
	N unstable.RawMessage `toml:"n"`
}

type rawTableConfig struct {
	Plugin unstable.RawMessage `toml:"plugin"`
}

// TestMarshalerInterfaceDisabled checks the default Encoder is unchanged: a
// RawMessage is a []byte and is emitted as an array of integers.
func TestMarshalerInterfaceDisabled(t *testing.T) {
	b, err := toml.Marshal(rawValueConfig{N: unstable.RawMessage("42")})
	assert.NoError(t, err)
	assert.Equal(t, "n = [52, 50]\n", string(b))

	// Even on an Encoder, without the toggle the behavior is the byte array.
	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(rawValueConfig{N: unstable.RawMessage("42")})
	assert.NoError(t, err)
	assert.Equal(t, "n = [52, 50]\n", buf.String())
}

// TestMarshalerInterfaceValueShapes checks single-value RawMessages emit inline.
func TestMarshalerInterfaceValueShapes(t *testing.T) {
	examples := []struct {
		desc     string
		raw      string
		expected string
	}{
		{"integer", "42", "n = 42\n"},
		{"negative integer", "-17", "n = -17\n"},
		{"float", "3.14", "n = 3.14\n"},
		{"bool", "true", "n = true\n"},
		{"basic string", `"hello"`, "n = \"hello\"\n"},
		{"literal string", `'lit'`, "n = 'lit'\n"},
		{"hex integer kept verbatim", "0xDEADBEEF", "n = 0xDEADBEEF\n"},
		{"array", "[1, 2, 3]", "n = [1, 2, 3]\n"},
		{"inline table", `{a = 1, b = "x"}`, "n = {a = 1, b = \"x\"}\n"},
		{"datetime", "1979-05-27T07:32:00Z", "n = 1979-05-27T07:32:00Z\n"},
		{"surrounding whitespace trimmed", "  42  ", "n = 42\n"},
		{"trailing newline trimmed", "42\n", "n = 42\n"},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			out, err := encodeRaw(t, rawValueConfig{N: unstable.RawMessage(e.raw)})
			assert.NoError(t, err)
			assert.Equal(t, e.expected, out)
		})
	}
}

// TestMarshalerInterfaceSingleTable checks a key-value body emits as a table.
func TestMarshalerInterfaceSingleTable(t *testing.T) {
	out, err := encodeRaw(t, rawTableConfig{
		Plugin: unstable.RawMessage("name = \"example\"\nversion = \"1.0\"\n"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "[plugin]\nname = \"example\"\nversion = \"1.0\"\n", out)
}

// TestMarshalerInterfaceDottedKeysPreserved checks dotted keys and odd
// formatting inside a table body are spliced verbatim.
func TestMarshalerInterfaceDottedKeysPreserved(t *testing.T) {
	body := "sub.key   =   'value'\nhex = 0xFF\n"
	out, err := encodeRaw(t, rawTableConfig{Plugin: unstable.RawMessage(body)})
	assert.NoError(t, err)
	assert.Equal(t, "[plugin]\nsub.key   =   'value'\nhex = 0xFF\n", out)
}

// TestMarshalerInterfaceArrayOfTables checks []RawMessage of table bodies emits
// as an array of tables.
func TestMarshalerInterfaceArrayOfTables(t *testing.T) {
	type config struct {
		Item []unstable.RawMessage `toml:"item"`
	}
	out, err := encodeRaw(t, config{Item: []unstable.RawMessage{
		unstable.RawMessage("a = 1\n"),
		unstable.RawMessage("a = 2\nb = 3\n"),
	}})
	assert.NoError(t, err)
	expected := "[[item]]\na = 1\n\n[[item]]\na = 2\nb = 3\n"
	assert.Equal(t, expected, out)
}

// TestMarshalerInterfaceArrayOfValues checks a []RawMessage of single values
// emits as a plain inline array, not as an array of tables.
func TestMarshalerInterfaceArrayOfValues(t *testing.T) {
	type config struct {
		Item []unstable.RawMessage `toml:"item"`
	}
	out, err := encodeRaw(t, config{Item: []unstable.RawMessage{
		unstable.RawMessage("1"),
		unstable.RawMessage("2"),
	}})
	assert.NoError(t, err)
	assert.Equal(t, "item = [1, 2]\n", out)
}

// TestMarshalerInterfaceNestedField checks a RawMessage under a sub-table emits
// under the right key path.
func TestMarshalerInterfaceNestedField(t *testing.T) {
	type inner struct {
		Plugin unstable.RawMessage `toml:"plugin"`
	}
	type config struct {
		Outer inner `toml:"outer"`
	}
	out, err := encodeRaw(t, config{Outer: inner{
		Plugin: unstable.RawMessage("a = 1\n"),
	}})
	assert.NoError(t, err)
	assert.Equal(t, "[outer]\n[outer.plugin]\na = 1\n", out)
}

// TestMarshalerInterfacePointerField checks *RawMessage works and nil is
// skipped.
func TestMarshalerInterfacePointerField(t *testing.T) {
	type config struct {
		A *unstable.RawMessage `toml:"a"`
		B *unstable.RawMessage `toml:"b"`
	}
	raw := unstable.RawMessage("42")
	out, err := encodeRaw(t, config{A: &raw, B: nil})
	assert.NoError(t, err)
	assert.Equal(t, "a = 42\n", out)
}

// TestMarshalerInterfaceMapValues checks RawMessage map values are classified
// individually.
func TestMarshalerInterfaceMapValues(t *testing.T) {
	type config struct {
		M map[string]unstable.RawMessage `toml:"m"`
	}
	out, err := encodeRaw(t, config{M: map[string]unstable.RawMessage{
		"x": unstable.RawMessage("a = 1\n"),
	}})
	assert.NoError(t, err)
	assert.Equal(t, "[m]\n[m.x]\na = 1\n", out)
}

// customMarshaler is a non-RawMessage type implementing unstable.Marshaler with
// a value receiver.
type customMarshaler struct {
	body string
}

func (c customMarshaler) MarshalTOML() ([]byte, error) {
	return []byte(c.body), nil
}

// ptrMarshaler implements unstable.Marshaler with a pointer receiver.
type ptrMarshaler struct {
	body string
}

func (c *ptrMarshaler) MarshalTOML() ([]byte, error) {
	return []byte(c.body), nil
}

func TestMarshalerInterfaceCustomTypes(t *testing.T) {
	t.Run("value receiver", func(t *testing.T) {
		type config struct {
			X customMarshaler `toml:"x"`
		}
		out, err := encodeRaw(t, config{X: customMarshaler{body: "v = 1\n"}})
		assert.NoError(t, err)
		assert.Equal(t, "[x]\nv = 1\n", out)
	})

	t.Run("pointer receiver", func(t *testing.T) {
		type config struct {
			X ptrMarshaler `toml:"x"`
		}
		out, err := encodeRaw(t, config{X: ptrMarshaler{body: "42"}})
		assert.NoError(t, err)
		assert.Equal(t, "x = 42\n", out)
	})

	t.Run("pointer receiver behind pointer", func(t *testing.T) {
		type config struct {
			X *ptrMarshaler `toml:"x"`
		}
		out, err := encodeRaw(t, config{X: &ptrMarshaler{body: "k = 1\n"}})
		assert.NoError(t, err)
		assert.Equal(t, "[x]\nk = 1\n", out)
	})
}

// errMarshaler always fails.
type errMarshaler struct{}

func (errMarshaler) MarshalTOML() ([]byte, error) {
	return nil, errors.New("marshal boom")
}

func TestMarshalerInterfaceErrorPropagation(t *testing.T) {
	t.Run("table position", func(t *testing.T) {
		type config struct {
			X errMarshaler `toml:"x"`
		}
		_, err := encodeRaw(t, config{})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "marshal boom"), err.Error())
	})

	t.Run("inline array element", func(t *testing.T) {
		type config struct {
			X []errMarshaler `toml:"x"`
		}
		_, err := encodeRaw(t, config{X: []errMarshaler{{}}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "marshal boom"), err.Error())
	})
}

// TestMarshalerInterfaceArrayOfTablesPointerReceiver exercises a slice of a
// pointer-receiver Marshaler (its elements are addressable).
func TestMarshalerInterfaceArrayOfTablesPointerReceiver(t *testing.T) {
	type config struct {
		Item []ptrMarshaler `toml:"item"`
	}
	out, err := encodeRaw(t, config{Item: []ptrMarshaler{
		{body: "a = 1\n"},
		{body: "a = 2\n"},
	}})
	assert.NoError(t, err)
	assert.Equal(t, "[[item]]\na = 1\n\n[[item]]\na = 2\n", out)
}

// TestMarshalerInterfaceCommentedMultilineValue checks a commented multiline
// raw value has every physical line prefixed with the comment marker, sharing
// encodeKeyValue's commented handling rather than emitting invalid TOML.
func TestMarshalerInterfaceCommentedMultilineValue(t *testing.T) {
	type config struct {
		A unstable.RawMessage `toml:"a,commented"`
	}
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).
		EnableMarshalerInterface().
		SetArraysMultiline(true).
		Encode(config{A: unstable.RawMessage("[\n  1,\n  2,\n]")})
	assert.NoError(t, err)
	assert.Equal(t, "# a = [\n#   1,\n#   2,\n# ]\n", buf.String())
}

// flakyMarshaler succeeds on its first call (used by the encoder to classify
// the shape) and fails on the second (the emit), exercising the emit-time error
// paths that the eager classification otherwise makes unreachable.
type flakyMarshaler struct {
	calls *int
	body  string
}

func (f flakyMarshaler) MarshalTOML() ([]byte, error) {
	*f.calls++
	if *f.calls == 1 {
		return []byte(f.body), nil
	}
	return nil, errors.New("flaky boom")
}

func TestMarshalerInterfaceEmitError(t *testing.T) {
	t.Run("table body", func(t *testing.T) {
		calls := 0
		type config struct {
			X flakyMarshaler `toml:"x"`
		}
		_, err := encodeRaw(t, config{X: flakyMarshaler{calls: &calls, body: "a = 1\n"}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "flaky boom"), err.Error())
	})

	t.Run("inline value", func(t *testing.T) {
		calls := 0
		type config struct {
			X flakyMarshaler `toml:"x"`
		}
		_, err := encodeRaw(t, config{X: flakyMarshaler{calls: &calls, body: "42"}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "flaky boom"), err.Error())
	})
}

// TestMarshalerInterfaceValueSliceField marshals a plain value slice with the
// interface enabled: it is a regular array, not an array of tables.
func TestMarshalerInterfaceValueSliceField(t *testing.T) {
	type config struct {
		Nums []int `toml:"nums"`
	}
	out, err := encodeRaw(t, config{Nums: []int{1, 2, 3}})
	assert.NoError(t, err)
	assert.Equal(t, "nums = [1, 2, 3]\n", out)
}

// TestMarshalerInterfaceNilSliceElement covers a nil (unresolvable) array
// element while the interface is enabled.
func TestMarshalerInterfaceNilSliceElement(t *testing.T) {
	type config struct {
		Items []interface{} `toml:"items"`
	}
	_, err := encodeRaw(t, config{Items: []interface{}{nil}})
	assert.Error(t, err)
}

func TestMarshalerInterfaceEmptyOmitted(t *testing.T) {
	type config struct {
		A string              `toml:"a"`
		E unstable.RawMessage `toml:"e"`
		B string              `toml:"b"`
	}
	t.Run("nil", func(t *testing.T) {
		out, err := encodeRaw(t, config{A: "x", E: nil, B: "y"})
		assert.NoError(t, err)
		assert.Equal(t, "a = 'x'\nb = 'y'\n", out)
	})
	t.Run("whitespace only", func(t *testing.T) {
		out, err := encodeRaw(t, config{A: "x", E: unstable.RawMessage("  \n\t"), B: "y"})
		assert.NoError(t, err)
		assert.Equal(t, "a = 'x'\nb = 'y'\n", out)
	})
}

func TestMarshalerInterfaceTableContentInInlineErrors(t *testing.T) {
	t.Run("array element with table content", func(t *testing.T) {
		// Mixed shapes: not an array of tables, so the table-shaped element is
		// pushed into an inline array, which is invalid.
		type config struct {
			X []unstable.RawMessage `toml:"x"`
		}
		_, err := encodeRaw(t, config{X: []unstable.RawMessage{
			unstable.RawMessage("k = 1\n"),
			unstable.RawMessage("42"),
		}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "inline value"), err.Error())
	})

	t.Run("inline table member with table content", func(t *testing.T) {
		type config struct {
			X unstable.RawMessage `toml:"x"`
		}
		var buf bytes.Buffer
		err := toml.NewEncoder(&buf).
			EnableMarshalerInterface().
			SetTablesInline(true).
			Encode(config{X: unstable.RawMessage("k = 1\n")})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "inline value"), err.Error())
	})

	t.Run("empty in inline array", func(t *testing.T) {
		type config struct {
			X []unstable.RawMessage `toml:"x"`
		}
		_, err := encodeRaw(t, config{X: []unstable.RawMessage{
			unstable.RawMessage(""),
		}})
		assert.Error(t, err)
	})
}

// TestRawMessageRoundTrip decodes a document capturing raw bytes, re-encodes it,
// and checks the re-encoded document decodes to the same structure.
func TestRawMessageRoundTrip(t *testing.T) {
	examples := []struct {
		desc string
		doc  string
	}{
		{
			desc: "flat table",
			doc:  "[plugin]\nname = \"example\"\nversion = \"1.0\"\n",
		},
		{
			desc: "scalar value",
			doc:  "answer = 42\n",
		},
		{
			desc: "array value",
			doc:  "ports = [8000, 8001, 8002]\n",
		},
		{
			desc: "inline table value",
			doc:  "point = {x = 1, y = 2}\n",
		},
		{
			desc: "multiple tables",
			doc:  "[plugin]\nname = \"a\"\n\n[server]\nhost = \"localhost\"\nport = 8080\n",
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			// Capture every top-level key as a RawMessage.
			var captured map[string]unstable.RawMessage
			err := toml.NewDecoder(strings.NewReader(e.doc)).
				EnableUnmarshalerInterface().
				Decode(&captured)
			assert.NoError(t, err)

			reencoded, err := encodeRaw(t, captured)
			assert.NoError(t, err)

			// Both the original and the re-encoded document must decode to the
			// same generic structure.
			var want, got map[string]interface{}
			assert.NoError(t, toml.Unmarshal([]byte(e.doc), &want))
			assert.NoError(t, toml.Unmarshal([]byte(reencoded), &got))
			assert.True(t, reflect.DeepEqual(want, got),
				"round-trip mismatch\noriginal:\n%s\nre-encoded:\n%s\nwant=%#v\ngot=%#v",
				e.doc, reencoded, want, got)
		})
	}
}

// TestRawMessageRoundTripArrayTables captures each [[item]] element into a
// []RawMessage, re-encodes it, and checks the structure survives.
func TestRawMessageRoundTripArrayTables(t *testing.T) {
	doc := "[[item]]\na = 1\n\n[[item]]\na = 2\nb = 3\n"

	type config struct {
		Item []unstable.RawMessage `toml:"item"`
	}
	var captured config
	err := toml.NewDecoder(strings.NewReader(doc)).
		EnableUnmarshalerInterface().
		Decode(&captured)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(captured.Item))

	reencoded, err := encodeRaw(t, captured)
	assert.NoError(t, err)

	var want, got map[string]interface{}
	assert.NoError(t, toml.Unmarshal([]byte(doc), &want))
	assert.NoError(t, toml.Unmarshal([]byte(reencoded), &got))
	assert.True(t, reflect.DeepEqual(want, got),
		"round-trip mismatch\noriginal:\n%s\nre-encoded:\n%s", doc, reencoded)
}

// TestRawMessageRoundTripStruct exercises the documented struct-field flow end
// to end.
func TestRawMessageRoundTripStruct(t *testing.T) {
	doc := "[plugin]\nname = \"example\"\nversion = \"1.0\"\nenabled = true\n"

	type config struct {
		Plugin unstable.RawMessage `toml:"plugin"`
	}
	var cfg config
	err := toml.NewDecoder(strings.NewReader(doc)).
		EnableUnmarshalerInterface().
		Decode(&cfg)
	assert.NoError(t, err)

	out, err := encodeRaw(t, cfg)
	assert.NoError(t, err)
	assert.Equal(t, "[plugin]\nname = \"example\"\nversion = \"1.0\"\nenabled = true\n", out)
}

// TestRawMessageNestedSubTableHeadersLimitation pins the documented limitation:
// a captured table body that contains its own sub-table headers keeps those
// headers relative to the captured root, so re-emitting it under a deeper key
// does not re-root them.
func TestRawMessageNestedSubTableHeadersLimitation(t *testing.T) {
	doc := "[outer]\na = 1\n[outer.sub]\nb = 2\n"

	type config struct {
		Outer unstable.RawMessage `toml:"outer"`
	}
	var cfg config
	err := toml.NewDecoder(strings.NewReader(doc)).
		EnableUnmarshalerInterface().
		Decode(&cfg)
	assert.NoError(t, err)

	// The captured body uses a header relative to [outer].
	assert.True(t, strings.Contains(string(cfg.Outer), "[sub]"), string(cfg.Outer))

	out, err := encodeRaw(t, cfg)
	assert.NoError(t, err)
	// The relative [sub] header is spliced verbatim (not rewritten to
	// [outer.sub]). This is the documented behavior.
	assert.Equal(t, "[outer]\na = 1\n[sub]\nb = 2\n", out)
}

func BenchmarkMarshalRawMessage(b *testing.B) {
	b.Run("table", func(b *testing.B) {
		v := rawTableConfig{
			Plugin: unstable.RawMessage("name = \"example\"\nversion = \"1.0\"\nenabled = true\n"),
		}
		benchmarkEncodeRaw(b, v)
	})

	b.Run("scalar", func(b *testing.B) {
		benchmarkEncodeRaw(b, rawValueConfig{N: unstable.RawMessage("42")})
	})

	b.Run("inline-table", func(b *testing.B) {
		benchmarkEncodeRaw(b, rawValueConfig{N: unstable.RawMessage(`{a = 1, b = "x", c = [1, 2, 3]}`)})
	})

	b.Run("array-of-tables", func(b *testing.B) {
		type config struct {
			Item []unstable.RawMessage `toml:"item"`
		}
		v := config{Item: []unstable.RawMessage{
			unstable.RawMessage("a = 1\nb = 2\n"),
			unstable.RawMessage("a = 3\nb = 4\n"),
			unstable.RawMessage("a = 5\nb = 6\n"),
		}}
		benchmarkEncodeRaw(b, v)
	})
}

func benchmarkEncodeRaw(b *testing.B, v interface{}) {
	b.Helper()
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf).EnableMarshalerInterface()

	// Warm the per-type caches and compute the output size.
	buf.Reset()
	if err := enc.Encode(v); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(buf.Len()))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := enc.Encode(v); err != nil {
			b.Fatal(err)
		}
	}
}

func ExampleEncoder_EnableMarshalerInterface() {
	// A RawMessage captured (or hand-built) as raw TOML is spliced back into
	// the document verbatim.
	type Config struct {
		Plugin unstable.RawMessage `toml:"plugin"`
	}

	cfg := Config{
		Plugin: unstable.RawMessage("name = \"example\"\nversion = \"1.0\"\n"),
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).
		EnableMarshalerInterface().
		Encode(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

	// Output:
	// [plugin]
	// name = "example"
	// version = "1.0"
}
