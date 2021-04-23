package toml_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestMarshal(t *testing.T) {
	t.Parallel()

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
			expected: "hello = 'world'",
		},
		{
			desc: "map with new line in key",
			v: map[string]string{
				"hel\nlo": "world",
			},
			err: true,
		},
		{
			desc: `map with " in key`,
			v: map[string]string{
				`hel"lo`: "world",
			},
			expected: `'hel"lo' = 'world'`,
		},
		{
			desc: "map in map and string",
			v: map[string]map[string]string{
				"table": {
					"hello": "world",
				},
			},
			expected: `
[table]
hello = 'world'`,
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
			expected: `
[this]
[this.is]
a = 'test'`,
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
			expected: `
[this]
also = 'that'
[this.is]
a = 'test'`,
		},
		{
			desc: "simple string array",
			v: map[string][]string{
				"array": {"one", "two", "three"},
			},
			expected: `array = ['one', 'two', 'three']`,
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
			expected: `array = [['one', 'two'], ['three']]`,
		},
		{
			desc: "mixed strings and nested string arrays",
			v: map[string][]interface{}{
				"array": {"a string", []string{"one", "two"}, "last"},
			},
			expected: `array = ['a string', ['one', 'two'], 'last']`,
		},
		{
			desc: "array of maps",
			v: map[string][]map[string]string{
				"top": {
					{"map1.1": "v1.1"},
					{"map2.1": "v2.1"},
				},
			},
			expected: `
[[top]]
'map1.1' = 'v1.1'
[[top]]
'map2.1' = 'v2.1'
`,
		},
		{
			desc: "map with two keys",
			v: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: `
key1 = 'value1'
key2 = 'value2'`,
		},
		{
			desc: "simple struct",
			v: struct {
				A string
			}{
				A: "foo",
			},
			expected: `A = 'foo'`,
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
			expected: `
[A]
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
			expected: `
[root]
[[root.nested]]
name = 'Bob'
[[root.nested]]
name = 'Alice'
`,
		},
		{
			desc: "string escapes",
			v: map[string]interface{}{
				"a": `'"\`,
			},
			expected: `a = "'\"\\"`,
		},
		{
			desc: "string utf8 low",
			v: map[string]interface{}{
				"a": "'Ę",
			},
			expected: `a = "'Ę"`,
		},
		{
			desc: "string utf8 low 2",
			v: map[string]interface{}{
				"a": "'\u10A85",
			},
			expected: "a = \"'\u10A85\"",
		},
		{
			desc: "string utf8 low 2",
			v: map[string]interface{}{
				"a": "'\u10A85",
			},
			expected: "a = \"'\u10A85\"",
		},
		{
			desc: "emoji",
			v: map[string]interface{}{
				"a": "'😀",
			},
			expected: "a = \"'😀\"",
		},
		{
			desc: "control char",
			v: map[string]interface{}{
				"a": "'\u001A",
			},
			expected: `a = "'\u001A"`,
		},
		{
			desc: "multi-line string",
			v: map[string]interface{}{
				"a": "hello\nworld",
			},
			expected: `a = "hello\nworld"`,
		},
		{
			desc: "multi-line forced",
			v: struct {
				A string `multiline:"true"`
			}{
				A: "hello\nworld",
			},
			expected: `A = """
hello
world"""`,
		},
		{
			desc: "inline field",
			v: struct {
				A map[string]string `inline:"true"`
				B map[string]string
			}{
				A: map[string]string{
					"isinline": "yes",
				},
				B: map[string]string{
					"isinline": "no",
				},
			},
			expected: `
A = {isinline = 'yes'}
[B]
isinline = 'no'
`,
		},
		{
			desc: "mutiline array int",
			v: struct {
				A []int `multiline:"true"`
				B []int
			}{
				A: []int{1, 2, 3, 4},
				B: []int{1, 2, 3, 4},
			},
			expected: `
A = [
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
				A [][]int `multiline:"true"`
			}{
				A: [][]int{{1, 2}, {3, 4}},
			},
			expected: `
A = [
  [1, 2],
  [3, 4]
]
`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			b, err := toml.Marshal(e.v)
			if e.err {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			equalStringsIgnoreNewlines(t, e.expected, string(b))

			// make sure the output is always valid TOML
			defaultMap := map[string]interface{}{}
			err = toml.Unmarshal(b, &defaultMap)
			require.NoError(t, err)

			testWithAllFlags(t, func(t *testing.T, flags int) {
				t.Helper()

				var buf bytes.Buffer
				enc := toml.NewEncoder(&buf)
				setFlags(enc, flags)

				err := enc.Encode(e.v)
				require.NoError(t, err)

				inlineMap := map[string]interface{}{}
				err = toml.Unmarshal(buf.Bytes(), &inlineMap)
				require.NoError(t, err)

				require.Equal(t, defaultMap, inlineMap)
			})
		})
	}
}

type flagsSetters []struct {
	name string
	f    func(enc *toml.Encoder, flag bool)
}

var allFlags = flagsSetters{
	{"arrays-multiline", (*toml.Encoder).SetArraysMultiline},
	{"tables-inline", (*toml.Encoder).SetTablesInline},
}

func setFlags(enc *toml.Encoder, flags int) {
	for i := 0; i < len(allFlags); i++ {
		enabled := flags&1 > 0
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

func equalStringsIgnoreNewlines(t *testing.T, expected string, actual string) {
	t.Helper()
	cutset := "\n"
	assert.Equal(t, strings.Trim(expected, cutset), strings.Trim(actual, cutset))
}

func TestIssue436(t *testing.T) {
	t.Parallel()

	data := []byte(`{"a": [ { "b": { "c": "d" } } ]}`)

	var v interface{}
	err := json.Unmarshal(data, &v)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(v)
	require.NoError(t, err)

	expected := `
[[a]]
[a.b]
c = 'd'
`
	equalStringsIgnoreNewlines(t, expected, buf.String())
}

func TestIssue424(t *testing.T) {
	t.Parallel()

	type Message1 struct {
		Text string
	}

	type Message2 struct {
		Text string `multiline:"true"`
	}

	msg1 := Message1{"Hello\\World"}
	msg2 := Message2{"Hello\\World"}

	toml1, err := toml.Marshal(msg1)
	require.NoError(t, err)

	toml2, err := toml.Marshal(msg2)
	require.NoError(t, err)

	msg1parsed := Message1{}
	err = toml.Unmarshal(toml1, &msg1parsed)
	require.NoError(t, err)
	require.Equal(t, msg1, msg1parsed)

	msg2parsed := Message2{}
	err = toml.Unmarshal(toml2, &msg2parsed)
	require.NoError(t, err)
	require.Equal(t, msg2, msg2parsed)
}
