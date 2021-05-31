package toml

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
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
			desc:  "float zero",
			input: `0.0`,
			kind:  ast.Float,
		},
		{
			desc:  "float positive zero",
			input: `+0.0`,
			kind:  ast.Float,
		},
		{
			desc:  "float negative zero",
			input: `-0.0`,
			kind:  ast.Float,
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
		e := e
		t.Run(e.desc, func(t *testing.T) {

			p := parser{}
			p.Reset([]byte(`A = ` + e.input))
			p.NextExpression()
			err := p.Error()
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				expected := astNode{
					Kind: ast.KeyValue,
					Children: []astNode{
						{Kind: e.kind, Data: []byte(e.input)},
						{Kind: ast.Key, Data: []byte(`A`)},
					},
				}
				compareNode(t, expected, p.Expression())
			}
		})
	}
}

type (
	astNode struct {
		Kind     ast.Kind
		Data     []byte
		Children []astNode
	}
)

func compareNode(t *testing.T, e astNode, n ast.Node) {
	t.Helper()
	require.Equal(t, e.Kind, n.Kind)
	require.Equal(t, e.Data, n.Data)

	compareIterator(t, e.Children, n.Children())
}

func compareIterator(t *testing.T, expected []astNode, actual ast.Iterator) {
	t.Helper()
	idx := 0

	for actual.Next() {
		n := actual.Node()

		if idx >= len(expected) {
			t.Fatal("extra child in actual tree")
		}
		e := expected[idx]

		compareNode(t, e, n)

		idx++
	}

	if idx < len(expected) {
		t.Fatal("missing children in actual", "idx =", idx, "expected =", len(expected))
	}
}

//nolint:funlen
func TestParser_AST(t *testing.T) {

	examples := []struct {
		desc  string
		input string
		ast   astNode
		err   bool
	}{
		{
			desc:  "simple string assignment",
			input: `A = "hello"`,
			ast: astNode{
				Kind: ast.KeyValue,
				Children: []astNode{
					{
						Kind: ast.String,
						Data: []byte(`hello`),
					},
					{
						Kind: ast.Key,
						Data: []byte(`A`),
					},
				},
			},
		},
		{
			desc:  "simple bool assignment",
			input: `A = true`,
			ast: astNode{
				Kind: ast.KeyValue,
				Children: []astNode{
					{
						Kind: ast.Bool,
						Data: []byte(`true`),
					},
					{
						Kind: ast.Key,
						Data: []byte(`A`),
					},
				},
			},
		},
		{
			desc:  "array of strings",
			input: `A = ["hello", ["world", "again"]]`,
			ast: astNode{
				Kind: ast.KeyValue,
				Children: []astNode{
					{
						Kind: ast.Array,
						Children: []astNode{
							{
								Kind: ast.String,
								Data: []byte(`hello`),
							},
							{
								Kind: ast.Array,
								Children: []astNode{
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
					{
						Kind: ast.Key,
						Data: []byte(`A`),
					},
				},
			},
		},
		{
			desc:  "array of arrays of strings",
			input: `A = ["hello", "world"]`,
			ast: astNode{
				Kind: ast.KeyValue,
				Children: []astNode{
					{
						Kind: ast.Array,
						Children: []astNode{
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
					{
						Kind: ast.Key,
						Data: []byte(`A`),
					},
				},
			},
		},
		{
			desc:  "inline table",
			input: `name = { first = "Tom", last = "Preston-Werner" }`,
			ast: astNode{
				Kind: ast.KeyValue,
				Children: []astNode{
					{
						Kind: ast.InlineTable,
						Children: []astNode{
							{
								Kind: ast.KeyValue,
								Children: []astNode{
									{Kind: ast.String, Data: []byte(`Tom`)},
									{Kind: ast.Key, Data: []byte(`first`)},
								},
							},
							{
								Kind: ast.KeyValue,
								Children: []astNode{
									{Kind: ast.String, Data: []byte(`Preston-Werner`)},
									{Kind: ast.Key, Data: []byte(`last`)},
								},
							},
						},
					},
					{
						Kind: ast.Key,
						Data: []byte(`name`),
					},
				},
			},
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {

			p := parser{}
			p.Reset([]byte(e.input))
			p.NextExpression()
			err := p.Error()
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				compareNode(t, e.ast, p.Expression())
			}
		})
	}
}
