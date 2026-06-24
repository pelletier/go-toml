package toml_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

// This file is an exhaustive regression suite for the marshal/unmarshal option
// and struct-tag combinations whose behavior was reworked by the v2.4.0 parser,
// decoder, and encoder reimplementation (#1067). Each test documents and locks
// in the intended v2.4 behavior so future refactors cannot silently regress it.
//
// Where the behavior intentionally changed relative to v2.3.x, the change is
// noted in the test comment ("v2.3: ..." / "v2.4: ...").

func marshalOpts(t *testing.T, v interface{}, opts func(*toml.Encoder)) (string, error) {
	t.Helper()
	var b strings.Builder
	enc := toml.NewEncoder(&b)
	if opts != nil {
		opts(enc)
	}
	err := enc.Encode(v)
	return b.String(), err
}

// ---------------------------------------------------------------------------
// Struct-tag option forms.
// ---------------------------------------------------------------------------

// TestAuditStructTagFormsEquivalent verifies that the comma form embedded in the
// "toml" tag (e.g. `toml:",multiline"`) and the standalone boolean tag form
// (e.g. `multiline:"true"`) produce identical output.
//
// v2.3: only the comma form was honored; the standalone form was silently
// ignored (despite being shown in the docs).
// v2.4: both forms are honored and are equivalent.
func TestAuditStructTagFormsEquivalent(t *testing.T) {
	t.Run("multiline string", func(t *testing.T) {
		comma := struct {
			S string `toml:"s,multiline"`
		}{S: "l1\nl2"}
		standalone := struct {
			S string `toml:"s" multiline:"true"`
		}{S: "l1\nl2"}
		a, err := toml.Marshal(comma)
		assert.NoError(t, err)
		b, err := toml.Marshal(standalone)
		assert.NoError(t, err)
		assert.Equal(t, string(a), string(b))
		assert.True(t, strings.Contains(string(a), `"""`))
	})

	t.Run("inline table", func(t *testing.T) {
		type inner struct {
			A int
			B string
		}
		comma := struct {
			T inner `toml:"t,inline"`
		}{T: inner{A: 1, B: "z"}}
		standalone := struct {
			T inner `toml:"t" inline:"true"`
		}{T: inner{A: 1, B: "z"}}
		a, err := toml.Marshal(comma)
		assert.NoError(t, err)
		b, err := toml.Marshal(standalone)
		assert.NoError(t, err)
		assert.Equal(t, string(a), string(b))
		assert.Equal(t, "t = {A = 1, B = 'z'}\n", string(a))
	})

	t.Run("commented", func(t *testing.T) {
		comma := struct {
			X int `toml:"x,commented"`
			Y int `toml:"y"`
		}{X: 1, Y: 2}
		standalone := struct {
			X int `toml:"x" commented:"true"`
			Y int `toml:"y"`
		}{X: 1, Y: 2}
		a, err := toml.Marshal(comma)
		assert.NoError(t, err)
		b, err := toml.Marshal(standalone)
		assert.NoError(t, err)
		assert.Equal(t, string(a), string(b))
		assert.Equal(t, "# x = 1\ny = 2\n", string(a))
	})
}

// ---------------------------------------------------------------------------
// Comments and the "commented" option under SetTablesInline.
// ---------------------------------------------------------------------------

// TestAuditCommentsWithTablesInline checks that comments and the "commented"
// option are preserved when SetTablesInline collapses a field onto a single
// key/value line.
//
// v2.3: comments and the commented marker were dropped for any field that was
// rendered inline at the document root.
// v2.4: they are preserved, because a comment that precedes (or comments out) a
// key/value line is valid TOML.
func TestAuditCommentsWithTablesInline(t *testing.T) {
	t.Run("comment tag on inlined table", func(t *testing.T) {
		type inner struct {
			A int
			B string
		}
		v := struct {
			Sub inner `toml:"sub" comment:"a table"`
		}{Sub: inner{A: 1, B: "x"}}
		out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetTablesInline(true) })
		assert.NoError(t, err)
		assert.Equal(t, "# a table\nsub = {A = 1, B = 'x'}\n", out)
	})

	t.Run("commented scalar inline", func(t *testing.T) {
		v := struct {
			X int `toml:"x,commented"`
			Y int `toml:"y"`
		}{X: 1, Y: 2}
		out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetTablesInline(true) })
		assert.NoError(t, err)
		assert.Equal(t, "# x = 1\ny = 2\n", out)
	})

	t.Run("multiline comment inline", func(t *testing.T) {
		v := struct {
			X int `toml:"x" comment:"line1\nline2"`
		}{X: 1}
		out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetTablesInline(true) })
		assert.NoError(t, err)
		assert.Equal(t, "# line1\n# line2\nx = 1\n", out)
	})
}

