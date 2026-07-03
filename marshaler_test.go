package toml_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

type marshalTextKey struct {
	A string
	B string
}

func (k marshalTextKey) MarshalText() ([]byte, error) {
	return []byte(k.A + "-" + k.B), nil
}

type marshalBadTextKey struct{}

func (k marshalBadTextKey) MarshalText() ([]byte, error) {
	return nil, errors.New("error")
}

func toFloat(x interface{}) float64 {
	// Shortened version of testify/toFloat
	var xf float64
	switch xn := x.(type) {
	case float32:
		xf = float64(xn)
	case float64:
		xf = xn
	}
	return xf
}

func inDelta(t *testing.T, expected, actual interface{}, delta float64) {
	t.Helper()
	dt := toFloat(expected) - toFloat(actual)
	assert.True(t,
		dt < -delta && dt < delta,
		"Difference between %v and %v is %v, but difference was %v",
		expected, actual, delta, dt,
	)
}

func TestMarshal(t *testing.T) {
	someInt := 42

	type structInline struct {
		A interface{} `toml:",inline"`
	}

	type comments struct {
		One   int
		Two   int   `comment:"Before kv"`
		Three []int `comment:"Before array"`
	}

	examples := []struct {
		desc     string
		v        interface{}
		expected string
		err      bool
	}{
		{
			desc: "simple map and string",
			v: map[string]string{
				"hello": "world",
			},
			expected: "hello = 'world'\n",
		},
		{
			desc: "map with new line in key",
			v: map[string]string{
				"hel\nlo": "world",
			},
			expected: "\"hel\\nlo\" = 'world'\n",
		},
		{
			desc: `map with " in key`,
			v: map[string]string{
				`hel"lo`: "world",
			},
			expected: "'hel\"lo' = 'world'\n",
		},
		{
			desc: "map in map and string",
			v: map[string]map[string]string{
				"table": {
					"hello": "world",
				},
			},
			expected: `[table]
hello = 'world'
`,
		},
		{
			desc: "map in map in map and string",
			v: map[string]map[string]map[string]string{
				"this": {
					"is": {
						"a": "test",
					},
				},
			},
			expected: `[this]
[this.is]
a = 'test'
`,
		},
		{
			desc: "map in map in map and string with values",
			v: map[string]interface{}{
				"this": map[string]interface{}{
					"is": map[string]string{
						"a": "test",
					},
					"also": "that",
				},
			},
			expected: `[this]
also = 'that'

[this.is]
a = 'test'
`,
		},
		{
			desc: `map with text key`,
			v: map[marshalTextKey]string{
				{A: "a", B: "1"}: "value 1",
				{A: "a", B: "2"}: "value 2",
				{A: "b", B: "1"}: "value 3",
			},
			expected: `a-1 = 'value 1'
a-2 = 'value 2'
b-1 = 'value 3'
`,
		},
		{
			desc: `table with text key`,
			v: map[marshalTextKey]map[string]string{
				{A: "a", B: "1"}: {"value": "foo"},
			},
			expected: `[a-1]
value = 'foo'
`,
		},
		{
			desc: `map with ptr text key`,
			v: map[*marshalTextKey]string{
				{A: "a", B: "1"}: "value 1",
				{A: "a", B: "2"}: "value 2",
				{A: "b", B: "1"}: "value 3",
			},
			expected: `a-1 = 'value 1'
a-2 = 'value 2'
b-1 = 'value 3'
`,
		},
		{
			desc: `map with bad text key`,
			v: map[marshalBadTextKey]string{
				{}: "value 1",
			},
			err: true,
		},
		{
			desc: `map with bad ptr text key`,
			v: map[*marshalBadTextKey]string{
				{}: "value 1",
			},
			err: true,
		},
		{
			desc: "simple string array",
			v: map[string][]string{
				"array": {"one", "two", "three"},
			},
			expected: `array = ['one', 'two', 'three']
`,
		},
		{
			desc:     "empty string array",
			v:        map[string][]string{},
			expected: ``,
		},
		{
			desc:     "map",
			v:        map[string][]string{},
			expected: ``,
		},
		{
			desc: "nested string arrays",
			v: map[string][][]string{
				"array": {{"one", "two"}, {"three"}},
			},
			expected: `array = [['one', 'two'], ['three']]
`,
		},
		{
			desc: "mixed strings and nested string arrays",
			v: map[string][]interface{}{
				"array": {"a string", []string{"one", "two"}, "last"},
			},
			expected: `array = ['a string', ['one', 'two'], 'last']
`,
		},
		{
			desc: "array of maps",
			v: map[string][]map[string]string{
				"top": {
					{"map1.1": "v1.1"},
					{"map2.1": "v2.1"},
				},
			},
			expected: `[[top]]
'map1.1' = 'v1.1'

[[top]]
'map2.1' = 'v2.1'
`,
		},
		{
			desc: "fixed size string array",
			v: map[string][3]string{
				"array": {"one", "two", "three"},
			},
			expected: `array = ['one', 'two', 'three']
`,
		},
		{
			desc: "fixed size nested string arrays",
			v: map[string][2][2]string{
				"array": {{"one", "two"}, {"three"}},
			},
			expected: `array = [['one', 'two'], ['three', '']]
`,
		},
		{
			desc: "mixed strings and fixed size nested string arrays",
			v: map[string][]interface{}{
				"array": {"a string", [2]string{"one", "two"}, "last"},
			},
			expected: `array = ['a string', ['one', 'two'], 'last']
`,
		},
		{
			desc: "fixed size array of maps",
			v: map[string][2]map[string]string{
				"ftop": {
					{"map1.1": "v1.1"},
					{"map2.1": "v2.1"},
				},
			},
			expected: `[[ftop]]
'map1.1' = 'v1.1'

[[ftop]]
'map2.1' = 'v2.1'
`,
		},
		{
			desc: "map with two keys",
			v: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: `key1 = 'value1'
key2 = 'value2'
`,
		},
		{
			desc: "simple struct",
			v: struct {
				A string
			}{
				A: "foo",
			},
			expected: `A = 'foo'
`,
		},
		{
			desc: "one level of structs within structs",
			v: struct {
				A interface{}
			}{
				A: struct {
					K1 string
					K2 string
				}{
					K1: "v1",
					K2: "v2",
				},
			},
			expected: `[A]
K1 = 'v1'
K2 = 'v2'
`,
		},
		{
			desc: "structs in array with interfaces",
			v: map[string]interface{}{
				"root": map[string]interface{}{
					"nested": []interface{}{
						map[string]interface{}{"name": "Bob"},
						map[string]interface{}{"name": "Alice"},
					},
				},
			},
			expected: `[root]
[[root.nested]]
name = 'Bob'

[[root.nested]]
name = 'Alice'
`,
		},
		{
			desc: "string escapes",
			v: map[string]interface{}{
				"a": "'\b\f\r\t\"\\",
			},
			expected: `a = "'\b\f\r\t\"\\"
`,
		},
		{
			desc: "string utf8 low",
			v: map[string]interface{}{
				"a": "'Ę",
			},
			expected: `a = "'Ę"
`,
		},
		{
			desc: "string utf8 low 2",
			v: map[string]interface{}{
				"a": "'\u10A85",
			},
			expected: "a = \"'\u10A85\"\n",
		},
		{
			desc: "string utf8 low 2",
			v: map[string]interface{}{
				"a": "'\u10A85",
			},
			expected: "a = \"'\u10A85\"\n",
		},
		{
			desc: "emoji",
			v: map[string]interface{}{
				"a": "'😀",
			},
			expected: "a = \"'😀\"\n",
		},
		{
			desc: "control char",
			v: map[string]interface{}{
				"a": "'\u001A",
			},
			expected: `a = "'\u001A"
`,
		},
		{
			desc: "multi-line string",
			v: map[string]interface{}{
				"a": "hello\nworld",
			},
			expected: `a = "hello\nworld"
`,
		},
		{
			desc: "multi-line forced",
			v: struct {
				A string `toml:",multiline"`
			}{
				A: "hello\nworld",
			},
			expected: `A = """
hello
world"""
`,
		},
		{
			desc: "multi-line quotation",
			v: struct {
				A string `toml:",multiline"`
			}{
				A: "hello\n\"world\"",
			},
			expected: `A = """
hello
"world""""
`,
		},
		{
			desc: "multi-line triple quotation",
			v: struct {
				A string `toml:",multiline"`
			}{
				A: "hello\n\"\"\"world\"",
			},
			expected: `A = """
hello
\"\"\"world""""
`,
		},
		{
			desc: "multi-line triple quotation",
			v: struct {
				A string `toml:",multiline"`
			}{
				A: "hello\n\"world\"\"\"",
			},
			expected: `A = """
hello
"world\"\"\""""
`,
		},
		{
			desc: "multi-line sextuple quotation",
			v: struct {
				A string `toml:",multiline"`
			}{
				A: "hello\n\"\"\"\"\"\"world\"",
			},
			expected: `A = """
hello
\"\"\"\"\"\"world""""
`,
		},
		{
			// Regression for #1075: the multiline option must not wrap a
			// single-line value in a """...""" block.
			desc: "multi-line option ignored without newline",
			v: struct {
				A string `toml:",multiline"`
				B string `toml:",multiline"`
			}{
				A: "2",
				B: "with ' quote",
			},
			expected: `A = '2'
B = "with ' quote"
`,
		},
		{
			desc: "inline field",
			v: struct {
				A map[string]string `toml:",inline"`
				B map[string]string
			}{
				A: map[string]string{
					"isinline": "yes",
				},
				B: map[string]string{
					"isinline": "no",
				},
			},
			expected: `A = {isinline = 'yes'}

[B]
isinline = 'no'
`,
		},
		{
			desc: "mutiline array int",
			v: struct {
				A []int `toml:",multiline"`
				B []int
			}{
				A: []int{1, 2, 3, 4},
				B: []int{1, 2, 3, 4},
			},
			expected: `A = [
  1,
  2,
  3,
  4
]
B = [1, 2, 3, 4]
`,
		},
		{
			desc: "mutiline array in array",
			v: struct {
				A [][]int `toml:",multiline"`
			}{
				A: [][]int{{1, 2}, {3, 4}},
			},
			expected: `A = [
  [1, 2],
  [3, 4]
]
`,
		},
		{
			desc: "nil interface not supported at root",
			v:    nil,
			err:  true,
		},
		{
			desc: "nil interface not supported in slice",
			v: map[string]interface{}{
				"a": []interface{}{"a", nil, 2},
			},
			err: true,
		},
		{
			desc: "nil pointer in slice uses zero value",
			v: struct {
				A []*int
			}{
				A: []*int{nil},
			},
			expected: `A = [0]
`,
		},
		{
			desc: "nil pointer in slice uses zero value",
			v: struct {
				A []*int
			}{
				A: []*int{nil},
			},
			expected: `A = [0]
`,
		},
		{
			desc: "pointer in slice",
			v: struct {
				A []*int
			}{
				A: []*int{&someInt},
			},
			expected: `A = [42]
`,
		},
		{
			desc: "inline table in inline table",
			v: structInline{
				A: structInline{
					A: structInline{
						A: "hello",
					},
				},
			},
			expected: `A = {A = {A = 'hello'}}
`,
		},
		{
			desc: "empty slice in map",
			v: map[string][]string{
				"a": {},
			},
			expected: `a = []
`,
		},
		{
			desc: "map in slice",
			v: map[string][]map[string]string{
				"a": {{"hello": "world"}},
			},
			expected: `[[a]]
hello = 'world'
`,
		},
		{
			desc: "newline in map in slice",
			v: map[string][]map[string]string{
				"a\n": {{"hello": "world"}},
			},
			expected: `[["a\n"]]
hello = 'world'
`,
		},
		{
			desc: "newline in map in slice",
			v: map[string][]map[string]*customTextMarshaler{
				"a": {{"hello": &customTextMarshaler{1}}},
			},
			err: true,
		},
		{
			desc: "empty slice of empty struct",
			v: struct {
				A []struct{}
			}{
				A: []struct{}{},
			},
			expected: `A = []
`,
		},
		{
			desc: "nil field is ignored",
			v: struct {
				A interface{}
			}{
				A: nil,
			},
			expected: ``,
		},
		{
			desc: "private fields are ignored",
			v: struct {
				Public  string
				private string
			}{
				Public:  "shown",
				private: "hidden",
			},
			expected: `Public = 'shown'
`,
		},
		{
			desc: "fields tagged - are ignored",
			v: struct {
				Public  string `toml:"-"`
				private string
			}{
				Public: "hidden",
			},
			expected: ``,
		},
		{
			desc: "nil interface value in map is ignored",
			v: map[string]interface{}{
				"A": nil,
			},
			expected: ``,
		},
		{
			desc: "nil pointer to struct in map produces empty table",
			v: map[string]*struct{}{
				"A": nil,
			},
			expected: `[A]
`,
		},
		{
			desc: "nil pointer to int in map produces zero value",
			v: map[string]*int{
				"A": nil,
			},
			expected: `A = 0
`,
		},
		{
			desc: "nil pointer to string in map produces empty string",
			v: map[string]*string{
				"A": nil,
			},
			expected: `A = ''
`,
		},
		{
			desc: "new line in table key",
			v: map[string]interface{}{
				"hello\nworld": 42,
			},
			expected: `"hello\nworld" = 42
`,
		},
		{
			desc: "new line in parent of nested table key",
			v: map[string]interface{}{
				"hello\nworld": map[string]interface{}{
					"inner": 42,
				},
			},
			expected: `["hello\nworld"]
inner = 42
`,
		},
		{
			desc: "new line in nested table key",
			v: map[string]interface{}{
				"parent": map[string]interface{}{
					"in\ner": map[string]interface{}{
						"foo": 42,
					},
				},
			},
			expected: `[parent]
[parent."in\ner"]
foo = 42
`,
		},
		{
			desc: "int map key",
			v:    map[int]interface{}{1: "a"},
			expected: `1 = 'a'
`,
		},
		{
			desc: "int8 map key",
			v:    map[int8]interface{}{1: "a"},
			expected: `1 = 'a'
`,
		},
		{
			desc: "int64 map key",
			v:    map[int64]interface{}{1: "a"},
			expected: `1 = 'a'
`,
		},
		{
			desc: "uint map key",
			v:    map[uint]interface{}{1: "a"},
			expected: `1 = 'a'
`,
		},
		{
			desc: "uint8 map key",
			v:    map[uint8]interface{}{1: "a"},
			expected: `1 = 'a'
`,
		},
		{
			desc: "uint64 map key",
			v:    map[uint64]interface{}{1: "a"},
			expected: `1 = 'a'
`,
		},
		{
			desc: "float32 map key",
			v: map[float32]interface{}{
				1.1:    "a",
				1.0020: "b",
			},
			expected: `'1.002' = 'b'
'1.1' = 'a'
`,
		},
		{
			desc: "float64 map key",
			v: map[float64]interface{}{
				1.1:    "a",
				1.0020: "b",
			},
			expected: `'1.002' = 'b'
'1.1' = 'a'
`,
		},
		{
			desc: "invalid map key",
			v:    map[struct{ int }]interface{}{{1}: "a"},
			err:  true,
		},
		{
			desc:     "invalid map key but empty",
			v:        map[struct{ int }]interface{}{},
			expected: "",
		},
		{
			desc: "unhandled type",
			v: struct {
				A chan int
			}{
				A: make(chan int),
			},
			err: true,
		},
		{
			desc: "time",
			v: struct {
				T time.Time
			}{
				T: time.Time{},
			},
			expected: `T = 0001-01-01T00:00:00Z
`,
		},
		{
			desc: "time nano",
			v: struct {
				T time.Time
			}{
				T: time.Date(1979, time.May, 27, 0, 32, 0, 999999000, time.UTC),
			},
			expected: `T = 1979-05-27T00:32:00.999999Z
`,
		},
		{
			desc: "bool",
			v: struct {
				A bool
				B bool
			}{
				A: false,
				B: true,
			},
			expected: `A = false
B = true
`,
		},
		{
			desc: "numbers",
			v: struct {
				A float32
				B uint64
				C uint32
				D uint16
				E uint8
				F uint
				G int64
				H int32
				I int16
				J int8
				K int
				L float64
			}{
				A: 1.1,
				B: 42,
				C: 42,
				D: 42,
				E: 42,
				F: 42,
				G: 42,
				H: 42,
				I: 42,
				J: 42,
				K: 42,
				L: 2.2,
			},
			expected: `A = 1.1
B = 42
C = 42
D = 42
E = 42
F = 42
G = 42
H = 42
I = 42
J = 42
K = 42
L = 2.2
`,
		},
		{
			desc: "comments",
			v: struct {
				Table comments `comment:"Before table"`
			}{
				Table: comments{
					One:   1,
					Two:   2,
					Three: []int{1, 2, 3},
				},
			},
			expected: `# Before table
[Table]
One = 1
# Before kv
Two = 2
# Before array
Three = [1, 2, 3]
`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			b, err := toml.Marshal(e.v)
			if e.err {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, e.expected, string(b))

			// make sure the output is always valid TOML
			defaultMap := map[string]interface{}{}
			err = toml.Unmarshal(b, &defaultMap)
			assert.NoError(t, err)

			testWithAllFlags(t, func(t *testing.T, flags int) {
				t.Helper()

				var buf bytes.Buffer
				enc := toml.NewEncoder(&buf)
				setFlags(enc, flags)

				err := enc.Encode(e.v)
				assert.NoError(t, err)

				inlineMap := map[string]interface{}{}
				err = toml.Unmarshal(buf.Bytes(), &inlineMap)
				assert.NoError(t, err)

				assert.Equal(t, defaultMap, inlineMap)
			})
		})
	}
}

