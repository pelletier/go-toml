package toml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalSimple(t *testing.T) {
	x := struct{ Foo string }{}
	err := Unmarshal([]byte(`Foo = "hello"`), &x)
	require.NoError(t, err)
	assert.Equal(t, "hello", x.Foo)
}

func TestUnmarshalNestedStructs(t *testing.T) {
	x := struct{ Foo struct{ Bar string } }{}
	err := Unmarshal([]byte(`Foo.Bar = "hello"`), &x)
	require.NoError(t, err)
	assert.Equal(t, "hello", x.Foo.Bar)
}

func TestUnmarshalNestedStructsMultipleExpressions(t *testing.T) {
	x := struct {
		A struct{ B string }
		C string
	}{}
	err := Unmarshal([]byte(`A.B = "hello"
C = "test"`), &x)
	require.NoError(t, err)
	assert.Equal(t, "hello", x.A.B)
	assert.Equal(t, "test", x.C)
}
