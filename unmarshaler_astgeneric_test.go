package toml_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// captured records the raw TOML bytes delivered to it through the unmarshaler
// interface.
type captured struct {
	raw string
}

func (c *captured) UnmarshalTOML(data []byte) error {
	c.raw = string(data)
	return nil
}

// TestUnmarshalerInterfaceTargets exercises the raw-capture machinery of the
// unmarshaler interface: a type implementing it receives the raw TOML of its
// (sub)table, whether it sits in a struct field, a map value or an array table.
func TestUnmarshalerInterfaceTargets(t *testing.T) {
	type Doc struct {
		A *captured            `toml:"a"`
		B captured             `toml:"b"`
		M map[string]captured  `toml:"m"`
		S []captured           `toml:"s"`
		N map[string]*captured `toml:"n"`
	}

	doc := `
[a]
x = 1

[b]
y = 2

[m.one]
z = 3

[m.two]
z = 4

[[s]]
w = 5

[[s]]
w = 6

[n.deep]
q = 7
`

	var d Doc
	dec := toml.NewDecoder(strings.NewReader(doc)).EnableUnmarshalerInterface()
	if err := dec.Decode(&d); err != nil {
		t.Fatal(err)
	}
	if d.A == nil || !strings.Contains(d.A.raw, "x = 1") {
		t.Fatalf("A = %#v", d.A)
	}
	if !strings.Contains(d.B.raw, "y = 2") {
		t.Fatalf("B = %#v", d.B)
	}
	if !strings.Contains(d.M["one"].raw, "z = 3") || !strings.Contains(d.M["two"].raw, "z = 4") {
		t.Fatalf("M = %#v", d.M)
	}
	if len(d.S) != 2 || !strings.Contains(d.S[0].raw, "w = 5") || !strings.Contains(d.S[1].raw, "w = 6") {
		t.Fatalf("S = %#v", d.S)
	}
	if d.N["deep"] == nil || !strings.Contains(d.N["deep"].raw, "q = 7") {
		t.Fatalf("N = %#v", d.N)
	}
}

// TestUnmarshalInvalidDateTimeIntoTyped decodes well-formed-looking but invalid
// dates and times into typed fields, exercising the parse-error branches of the
// date/time assignments.
func TestUnmarshalInvalidDateTimeIntoTyped(t *testing.T) {
	cases := []struct {
		desc string
		doc  string
	}{
		{"bad local date", "t = 1979-13-45"},
		{"bad local time", "t = 25:99:99"},
		{"bad local datetime", "t = 1979-13-45T25:99:99"},
		{"bad offset datetime", "t = 1979-13-45T25:99:99Z"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var s struct {
				T time.Time `toml:"t"`
			}
			if err := toml.Unmarshal([]byte(c.doc), &s); err == nil {
				t.Fatalf("expected an error decoding %q", c.doc)
			}
		})
	}
}

// strKey is a named string type so that map[strKey]interface{} is not the
// exact map[string]interface{}, forcing the reflection-based decode path.
type strKey string

// TestUnmarshalRichDocIntoNamedKeyMap decodes a large real-world document into
// a map with a named key type, driving the reflection-based document loop
// (table headers, array tables, dotted keys, scalar/array/inline-table values)
// across a broad range of inputs.
func TestUnmarshalRichDocIntoNamedKeyMap(t *testing.T) {
	data, err := os.ReadFile("benchmark/benchmark.toml")
	if err != nil {
		t.Fatal(err)
	}
	var m map[strKey]interface{}
	if err := toml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal("expected a non-empty map")
	}
}

