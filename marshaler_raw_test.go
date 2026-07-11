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

	t.Run("nested table position", func(t *testing.T) {
		type inner struct {
			X errMarshaler `toml:"x"`
		}
		type config struct {
			A inner `toml:"a"`
		}
		_, err := encodeRaw(t, config{})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "marshal boom"), err.Error())
	})

	t.Run("nested table position with omitted super-tables", func(t *testing.T) {
		type inner struct {
			X errMarshaler `toml:"x"`
		}
		type config struct {
			A inner `toml:"a"`
		}
		var buf bytes.Buffer
		err := toml.NewEncoder(&buf).
			EnableMarshalerInterface().
			SetOmitEmptySuperTables(true).
			Encode(config{})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "marshal boom"), err.Error())
	})

	t.Run("bad map key under a table", func(t *testing.T) {
		type config struct {
			A map[bool]int `toml:"a"`
		}
		_, err := encodeRaw(t, config{A: map[bool]int{true: 1}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "map with key type"), err.Error())
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

// TestMarshalerInterfacePlainArrayOfTables marshals an array of plain
// (non-Marshaler) tables with the interface enabled: the elements go through
// the regular structural encoding.
func TestMarshalerInterfacePlainArrayOfTables(t *testing.T) {
	type item struct {
		V int64 `toml:"v"`
	}
	type config struct {
		Item []item `toml:"item"`
	}
	out, err := encodeRaw(t, config{Item: []item{{V: 1}, {V: 2}}})
	assert.NoError(t, err)
	assert.Equal(t, "[[item]]\nv = 1\n\n[[item]]\nv = 2\n", out)
}

// TestMarshalerInterfacePlainArrayOfTablesError propagates an encoding error
// out of a plain (non-Marshaler) element of an array of tables while the
// interface is enabled.
func TestMarshalerInterfacePlainArrayOfTablesError(t *testing.T) {
	type item struct {
		C chan int `toml:"c"`
	}
	type config struct {
		Item []item `toml:"item"`
	}
	_, err := encodeRaw(t, config{Item: []item{{C: make(chan int)}}})
	assert.Error(t, err)
}

// budgetMarshaler succeeds for a fixed number of calls, then errors. Unlike
// flakyMarshaler (which fails on its second call), the budget can be set to
// survive every classification pass and fail only on the final emit call.
type budgetMarshaler struct {
	calls  *int
	body   string
	budget int
}

func (m budgetMarshaler) MarshalTOML() ([]byte, error) {
	*m.calls++
	if *m.calls > m.budget {
		return nil, errors.New("budget boom")
	}
	return []byte(m.body), nil
}

// TestMarshalerInterfaceArrayOfTablesEmitError exercises the emit-time
// MarshalTOML error inside an array of tables: classification succeeds, the
// call producing the spliced bytes fails.
func TestMarshalerInterfaceArrayOfTablesEmitError(t *testing.T) {
	calls := 0
	type config struct {
		Item []budgetMarshaler `toml:"item"`
	}
	// The encoder classifies the slice as an array of tables before emitting:
	// the budget covers the classification calls so the failure lands on the
	// emit call inside the array-table writer.
	_, err := encodeRaw(t, config{Item: []budgetMarshaler{
		{calls: &calls, body: "a = 1\n", budget: 3},
	}})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "budget boom"), err.Error())
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

// TestMarshalerInterfaceInvalidContent checks that bytes which are neither a
// value nor valid table content produce an error instead of silently splicing
// an invalid document, mirroring json.RawMessage's validation.
func TestMarshalerInterfaceInvalidContent(t *testing.T) {
	type config struct {
		X unstable.RawMessage `toml:"x"`
	}
	for _, raw := range []string{"!!! not toml", "[1, 2", "42 garbage", "a = ", "= 1"} {
		t.Run(raw, func(t *testing.T) {
			out, err := encodeRaw(t, config{X: unstable.RawMessage(raw)})
			assert.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), "invalid TOML"), err.Error())
			assert.Equal(t, "", out)
		})
	}

	t.Run("array of tables element", func(t *testing.T) {
		type config struct {
			Item []unstable.RawMessage `toml:"item"`
		}
		// Both elements classify as table-shaped, but the second is invalid.
		_, err := encodeRaw(t, config{Item: []unstable.RawMessage{
			unstable.RawMessage("a = 1\n"),
			unstable.RawMessage("b ="),
		}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "invalid TOML"), err.Error())
	})
}