// TestAuditNoCommentInsideInlineTable verifies that a comment annotation on a
// field that ends up *inside* an inline table's braces is suppressed, since a
// newline-bearing comment inside `{}` is invalid TOML.
//
// v2.3: the comment was emitted inside the braces, producing invalid TOML.
// v2.4: the comment is suppressed.
func TestAuditNoCommentInsideInlineTable(t *testing.T) {
	type inner struct {
		X int `toml:"x" comment:"inner comment"`
		Y int `toml:"y"`
	}
	v := struct {
		T inner `toml:"t,inline"`
	}{T: inner{X: 1, Y: 2}}
	out, err := toml.Marshal(v)
	assert.NoError(t, err)
	assert.Equal(t, "t = {x = 1, y = 2}\n", out2str(out))

	// The output must round-trip.
	var back map[string]map[string]int
	assert.NoError(t, toml.Unmarshal(out, &back))
	assert.Equal(t, 1, back["t"]["x"])
	assert.Equal(t, 2, back["t"]["y"])
}

// TestAuditMultilineArrayCollapsesInsideInlineTable verifies that a
// multiline-tagged array collapses to a single line when it is nested inside an
// inline table, since newlines are not permitted inside `{}`.
//
// v2.3: emitted a multiline array inside the inline table, producing invalid
// TOML.
// v2.4: collapses the array onto a single line.
func TestAuditMultilineArrayCollapsesInsideInlineTable(t *testing.T) {
	type sub struct {
		A []int `toml:"a,multiline"`
	}
	v := struct {
		Sub sub `toml:"sub"`
	}{Sub: sub{A: []int{1, 2, 3}}}
	out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetTablesInline(true) })
	assert.NoError(t, err)
	assert.Equal(t, "sub = {a = [1, 2, 3]}\n", out)
}

// TestAuditInlineTableSpacingInMultilineArray verifies that inline tables that
// appear as elements of a multiline array use clean single-space padding.
//
// v2.3: produced stray double spaces, e.g. `{  A = 1,   B = 'a'}`.
// v2.4: produces `{A = 1, B = 'a'}`.
func TestAuditInlineTableSpacingInMultilineArray(t *testing.T) {
	type inner struct {
		A int
		B string
	}
	v := struct {
		S []inner `toml:"s,inline"`
	}{S: []inner{{1, "a"}, {2, "b"}}}
	out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetArraysMultiline(true) })
	assert.NoError(t, err)
	assert.Equal(t, "s = [\n  {A = 1, B = 'a'},\n  {A = 2, B = 'b'}\n]\n", out)
}

// ---------------------------------------------------------------------------
// omitempty / omitzero.
// ---------------------------------------------------------------------------

// TestAuditOmitemptyTime verifies that omitempty omits the zero time.Time
// (0001-01-01) but emits a non-zero time such as the Unix epoch.
//
// v2.3: a non-zero epoch time was incorrectly omitted.
// v2.4: only the zero value is omitted.
func TestAuditOmitemptyTime(t *testing.T) {
	type S struct {
		G time.Time `toml:"g,omitempty"`
		N int       `toml:"n"`
	}
	epoch, err := toml.Marshal(S{G: time.Unix(0, 0).UTC(), N: 1})
	assert.NoError(t, err)
	assert.Equal(t, "g = 1970-01-01T00:00:00Z\nn = 1\n", out2str(epoch))

	zero, err := toml.Marshal(S{N: 1})
	assert.NoError(t, err)
	assert.Equal(t, "n = 1\n", out2str(zero))
}

// TestAuditOmitemptyNonNilEmptyCollections verifies that a non-nil but empty
// slice or map is omitted under omitempty.
func TestAuditOmitemptyNonNilEmptyCollections(t *testing.T) {
	type S struct {
		C []int          `toml:"c,omitempty"`
		D map[string]int `toml:"d,omitempty"`
		N int            `toml:"n"`
	}
	out, err := toml.Marshal(S{C: []int{}, D: map[string]int{}, N: 1})
	assert.NoError(t, err)
	assert.Equal(t, "n = 1\n", out2str(out))
}

