package toml

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestParser_AST_Numbers(t *testing.T) {
	examples := []struct {
		desc  string
		input string
		kind  ast.Kind
		err   bool
	}{
		{
			desc:  "integer just digits",
			input: `1234`,
			kind:  ast.Integer,
		},
		{
			desc:  "integer zero",
			input: `0`,
			kind:  ast.Integer,
		},
		{
			desc:  "integer sign",
			input: `+99`,
			kind:  ast.Integer,
		},
		{
			desc:  "integer hex uppercase",
			input: `0xDEADBEEF`,
			kind:  ast.Integer,
		},
		{
			desc:  "integer hex lowercase",
			input: `0xdead_beef`,
			kind:  ast.Integer,
		},
		{
			desc:  "integer octal",
			input: `0o01234567`,
			kind:  ast.Integer,
		},
		{
			desc:  "integer binary",
			input: `0b11010110`,
			kind:  ast.Integer,
		},
		{
			desc:  "float pi",
			input: `3.1415`,
			kind:  ast.Float,
		},
		{
			desc:  "float negative",
			input: `-0.01`,
			kind:  ast.Float,
		},
		{
			desc:  "float signed exponent",
			input: `5e+22`,
			kind:  ast.Float,
		},
		{
			desc:  "float exponent lowercase",
			input: `1e06`,
			kind:  ast.Float,
		},
		{
			desc:  "float exponent uppercase",
			input: `-2E-2`,
			kind:  ast.Float,
		},
		{
			desc:  "float fractional with exponent",
			input: `6.626e-34`,
			kind:  ast.Float,
		},
		{
			desc:  "float underscores",
			input: `224_617.445_991_228`,
			kind:  ast.Float,
		},
		{
			desc:  "inf",
			input: `inf`,
			kind:  ast.Float,
		},
		{
			desc:  "inf negative",
			input: `-inf`,
			kind:  ast.Float,
		},
		{
			desc:  "inf positive",
			input: `+inf`,
			kind:  ast.Float,
		},
		{
			desc:  "nan",
			input: `nan`,
			kind:  ast.Float,
		},
		{
			desc:  "nan negative",
			input: `-nan`,
			kind:  ast.Float,
		},
		{
			desc:  "nan positive",
			input: `+nan`,
			kind:  ast.Float,
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			p := parser{}
			err := p.parse([]byte(`A = ` + e.input))
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				expected := ast.Root{
					ast.Node{
						Kind: ast.KeyValue,
						Children: []ast.Node{
							{Kind: ast.Key, Data: []byte(`A`)},
							{Kind: e.kind, Data: []byte(e.input)},
						},
					},
				}

				require.Equal(t, expected, p.tree)
			}
		})
	}
}

func TestParser_AST(t *testing.T) {
	examples := []struct {
		desc  string
		input string
		ast   ast.Root
		err   bool
	}{
		{
			desc:  "simple string assignment",
			input: `A = "hello"`,
			ast: ast.Root{
				ast.Node{
					Kind: ast.KeyValue,
					Children: []ast.Node{
						{
							Kind: ast.Key,
							Data: []byte(`A`),
						},
						{
							Kind: ast.String,
							Data: []byte(`hello`),
						},
					},
				},
			},
		},
		{
			desc:  "simple bool assignment",
			input: `A = true`,
			ast: ast.Root{
				ast.Node{
					Kind: ast.KeyValue,
					Children: []ast.Node{
						{
							Kind: ast.Key,
							Data: []byte(`A`),
						},
						{
							Kind: ast.Bool,
							Data: []byte(`true`),
						},
					},
				},
			},
		},
		{
			desc:  "array of strings",
			input: `A = ["hello", ["world", "again"]]`,
			ast: ast.Root{
				ast.Node{
					Kind: ast.KeyValue,
					Children: []ast.Node{
						{
							Kind: ast.Key,
							Data: []byte(`A`),
						},
						{
							Kind: ast.Array,
							Children: []ast.Node{
								{
									Kind: ast.String,
									Data: []byte(`hello`),
								},
								{
									Kind: ast.Array,
									Children: []ast.Node{
										{
											Kind: ast.String,
											Data: []byte(`world`),
										},
										{
											Kind: ast.String,
											Data: []byte(`again`),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:  "array of arrays of strings",
			input: `A = ["hello", "world"]`,
			ast: ast.Root{
				ast.Node{
					Kind: ast.KeyValue,
					Children: []ast.Node{
						{
							Kind: ast.Key,
							Data: []byte(`A`),
						},
						{
							Kind: ast.Array,
							Children: []ast.Node{
								{
									Kind: ast.String,
									Data: []byte(`hello`),
								},
								{
									Kind: ast.String,
									Data: []byte(`world`),
								},
							},
						},
					},
				},
			},
		},
		{
			desc:  "inline table",
			input: `name = { first = "Tom", last = "Preston-Werner" }`,
			ast: ast.Root{
				ast.Node{
					Kind: ast.KeyValue,
					Children: []ast.Node{
						{
							Kind: ast.Key,
							Data: []byte(`name`),
						},
						{
							Kind: ast.InlineTable,
							Children: []ast.Node{
								{
									Kind: ast.KeyValue,
									Children: []ast.Node{
										{Kind: ast.Key, Data: []byte(`first`)},
										{Kind: ast.String, Data: []byte(`Tom`)},
									},
								},
								{
									Kind: ast.KeyValue,
									Children: []ast.Node{
										{Kind: ast.Key, Data: []byte(`last`)},
										{Kind: ast.String, Data: []byte(`Preston-Werner`)},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			p := parser{}
			err := p.parse([]byte(e.input))
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, e.ast, p.tree)
			}
		})
	}
}