// TestMarshalerInterfaceCommentedTable checks a commented table-shaped
// RawMessage has its body comment-prefixed line by line: it must not leak into
// the document as live keys of the parent table.
func TestMarshalerInterfaceCommentedTable(t *testing.T) {
	t.Run("single table", func(t *testing.T) {
		type config struct {
			X unstable.RawMessage `toml:"x,commented"`
		}
		out, err := encodeRaw(t, config{X: unstable.RawMessage("a = 1\nb = 2\n")})
		assert.NoError(t, err)
		assert.Equal(t, "# [x]\n# a = 1\n# b = 2\n", out)

		// The commented content must not decode into live keys.
		var m map[string]interface{}
		assert.NoError(t, toml.Unmarshal([]byte(out), &m))
		assert.Equal(t, 0, len(m))
	})

	t.Run("array of tables", func(t *testing.T) {
		type config struct {
			Item []unstable.RawMessage `toml:"item,commented"`
		}
		out, err := encodeRaw(t, config{Item: []unstable.RawMessage{
			unstable.RawMessage("a = 1\n"),
			unstable.RawMessage("a = 2\n"),
		}})
		assert.NoError(t, err)
		assert.Equal(t, "# [[item]]\n# a = 1\n\n# [[item]]\n# a = 2\n", out)

		var m map[string]interface{}
		assert.NoError(t, toml.Unmarshal([]byte(out), &m))
		assert.Equal(t, 0, len(m))
	})
}

// shiftyMarshaler returns a different body on each call, exercising the
// emit-time validation that guards against non-deterministic implementations:
// the encoder classifies on a first call and emits the bytes of a second one.
type shiftyMarshaler struct {
	calls  *int
	bodies []string
}

func (s shiftyMarshaler) MarshalTOML() ([]byte, error) {
	i := *s.calls
	if i >= len(s.bodies) {
		i = len(s.bodies) - 1
	}
	*s.calls++
	return []byte(s.bodies[i]), nil
}

func TestMarshalerInterfaceNonDeterministic(t *testing.T) {
	t.Run("table then value", func(t *testing.T) {
		calls := 0
		type config struct {
			X shiftyMarshaler `toml:"x"`
		}
		// Classified as a table body, but the emit call returns a bare value,
		// which is not valid after a [x] header.
		_, err := encodeRaw(t, config{X: shiftyMarshaler{calls: &calls, bodies: []string{"a = 1\n", "42"}}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "invalid TOML"), err.Error())
	})

	t.Run("value then table", func(t *testing.T) {
		calls := 0
		type config struct {
			X shiftyMarshaler `toml:"x"`
		}
		// Classified as a value, but the emit call returns table content,
		// which must not be spliced into a `key = ` position.
		_, err := encodeRaw(t, config{X: shiftyMarshaler{calls: &calls, bodies: []string{"42", "a = 1\n"}}})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "inline value"), err.Error())
	})

	t.Run("table then empty", func(t *testing.T) {
		calls := 0
		type config struct {
			X shiftyMarshaler `toml:"x"`
		}
		// Classified as a table body, but the emit call returns nothing:
		// empty content is valid TOML, so the header stands alone.
		out, err := encodeRaw(t, config{X: shiftyMarshaler{calls: &calls, bodies: []string{"a = 1\n", ""}}})
		assert.NoError(t, err)
		assert.Equal(t, "[x]\n", out)
	})
}

