package toml

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

// TestUnmarshalManyKeysTable exercises the seen-tracker's spill to its hash
// index (tables with more keys than fit the sibling chains), for both the
// fused generic path and the reflection path, including duplicate detection
// after the spill.
func TestUnmarshalManyKeysTable(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&sb, "key%d = %d\n", i, i)
	}
	doc := sb.String()

	t.Run("map", func(t *testing.T) {
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(doc), &m))
		assert.Equal(t, 100, len(m))
		assert.Equal(t, interface{}(int64(99)), m["key99"])
	})

	t.Run("struct", func(t *testing.T) {
		var s struct {
			Key0  int64
			Key42 int64
			Key99 int64
		}
		assert.NoError(t, Unmarshal([]byte(doc), &s))
		assert.Equal(t, int64(42), s.Key42)
		assert.Equal(t, int64(99), s.Key99)
	})

	t.Run("duplicate after spill", func(t *testing.T) {
		m := map[string]interface{}{}
		err := Unmarshal([]byte(doc+"key12 = 1\n"), &m)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "already"), "unexpected error: %v", err)
	})

	t.Run("array table refresh under root", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 40; i++ {
			fmt.Fprintf(&sb, "key%d = %d\n", i, i)
		}
		for i := 0; i < 3; i++ {
			sb.WriteString("[[elem]]\nname = 'x'\n")
		}
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(sb.String()), &m))
		assert.Equal(t, 3, len(m["elem"].([]interface{})))
	})
}

// TestUnmarshalGenericValueShapes covers the generic decoding of every scalar
// kind (through interface{} struct fields, which use the AST path), long
// strings beyond the slab limit, and arrays larger than the slab cutoff.
func TestUnmarshalGenericValueShapes(t *testing.T) {
	long := strings.Repeat("x", 600)
	var arr strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&arr, "%d,", i)
	}
	doc := fmt.Sprintf(`
str = "hello"
longstr = "%s"
int = 42
bigint = 123456
negint = -7
float = 1.25
bool = true
date = 2021-03-30
time = 11:21:00
datetime = 2021-03-30T11:21:00Z
localdt = 2021-03-30T11:21:00
bigarray = [%s]
nested = { a.b = 1, c = [{ d = 2 }, { d = 3 }] }
`, long, arr.String())

	type target struct {
		Str      interface{}
		Longstr  interface{}
		Int      interface{}
		Bigint   interface{}
		Negint   interface{}
		Float    interface{}
		Bool     interface{}
		Date     interface{}
		Time     interface{}
		Datetime interface{}
		Localdt  interface{}
		Bigarray interface{}
		Nested   interface{}
	}

	check := func(t *testing.T, get func(string) interface{}) {
		t.Helper()
		assert.Equal(t, interface{}("hello"), get("str"))
		assert.Equal(t, interface{}(long), get("longstr"))
		assert.Equal(t, interface{}(int64(42)), get("int"))
		assert.Equal(t, interface{}(int64(123456)), get("bigint"))
		assert.Equal(t, interface{}(int64(-7)), get("negint"))
		assert.Equal(t, interface{}(1.25), get("float"))
		assert.Equal(t, interface{}(true), get("bool"))
		assert.Equal(t, interface{}(LocalDate{2021, 3, 30}), get("date"))
		assert.Equal(t, 40, len(get("bigarray").([]interface{})))
		nested := get("nested").(map[string]interface{})
		assert.Equal(t, interface{}(int64(1)), nested["a"].(map[string]interface{})["b"])
		cs := nested["c"].([]interface{})
		assert.Equal(t, interface{}(int64(3)), cs[1].(map[string]interface{})["d"])
		dt := get("datetime").(time.Time)
		assert.Equal(t, 2021, dt.Year())
	}

	t.Run("map target (fused path)", func(t *testing.T) {
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(doc), &m))
		check(t, func(k string) interface{} { return m[k] })
	})

	t.Run("map target with unmarshaler interface (AST path)", func(t *testing.T) {
		m := map[string]interface{}{}
		d := NewDecoder(strings.NewReader(doc))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&m))
		check(t, func(k string) interface{} { return m[k] })
	})

	t.Run("struct with interface fields (AST path)", func(t *testing.T) {
		var s target
		d := NewDecoder(strings.NewReader(doc))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&s))
		byName := map[string]interface{}{
			"str": s.Str, "longstr": s.Longstr, "int": s.Int,
			"bigint": s.Bigint, "negint": s.Negint, "float": s.Float,
			"bool": s.Bool, "date": s.Date, "time": s.Time,
			"datetime": s.Datetime, "localdt": s.Localdt,
			"bigarray": s.Bigarray, "nested": s.Nested,
		}
		check(t, func(k string) interface{} { return byName[k] })
	})
}