type flagsSetters []struct {
	name string
	f    func(enc *toml.Encoder, flag bool) *toml.Encoder
}

var allFlags = flagsSetters{
	{"arrays-multiline", (*toml.Encoder).SetArraysMultiline},
	{"tables-inline", (*toml.Encoder).SetTablesInline},
	{"indent-tables", (*toml.Encoder).SetIndentTables},
	{"omit-empty-super-tables", (*toml.Encoder).SetOmitEmptySuperTables},
}

func setFlags(enc *toml.Encoder, flags int) {
	// testWithFlags builds the bitmask by shifting at each recursion level, so
	// the first setter owns the most significant bit.
	for i := 0; i < len(allFlags); i++ {
		enabled := flags&(1<<(len(allFlags)-1-i)) > 0
		allFlags[i].f(enc, enabled)
	}
}

func testWithAllFlags(t *testing.T, testfn func(t *testing.T, flags int)) {
	t.Helper()
	testWithFlags(t, 0, allFlags, testfn)
}

func testWithFlags(t *testing.T, flags int, setters flagsSetters, testfn func(t *testing.T, flags int)) {
	t.Helper()

	if len(setters) == 0 {
		testfn(t, flags)

		return
	}

	s := setters[0]

	for _, enabled := range []bool{false, true} {
		name := fmt.Sprintf("%s=%t", s.name, enabled)
		newFlags := flags << 1

		if enabled {
			newFlags++
		}

		t.Run(name, func(t *testing.T) {
			testWithFlags(t, newFlags, setters[1:], testfn)
		})
	}
}

