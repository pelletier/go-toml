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
	t.Run("slice of string", func(t *testing.T) {
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
	})

	t.Run("slice of interfaces for strings", func(t *testing.T) {
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
			Foo []interface{}
		}

		x := Doc{}
		err := unmarshaler.FromAst(root, &x)
		require.NoError(t, err)
		assert.Equal(t, Doc{Foo: []interface{}{"hello", "world"}}, x)
	})

	t.Run("slice of interfaces with slices", func(t *testing.T) {
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
								Kind: ast.Array,
								Children: []ast.Node{
									{
										Kind: ast.String,
										Data: []byte(`inner1`),
									},
									{
										Kind: ast.String,
										Data: []byte(`inner2`),
									},
								},
							},
						},
					},
				},
			},
		}

		type Doc struct {
			Foo []interface{}
		}

		x := Doc{}
		err := unmarshaler.FromAst(root, &x)
		require.NoError(t, err)
		assert.Equal(t, Doc{Foo: []interface{}{"hello", []interface{}{"inner1", "inner2"}}}, x)
	})
}
