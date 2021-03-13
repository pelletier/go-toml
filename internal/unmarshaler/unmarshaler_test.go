package unmarshaler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/pelletier/go-toml/v2/internal/unmarshaler"
)

func TestFromAst_KV(t *testing.T) {
	root := ast.Root{
		ast.Node{
			Kind: ast.KeyValue,
			Children: []ast.Node{
				{
					Kind: ast.Key,
					Data: []byte(`Foo`),
				},
				{
					Kind: ast.String,
					Data: []byte(`hello`),
				},
			},
		},
	}

	type Doc struct {
		Foo string
	}

	x := Doc{}
	err := unmarshaler.FromAst(root, &x)
	require.NoError(t, err)
	assert.Equal(t, Doc{Foo: "hello"}, x)
}

func TestFromAst_Slice(t *testing.T) {
	root := ast.Root{
		ast.Node{
			Kind: ast.KeyValue,
			Children: []ast.Node{
				{
					Kind: ast.Key,
					Data: []byte(`Foo`),
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
	}

	type Doc struct {
		Foo []string
	}

	x := Doc{}
	err := unmarshaler.FromAst(root, &x)
	require.NoError(t, err)
	assert.Equal(t, Doc{Foo: []string{"hello", "world"}}, x)
}