func TestMarshalFloats(t *testing.T) {
	v := map[string]float32{
		"nan":  float32(math.NaN()),
		"+inf": float32(math.Inf(1)),
		"-inf": float32(math.Inf(-1)),
	}

	expected := `'+inf' = inf
-inf = -inf
nan = nan
`

	actual, err := toml.Marshal(v)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(actual))

	v64 := map[string]float64{
		"nan":  math.NaN(),
		"+inf": math.Inf(1),
		"-inf": math.Inf(-1),
	}

	actual, err = toml.Marshal(v64)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}

func TestMarshalIndentTables(t *testing.T) {
	examples := []struct {
		desc     string
		v        interface{}
		expected string
	}{
		{
			desc: "one kv",
			v: map[string]interface{}{
				"foo": "bar",
			},
			expected: `foo = 'bar'
`,
		},
		{
			desc: "one level table",
			v: map[string]map[string]string{
				"foo": {
					"one": "value1",
					"two": "value2",
				},
			},
			expected: `[foo]
  one = 'value1'
  two = 'value2'
`,
		},
		{
			desc: "two levels table",
			v: map[string]interface{}{
				"root": "value0",
				"level1": map[string]interface{}{
					"one": "value1",
					"level2": map[string]interface{}{
						"two": "value2",
					},
				},
			},
			expected: `root = 'value0'

[level1]
  one = 'value1'

  [level1.level2]
    two = 'value2'
`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			var buf strings.Builder
			enc := toml.NewEncoder(&buf)
			enc.SetIndentTables(true)
			err := enc.Encode(e.v)
			assert.NoError(t, err)
			assert.Equal(t, e.expected, buf.String())
		})
	}
}

func TestMarshalOmitEmptySuperTables(t *testing.T) {
	type D struct{ S string }
	type C struct{ D D }
	type B struct{ C C }

	examples := []struct {
		desc     string
		v        interface{}
		indent   bool
		expected string
	}{
		{
			desc: "chain of super-tables",
			v:    struct{ B B }{B{C{D{S: "foo"}}}},
			expected: `[B.C.D]
S = 'foo'
`,
		},
		{
			desc: "super-table with its own key-value keeps its header",
			v: struct {
				B struct {
					X int
					C C
				}
			}{B: struct {
				X int
				C C
			}{X: 1, C: C{D: D{S: "foo"}}}},
			expected: `[B]
X = 1

[B.C.D]
S = 'foo'
`,
		},
		{
			desc: "empty table keeps its header",
			v:    struct{ B struct{} }{},
			expected: `[B]
`,
		},
		{
			desc: "empty table under a super-table",
			v:    struct{ B struct{ C struct{} } }{},
			expected: `[B.C]
`,
		},
		{
			desc: "array of tables under super-tables",
			v: struct{ B struct{ C []D } }{B: struct{ C []D }{
				C: []D{{S: "x"}, {S: "y"}},
			}},
			expected: `[[B.C]]
S = 'x'

[[B.C]]
S = 'y'
`,
		},
		{
			desc: "super-table with a comment keeps its header",
			v: struct {
				B struct{ C C } `comment:"hello"`
			}{B: struct{ C C }{C: C{D: D{S: "foo"}}}},
			expected: `# hello
[B]
[B.C.D]
S = 'foo'
`,
		},
		{
			desc: "commented super-tables are still omitted",
			v: struct {
				B struct{ C C } `toml:",commented"`
			}{B: struct{ C C }{C: C{D: D{S: "foo"}}}},
			expected: `# [B.C.D]
# S = 'foo'
`,
		},
		{
			desc: "two sibling sub-tables",
			v: struct{ B struct{ C, D D } }{B: struct{ C, D D }{
				C: D{S: "one"},
				D: D{S: "two"},
			}},
			expected: `[B.C]
S = 'one'

[B.D]
S = 'two'
`,
		},
		{
			desc: "maps",
			v: map[string]map[string]map[string]string{
				"x": {"y": {"z": "w"}},
			},
			expected: `[x.y]
z = 'w'
`,
		},
		{
			desc: "table emptied by omitempty",
			v: struct {
				B struct {
					X string `toml:",omitempty"`
					C C
				}
			}{B: struct {
				X string `toml:",omitempty"`
				C C
			}{C: C{D: D{S: "foo"}}}},
			expected: `[B.C.D]
S = 'foo'
`,
		},
		{
			desc:   "omitted tables do not indent their children",
			v:      struct{ B B }{B{C{D{S: "foo"}}}},
			indent: true,
			expected: `[B.C.D]
  S = 'foo'
`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			var buf strings.Builder
			enc := toml.NewEncoder(&buf)
			enc.SetOmitEmptySuperTables(true)
			enc.SetIndentTables(e.indent)
			err := enc.Encode(e.v)
			assert.NoError(t, err)
			assert.Equal(t, e.expected, buf.String())

			// The omitted headers must not change the meaning of the
			// document: both outputs decode to the same map.
			def, err := toml.Marshal(e.v)
			assert.NoError(t, err)
			defaultMap := map[string]interface{}{}
			assert.NoError(t, toml.Unmarshal(def, &defaultMap))
			omitMap := map[string]interface{}{}
			assert.NoError(t, toml.Unmarshal([]byte(buf.String()), &omitMap))
			assert.Equal(t, defaultMap, omitMap)
		})
	}
}

type customTextMarshaler struct {
	value int64
}

func (c *customTextMarshaler) MarshalText() ([]byte, error) {
	if c.value == 1 {
		return nil, errors.New("cannot represent 1 because this is a silly test")
	}
	return []byte(fmt.Sprintf("::%d", c.value)), nil
}

func TestMarshalTextMarshaler_NoRoot(t *testing.T) {
	c := customTextMarshaler{}
	_, err := toml.Marshal(&c)
	assert.Error(t, err)
}

func TestMarshalTextMarshaler_Error(t *testing.T) {
	m := map[string]interface{}{"a": &customTextMarshaler{value: 1}}
	_, err := toml.Marshal(m)
	assert.Error(t, err)
}

func TestMarshalTextMarshaler_ErrorInline(t *testing.T) {
	type s struct {
		A map[string]interface{} `inline:"true"`
	}

	d := s{
		A: map[string]interface{}{"a": &customTextMarshaler{value: 1}},
	}

	_, err := toml.Marshal(d)
	assert.Error(t, err)
}

func TestMarshalTextMarshaler(t *testing.T) {
	m := map[string]interface{}{"a": &customTextMarshaler{value: 2}}
	r, err := toml.Marshal(m)
	assert.NoError(t, err)
	assert.Equal(t, "a = '::2'\n", string(r))
}

type brokenWriter struct{}

func (b *brokenWriter) Write([]byte) (int, error) {
	return 0, errors.New("dead")
}

func TestEncodeToBrokenWriter(t *testing.T) {
	w := brokenWriter{}
	enc := toml.NewEncoder(&w)
	err := enc.Encode(map[string]string{"hello": "world"})
	assert.Error(t, err)
}

func TestEncoderSetIndentSymbol(t *testing.T) {
	var w strings.Builder
	enc := toml.NewEncoder(&w)
	enc.SetIndentTables(true)
	enc.SetIndentSymbol(">>>")
	err := enc.Encode(map[string]map[string]string{"parent": {"hello": "world"}})
	assert.NoError(t, err)
	expected := `[parent]
>>>hello = 'world'
`
	assert.Equal(t, expected, w.String())
}

func TestEncoderSetMarshalJSONNumbers(t *testing.T) {
	var w strings.Builder
	enc := toml.NewEncoder(&w)
	enc.SetMarshalJSONNumbers(true)
	err := enc.Encode(map[string]interface{}{
		"A": json.Number("1.1"),
		"B": json.Number("42e-3"),
		"C": json.Number("42"),
		"D": json.Number("0"),
		"E": json.Number("0.0"),
		"F": json.Number(""),
	})
	assert.NoError(t, err)
	expected := `A = 1.1
B = 0.042
C = 42
D = 0
E = 0.0
F = 0
`
	assert.Equal(t, expected, w.String())
}

