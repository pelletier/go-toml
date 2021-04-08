package toml_test

import (
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
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
			desc: "simple string array",
			v: map[string][]string{
				"array": {"one", "two", "three"},
			},
			expected: `array = ['one', 'two', 'three']`,
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
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
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