// TestUnmarshalComprehensiveTypedStruct decodes every value kind into typed
// struct fields, driving the typed branches of the scalar assignments, fixed
// arrays, nested structs and typed maps.
func TestUnmarshalComprehensiveTypedStruct(t *testing.T) {
	type Sub struct {
		X int
	}
	type Typed struct {
		I   int                `toml:"i"`
		U   uint               `toml:"u"`
		S   string             `toml:"s"`
		F   float64            `toml:"f"`
		B   bool               `toml:"b"`
		T   time.Time          `toml:"t"`
		TT  time.Time          `toml:"tt"`
		LDT toml.LocalDateTime `toml:"ldt"`
		LD  toml.LocalDate     `toml:"ld"`
		LT  toml.LocalTime     `toml:"lt"`
		Arr [3]int             `toml:"arr"`
		Sl  []string           `toml:"sl"`
		Sub Sub                `toml:"sub"`
		M   map[string]int     `toml:"m"`
	}

	doc := `
i = -5
u = 9
s = "hi"
f = 2.5
b = true
t = 1979-05-27T07:32:00Z
tt = 1979-05-27
ldt = 1979-05-27T07:32:00
ld = 1979-05-27
lt = 07:32:00
arr = [1, 2, 3]
sl = ["a", "b"]
sub = { x = 1 }
m = { one = 1, two = 2 }
`

	var v Typed
	if err := toml.Unmarshal([]byte(doc), &v); err != nil {
		t.Fatal(err)
	}
	if v.I != -5 || v.U != 9 || v.S != "hi" || v.F != 2.5 || !v.B {
		t.Fatalf("scalars: %#v", v)
	}
	if v.T.IsZero() || v.TT.IsZero() {
		t.Fatalf("times: %#v", v)
	}
	if v.Arr != [3]int{1, 2, 3} || len(v.Sl) != 2 || v.Sub.X != 1 || v.M["two"] != 2 {
		t.Fatalf("containers: %#v", v)
	}
}

// TestUnmarshalTypeMismatchErrors drives the type-mismatch error branches of
// the reflection decode path (including key highlighting for nested keys).
func TestUnmarshalTypeMismatchErrors(t *testing.T) {
	cases := []struct {
		desc string
		doc  string
		into func() interface{}
	}{
		{"string into int", `i = "x"`, func() interface{} { return &struct{ I int }{} }},
		{"int into string", `s = 1`, func() interface{} { return &struct{ S string }{} }},
		{"bool into float", `f = true`, func() interface{} { return &struct{ F float64 }{} }},
		{"array into int", `i = [1, 2]`, func() interface{} { return &struct{ I int }{} }},
		{"table into int", `i = { a = 1 }`, func() interface{} { return &struct{ I int }{} }},
		{"dotted into scalar", "i.a = 1", func() interface{} { return &struct{ I int }{} }},
		{"datetime into int", `i = 1979-05-27T07:32:00Z`, func() interface{} { return &struct{ I int }{} }},
		{"local date into int", `i = 1979-05-27`, func() interface{} { return &struct{ I int }{} }},
		{"local time into int", `i = 07:32:00`, func() interface{} { return &struct{ I int }{} }},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if err := toml.Unmarshal([]byte(c.doc), c.into()); err == nil {
				t.Fatalf("expected an error decoding %q", c.doc)
			}
		})
	}
}

// These tests exercise the reflection-based ("AST") generic decoding path,
// which handles generic values (interface{} / map[string]interface{}) that
// appear nested inside a non-generic target. Top-level generic targets take a
// dedicated single-pass path, so without these the nested path would be
// covered only indirectly.