func TestEncoderOmitempty(t *testing.T) {
	type doc struct {
		String  string            `toml:",omitempty,multiline"`
		Bool    bool              `toml:",omitempty,multiline"`
		Int     int               `toml:",omitempty,multiline"`
		Int8    int8              `toml:",omitempty,multiline"`
		Int16   int16             `toml:",omitempty,multiline"`
		Int32   int32             `toml:",omitempty,multiline"`
		Int64   int64             `toml:",omitempty,multiline"`
		Uint    uint              `toml:",omitempty,multiline"`
		Uint8   uint8             `toml:",omitempty,multiline"`
		Uint16  uint16            `toml:",omitempty,multiline"`
		Uint32  uint32            `toml:",omitempty,multiline"`
		Uint64  uint64            `toml:",omitempty,multiline"`
		Float32 float32           `toml:",omitempty,multiline"`
		Float64 float64           `toml:",omitempty,multiline"`
		MapNil  map[string]string `toml:",omitempty,multiline"`
		Slice   []string          `toml:",omitempty,multiline"`
		Ptr     *string           `toml:",omitempty,multiline"`
		Iface   interface{}       `toml:",omitempty,multiline"`
		Struct  struct{}          `toml:",omitempty,multiline"`
		Inline  struct {
			String string `toml:",omitempty,multiline"`
		} `toml:",inline"`
	}

	d := doc{}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	expected := `Inline = {}
`

	assert.Equal(t, expected, string(b))
}

func TestEncoderOmitemptyNonNilEmptyCollections(t *testing.T) {
	// Regression for #1075: a struct whose only non-zero fields are non-nil
	// but empty maps/slices (as produced by some YAML decoders/cloners) must
	// be treated as empty by omitempty, so no empty table header is emitted.
	// reflect.Value.IsZero would report such a struct as non-zero.
	type base struct {
		Rules map[string]string `toml:"rules,omitempty"`
	}
	type nested struct {
		base
		Overrides []string `toml:"overrides,omitempty"`
	}
	type group struct {
		Nested nested `toml:"nested,omitempty"`
	}
	type doc struct {
		Group group  `toml:"group,omitempty"`
		Keep  string `toml:"keep,omitempty"`
	}

	d := doc{
		Group: group{Nested: nested{
			base:      base{Rules: map[string]string{}},
			Overrides: []string{},
		}},
		Keep: "x",
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)
	assert.Equal(t, "keep = 'x'\n", string(b))
}

func TestEncoderOmitemptyScalarStructStillEmittedWhenSet(t *testing.T) {
	// A struct that encodes as a scalar (here, via a TextMarshaler) must keep
	// the zero-value-based emptiness check: non-zero values are still emitted,
	// zero values are omitted.
	type doc struct {
		When time.Time `toml:"when,omitempty"`
		Keep string    `toml:"keep,omitempty"`
	}

	zero, err := toml.Marshal(doc{Keep: "x"})
	assert.NoError(t, err)
	assert.Equal(t, "keep = 'x'\n", string(zero))

	set, err := toml.Marshal(doc{When: time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC), Keep: "x"})
	assert.NoError(t, err)
	assert.Equal(t, "when = 2026-06-22T00:00:00Z\nkeep = 'x'\n", string(set))
}

func TestEncoderOmitzero(t *testing.T) {
	type doc struct {
		String  string            `toml:",omitzero,multiline"`
		Bool    bool              `toml:",omitzero,multiline"`
		Int     int               `toml:",omitzero,multiline"`
		Int8    int8              `toml:",omitzero,multiline"`
		Int16   int16             `toml:",omitzero,multiline"`
		Int32   int32             `toml:",omitzero,multiline"`
		Int64   int64             `toml:",omitzero,multiline"`
		Uint    uint              `toml:",omitzero,multiline"`
		Uint8   uint8             `toml:",omitzero,multiline"`
		Uint16  uint16            `toml:",omitzero,multiline"`
		Uint32  uint32            `toml:",omitzero,multiline"`
		Uint64  uint64            `toml:",omitzero,multiline"`
		Float32 float32           `toml:",omitzero,multiline"`
		Float64 float64           `toml:",omitzero,multiline"`
		MapNil  map[string]string `toml:",omitzero,multiline"`
		Slice   []string          `toml:",omitzero,multiline"`
		Ptr     *string           `toml:",omitzero,multiline"`
		Iface   interface{}       `toml:",omitzero,multiline"`
		Struct  struct{}          `toml:",omitzero,multiline"`
		Time    time.Time         `toml:",omitzero,multiline"`
		IP      netip.Addr        `toml:",omitzero,multiline"`
		Inline  struct {
			String string `toml:",omitzero,multiline"`
		} `toml:",inline"`
	}

	d := doc{}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	expected := `Inline = {}
`

	assert.Equal(t, expected, string(b))
}