// TestUnmarshalFusedLineEndings covers CRLF and comment handling inside the
// natively decoded containers, and bare-CR rejection.
func TestUnmarshalFusedLineEndings(t *testing.T) {
	m := map[string]interface{}{}
	doc := "a = [ # comment\r\n1, # c\r\n2 ]\r\nb = { # comment\r\nx = 1, # c\r\n}\r\n"
	assert.NoError(t, Unmarshal([]byte(doc), &m))
	assert.Equal(t, 2, len(m["a"].([]interface{})))
	assert.Equal(t, interface{}(int64(1)), m["b"].(map[string]interface{})["x"])

	for _, bad := range []string{"a = [1,\r2]", "a = {x = 1,\ry = 2}"} {
		assert.Error(t, Unmarshal([]byte(bad), &map[string]interface{}{}))
	}
}

// TestUnmarshalFusedStructErrors covers the scanning error branches of the
// fused document loop for reflection targets, and the strict-mode reporting
// on both the fused path and the AST path (unmarshaler interface enabled).
func TestUnmarshalFusedStructErrors(t *testing.T) {
	type target struct {
		A int64 `toml:"a"`
	}

	bad := []string{
		"a = 1\rb = 2",   // bare CR between expressions
		"# comment\rx",   // bare CR terminating a comment
		"[tbl\na = 1",    // header syntax error through the delegated parser
		"a = 1 trailing", // garbage after a scalar value
		"a.b = 1",        // dotted key descending into an integer field
	}
	for _, doc := range bad {
		var s target
		assert.Error(t, Unmarshal([]byte(doc), &s), "expected error for %q", doc)
	}

	t.Run("strict unknown field fused", func(t *testing.T) {
		var s target
		d := NewDecoder(strings.NewReader("a = 1\nunknown = 2\nun.known = 3\n"))
		d.DisallowUnknownFields()
		err := d.Decode(&s)
		assert.Error(t, err)
		var missing *StrictMissingError
		assert.True(t, errors.As(err, &missing))
		assert.Equal(t, 2, len(missing.Errors))
	})

	t.Run("strict unknown field with unmarshaler interface", func(t *testing.T) {
		var s target
		d := NewDecoder(strings.NewReader("a = 1\nunknown = 2\n"))
		d.DisallowUnknownFields()
		d.EnableUnmarshalerInterface()
		err := d.Decode(&s)
		assert.Error(t, err)
		var missing *StrictMissingError
		assert.True(t, errors.As(err, &missing))
		assert.Equal(t, 1, len(missing.Errors))
	})

	t.Run("fixed array overflow via dotted key", func(t *testing.T) {
		var s struct {
			Elem [1]struct{ V int64 }
		}
		err := Unmarshal([]byte("[[elem]]\nv = 1\n[[elem]]\nv = 2\n"), &s)
		assert.Error(t, err)
	})
}

// TestUnmarshalMoreErrorBranches sweeps assorted error and edge branches of
// the fused paths and integer parsing.
func TestUnmarshalMoreErrorBranches(t *testing.T) {
	type target struct {
		A []int64                     `toml:"a"`
		M map[string]int64            `toml:"m"`
		N map[string]map[string]int64 `toml:"n"`
	}

	bad := []string{
		"\rx = 1",            // CR at start of expression (struct loop)
		"a = [1,",            // container syntax error (struct target)
		"a = [1] x",          // garbage after container
		"b = 1\nb = [1]",     // duplicate key with container value
		"m = {x = 1, x = 2}", // duplicate inside container (struct target)
	}
	for _, doc := range bad {
		var s target
		assert.Error(t, Unmarshal([]byte(doc), &s), "expected error for %q", doc)
	}

	overflow := []string{
		"i = 0xFFFFFFFFFFFFFFFF",           // hex overflow
		"i = 0o7777777777777777777777",     // octal overflow
		"i = 0b" + strings.Repeat("1", 65), // binary overflow
	}
	for _, doc := range overflow {
		m := map[string]interface{}{}
		assert.Error(t, Unmarshal([]byte(doc), &m), "expected error for %q", doc)
	}

	good := []string{
		"\r\na = [1]\r\n",            // CRLF blank line handling in the struct loop
		"[m]\r\nk = 1",               // CRLF after header
		"[n.x]\nk = 1\n[n.y]\nk = 2", // dotted headers into nested maps
	}
	for _, doc := range good {
		var s target
		assert.NoError(t, Unmarshal([]byte(doc), &s), "expected success for %q", doc)
	}

	t.Run("unmarshaler interface without strict ignores unknowns", func(t *testing.T) {
		var s struct{ A int64 }
		d := NewDecoder(strings.NewReader("a = 1\nunknown = 2"))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&s))
		assert.Equal(t, int64(1), s.A)
	})

	t.Run("nil map field filled through cached table", func(t *testing.T) {
		var s struct {
			M map[string]int64 `toml:"m"`
		}
		assert.NoError(t, Unmarshal([]byte("[m]\nk = 1\nl = 2"), &s))
		assert.Equal(t, int64(2), s.M["l"])
	})
}