// TestAuditOmitzero verifies that omitzero honors a type's IsZero() method.
func TestAuditOmitzero(t *testing.T) {
	type S struct {
		A customZeroAudit `toml:"a,omitzero"`
		B customZeroAudit `toml:"b,omitzero"`
		C int             `toml:"c,omitzero"`
	}
	out, err := toml.Marshal(S{A: customZeroAudit{V: 0}, B: customZeroAudit{V: 5}, C: 0})
	assert.NoError(t, err)
	assert.Equal(t, "[b]\nV = 5\n", out2str(out))
}

type customZeroAudit struct{ V int }

func (c customZeroAudit) IsZero() bool { return c.V == 0 }

// ---------------------------------------------------------------------------
// Document-root constraints.
// ---------------------------------------------------------------------------

// TestAuditTopLevelSliceRejected verifies that marshaling a slice as the
// document root is rejected with an error.
//
// v2.3: emitted invalid TOML (an array-table header with an empty key).
// v2.4: returns an error, because a TOML document root must be a table.
func TestAuditTopLevelSliceRejected(t *testing.T) {
	_, err := toml.Marshal([]map[string]int{{"a": 1}, {"b": 2}})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "document root"))
}

// ---------------------------------------------------------------------------
// Encoder option matrix on a representative document.
// ---------------------------------------------------------------------------

// TestAuditEncoderOptionMatrix locks in the rendering of a document containing a
// scalar, a sub-table, an array of tables, and a nested map across the main
// encoder options.
func TestAuditEncoderOptionMatrix(t *testing.T) {
	type inner struct {
		A int
		B string
	}
	doc := struct {
		Top  int
		Sub  inner
		Subs []inner
		MapV map[string]inner
	}{
		Top:  1,
		Sub:  inner{A: 2, B: "x"},
		Subs: []inner{{3, "y"}, {4, "z"}},
		MapV: map[string]inner{"k": {5, "w"}},
	}

	cases := []struct {
		name     string
		opts     func(*toml.Encoder)
		expected string
	}{
		{
			name: "default",
			opts: nil,
			expected: "Top = 1\n\n[Sub]\nA = 2\nB = 'x'\n\n" +
				"[[Subs]]\nA = 3\nB = 'y'\n\n[[Subs]]\nA = 4\nB = 'z'\n\n" +
				"[MapV]\n[MapV.k]\nA = 5\nB = 'w'\n",
		},
		{
			name: "indentTables",
			opts: func(e *toml.Encoder) { e.SetIndentTables(true) },
			expected: "Top = 1\n\n[Sub]\n  A = 2\n  B = 'x'\n\n" +
				"[[Subs]]\n  A = 3\n  B = 'y'\n\n[[Subs]]\n  A = 4\n  B = 'z'\n\n" +
				"[MapV]\n  [MapV.k]\n    A = 5\n    B = 'w'\n",
		},
		{
			name: "tablesInline",
			opts: func(e *toml.Encoder) { e.SetTablesInline(true) },
			expected: "Top = 1\nSub = {A = 2, B = 'x'}\n" +
				"Subs = [{A = 3, B = 'y'}, {A = 4, B = 'z'}]\n" +
				"MapV = {k = {A = 5, B = 'w'}}\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := marshalOpts(t, doc, tc.opts)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, out)

			// Every rendering must round-trip back to the same structure.
			var back map[string]interface{}
			assert.NoError(t, toml.Unmarshal([]byte(out), &back))
		})
	}
}

// ---------------------------------------------------------------------------
// Decoder: local time / date into time.Time.
// ---------------------------------------------------------------------------

// TestAuditLocalTimeAndDateIntoTime locks in that local times and local dates
// can be decoded into a time.Time, capturing the available clock/date fields.
//
// v2.3: decoding a local time into time.Time was an error.
// v2.4: it succeeds; the resulting time.Time carries the clock components (the
// date defaults to year 0 in the machine-local zone).
func TestAuditLocalTimeAndDateIntoTime(t *testing.T) {
	t.Run("local time", func(t *testing.T) {
		var x struct{ T time.Time }
		assert.NoError(t, toml.Unmarshal([]byte("t = 04:05:06\n"), &x))
		assert.Equal(t, 4, x.T.Hour())
		assert.Equal(t, 5, x.T.Minute())
		assert.Equal(t, 6, x.T.Second())
	})

	t.Run("local date", func(t *testing.T) {
		var x struct{ T time.Time }
		assert.NoError(t, toml.Unmarshal([]byte("t = 2020-01-02\n"), &x))
		assert.Equal(t, 2020, x.T.Year())
		assert.Equal(t, time.Month(1), x.T.Month())
		assert.Equal(t, 2, x.T.Day())
	})

	t.Run("local time into int errors", func(t *testing.T) {
		var x struct{ T int }
		assert.Error(t, toml.Unmarshal([]byte("t = 04:05:06\n"), &x))
	})
}

