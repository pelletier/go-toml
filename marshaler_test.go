package toml_test

import (
	"bytes"
	"encoding/json"
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
			//nolint:godox
			// TODO: this test is flaky because output changes depending on
			//   the map iteration order.
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
			desc: "simple string slice",
			v: map[string][]string{
				"slice": {"one", "two", "three"},
			},
			expected: `slice = ['one', 'two', 'three']`,
		},
		{
			desc:     "empty string slice",
			v:        map[string][]string{},
			expected: ``,
		},
		{
			desc:     "map",
			v:        map[string][]string{},
			expected: ``,
		},
		{
			desc: "nested string slices",
			v: map[string][][]string{
				"slice": {{"one", "two"}, {"three"}},
			},
			expected: `slice = [['one', 'two'], ['three']]`,
		},
		{
			desc: "mixed strings and nested string slices",
			v: map[string][]interface{}{
				"slice": {"a string", []string{"one", "two"}, "last"},
			},
			expected: `slice = ['a string', ['one', 'two'], 'last']`,
		},
		{
			desc: "slice of maps",
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
			desc: "structs in slice with interfaces",
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
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			b, err := toml.Marshal(e.v)
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				equalStringsIgnoreNewlines(t, e.expected, string(b))
			}
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