// TestUnmarshalNestedGenericMap decodes into a map[string]interface{} struct
// field, driving the native nested-map descent for scalars, dotted keys,
// sub-tables, array tables, arrays and inline tables.
func TestUnmarshalNestedGenericMap(t *testing.T) {
	type S struct {
		Data map[string]interface{}
	}

	doc := `
[data]
s = "string"
i = 42
f = 3.5
b = true
odt = 1979-05-27T07:32:00Z
ldt = 1979-05-27T07:32:00
ld = 1979-05-27
lt = 07:32:00
arr = [1, 2, 3]
mixed = ["a", 1, true]
inline = { x = 1, y = 2 }
deep.one = "a"
deep.two = "b"

[data.sub]
nested = 1

[[data.items]]
n = 1

[[data.items]]
n = 2
`

	var s S
	if err := toml.Unmarshal([]byte(doc), &s); err != nil {
		t.Fatal(err)
	}

	m := s.Data
	if m["s"] != "string" {
		t.Fatalf("s = %#v", m["s"])
	}
	if m["i"] != int64(42) {
		t.Fatalf("i = %#v", m["i"])
	}
	if m["f"] != 3.5 {
		t.Fatalf("f = %#v", m["f"])
	}
	if m["b"] != true {
		t.Fatalf("b = %#v", m["b"])
	}
	if _, ok := m["odt"].(time.Time); !ok {
		t.Fatalf("odt = %#v", m["odt"])
	}
	if _, ok := m["ldt"].(toml.LocalDateTime); !ok {
		t.Fatalf("ldt = %#v", m["ldt"])
	}
	if _, ok := m["ld"].(toml.LocalDate); !ok {
		t.Fatalf("ld = %#v", m["ld"])
	}
	if _, ok := m["lt"].(toml.LocalTime); !ok {
		t.Fatalf("lt = %#v", m["lt"])
	}
	if arr, ok := m["arr"].([]interface{}); !ok || len(arr) != 3 {
		t.Fatalf("arr = %#v", m["arr"])
	}
	if inline, ok := m["inline"].(map[string]interface{}); !ok || inline["x"] != int64(1) {
		t.Fatalf("inline = %#v", m["inline"])
	}
	deep, ok := m["deep"].(map[string]interface{})
	if !ok || deep["one"] != "a" || deep["two"] != "b" {
		t.Fatalf("deep = %#v", m["deep"])
	}
	sub, ok := m["sub"].(map[string]interface{})
	if !ok || sub["nested"] != int64(1) {
		t.Fatalf("sub = %#v", m["sub"])
	}
	items, ok := m["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("items = %#v", m["items"])
	}
}

// TestUnmarshalInterfaceScalarFields decodes each scalar kind into an
// interface{} struct field, driving the interface branch of every scalar
// assignment.
func TestUnmarshalInterfaceScalarFields(t *testing.T) {
	type S struct {
		Str interface{} `toml:"str"`
		Int interface{} `toml:"int"`
		Flt interface{} `toml:"flt"`
		Bln interface{} `toml:"bln"`
		Odt interface{} `toml:"odt"`
		Ldt interface{} `toml:"ldt"`
		Ld  interface{} `toml:"ld"`
		Lt  interface{} `toml:"lt"`
		Arr interface{} `toml:"arr"`
		Tbl interface{} `toml:"tbl"`
	}

	doc := `
str = "hello"
int = 7
flt = 1.25
bln = false
odt = 1979-05-27T07:32:00Z
ldt = 1979-05-27T07:32:00
ld = 1979-05-27
lt = 07:32:00
arr = [1, 2]
tbl = { a = 1 }
`

	var s S
	if err := toml.Unmarshal([]byte(doc), &s); err != nil {
		t.Fatal(err)
	}
	if s.Str != "hello" || s.Int != int64(7) || s.Flt != 1.25 || s.Bln != false {
		t.Fatalf("scalars: %#v", s)
	}
	if _, ok := s.Odt.(time.Time); !ok {
		t.Fatalf("odt = %#v", s.Odt)
	}
	if _, ok := s.Ldt.(toml.LocalDateTime); !ok {
		t.Fatalf("ldt = %#v", s.Ldt)
	}
	if _, ok := s.Ld.(toml.LocalDate); !ok {
		t.Fatalf("ld = %#v", s.Ld)
	}
	if _, ok := s.Lt.(toml.LocalTime); !ok {
		t.Fatalf("lt = %#v", s.Lt)
	}
	if arr, ok := s.Arr.([]interface{}); !ok || len(arr) != 2 {
		t.Fatalf("arr = %#v", s.Arr)
	}
	if tbl, ok := s.Tbl.(map[string]interface{}); !ok || tbl["a"] != int64(1) {
		t.Fatalf("tbl = %#v", s.Tbl)
	}
}