// TestUnmarshalReplacePreexistingValues covers replacing pre-existing values
// held in interface fields and AST-path assignment errors.
func TestUnmarshalReplacePreexistingValues(t *testing.T) {
	t.Run("table replaces scalar in interface field", func(t *testing.T) {
		var s struct{ T interface{} }
		s.T = "old scalar"
		assert.NoError(t, Unmarshal([]byte("[t]\nk = 1"), &s))
		m, ok := s.T.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, interface{}(int64(1)), m["k"])
	})

	t.Run("type mismatch through unmarshaler interface path", func(t *testing.T) {
		var s struct{ A int64 }
		d := NewDecoder(strings.NewReader(`a = "not an int"`))
		d.EnableUnmarshalerInterface()
		assert.Error(t, d.Decode(&s))
	})

	t.Run("duplicate key through unmarshaler interface path", func(t *testing.T) {
		var s struct{ A int64 }
		d := NewDecoder(strings.NewReader("a = 1\na = 2"))
		d.EnableUnmarshalerInterface()
		assert.Error(t, d.Decode(&s))
	})
}

// TestUnmarshalCommentAndDottedStrictEdges covers comment scanning errors in
// the fused loops and dotted-key strict reporting through the AST path.
func TestUnmarshalCommentAndDottedStrictEdges(t *testing.T) {
	for _, doc := range []string{"# c\rx = 1", "a = {#\x01\nx = 1}"} {
		m := map[string]interface{}{}
		assert.Error(t, Unmarshal([]byte(doc), &m), "expected error for %q", doc)
	}

	var s struct{ A int64 }
	d := NewDecoder(strings.NewReader("un.known.key = 1"))
	d.DisallowUnknownFields()
	d.EnableUnmarshalerInterface()
	err := d.Decode(&s)
	var missing *StrictMissingError
	assert.True(t, errors.As(err, &missing))
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

// TestUnmarshalerInterfaceLoopBranches covers the expression-loop branches
// that only the unmarshaler-interface path uses: parser error wrapping and
// the empty-document epilogue for generic roots.
func TestUnmarshalerInterfaceLoopBranches(t *testing.T) {
	t.Run("syntax error", func(t *testing.T) {
		var s struct{ A int64 }
		d := NewDecoder(strings.NewReader("[unclosed\na = 1"))
		d.EnableUnmarshalerInterface()
		assert.Error(t, d.Decode(&s))
	})

	t.Run("empty document into nil map", func(t *testing.T) {
		var m map[string]interface{}
		d := NewDecoder(strings.NewReader(""))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&m))
		assert.True(t, m != nil)
	})

	t.Run("empty document into interface", func(t *testing.T) {
		var v interface{}
		d := NewDecoder(strings.NewReader(""))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&v))
		_, ok := v.(map[string]interface{})
		assert.True(t, ok)
	})
}

// TestValidateValueNodeScalar covers the guard for non-container nodes: only
// containers declare keys, so scalars validate trivially.
func TestValidateValueNodeScalar(t *testing.T) {
	d := getDecoder(false, false)
	defer putDecoder(d)
	assert.NoError(t, d.validateValueNode(&unstable.Node{Kind: unstable.String}))
	assert.NoError(t, d.validateValueNode(&unstable.Node{Kind: unstable.Integer}))
}

// lyingLenReader reports a smaller Len than it will deliver, exercising the
// finish-with-ReadAll branch of readDocument.
type lyingLenReader struct {
	r io.Reader
}

func (l *lyingLenReader) Read(p []byte) (int, error) { return l.r.Read(p) }
func (l *lyingLenReader) Len() int                   { return 2 }