func TestEncoderOmitzeroOpaqueStruct(t *testing.T) {
	type doc struct {
		Time time.Time  `toml:",omitzero"`
		IP   netip.Addr `toml:",omitzero"`
	}

	d := doc{
		Time: time.Date(2001, 2, 3, 4, 5, 6, 7, time.UTC),
		IP:   netip.MustParseAddr("192.168.178.35"),
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	expected := `Time = 2001-02-03T04:05:06.000000007Z
IP = '192.168.178.35'
`

	assert.Equal(t, expected, string(b))
}

// customZeroType has a custom IsZero method that returns true
// when Value is less than 10.
type customZeroType struct {
	Value int
}

func (c customZeroType) IsZero() bool {
	return c.Value < 10
}

// customZeroPointerType has a custom IsZero method on the pointer receiver.
type customZeroPointerType struct {
	Value int
}

func (c *customZeroPointerType) IsZero() bool {
	return c.Value < 10
}

func TestEncoderOmitzeroCustomIsZero(t *testing.T) {
	type doc struct {
		Custom customZeroType `toml:",omitzero"`
		Normal int            `toml:",omitzero"`
	}

	// Custom.Value = 5, which is < 10, so custom IsZero returns true
	d := doc{
		Custom: customZeroType{Value: 5},
		Normal: 0,
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Both fields should be omitted: Custom because custom IsZero returns true,
	// Normal because its reflect zero value is true.
	expected := ``

	assert.Equal(t, expected, string(b))
}

func TestEncoderOmitzeroCustomIsZeroNotZero(t *testing.T) {
	type doc struct {
		Custom customZeroType `toml:",omitzero"`
		Normal int            `toml:",omitzero"`
	}

	// Custom.Value = 15, which is >= 10, so custom IsZero returns false
	d := doc{
		Custom: customZeroType{Value: 15},
		Normal: 42,
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Both fields should be present
	expected := `Normal = 42

[Custom]
Value = 15
`

	assert.Equal(t, expected, string(b))
}

func TestEncoderOmitzeroCustomIsZeroPointerReceiver(t *testing.T) {
	type doc struct {
		Custom customZeroPointerType `toml:",omitzero"`
	}

	// Custom.Value = 5, which is < 10, so custom IsZero returns true
	d := doc{
		Custom: customZeroPointerType{Value: 5},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Field should be omitted because custom IsZero returns true
	expected := ``

	assert.Equal(t, expected, string(b))
}

func TestEncoderOmitzeroCustomIsZeroPointerReceiverNotZero(t *testing.T) {
	type doc struct {
		Custom customZeroPointerType `toml:",omitzero"`
	}

	// Custom.Value = 15, which is >= 10, so custom IsZero returns false
	d := doc{
		Custom: customZeroPointerType{Value: 15},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Field should be present
	expected := `[Custom]
Value = 15
`

	assert.Equal(t, expected, string(b))
}

// TestEncoderOmitzeroCustomIsZeroPointerReceiverAddressable tests the v.CanAddr() path
// by marshaling a pointer to a struct, which makes fields addressable.
func TestEncoderOmitzeroCustomIsZeroPointerReceiverAddressable(t *testing.T) {
	type doc struct {
		Custom customZeroPointerType `toml:",omitzero"`
	}

	// Custom.Value = 5, which is < 10, so custom IsZero returns true
	d := &doc{
		Custom: customZeroPointerType{Value: 5},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Field should be omitted because custom IsZero returns true
	expected := ``

	assert.Equal(t, expected, string(b))
}

// TestEncoderOmitzeroCustomIsZeroPointerReceiverAddressableNotZero tests the v.CanAddr() path
// when custom IsZero returns false.
func TestEncoderOmitzeroCustomIsZeroPointerReceiverAddressableNotZero(t *testing.T) {
	type doc struct {
		Custom customZeroPointerType `toml:",omitzero"`
	}

	// Custom.Value = 15, which is >= 10, so custom IsZero returns false
	d := &doc{
		Custom: customZeroPointerType{Value: 15},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Field should be present
	expected := `[Custom]
Value = 15
`

	assert.Equal(t, expected, string(b))
}

// TestEncoderOmitzeroCustomIsZeroInlineTable tests omitzero with inline tables.
func TestEncoderOmitzeroCustomIsZeroInlineTable(t *testing.T) {
	type doc struct {
		Custom customZeroType `toml:",omitzero,inline"`
	}

	// Custom.Value = 5, which is < 10, so custom IsZero returns true
	d := doc{
		Custom: customZeroType{Value: 5},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Field should be omitted
	expected := ``

	assert.Equal(t, expected, string(b))
}

// TestEncoderOmitzeroCustomIsZeroInlineTableNotZero tests omitzero with inline tables when not zero.
func TestEncoderOmitzeroCustomIsZeroInlineTableNotZero(t *testing.T) {
	type doc struct {
		Custom customZeroType `toml:",omitzero,inline"`
	}

	// Custom.Value = 15, which is >= 10, so custom IsZero returns false
	d := doc{
		Custom: customZeroType{Value: 15},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Field should be present as inline table
	expected := `Custom = {Value = 15}
`

	assert.Equal(t, expected, string(b))
}

// TestEncoderOmitzeroCustomIsZeroMixedTypes tests omitzero with a mix of custom and regular types.
func TestEncoderOmitzeroCustomIsZeroMixedTypes(t *testing.T) {
	type doc struct {
		Custom  customZeroType `toml:",omitzero"`
		Regular int            `toml:",omitzero"`
		NoOmit  customZeroType `toml:""`
		Pointer *int           `toml:",omitzero"`
	}

	d := doc{
		Custom:  customZeroType{Value: 5}, // IsZero returns true
		Regular: 0,                        // zero value
		NoOmit:  customZeroType{Value: 5}, // not omitted (no omitzero tag)
		Pointer: nil,                      // nil pointer
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Custom is omitted (custom IsZero true), Regular is omitted (zero value),
	// NoOmit is present (no omitzero tag), Pointer is omitted (nil)
	expected := `[NoOmit]
Value = 5
`

	assert.Equal(t, expected, string(b))
}

// TestEncoderOmitzeroCustomIsZeroSlice tests omitzero with slices containing custom types.
func TestEncoderOmitzeroCustomIsZeroSlice(t *testing.T) {
	type doc struct {
		Items []customZeroType `toml:",omitzero"`
	}

	// Nil slice should be omitted (IsZero returns true for nil slices)
	d := doc{
		Items: nil,
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	expected := ``

	assert.Equal(t, expected, string(b))

	// Empty but non-nil slice is NOT zero, so it's included
	d2 := doc{
		Items: []customZeroType{},
	}

	b2, err := toml.Marshal(d2)
	assert.NoError(t, err)

	expected2 := `Items = []
`

	assert.Equal(t, expected2, string(b2))
}

// TestEncoderOmitzeroCustomIsZeroNestedStruct tests omitzero with nested structs.
func TestEncoderOmitzeroCustomIsZeroNestedStruct(t *testing.T) {
	type inner struct {
		Custom customZeroType `toml:",omitzero"`
		Value  int            `toml:",omitzero"`
	}
	type doc struct {
		Inner inner `toml:",omitzero"`
	}

	// Inner struct has all zero fields, but the struct itself is not zero
	// (reflect.Value.IsZero checks if all fields are zero)
	d := doc{
		Inner: inner{
			Custom: customZeroType{Value: 5}, // custom IsZero returns true
			Value:  0,                        // zero value
		},
	}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	// Inner is present but its fields are omitted
	expected := `[Inner]
`

	assert.Equal(t, expected, string(b))
}

func TestEncoderTagFieldName(t *testing.T) {
	type doc struct {
		String string `toml:"hello"`
		OkSym  string `toml:"#"`
		Bad    string `toml:"\"` //nolint:govet
	}

	d := doc{String: "world"}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	expected := `hello = 'world'
'#' = ''
Bad = ''
`

	assert.Equal(t, expected, string(b))
}

func TestIssue436(t *testing.T) {
	data := []byte(`{"a": [ { "b": { "c": "d" } } ]}`)

	var v interface{}
	err := json.Unmarshal(data, &v)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(v)
	assert.NoError(t, err)

	expected := `[[a]]
[a.b]
c = 'd'
`
	assert.Equal(t, expected, buf.String())
}

func TestIssue424(t *testing.T) {
	type Message1 struct {
		Text string
	}

	type Message2 struct {
		Text string `multiline:"true"`
	}

	msg1 := Message1{"Hello\\World"}
	msg2 := Message2{"Hello\\World"}

	toml1, err := toml.Marshal(msg1)
	assert.NoError(t, err)

	toml2, err := toml.Marshal(msg2)
	assert.NoError(t, err)

	msg1parsed := Message1{}
	err = toml.Unmarshal(toml1, &msg1parsed)
	assert.NoError(t, err)
	assert.Equal(t, msg1, msg1parsed)

	msg2parsed := Message2{}
	err = toml.Unmarshal(toml2, &msg2parsed)
	assert.NoError(t, err)
	assert.Equal(t, msg2, msg2parsed)
}

func TestIssue567(t *testing.T) {
	var m map[string]interface{}
	err := toml.Unmarshal([]byte("A = 12:08:05"), &m)
	assert.NoError(t, err)
	assert.Equal(t,
		reflect.TypeOf(m["A"]), reflect.TypeOf(toml.LocalTime{}),
		"Expected type '%v', got: %v", reflect.TypeOf(m["A"]), reflect.TypeOf(toml.LocalTime{}),
	)
}

func TestIssue590(t *testing.T) {
	type CustomType int
	var cfg struct {
		Option CustomType `toml:"option"`
	}
	err := toml.Unmarshal([]byte("option = 42"), &cfg)
	assert.NoError(t, err)
}

func TestIssue571(t *testing.T) {
	type Foo struct {
		Float32 float32
		Float64 float64
	}

	const closeEnough = 1e-9

	foo := Foo{
		Float32: 42,
		Float64: 43,
	}
	b, err := toml.Marshal(foo)
	assert.NoError(t, err)

	var foo2 Foo
	err = toml.Unmarshal(b, &foo2)
	assert.NoError(t, err)

	inDelta(t, 42, foo2.Float32, closeEnough)
	inDelta(t, 43, foo2.Float64, closeEnough)
}

func TestIssue678(t *testing.T) {
	type Config struct {
		BigInt big.Int
	}

	cfg := &Config{
		BigInt: *big.NewInt(123),
	}

	out, err := toml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "BigInt = '123'\n", string(out))

	cfg2 := &Config{}
	err = toml.Unmarshal(out, cfg2)
	assert.NoError(t, err)
	assert.Equal(t, cfg, cfg2)
}

func TestIssue752(t *testing.T) {
	type Fooer interface {
		Foo() string
	}

	type Container struct {
		Fooer
	}

	c := Container{}

	out, err := toml.Marshal(c)
	assert.NoError(t, err)
	assert.Equal(t, "", string(out))
}

func TestIssue768(t *testing.T) {
	type cfg struct {
		Name string `comment:"This is a multiline comment.\nThis is line 2."`
	}

	out, err := toml.Marshal(&cfg{})
	assert.NoError(t, err)

	expected := `# This is a multiline comment.
# This is line 2.
Name = ''
`

	assert.Equal(t, expected, string(out))
}

func TestIssue786(t *testing.T) {
	type Dependencies struct {
		Dependencies         []string `toml:"dependencies,multiline,omitempty"`
		BuildDependencies    []string `toml:"buildDependencies,multiline,omitempty"`
		OptionalDependencies []string `toml:"optionalDependencies,multiline,omitempty"`
	}

	type Test struct {
		Dependencies Dependencies `toml:"dependencies,omitempty"`
	}

	x := Test{}
	b, err := toml.Marshal(x)
	assert.NoError(t, err)

	assert.Equal(t, "", string(b))

	type General struct {
		From      string `toml:"from,omitempty" json:"from,omitempty" comment:"from in graphite-web format, the local TZ is used"`
		Randomize bool   `toml:"randomize" json:"randomize" comment:"randomize starting time with [0,step)"`
	}

	type Custom struct {
		Name string `toml:"name" json:"name,omitempty" comment:"names for generator, braces are expanded like in shell"`
		Type string `toml:"type,omitempty" json:"type,omitempty" comment:"type of generator"`
		General
	}
	type Config struct {
		General
		Custom []Custom `toml:"custom,omitempty" json:"custom,omitempty" comment:"generators with custom parameters can be specified separately"`
	}

	buf := new(bytes.Buffer)
	config := &Config{General: General{From: "-2d", Randomize: true}}
	config.Custom = []Custom{{Name: "omit", General: General{Randomize: false}}}
	config.Custom = append(config.Custom, Custom{Name: "present", General: General{From: "-2d", Randomize: true}})
	encoder := toml.NewEncoder(buf)
	err = encoder.Encode(config)
	assert.NoError(t, err)

	expected := `# from in graphite-web format, the local TZ is used
from = '-2d'
# randomize starting time with [0,step)
randomize = true

# generators with custom parameters can be specified separately
[[custom]]
# names for generator, braces are expanded like in shell
name = 'omit'
# randomize starting time with [0,step)
randomize = false

[[custom]]
# names for generator, braces are expanded like in shell
name = 'present'
# from in graphite-web format, the local TZ is used
from = '-2d'
# randomize starting time with [0,step)
randomize = true
`

	assert.Equal(t, expected, buf.String())
}

func TestMarshalIssue888(t *testing.T) {
	type Thing struct {
		FieldA string `comment:"my field A"`
		FieldB string `comment:"my field B"`
	}

	type Cfg struct {
		Custom []Thing `comment:"custom config"`
	}

	buf := new(bytes.Buffer)

	config := Cfg{
		Custom: []Thing{
			{FieldA: "field a 1", FieldB: "field b 1"},
			{FieldA: "field a 2", FieldB: "field b 2"},
		},
	}

	encoder := toml.NewEncoder(buf).SetIndentTables(true)
	err := encoder.Encode(config)
	assert.NoError(t, err)

	expected := `# custom config
[[Custom]]
  # my field A
  FieldA = 'field a 1'
  # my field B
  FieldB = 'field b 1'

[[Custom]]
  # my field A
  FieldA = 'field a 2'
  # my field B
  FieldB = 'field b 2'
`

	assert.Equal(t, expected, buf.String())
}

func TestMarshalNestedAnonymousStructs(t *testing.T) {
	type Embedded struct {
		Value string `toml:"value" json:"value"`
		Top   struct {
			Value string `toml:"value" json:"value"`
		} `toml:"top" json:"top"`
	}

	type Named struct {
		Value string `toml:"value" json:"value"`
	}

	var doc struct {
		Embedded
		Named     `toml:"named" json:"named"`
		Anonymous struct {
			Value string `toml:"value" json:"value"`
		} `toml:"anonymous" json:"anonymous"`
	}

	expected := `value = ''

[top]
value = ''

[named]
value = ''

[anonymous]
value = ''
`

	result, err := toml.Marshal(doc)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestMarshalNestedAnonymousStructs_DuplicateField(t *testing.T) {
	type Embedded struct {
		Value string `toml:"value" json:"value"`
		Top   struct {
			Value string `toml:"value" json:"value"`
		} `toml:"top" json:"top"`
	}

	var doc struct {
		Value string `toml:"value" json:"value"`
		Embedded
	}
	doc.Embedded.Value = "shadowed"
	doc.Value = "shadows"

	expected := `value = 'shadows'

[top]
value = ''
`

	result, err := toml.Marshal(doc)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestMarshalNestedAnonymousStructs_PointerEmbedded(t *testing.T) {
	type Embedded struct {
		Value   string  `toml:"value" json:"value"`
		Omitted string  `toml:"omitted,omitempty"`
		Ptr     *string `toml:"ptr"`
	}

	type Named struct {
		Value string `toml:"value" json:"value"`
	}

	type Doc struct {
		*Embedded
		*Named    `toml:"named" json:"named"`
		Anonymous struct {
			*Embedded
			Value *string `toml:"value" json:"value"`
		} `toml:"anonymous,omitempty" json:"anonymous,omitempty"`
	}

	doc := &Doc{
		Embedded: &Embedded{Value: "foo"},
	}

	expected := `value = 'foo'
`

	result, err := toml.Marshal(doc)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestLocalTime(t *testing.T) {
	v := map[string]toml.LocalTime{
		"a": {
			Hour:       1,
			Minute:     2,
			Second:     3,
			Nanosecond: 4,
		},
	}

	expected := `a = 01:02:03.000000004
`

	out, err := toml.Marshal(v)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(out))
}

func TestMarshalUint64Overflow(t *testing.T) {
	// The TOML spec only asserts implementation to provide support for the
	// int64 range. To avoid generating TOML documents that would not be
	// supported by standard-compliant parsers, uint64 > max int64 cannot be
	// marshaled.
	x := map[string]interface{}{
		"foo": uint64(math.MaxInt64) + 1,
	}

	_, err := toml.Marshal(x)
	assert.Error(t, err)
}

func TestIndentWithInlineTable(t *testing.T) {
	x := map[string][]map[string]string{
		"one": {
			{"0": "0"},
			{"1": "1"},
		},
	}
	expected := `one = [
  {0 = '0'},
  {1 = '1'}
]
`
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	enc.SetTablesInline(true)
	enc.SetArraysMultiline(true)
	assert.NoError(t, enc.Encode(x))
	assert.Equal(t, expected, buf.String())
}

type C3 struct {
	Value  int   `toml:",commented"`
	Values []int `toml:",commented"`
}

type C2 struct {
	Int       int64
	String    string
	ArrayInts []int
	Structs   []C3
}

type C1 struct {
	Int       int64  `toml:",commented"`
	String    string `toml:",commented"`
	ArrayInts []int  `toml:",commented"`
	Structs   []C3   `toml:",commented"`
}

type Commented struct {
	Int    int64  `toml:",commented"`
	String string `toml:",commented"`

	C1 C1
	C2 C2 `toml:",commented"` // same as C1, but commented at top level
}

func TestMarshalCommented(t *testing.T) {
	c := Commented{
		Int:    42,
		String: "root",

		C1: C1{
			Int:       11,
			String:    "C1",
			ArrayInts: []int{1, 2, 3},
			Structs: []C3{
				{Value: 100},
				{Values: []int{4, 5, 6}},
			},
		},
		C2: C2{
			Int:       22,
			String:    "C2",
			ArrayInts: []int{1, 2, 3},
			Structs: []C3{
				{Value: 100},
				{Values: []int{4, 5, 6}},
			},
		},
	}

	out, err := toml.Marshal(c)
	assert.NoError(t, err)

	expected := `# Int = 42
# String = 'root'

[C1]
# Int = 11
# String = 'C1'
# ArrayInts = [1, 2, 3]

# [[C1.Structs]]
# Value = 100
# Values = []

# [[C1.Structs]]
# Value = 0
# Values = [4, 5, 6]

# [C2]
# Int = 22
# String = 'C2'
# ArrayInts = [1, 2, 3]

# [[C2.Structs]]
# Value = 100
# Values = []

# [[C2.Structs]]
# Value = 0
# Values = [4, 5, 6]
`

	assert.Equal(t, expected, string(out))
}

func TestMarshalIndentedCustomTypeArray(t *testing.T) {
	c := struct {
		Nested struct {
			NestedArray []struct {
				Value int
			}
		}
	}{
		Nested: struct {
			NestedArray []struct {
				Value int
			}
		}{
			NestedArray: []struct {
				Value int
			}{
				{Value: 1},
				{Value: 2},
			},
		},
	}

	expected := `[Nested]
  [[Nested.NestedArray]]
    Value = 1

  [[Nested.NestedArray]]
    Value = 2
`

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	assert.NoError(t, enc.Encode(c))
	assert.Equal(t, expected, buf.String())
}

func ExampleMarshal() {
	type MyConfig struct {
		Version int
		Name    string
		Tags    []string
	}

	cfg := MyConfig{
		Version: 2,
		Name:    "go-toml",
		Tags:    []string{"go", "toml"},
	}

	b, err := toml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	// Output:
	// Version = 2
	// Name = 'go-toml'
	// Tags = ['go', 'toml']
}

func ExampleEncoder_SetOmitEmptySuperTables() {
	type Credentials struct {
		User     string
		Password string
	}
	type Database struct {
		Credentials Credentials
	}
	type Config struct {
		Database Database
	}

	cfg := Config{
		Database: Database{
			Credentials: Credentials{
				User:     "root",
				Password: "hunter2",
			},
		},
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetOmitEmptySuperTables(true)
	if err := enc.Encode(cfg); err != nil {
		panic(err)
	}
	fmt.Println(buf.String())

	// Output:
	// [Database.Credentials]
	// User = 'root'
	// Password = 'hunter2'
}

// Example that uses the 'commented' field tag option to generate an example
// configuration file that has commented out sections (example from
// go-graphite/graphite-clickhouse).
func ExampleMarshal_commented() {
	type Common struct {
		Listen               string        `toml:"listen"                     comment:"general listener"`
		PprofListen          string        `toml:"pprof-listen"               comment:"listener to serve /debug/pprof requests. '-pprof' argument overrides it"`                    //nolint:lll
		MaxMetricsPerTarget  int           `toml:"max-metrics-per-target"     comment:"limit numbers of queried metrics per target in /render requests, 0 or negative = unlimited"` //nolint:lll
		MemoryReturnInterval time.Duration `toml:"memory-return-interval"     comment:"daemon will return the freed memory to the OS when it>0"`
	}

	type Costs struct {
		Cost       *int           `toml:"cost"        comment:"default cost (for wildcarded equivalence or matched with regex, or if no value cost set)"`
		ValuesCost map[string]int `toml:"values-cost" comment:"cost with some value (for equivalence without wildcards) (additional tuning, usually not needed)"` //nolint:lll
	}

	type ClickHouse struct {
		URL string `toml:"url" comment:"default url, see https://clickhouse.tech/docs/en/interfaces/http. Can be overwritten with query-params"`

		RenderMaxQueries        int               `toml:"render-max-queries" comment:"Max queries to render queries"`
		RenderConcurrentQueries int               `toml:"render-concurrent-queries" comment:"Concurrent queries to render queries"`
		TaggedCosts             map[string]*Costs `toml:"tagged-costs,commented"`
		TreeTable               string            `toml:"tree-table,commented"`
		ReverseTreeTable        string            `toml:"reverse-tree-table,commented"`
		DateTreeTable           string            `toml:"date-tree-table,commented"`
		DateTreeTableVersion    int               `toml:"date-tree-table-version,commented"`
		TreeTimeout             time.Duration     `toml:"tree-timeout,commented"`
		TagTable                string            `toml:"tag-table,commented"`
		ExtraPrefix             string            `toml:"extra-prefix"             comment:"add extra prefix (directory in graphite) for all metrics, w/o trailing dot"` //nolint:lll
		ConnectTimeout          time.Duration     `toml:"connect-timeout"          comment:"TCP connection timeout"`
		DataTableLegacy         string            `toml:"data-table,commented"`
		RollupConfLegacy        string            `toml:"rollup-conf,commented"`
		MaxDataPoints           int               `toml:"max-data-points"          comment:"max points per metric when internal-aggregation=true"`
		InternalAggregation     bool              `toml:"internal-aggregation"     comment:"ClickHouse-side aggregation, see doc/aggregation.md"`
	}

	type Tags struct {
		Rules      string `toml:"rules"`
		Date       string `toml:"date"`
		ExtraWhere string `toml:"extra-where"`
		InputFile  string `toml:"input-file"`
		OutputFile string `toml:"output-file"`
	}

	type Config struct {
		Common     Common     `toml:"common"`
		ClickHouse ClickHouse `toml:"clickhouse"`
		Tags       Tags       `toml:"tags,commented"`
	}

	cfg := &Config{
		Common: Common{
			Listen:               ":9090",
			PprofListen:          "",
			MaxMetricsPerTarget:  15000, // This is arbitrary value to protect CH from overload
			MemoryReturnInterval: 0,
		},
		ClickHouse: ClickHouse{
			URL:                 "http://localhost:8123?cancel_http_readonly_queries_on_client_close=1",
			ExtraPrefix:         "",
			ConnectTimeout:      time.Second,
			DataTableLegacy:     "",
			RollupConfLegacy:    "auto",
			MaxDataPoints:       1048576,
			InternalAggregation: true,
		},
		Tags: Tags{},
	}

	out, err := toml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	err = toml.Unmarshal(out, &cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	// Output:
	// [common]
	// # general listener
	// listen = ':9090'
	// # listener to serve /debug/pprof requests. '-pprof' argument overrides it
	// pprof-listen = ''
	// # limit numbers of queried metrics per target in /render requests, 0 or negative = unlimited
	// max-metrics-per-target = 15000
	// # daemon will return the freed memory to the OS when it>0
	// memory-return-interval = 0
	//
	// [clickhouse]
	// # default url, see https://clickhouse.tech/docs/en/interfaces/http. Can be overwritten with query-params
	// url = 'http://localhost:8123?cancel_http_readonly_queries_on_client_close=1'
	// # Max queries to render queries
	// render-max-queries = 0
	// # Concurrent queries to render queries
	// render-concurrent-queries = 0
	// # tree-table = ''
	// # reverse-tree-table = ''
	// # date-tree-table = ''
	// # date-tree-table-version = 0
	// # tree-timeout = 0
	// # tag-table = ''
	// # add extra prefix (directory in graphite) for all metrics, w/o trailing dot
	// extra-prefix = ''
	// # TCP connection timeout
	// connect-timeout = 1000000000
	// # data-table = ''
	// # rollup-conf = 'auto'
	// # max points per metric when internal-aggregation=true
	// max-data-points = 1048576
	// # ClickHouse-side aggregation, see doc/aggregation.md
	// internal-aggregation = true
	//
	// # [tags]
	// # rules = ''
	// # date = ''
	// # extra-where = ''
	// # input-file = ''
	// # output-file = ''
}

func TestReadmeComments(t *testing.T) {
	type TLS struct {
		Cipher  string `toml:"cipher"`
		Version string `toml:"version"`
	}
	type Config struct {
		Host string `toml:"host" comment:"Host IP to connect to."`
		Port int    `toml:"port" comment:"Port of the remote server."`
		TLS  TLS    `toml:"TLS,commented" comment:"Encryption parameters (optional)"`
	}
	example := Config{
		Host: "127.0.0.1",
		Port: 4242,
		TLS: TLS{
			Cipher:  "AEAD-AES128-GCM-SHA256",
			Version: "TLS 1.3",
		},
	}
	out, err := toml.Marshal(example)
	assert.NoError(t, err)

	expected := `# Host IP to connect to.
host = '127.0.0.1'
# Port of the remote server.
port = 4242

# Encryption parameters (optional)
# [TLS]
# cipher = 'AEAD-AES128-GCM-SHA256'
# version = 'TLS 1.3'
`
	assert.Equal(t, expected, string(out))
}

// TestMarshalIssue975 tests that nil pointer values in maps are marshaled as
// empty tables, allowing round-trip marshaling to work correctly.
// See https://github.com/pelletier/go-toml/issues/975
func TestMarshalIssue975(t *testing.T) {
	// Test case from the issue: map[string]*struct{}
	oldMap := map[string]*struct{}{
		"foo": nil,
	}

	doc, err := toml.Marshal(&oldMap)
	assert.NoError(t, err)
	assert.Equal(t, "[foo]\n", string(doc))

	var newMap map[string]*struct{}
	err = toml.Unmarshal(doc, &newMap)
	assert.NoError(t, err)

	// Verify the key is preserved after round-trip
	_, exists := newMap["foo"]
	assert.True(t, exists, "key 'foo' should exist after round-trip")
}

type failKey struct{}

func (failKey) MarshalText() ([]byte, error) {
	return nil, errors.New("nope")
}

type ptrTextKey struct {
	V string
}

func (k *ptrTextKey) MarshalText() ([]byte, error) {
	return []byte(k.V), nil
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("disk full")
}

func TestMarshalRootErrors(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) {
		_, err := toml.Marshal(nil)
		assert.Error(t, err)
	})
	t.Run("nil pointer", func(t *testing.T) {
		_, err := toml.Marshal((*int)(nil))
		assert.Error(t, err)
	})
	t.Run("scalar root", func(t *testing.T) {
		_, err := toml.Marshal(42)
		assert.Error(t, err)
	})
	t.Run("value-kind struct root", func(t *testing.T) {
		_, err := toml.Marshal(time.Time{})
		assert.Error(t, err)
	})
	t.Run("write error", func(t *testing.T) {
		err := toml.NewEncoder(failWriter{}).Encode(map[string]int{"a": 1})
		assert.Error(t, err)
	})
}

func TestMarshalNilTableValues(t *testing.T) {
	type sub struct {
		X int
	}
	t.Run("nil struct pointer map value", func(t *testing.T) {
		out, err := toml.Marshal(map[string]*sub{"a": nil})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "[a]"))
	})
	t.Run("nil scalar pointer map value", func(t *testing.T) {
		out, err := toml.Marshal(map[string]*int{"a": nil})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "a = 0"))
	})
	t.Run("nil pointer array element encodes zero value", func(t *testing.T) {
		out, err := toml.Marshal(map[string][]*sub{"a": {nil}})
		assert.NoError(t, err)
		assert.Equal(t, "a = [{X = 0}]\n", string(out))
	})
}

func TestMarshalMapKeys(t *testing.T) {
	t.Run("nil interface key", func(t *testing.T) {
		_, err := toml.Marshal(map[interface{}]int{nil: 1})
		assert.Error(t, err)
	})
	t.Run("text marshaler key", func(t *testing.T) {
		out, err := toml.Marshal(map[time.Time]int{{}: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "0001-01-01"))
	})
	t.Run("pointer receiver text marshaler key", func(t *testing.T) {
		out, err := toml.Marshal(map[ptrTextKey]int{{V: "k"}: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "k = 1"))
	})
	t.Run("failing text marshaler key", func(t *testing.T) {
		_, err := toml.Marshal(map[failKey]int{{}: 1})
		assert.Error(t, err)
	})
	t.Run("integer keys", func(t *testing.T) {
		out, err := toml.Marshal(map[int64]int{-1: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "-1 = 1"))
		out, err = toml.Marshal(map[uint64]int{7: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "7 = 1"))
	})
}

type selfEmbed struct {
	*selfEmbed //nolint:unused // exercises recursive embedded type plans
	X          int
}

type planInner struct {
	X int
	Y int
}

type planOuter struct {
	planInner
	X int
}

type tagEmbed struct {
	planInner `toml:",inline"`
}

func TestMarshalEncodePlans(t *testing.T) {
	t.Run("recursive embedded type", func(t *testing.T) {
		v := selfEmbed{X: 1}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 1"))
	})
	t.Run("shadowed embedded field", func(t *testing.T) {
		v := planOuter{planInner: planInner{X: 5, Y: 2}}
		v.X = 9
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 9"))
		assert.True(t, strings.Contains(string(out), "Y = 2"))
		assert.False(t, strings.Contains(string(out), "X = 5"))
	})
	t.Run("tagged embedded without name", func(t *testing.T) {
		v := tagEmbed{planInner{X: 3, Y: 4}}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 3"))
	})
	t.Run("nil embedded pointer", func(t *testing.T) {
		v := selfEmbed{X: 2}
		out, err := toml.Marshal(&v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 2"))
	})
}

type myInt int

type embedNonStruct struct {
	myInt
	A int
}

func TestMarshalEmbeddedNonStructUnexported(t *testing.T) {
	out, err := toml.Marshal(embedNonStruct{myInt: 1, A: 2})
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(out), "myInt"))
	assert.True(t, strings.Contains(string(out), "A = 2"))
}

func TestMarshalOmitemptyUncommonKind(t *testing.T) {
	v := struct {
		F func() `toml:",omitempty"`
	}{F: func() {}}
	_, err := toml.Marshal(v)
	assert.Error(t, err)
}

type ptrText struct {
	V string
}

func (p *ptrText) MarshalText() ([]byte, error) {
	return []byte(p.V), nil
}

func TestMarshalPtrReceiverTextMarshalerMapValue(t *testing.T) {
	out, err := toml.Marshal(map[string]ptrText{"a": {V: "x"}})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), `a = 'x'`))
}

func TestMarshalJSONNumbers(t *testing.T) {
	enc := func(v interface{}) (string, error) {
		var buf strings.Builder
		err := toml.NewEncoder(&buf).SetMarshalJSONNumbers(true).Encode(v)
		return buf.String(), err
	}
	t.Run("invalid", func(t *testing.T) {
		_, err := enc(map[string]json.Number{"a": json.Number("abc")})
		assert.Error(t, err)
	})
	t.Run("empty is zero", func(t *testing.T) {
		out, err := enc(map[string]json.Number{"a": json.Number("")})
		assert.NoError(t, err)
		assert.Equal(t, "a = 0\n", out)
	})
	t.Run("float", func(t *testing.T) {
		out, err := enc(map[string]json.Number{"a": json.Number("1.5")})
		assert.NoError(t, err)
		assert.Equal(t, "a = 1.5\n", out)
	})
}

func TestMarshalElementErrors(t *testing.T) {
	t.Run("array element error", func(t *testing.T) {
		_, err := toml.Marshal(map[string][]interface{}{"a": {func() {}}})
		assert.Error(t, err)
	})
	t.Run("multiline array element error", func(t *testing.T) {
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.SetArraysMultiline(true)
		err := enc.Encode(map[string][]interface{}{"a": {func() {}}})
		assert.Error(t, err)
	})
	t.Run("inline table element error", func(t *testing.T) {
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.SetTablesInline(true)
		err := enc.Encode(map[string]map[string]interface{}{"a": {"b": func() {}}})
		assert.Error(t, err)
	})
	t.Run("multiline array of arrays", func(t *testing.T) {
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.SetArraysMultiline(true)
		err := enc.Encode(map[string][]interface{}{"a": {[]interface{}{int64(1), int64(2)}, "x"}})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(buf.String(), "1"))
	})
}

func TestMarshalMultilineArrayIndentWithoutIndentTables(t *testing.T) {
	// Regression for #1075: when tables are not indented, a multiline array
	// nested under tables must have its elements indented relative to its key
	// (column zero) rather than to the table nesting depth.
	type exclusions struct {
		Paths []string `toml:"paths"`
	}
	type linters struct {
		Exclusions exclusions `toml:"exclusions"`
	}
	v := struct {
		Linters linters `toml:"linters"`
	}{
		Linters: linters{
			Exclusions: exclusions{
				Paths: []string{"third_party$", "builtin$"},
			},
		},
	}

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	enc.SetArraysMultiline(true)
	assert.NoError(t, enc.Encode(v))

	expected := `[linters]
[linters.exclusions]
paths = [
  'third_party$',
  'builtin$'
]
`
	assert.Equal(t, expected, buf.String())

	// With SetIndentTables, the key and its array elements are indented
	// consistently to the table nesting depth.
	var buf2 strings.Builder
	enc2 := toml.NewEncoder(&buf2)
	enc2.SetArraysMultiline(true)
	enc2.SetIndentTables(true)
	assert.NoError(t, enc2.Encode(v))

	expectedIndented := `[linters]
  [linters.exclusions]
    paths = [
      'third_party$',
      'builtin$'
    ]
`
	assert.Equal(t, expectedIndented, buf2.String())
}

// TestMarshalIssue1075 reproduces the exact serialization reported in #1075.
// It mirrors golangci-lint's "version two" config: every field tagged
// `,multiline,omitempty`, an embedded struct, a non-nil but empty slice, and
// the default encoder. Before the fix this produced multiline-wrapped short
// strings (version = """\n2"""), array elements indented to the table depth,
// and empty [linters.settings(.tagliatelle.case)] tables. The expected output
// below is byte-for-byte golangci-lint's empty.golden.toml.
func TestMarshalIssue1075(t *testing.T) {
	strptr := func(s string) *string { return &s }

	type tagBase struct {
		Rules         map[string]string `toml:"rules,multiline,omitempty"`
		UseFieldName  *bool             `toml:"use-field-name,multiline,omitempty"`
		IgnoredFields []string          `toml:"ignored-fields,multiline,omitempty"`
	}
	type tagCase struct {
		tagBase
		Overrides []string `toml:"overrides,multiline,omitempty"`
	}
	type tagSettings struct {
		Case tagCase `toml:"case,multiline,omitempty"`
	}
	type lintersSettings struct {
		Tagliatelle tagSettings `toml:"tagliatelle,multiline,omitempty"`
	}
	type lintersExclusions struct {
		Generated *string  `toml:"generated,multiline,omitempty"`
		Presets   []string `toml:"presets,multiline,omitempty"`
		Paths     []string `toml:"paths,multiline,omitempty"`
	}
	type linters struct {
		Settings   lintersSettings   `toml:"settings,multiline,omitempty"`
		Exclusions lintersExclusions `toml:"exclusions,multiline,omitempty"`
	}
	type formattersExclusions struct {
		Generated *string  `toml:"generated,multiline,omitempty"`
		Paths     []string `toml:"paths,multiline,omitempty"`
	}
	type formatters struct {
		Exclusions formattersExclusions `toml:"exclusions,multiline,omitempty"`
	}
	type config struct {
		Version    *string    `toml:"version,multiline,omitempty"`
		Linters    linters    `toml:"linters,multiline,omitempty"`
		Formatters formatters `toml:"formatters,multiline,omitempty"`
	}

	c := config{
		Version: strptr("2"),
		Linters: linters{
			Settings: lintersSettings{
				// Non-nil but empty slice: the exact trigger for the empty
				// [linters.settings.tagliatelle.case] tables.
				Tagliatelle: tagSettings{Case: tagCase{Overrides: []string{}}},
			},
			Exclusions: lintersExclusions{
				Generated: strptr("lax"),
				Presets:   []string{"comments", "common-false-positives", "legacy", "std-error-handling"},
				Paths:     []string{"third_party$", "builtin$", "examples$"},
			},
		},
		Formatters: formatters{
			Exclusions: formattersExclusions{
				Generated: strptr("lax"),
				Paths:     []string{"third_party$", "builtin$", "examples$"},
			},
		},
	}

	expected := `version = '2'

[linters]
[linters.exclusions]
generated = 'lax'
presets = [
  'comments',
  'common-false-positives',
  'legacy',
  'std-error-handling'
]
paths = [
  'third_party$',
  'builtin$',
  'examples$'
]

[formatters]
[formatters.exclusions]
generated = 'lax'
paths = [
  'third_party$',
  'builtin$',
  'examples$'
]
`

	b, err := toml.Marshal(c)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(b))
}

func TestMarshalStringEscapes(t *testing.T) {
	t.Run("invalid utf8 in basic string", func(t *testing.T) {
		out, err := toml.Marshal(map[string]string{"a": "\xff"})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "\\u00FF"))
	})
	t.Run("multiline escapes", func(t *testing.T) {
		v := struct {
			S string `toml:",multiline"`
		}{S: "a\"\"\"\"b\\c\bd\fe\rf\tg\x01h\xffi\nj"}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		s := string(out)
		assert.True(t, strings.Contains(s, "\\\"\\\"\\\"\\\""))
		assert.True(t, strings.Contains(s, "\\\\"))
		assert.True(t, strings.Contains(s, "\\b"))
		assert.True(t, strings.Contains(s, "\\f"))
		assert.True(t, strings.Contains(s, "\\r"))
		assert.True(t, strings.Contains(s, "\\u0001"))
		assert.True(t, strings.Contains(s, "\\u00FF"))
	})
	t.Run("multiline short quote runs kept", func(t *testing.T) {
		v := struct {
			S string `toml:",multiline"`
		}{S: `a""b`}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), `a""b`))
	})
}