// TestUnmarshalDottedInterfaceField decodes dotted keys into an interface{}
// field, driving the creation and reuse of a generic map held in an interface.
func TestUnmarshalDottedInterfaceField(t *testing.T) {
	type S struct {
		Prof interface{} `toml:"prof"`
	}

	var s S
	if err := toml.Unmarshal([]byte("prof.a = 1\nprof.b = 2\n"), &s); err != nil {
		t.Fatal(err)
	}
	m, ok := s.Prof.(map[string]interface{})
	if !ok || m["a"] != int64(1) || m["b"] != int64(2) {
		t.Fatalf("prof = %#v", s.Prof)
	}
}

// TestUnmarshalRedefinitionIntoStruct exercises the reflection decode path's
// handling of duplicate-key and table-redefinition errors, which are reported
// as DecodeError with the offending key and position.
func TestUnmarshalRedefinitionIntoStruct(t *testing.T) {
	type Inner struct {
		X int
	}
	type S struct {
		A int
		B Inner
		C map[string]interface{}
		T []Inner
	}

	cases := []struct {
		desc string
		doc  string
		msg  string
		key  toml.Key
	}{
		{"duplicate key", "a = 1\na = 2\n", "toml: key a is already defined", toml.Key{"a"}},
		{"duplicate dotted key", "b.x = 1\nb.x = 2\n", "toml: key x is already defined", toml.Key{"b", "x"}},
		{"redefined table", "[b]\nx = 1\n[b]\nx = 2\n", "toml: table b already exists", toml.Key{"b"}},
		{"table over value", "a = 1\n[a]\n", "toml: key a should be a table, not a value", toml.Key{"a"}},
		{"array table over table", "[b]\n[[b]]\n", "toml: key b already exists as a table, but should be an array table", toml.Key{"b"}},
		{"duplicate key in inline table", "c = { x = 1, x = 2 }\n", "toml: key x is already defined", toml.Key{"c"}},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var s S
			err := toml.Unmarshal([]byte(c.doc), &s)

			var de *toml.DecodeError
			if !errors.As(err, &de) {
				t.Fatalf("expected *toml.DecodeError, got %T (%v)", err, err)
			}
			if de.Error() != c.msg {
				t.Fatalf("message = %q, want %q", de.Error(), c.msg)
			}
			if got := de.Key(); !keyEqual(got, c.key) {
				t.Fatalf("key = %v, want %v", got, c.key)
			}
		})
	}
}

// TestUnmarshalIntoNamedKeyMap decodes a document with generic values into a
// map whose key type is a named string type. Because that type is not the
// exact map[string]interface{}, the whole document goes through the
// reflection-based decode path (table headers, array tables, dotted keys,
// scalar/array/inline-table values into interface{} elements) rather than the
// single-pass generic path.
func TestUnmarshalIntoNamedKeyMap(t *testing.T) {
	doc := `
s = "string"
i = 42
f = 3.5
b = true
odt = 1979-05-27T07:32:00Z
ldt = 1979-05-27T07:32:00
ld = 1979-05-27
lt = 07:32:00
arr = [1, 2, 3]
nested_arr = [[1, 2], [3, 4]]
inline = { x = 1, y = { z = 2 } }
dotted.a = 1
dotted.b = 2

[table]
k = "v"

[table.sub]
deep = true

[[items]]
n = 1

[[items]]
n = 2
`

	var m map[strKey]interface{}
	if err := toml.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatal(err)
	}

	if m["s"] != "string" || m["i"] != int64(42) || m["b"] != true {
		t.Fatalf("scalars: %#v", m)
	}
	if _, ok := m["odt"].(time.Time); !ok {
		t.Fatalf("odt = %#v", m["odt"])
	}
	if arr, ok := m["arr"].([]interface{}); !ok || len(arr) != 3 {
		t.Fatalf("arr = %#v", m["arr"])
	}
	if na, ok := m["nested_arr"].([]interface{}); !ok || len(na) != 2 {
		t.Fatalf("nested_arr = %#v", m["nested_arr"])
	}
	if inline, ok := m["inline"].(map[string]interface{}); !ok || inline["x"] != int64(1) {
		t.Fatalf("inline = %#v", m["inline"])
	}
	if tbl, ok := m["table"].(map[string]interface{}); !ok || tbl["k"] != "v" {
		t.Fatalf("table = %#v", m["table"])
	}
	if items, ok := m["items"].([]interface{}); !ok || len(items) != 2 {
		t.Fatalf("items = %#v", m["items"])
	}
}