// TestReadDocumentGrowingReader covers readers whose Len underestimates the
// actual content.
func TestReadDocumentGrowingReader(t *testing.T) {
	var m map[string]interface{}
	d := NewDecoder(&lyingLenReader{r: strings.NewReader("key = 'longer than two bytes'")})
	assert.NoError(t, d.Decode(&m))
	assert.Equal(t, interface{}("longer than two bytes"), m["key"])
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

// TestUnmarshalerInterfaceDottedGeneric covers setAnyKey's dotted-key walk
// and error propagation on the AST generic path.
func TestUnmarshalerInterfaceDottedGeneric(t *testing.T) {
	m := map[string]interface{}{}
	d := NewDecoder(strings.NewReader("a.b.c = 1\na.b.d = 'x'"))
	d.EnableUnmarshalerInterface()
	assert.NoError(t, d.Decode(&m))
	ab := m["a"].(map[string]interface{})["b"].(map[string]interface{})
	assert.Equal(t, interface{}(int64(1)), ab["c"])

	m2 := map[string]interface{}{}
	d2 := NewDecoder(strings.NewReader("a.b = 2021-13-45"))
	d2.EnableUnmarshalerInterface()
	assert.Error(t, d2.Decode(&m2))
}

// TestArrayTableRefreshUnderSpilledParent covers refreshing an array table
// whose parent's children have spilled to the tracker's hash index, and the
// sibling-chain swap when they have not.
func TestArrayTableRefreshUnderSpilledParent(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 70; i++ {
		fmt.Fprintf(&sb, "key%d = %d\n", i, i)
	}
	sb.WriteString("[[t]]\nx = 1\n[[t]]\nx = 2\n")
	m := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(sb.String()), &m))
	assert.Equal(t, 2, len(m["t"].([]interface{})))

	// Unspilled parent, refresh target not at the chain head.
	doc := "a = 1\n[[t]]\nx = 1\nb = 2\n[[t]]\nx = 2\n"
	m2 := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(doc), &m2))
	assert.Equal(t, 2, len(m2["t"].([]interface{})))
}

type errAfterReader struct{ n int }

func (e *errAfterReader) Read(p []byte) (int, error) {
	if e.n == 0 {
		return 0, errors.New("boom")
	}
	p[0] = 'a'
	e.n--
	return 1, nil
}
func (e *errAfterReader) Len() int { return 8 }

// TestReadDocumentReadError covers the error branch of the sized read.
func TestReadDocumentReadError(t *testing.T) {
	var m map[string]interface{}
	d := NewDecoder(&errAfterReader{n: 1})
	assert.Error(t, d.Decode(&m))
}

// TestSetAnyKeyValueError covers error propagation through the AST generic
// inline-table walk (an impossible date inside a nested inline table).
func TestSetAnyKeyValueError(t *testing.T) {
	var s struct{ V interface{} }
	d := NewDecoder(strings.NewReader("v = { nested = { d = 2021-13-45 } }"))
	d.EnableUnmarshalerInterface()
	assert.Error(t, d.Decode(&s))
}

// TestRawValueWithoutSpan covers the fallback of rawValue when neither an
// expression node nor a fused value span is available.
func TestRawValueWithoutSpan(t *testing.T) {
	d := getDecoder(false, false)
	defer putDecoder(d)
	doc := []byte("x = [1]")
	d.p.Reset(doc)
	d.fusedValueSpan = nil
	node := &unstable.Node{Kind: unstable.Array, Raw: d.p.Range(doc[4:7])}
	assert.Equal(t, "[1]", string(d.rawValue(nil, node)))
}

// TestSetAnyKeyBranches covers setAnyKey (the generic decode of inline
// tables reached through struct-held maps): dotted keys and error
// propagation.
func TestSetAnyKeyBranches(t *testing.T) {
	type target struct {
		M map[string]interface{} `toml:"m"`
	}

	var s target
	assert.NoError(t, Unmarshal([]byte("[m]\nv = { a.b = 1, c = 2 }"), &s))
	v := s.M["v"].(map[string]interface{})
	assert.Equal(t, interface{}(int64(1)),
		v["a"].(map[string]interface{})["b"])
	assert.Equal(t, interface{}(int64(2)), v["c"])

	var s2 target
	assert.Error(t, Unmarshal([]byte("[m]\nv = { d = 2021-13-45 }"), &s2))
}

// TestRawValueWithSpan covers the fused-path branch of rawValue, which
// returns the exact span of the current key-value's container.
func TestRawValueWithSpan(t *testing.T) {
	d := getDecoder(false, false)
	defer putDecoder(d)
	doc := []byte("x = [1]")
	d.p.Reset(doc)
	d.fusedValueSpan = doc[4:7]
	node := &unstable.Node{Kind: unstable.Array, Raw: d.p.Range(doc[4:7])}
	assert.Equal(t, "[1]", string(d.rawValue(nil, node)))
	d.fusedValueSpan = nil

	// A non-key-value expression context takes the best-effort span.
	arrExpr := &unstable.Node{Kind: unstable.Array}
	assert.Equal(t, "[1]", string(d.rawValue(arrExpr, node)))
}

