package unmarshaler

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/stretchr/testify/require"
)

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