// TestUnmarshalInvalidIntoStruct decodes malformed and conflicting documents
// into a struct target, which uses the reflection-based parser and seen-tracker
// (the single-pass generic path uses a different scanner), exercising their
// error branches.
func TestUnmarshalInvalidIntoStruct(t *testing.T) {
	docs := []string{
		// Structural parser errors.
		"[a",            // table name not closed
		"[[a]",          // array table name not closed (second ])
		"[[a",           // array table name not closed
		"[]",            // empty table name
		"a",             // key without '='
		"a =",           // value missing
		"a = 1 b",       // trailing characters after value
		"a = 1 #c\rd\n", // carriage return inside a comment
		"= 1",           // missing key
		"a..b = 1",      // empty dotted key part
		// Seen-tracker conflicts (reflection path).
		"[a]\nx = 1\n[a]\n",   // duplicate table
		"a = 1\n[a]\n",        // table over value
		"a.b = 1\n[a]\n",      // table over dotted-key table
		"[[a]]\n[a]\n",        // table over array table
		"a = 1\n[a.b]\n",      // table intermediate over value
		"[a]\n[[a]]\n",        // array table over table
		"a = 1\n[[a.b]]\n",    // array table intermediate over value
		"a = 1\na = 2\n",      // duplicate key
		"a = 1\na.b = 2\n",    // dotted key over value
		"a = { b = 1, b = 2}", // duplicate key in inline table
	}
	for _, doc := range docs {
		t.Run(doc, func(t *testing.T) {
			var s struct {
				A int
			}
			if err := toml.Unmarshal([]byte(doc), &s); err == nil {
				t.Fatalf("expected an error decoding %q", doc)
			}
		})
	}
}

// TestUnmarshalNumericForms decodes the integer and float syntaxes, including
// radix prefixes and the special floats, into a generic map.
func TestUnmarshalNumericForms(t *testing.T) {
	doc := `
hex = 0xDEADbeef
oct = 0o755
bin = 0b1101
under = 1_000_000
pos = +42
neg = -17
zero = 0
flt = 3.14
exp = 1e10
expsign = -2.5E-3
inf = inf
ninf = -inf
pinf = +inf
nan = nan
`
	var m map[string]interface{}
	if err := toml.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatal(err)
	}
	if m["hex"] != int64(0xDEADBEEF) || m["oct"] != int64(0o755) || m["bin"] != int64(0b1101) {
		t.Fatalf("radix: %#v", m)
	}
	if m["under"] != int64(1000000) || m["pos"] != int64(42) || m["neg"] != int64(-17) {
		t.Fatalf("ints: %#v", m)
	}
}

// TestUnmarshalInvalidNumbers decodes malformed numbers into a generic map,
// exercising the numeric scanner's error branches.
func TestUnmarshalInvalidNumbers(t *testing.T) {
	docs := []string{
		"a = +0x1",   // sign not allowed with radix prefix
		"a = 0x",     // radix prefix without digits
		"a = 0b2",    // invalid binary digit (prefix without valid digit)
		"a = 1__0",   // double underscore
		"a = 1.",     // decimal point without digit
		"a = 1e",     // exponent without digit
		"a = 01",     // leading zero
		"a = +",      // sign without number
		"a = i",      // truncated inf
		"a = 1e2000", // exponent overflows float64 (fast-path fallback)
	}
	for _, doc := range docs {
		t.Run(doc, func(t *testing.T) {
			var m map[string]interface{}
			if err := toml.Unmarshal([]byte(doc), &m); err == nil {
				t.Fatalf("expected an error decoding %q", doc)
			}
		})
	}
}

