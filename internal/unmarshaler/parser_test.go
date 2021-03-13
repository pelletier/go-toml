package unmarshaler

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestParser_Simple(t *testing.T) {
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