// TestMarshalerInterfaceRoot checks a Marshaler at the document root emits its
// bytes as the whole document: the encode counterpart of the decoder
// delivering the whole document to a root Unmarshaler.
func TestMarshalerInterfaceRoot(t *testing.T) {
	t.Run("document", func(t *testing.T) {
		doc := "a = 1\n[t]\nb = 2"
		out, err := encodeRaw(t, unstable.RawMessage(doc))
		assert.NoError(t, err)
		assert.Equal(t, doc+"\n", out)
	})

	t.Run("headers are absolute at the root", func(t *testing.T) {
		// Unlike a body re-emitted under a [key] header, a root document
		// keeps its headers at the position they name, so nested tables
		// round-trip exactly.
		doc := "[outer]\na = 1\n\n[outer.sub]\nb = 2"
		out, err := encodeRaw(t, unstable.RawMessage(doc))
		assert.NoError(t, err)
		assert.Equal(t, doc+"\n", out)
	})

	t.Run("numeric table header", func(t *testing.T) {
		// At the root the bytes are parsed as a document, so [1] is a table
		// named "1", not an array value.
		out, err := encodeRaw(t, unstable.RawMessage("[1]\na = 1"))
		assert.NoError(t, err)
		assert.Equal(t, "[1]\na = 1\n", out)
	})

	t.Run("pointer", func(t *testing.T) {
		raw := unstable.RawMessage("a = 1")
		out, err := encodeRaw(t, &raw)
		assert.NoError(t, err)
		assert.Equal(t, "a = 1\n", out)
	})

	t.Run("empty", func(t *testing.T) {
		out, err := encodeRaw(t, unstable.RawMessage(" \n\t"))
		assert.NoError(t, err)
		assert.Equal(t, "", out)
	})

	t.Run("single value errors", func(t *testing.T) {
		_, err := encodeRaw(t, unstable.RawMessage("42"))
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "document root"), err.Error())
	})

	t.Run("invalid content errors", func(t *testing.T) {
		_, err := encodeRaw(t, unstable.RawMessage("!!! not toml"))
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "invalid TOML"), err.Error())
	})

	t.Run("marshal error propagates", func(t *testing.T) {
		_, err := encodeRaw(t, errMarshaler{})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "marshal boom"), err.Error())
	})

	t.Run("disabled keeps the baseline error", func(t *testing.T) {
		var buf bytes.Buffer
		err := toml.NewEncoder(&buf).Encode(unstable.RawMessage("a = 1"))
		assert.Error(t, err)
	})
}

// TestRawMessageRoundTripRoot captures a whole document into a root RawMessage
// (as introduced for the decoder in #994) and re-encodes it.
func TestRawMessageRoundTripRoot(t *testing.T) {
	doc := "a = 1\nc = [1, 2]\n\n[t]\nb = 2\n\n[t.sub]\nd = 'x'\n"

	var raw unstable.RawMessage
	err := toml.NewDecoder(strings.NewReader(doc)).
		EnableUnmarshalerInterface().
		Decode(&raw)
	assert.NoError(t, err)

	reencoded, err := encodeRaw(t, raw)
	assert.NoError(t, err)

	var want, got map[string]interface{}
	assert.NoError(t, toml.Unmarshal([]byte(doc), &want))
	assert.NoError(t, toml.Unmarshal([]byte(reencoded), &got))
	assert.True(t, reflect.DeepEqual(want, got),
		"round-trip mismatch\noriginal:\n%s\nre-encoded:\n%s\nwant=%#v\ngot=%#v",
		doc, reencoded, want, got)
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

// TestMarshalerInterfaceOmitEmptySuperTables checks that the shape of
// Marshaler entries is taken into account when deciding whether a table only
// contains sub-tables and can lose its header.
func TestMarshalerInterfaceOmitEmptySuperTables(t *testing.T) {
	t.Run("table-shaped raw counts as a sub-table", func(t *testing.T) {
		var v struct {
			B struct {
				C unstable.RawMessage `toml:"c"`
			}
		}
		v.B.C = unstable.RawMessage("a = 1\nb = 2")

		var buf bytes.Buffer
		err := toml.NewEncoder(&buf).
			EnableMarshalerInterface().
			SetOmitEmptySuperTables(true).
			Encode(v)
		assert.NoError(t, err)
		assert.Equal(t, "[B.c]\na = 1\nb = 2\n", buf.String())
	})

	t.Run("value-shaped raw keeps the super-table header", func(t *testing.T) {
		var v struct {
			B struct {
				N unstable.RawMessage `toml:"n"`
				C struct{ S string }
			}
		}
		v.B.N = unstable.RawMessage("42")
		v.B.C.S = "foo"

		var buf bytes.Buffer
		err := toml.NewEncoder(&buf).
			EnableMarshalerInterface().
			SetOmitEmptySuperTables(true).
			Encode(v)
		assert.NoError(t, err)
		assert.Equal(t, "[B]\nn = 42\n\n[B.C]\nS = 'foo'\n", buf.String())
	})
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