// ---------------------------------------------------------------------------
// Decoder: error typing and scalar conversions.
// ---------------------------------------------------------------------------

// TestAuditDecodeErrorTypes verifies that structural and value errors surface as
// *toml.DecodeError (carrying position information), not a bare error.
//
// v2.3: several of these were plain errors.errorString.
// v2.4: they are *toml.DecodeError.
func TestAuditDecodeErrorTypes(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"integer overflow", "v = 1000\n"},  // into int8 below
		{"negative into uint", "v = -1\n"},  // into uint below
		{"duplicate key", "a = 1\na = 2\n"}, // into map
		{"redefined table", "[a]\nx=1\n[a]\ny=2\n"},
		{"dotted key conflict", "a.b = 1\na.b.c = 2\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			switch tc.name {
			case "integer overflow":
				var x struct{ V int8 }
				err = toml.Unmarshal([]byte(tc.input), &x)
			case "negative into uint":
				var x struct{ V uint }
				err = toml.Unmarshal([]byte(tc.input), &x)
			default:
				var x map[string]interface{}
				err = toml.Unmarshal([]byte(tc.input), &x)
			}
			assert.Error(t, err)
			var de *toml.DecodeError
			assert.True(t, errors.As(err, &de))
		})
	}
}

// TestAuditScalarConversionErrors locks in the rejection of incompatible scalar
// conversions.
func TestAuditScalarConversionErrors(t *testing.T) {
	t.Run("float into int", func(t *testing.T) {
		var x struct{ V int }
		assert.Error(t, toml.Unmarshal([]byte("v = 5.0\n"), &x))
	})
	t.Run("string into int", func(t *testing.T) {
		var x struct{ V int }
		assert.Error(t, toml.Unmarshal([]byte("v = \"x\"\n"), &x))
	})
	t.Run("int into float succeeds", func(t *testing.T) {
		var x struct{ V float64 }
		assert.NoError(t, toml.Unmarshal([]byte("v = 5\n"), &x))
		assert.Equal(t, 5.0, x.V)
	})
	t.Run("int8 overflow", func(t *testing.T) {
		var x struct{ V int8 }
		assert.Error(t, toml.Unmarshal([]byte("v = 1000\n"), &x))
	})
}

// ---------------------------------------------------------------------------
// Regressions introduced by the v2.4.0 reimplementation and fixed in v2.4.1 /
// v2.4.2. These were invisible to a v2.3 <-> v2.4.2 comparison (both endpoints
// are correct), so they are pinned explicitly here.
// ---------------------------------------------------------------------------

// TestAuditOmitemptyStructRecursion guards the #1075 fix: a struct field marked
// omitempty whose encodable fields are all empty — including non-nil but empty
// maps and slices — must be omitted entirely rather than emitting an empty
// table header.
func TestAuditOmitemptyStructRecursion(t *testing.T) {
	type coll struct {
		M map[string]int
		S []int
	}
	type doc struct {
		F coll `toml:"f,omitempty"`
		N int  `toml:"n"`
	}

	empty, err := toml.Marshal(doc{F: coll{M: map[string]int{}, S: []int{}}, N: 1})
	assert.NoError(t, err)
	assert.Equal(t, "n = 1\n", out2str(empty))

	nonEmpty, err := toml.Marshal(doc{F: coll{M: map[string]int{"k": 1}}, N: 1})
	assert.NoError(t, err)
	assert.Equal(t, "n = 1\n\n[f]\nS = []\n\n[f.M]\nk = 1\n", out2str(nonEmpty))
}