// TestUnmarshalerInterfaceNodePaths drives the node-driven expression loop
// (only used with the unmarshaler interface) through its error and
// line-ending branches, raw captures stored under maps, and pre-existing
// interface values replaced by tables.
func TestUnmarshalerInterfaceNodePaths(t *testing.T) {
	bad := []string{
		"\rx = 1",             // bare CR at expression level
		"[a]\n[a]",            // duplicate table
		"[[a]]\n[a]",          // table after array table
		"a.b = 1\n[a.b]",      // table over dotted key
		"a = 1\n[a]",          // table over value
		"[a]\n[[a]]",          // array table over table
		"a = {x = 1,\rb = 2}", // bare CR in inline table
	}
	for _, doc := range bad {
		var s struct{ A int64 }
		d := NewDecoder(strings.NewReader(doc))
		d.EnableUnmarshalerInterface()
		assert.Error(t, d.Decode(&s), "expected error for %q", doc)
	}

	t.Run("crlf in inline table", func(t *testing.T) {
		m := map[string]interface{}{}
		d := NewDecoder(strings.NewReader("a = {x = 1,\r\ny = 2}"))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&m))
		assert.Equal(t, 2, len(m["a"].(map[string]interface{})))
	})

	t.Run("raw capture under map", func(t *testing.T) {
		var s struct {
			M map[string]unstable.RawMessage `toml:"m"`
		}
		d := NewDecoder(strings.NewReader("[m.x]\na = 1\n[m.y]\nb = 2"))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&s))
		assert.True(t, strings.Contains(string(s.M["x"]), "a = 1"), "got %q", s.M["x"])
	})

	t.Run("table replaces scalar interface via node walk", func(t *testing.T) {
		var s struct{ T interface{} }
		s.T = "old"
		d := NewDecoder(strings.NewReader("[t]\nk = 1"))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&s))
		m, ok := s.T.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, interface{}(int64(1)), m["k"])
	})
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

// TestParseIntegerOverflows covers the radix-specific overflow guards
// directly (the scanner accepts the tokens; conversion rejects them).
func TestParseIntegerOverflows(t *testing.T) {
	for _, s := range []string{
		"0xFFFFFFFFFFFFFFFF",
		"0o1777777777777777777777",
		"0b" + strings.Repeat("1", 64),
	} {
		_, err := parseInteger([]byte(s))
		assert.Error(t, err, "expected overflow for %q", s)
	}
}

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

// TestMoreLineEndingAndHeaderEdges sweeps remaining line-ending and header
// branches on both document loops.
func TestMoreLineEndingAndHeaderEdges(t *testing.T) {
	t.Run("crlf blank line, interface loop", func(t *testing.T) {
		m := map[string]interface{}{}
		d := NewDecoder(strings.NewReader("\r\na = 1\r\n"))
		d.EnableUnmarshalerInterface()
		assert.NoError(t, d.Decode(&m))
		assert.Equal(t, interface{}(int64(1)), m["a"])
	})

	t.Run("invalid comment, fused generic loop", func(t *testing.T) {
		m := map[string]interface{}{}
		assert.Error(t, Unmarshal([]byte("# \x01\nx = 1"), &m))
	})

	t.Run("array table over scalar interface", func(t *testing.T) {
		var s struct{ T interface{} }
		s.T = "old"
		assert.NoError(t, Unmarshal([]byte("[[t]]\nk = 1\n[[t]]\nk = 2"), &s))
		arr, ok := s.T.([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(arr))
	})
}

// TestFastParseFloatRejects covers the scanner-shape early-outs that the
// parser normally filters before the fast path sees them.
func TestFastParseFloatRejects(t *testing.T) {
	for _, s := range []string{"1..2", "-e1", "1e+"} {
		_, ok := fastParseFloat([]byte(s))
		assert.False(t, ok, "expected reject for %q", s)
	}
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

// TestTableThroughScalarInterface covers replacing a scalar held in an
// interface when it is an intermediate step of a deeper table header.
func TestTableThroughScalarInterface(t *testing.T) {
	var s struct{ T interface{} }
	s.T = "old scalar"
	assert.NoError(t, Unmarshal([]byte("[t.sub]\nk = 1"), &s))
	m := s.T.(map[string]interface{})
	sub := m["sub"].(map[string]interface{})
	assert.Equal(t, interface{}(int64(1)), sub["k"])
}