type failPtrKey struct{ V int }

func (*failPtrKey) MarshalText() ([]byte, error) {
	return nil, errors.New("nope")
}

func TestMarshalMapKeysMore(t *testing.T) {
	t.Run("failing pointer receiver text marshaler key", func(t *testing.T) {
		_, err := toml.Marshal(map[failPtrKey]int{{V: 1}: 1})
		assert.Error(t, err)
	})
}

type commentedTagged struct {
	A int `commented:"true"`
}

func TestMarshalStandaloneCommentedTag(t *testing.T) {
	out, err := toml.Marshal(commentedTagged{A: 1})
	assert.NoError(t, err)
	assert.Equal(t, "# A = 1\n", string(out))
}

type nilEmbedInner struct {
	Z int
}

type nilEmbedOuter struct {
	*nilEmbedInner
	X int
}

func TestMarshalNilEmbeddedPointer(t *testing.T) {
	out, err := toml.Marshal(nilEmbedOuter{X: 1})
	assert.NoError(t, err)
	assert.Equal(t, "X = 1\n", string(out))
}

func TestMarshalInterfaceHoldingNilPointers(t *testing.T) {
	type sub struct{ X int }
	t.Run("nil struct pointer in interface", func(t *testing.T) {
		out, err := toml.Marshal(map[string]interface{}{"a": (*sub)(nil)})
		assert.NoError(t, err)
		// Interface-held nil pointers encode as the zero value of their
		// element type, like nil pointer map values.
		assert.Equal(t, "a = {X = 0}\n", string(out))
	})
	t.Run("nil map pointer in interface", func(t *testing.T) {
		out, err := toml.Marshal(map[string]interface{}{"a": (*map[string]int)(nil)})
		assert.NoError(t, err)
		assert.Equal(t, "a = {}\n", string(out))
	})
	t.Run("nil slice pointer in interface", func(t *testing.T) {
		out, err := toml.Marshal(map[string]interface{}{"a": (*[]int)(nil)})
		assert.NoError(t, err)
		assert.Equal(t, "a = []\n", string(out))
	})
}

func TestMarshalNestedTableValueError(t *testing.T) {
	_, err := toml.Marshal(map[string]map[string]interface{}{"a": {"b": func() {}}})
	assert.Error(t, err)
}

func TestMarshalInlineTableKeyError(t *testing.T) {
	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	enc.SetTablesInline(true)
	err := enc.Encode(map[string]map[failKey]int{"a": {{}: 1}})
	assert.Error(t, err)
}