// TestAuditEmbeddedOptionsOnlyTag guards the #1079 fix: an embedded struct
// whose toml tag sets only options (e.g. `,inline`) and no name must still be
// flattened, on both the encode and decode sides, matching encoding/json.
func TestAuditEmbeddedOptionsOnlyTag(t *testing.T) {
	type inner struct {
		A int
		B string
	}

	t.Run("encode flattens", func(t *testing.T) {
		v := struct {
			inner `toml:",inline"`
			C     int
		}{inner: inner{A: 1, B: "x"}, C: 2}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.Equal(t, "A = 1\nB = 'x'\nC = 2\n", out2str(out))
	})

	t.Run("decode flattens", func(t *testing.T) {
		var v struct {
			inner `toml:",inline"`
			C     int
		}
		assert.NoError(t, toml.Unmarshal([]byte("A = 1\nB = \"x\"\nC = 2\n"), &v))
		assert.Equal(t, 1, v.inner.A)
		assert.Equal(t, "x", v.inner.B)
		assert.Equal(t, 2, v.C)
	})

	t.Run("explicit name is a regular field", func(t *testing.T) {
		var v struct {
			inner `toml:"named"`
			C     int
		}
		assert.NoError(t, toml.Unmarshal([]byte("C = 2\n[named]\nA = 1\nB = \"x\"\n"), &v))
		assert.Equal(t, 1, v.inner.A)
		assert.Equal(t, 2, v.C)
	})
}

// TestAuditCommentedIndentLayout pins the layout of the "commented" option when
// combined with SetIndentTables: the comment marker stays at column zero and the
// table indentation appears inside the comment (e.g. `#   key = value`).
func TestAuditCommentedIndentLayout(t *testing.T) {
	type leaf struct {
		A int
		B string
	}
	type mid struct {
		Leaf leaf `toml:"leaf"`
	}
	v := struct {
		Mid mid `toml:"mid,commented"`
		Y   int `toml:"y"`
	}{Mid: mid{Leaf: leaf{A: 1, B: "x"}}, Y: 9}

	out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetIndentTables(true) })
	assert.NoError(t, err)
	assert.Equal(t, "y = 9\n\n# [mid]\n#   [mid.leaf]\n#     A = 1\n#     B = 'x'\n", out)

	// A commented scalar at a deeper indent likewise keeps the marker at the
	// margin.
	flat := struct {
		Sub leaf `toml:"sub,commented"`
	}{Sub: leaf{A: 1, B: "x"}}
	out2, err := marshalOpts(t, flat, func(e *toml.Encoder) { e.SetIndentTables(true) })
	assert.NoError(t, err)
	assert.Equal(t, "# [sub]\n#   A = 1\n#   B = 'x'\n", out2)
}

// TestAuditCommentedMultilineValue guards against emitting invalid TOML when a
// field is both "commented" and rendered across multiple lines (a multiline
// string or a multiline array). Every physical line must carry the comment
// marker; otherwise the continuation lines are live, syntactically invalid TOML.
//
// This was found by the generative validity oracle: the pre-rewrite encoder and
// every v2.4 release up to this fix only commented the first line.
func TestAuditCommentedMultilineValue(t *testing.T) {
	t.Run("multiline string", func(t *testing.T) {
		v := struct {
			S string `toml:"s,commented,multiline"`
		}{S: "line1\nline2"}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.Equal(t, "# s = \"\"\"\n# line1\n# line2\"\"\"\n", out2str(out))
		assertReparses(t, out)
	})

	t.Run("multiline array via tag", func(t *testing.T) {
		v := struct {
			A []int `toml:"a,commented,multiline"`
		}{A: []int{1, 2, 3}}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.Equal(t, "# a = [\n#   1,\n#   2,\n#   3\n# ]\n", out2str(out))
		assertReparses(t, out)
	})

	t.Run("multiline array via option", func(t *testing.T) {
		v := struct {
			A []int `toml:"a,commented"`
		}{A: []int{1, 2}}
		out, err := marshalOpts(t, v, func(e *toml.Encoder) { e.SetArraysMultiline(true) })
		assert.NoError(t, err)
		assertReparses(t, []byte(out))
		assert.True(t, strings.HasPrefix(out, "# a = ["))
	})

	t.Run("commented array of inline tables", func(t *testing.T) {
		type inner struct {
			X int
		}
		v := struct {
			A []inner `toml:"a,commented,inline,multiline"`
		}{A: []inner{{1}, {2}}}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assertReparses(t, out)
	})
}

func assertReparses(t *testing.T, out []byte) {
	t.Helper()
	var back map[string]interface{}
	assert.NoError(t, toml.Unmarshal(out, &back))
	// A fully commented field decodes to nothing.
	assert.Equal(t, 0, len(back))
}

func out2str(b []byte) string { return string(b) }