// TestUnmarshalStructFieldMatching exercises the struct field lookup paths:
// exact-case and case-folded matches, the byName-first path used when two
// fields fold to the same name, the in-place ASCII fold, and the non-ASCII and
// oversized-key fallbacks.
func TestUnmarshalStructFieldMatching(t *testing.T) {
	// Two fields folding to the same name force the byName-first resolution.
	type Collide struct {
		Lower int `toml:"foo"`
		Upper int `toml:"Foo"`
	}
	var c Collide
	if err := toml.Unmarshal([]byte("foo = 1\nFoo = 2\n"), &c); err != nil {
		t.Fatal(err)
	}
	if c.Lower != 1 || c.Upper != 2 {
		t.Fatalf("collide: %#v", c)
	}

	const long = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // > foldBufSize
	type Cases struct {
		MyField int // exact "MyField" and folded "myfield"
		Name    int // folded
		Naive   int `toml:"naïve"` // non-ASCII fold fallback
		Long    int `toml:"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
		Section struct {
			K int
		} `toml:"section"` // table header field lookup
	}
	doc := "MyField = 1\nNAME = 2\n\"NAÏVE\" = 3\n" +
		strings.ToUpper(long) + " = 4\n" +
		"[SECTION]\nk = 5\n"
	var cs Cases
	if err := toml.Unmarshal([]byte(doc), &cs); err != nil {
		t.Fatal(err)
	}
	if cs.MyField != 1 || cs.Name != 2 || cs.Naive != 3 || cs.Long != 4 || cs.Section.K != 5 {
		t.Fatalf("cases: %#v", cs)
	}
}

// TestUnmarshalBareCarriageReturn rejects a line that begins with a lone
// carriage return, exercising that branch of the generic document loop.
func TestUnmarshalBareCarriageReturn(t *testing.T) {
	var m map[string]interface{}
	if err := toml.Unmarshal([]byte("\rx = 1"), &m); err == nil {
		t.Fatal("expected an error for a lone carriage return")
	}
}

// TestUnmarshalLineEndingsAndHeaders covers CRLF and bare-CR handling on the
// reflection parser path, the empty array-table header error, multi-line inline
// tables, and table-header key folding for non-ASCII and oversized keys.
func TestUnmarshalLineEndingsAndHeaders(t *testing.T) {
	// CRLF line endings through the reflection (struct) parser.
	var s struct{ A, B int }
	if err := toml.Unmarshal([]byte("a = 1\r\nb = 2\r\n"), &s); err != nil {
		t.Fatal(err)
	}
	if s.A != 1 || s.B != 2 {
		t.Fatalf("crlf: %#v", s)
	}

	// A lone carriage return through the reflection parser is an error.
	if err := toml.Unmarshal([]byte("a = 1\rb = 2"), &struct{ A int }{}); err == nil {
		t.Fatal("expected an error for a lone carriage return")
	}

	// Empty array-table header name.
	if err := toml.Unmarshal([]byte("[[]]\n"), &struct{}{}); err == nil {
		t.Fatal("expected an error for an empty array-table header")
	}

	// Table-header key folding: oversized and non-ASCII (quoted) keys.
	type Hdr struct {
		Big struct {
			K int
		} `toml:"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
		Sec struct {
			K int
		} `toml:"séction"`
	}
	doc := "[AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA]\nk = 1\n" +
		"[\"SÉCTION\"]\nk = 2\n"
	var h Hdr
	if err := toml.Unmarshal([]byte(doc), &h); err != nil {
		t.Fatal(err)
	}
	if h.Big.K != 1 || h.Sec.K != 2 {
		t.Fatalf("headers: %#v", h)
	}

	// CRLF inside a multi-line inline table (TOML v1.1.0).
	var m map[string]interface{}
	if err := toml.Unmarshal([]byte("x = { a = 1,\r\n b = 2 }\n"), &m); err != nil {
		t.Fatal(err)
	}
	if inline, ok := m["x"].(map[string]interface{}); !ok || inline["b"] != int64(2) {
		t.Fatalf("inline = %#v", m["x"])
	}
}

