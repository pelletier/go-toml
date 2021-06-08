package toml_test

import (
	"encoding/json"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal_RecursiveTable(t *testing.T) {
	type Foo struct {
		I int
		F *Foo
	}

	examples := []struct {
		desc     string
		input    string
		expected string
		err      bool
	}{
		{
			desc: "simplest",
			input: `
				I=1
			`,
			expected: `{"I":1,"F":null}`,
		},
		{
			desc: "depth 1",
			input: `
				I=1
				[F]
				I=2
			`,
			expected: `{"I":1,"F":{"I":2,"F":null}}`,
		},
		{
			desc: "depth 3",
			input: `
				I=1
				[F]
				I=2
				[F.F]
				I=3
			`,
			expected: `{"I":1,"F":{"I":2,"F":{"I":3,"F":null}}}`,
		},
		{
			desc: "depth 4",
			input: `
				I=1
				[F]
				I=2
				[F.F]
				I=3
				[F.F.F]
				I=4
			`,
			expected: `{"I":1,"F":{"I":2,"F":{"I":3,"F":{"I":4,"F":null}}}}`,
		},
	}

	for _, ex := range examples {
		e := ex
		t.Run(e.desc, func(t *testing.T) {
			foo := Foo{}
			err := toml.Unmarshal([]byte(e.input), &foo)
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				j, err := json.Marshal(foo)
				require.NoError(t, err)
				assert.Equal(t, e.expected, string(j))
			}
		})
	}
}

func TestUnmarshal_RecursiveTableArray(t *testing.T) {
	type Foo struct {
		I int
		F []*Foo
	}

	examples := []struct {
		desc     string
		input    string
		expected string
		err      bool
	}{
		{
			desc: "simplest",
			input: `
				I=1
				F=[]
			`,
			expected: `{"I":1,"F":[]}`,
		},
		{
			desc: "depth 1",
			input: `
				I=1
				[[F]]
				I=2
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[]}]}`,
		},
		{
			desc: "depth 2",
			input: `
				I=1
				[[F]]
				I=2
				[[F.F]]
				I=3
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[{"I":3,"F":[]}]}]}`,
		},
		{
			desc: "depth 3",
			input: `
				I=1
				[[F]]
				I=2
				[[F.F]]
				I=3
				[[F.F.F]]
				I=4
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[{"I":3,"F":[{"I":4,"F":[]}]}]}]}`,
		},
		{
			desc: "depth 4",
			input: `
				I=1
				[[F]]
				I=2
				[[F.F]]
				I=3
				[[F.F.F]]
				I=4
				[[F.F.F.F]]
				I=5
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[{"I":3,"F":[{"I":4,"F":[{"I":5,"F":[]}]}]}]}]}`,
		},
	}

	for _, ex := range examples {
		e := ex
		t.Run(e.desc, func(t *testing.T) {
			foo := Foo{}
			err := toml.Unmarshal([]byte(e.input), &foo)
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				j, err := json.Marshal(foo)
				require.NoError(t, err)
				assert.Equal(t, e.expected, string(j))
			}
		})
	}
}