// TestUnmarshalConflictsIntoNamedMap decodes conflicting documents into a
// named-key map (the reflection path) where, unlike a typed struct, the values
// do not type-clash before the seen-tracker runs, so its table/array-table/key
// conflict branches are reached.
func TestUnmarshalConflictsIntoNamedMap(t *testing.T) {
	docs := []string{
		"[a]\nx = 1\n[a]\n",  // duplicate table
		"a = 1\n[a]\n",       // table over value
		"a.b = 1\n[a]\n",     // table over dotted-key table
		"[[a]]\n[a]\n",       // table over array table
		"a = 1\n[a.b]\n",     // table intermediate over value
		"[a]\n[[a]]\n",       // array table over table
		"a = 1\n[[a.b]]\n",   // array table intermediate over value
		"a = 1\na = 2\n",     // duplicate key
		"a = 1\na.b = 2\n",   // dotted key over value
		"a = {b = 1, b = 2}", // duplicate key in inline table
	}
	for _, doc := range docs {
		t.Run(doc, func(t *testing.T) {
			var m map[strKey]interface{}
			if err := toml.Unmarshal([]byte(doc), &m); err == nil {
				t.Fatalf("expected an error decoding %q", doc)
			}
		})
	}
}

// TestUnmarshalFusedEdgeCases covers a few branches of the single-pass generic
// path: a blank CRLF line, a nested array table, and a float exponent large
// enough to fall back from the fast float parser.
func TestUnmarshalFusedEdgeCases(t *testing.T) {
	var m map[string]interface{}
	doc := "a = 1\r\n\r\nb = 2\r\n" + // blank CRLF line between key-values
		"[[g.h]]\nx = 1\n[[g.h]]\nx = 2\n" // nested array table
	if err := toml.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] != int64(1) || m["b"] != int64(2) {
		t.Fatalf("kv: %#v", m)
	}
	g, ok := m["g"].(map[string]interface{})
	if !ok {
		t.Fatalf("g = %#v", m["g"])
	}
	if h, ok := g["h"].([]interface{}); !ok || len(h) != 2 {
		t.Fatalf("g.h = %#v", g["h"])
	}
}

// TestUnmarshalTableIntoPrepopulatedGeneric decodes table headers into targets
// whose generic slots already hold incompatible content, exercising the
// branches that replace such content with a fresh table.
func TestUnmarshalTableIntoPrepopulatedGeneric(t *testing.T) {
	// An interface field holding a non-container is replaced by a table.
	a := struct {
		X interface{} `toml:"x"`
	}{X: 42}
	if err := toml.Unmarshal([]byte("[x]\nk = 1\n"), &a); err != nil {
		t.Fatal(err)
	}
	if m, ok := a.X.(map[string]interface{}); !ok || m["k"] != int64(1) {
		t.Fatalf("X = %#v", a.X)
	}

	// A generic-map element holding a scalar is replaced by a descending table.
	b := struct {
		M map[string]interface{} `toml:"m"`
	}{M: map[string]interface{}{"sub": 42}}
	if err := toml.Unmarshal([]byte("[m.sub]\nk = 1\n"), &b); err != nil {
		t.Fatal(err)
	}
	if sub, ok := b.M["sub"].(map[string]interface{}); !ok || sub["k"] != int64(1) {
		t.Fatalf("M.sub = %#v", b.M["sub"])
	}

	// A generic-map element holding a struct is likewise replaced.
	type inner struct{ Y int }
	c := struct {
		M map[string]interface{} `toml:"m"`
	}{M: map[string]interface{}{"sub": inner{Y: 9}}}
	if err := toml.Unmarshal([]byte("[m.sub]\nk = 1\n"), &c); err != nil {
		t.Fatal(err)
	}
	if sub, ok := c.M["sub"].(map[string]interface{}); !ok || sub["k"] != int64(1) {
		t.Fatalf("M.sub (struct) = %#v", c.M["sub"])
	}
}

func keyEqual(a, b toml.Key) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
