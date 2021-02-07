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

func TestUnmarshalTable(t *testing.T) {
	x := struct {
		Foo struct {
			A string
			B string
			C string
		}
		Bar struct {
			D string
		}
		E string
	}{}
	err := Unmarshal([]byte(`

E = "E"
Foo.C = "C"

[Foo]
A = "A"
B = 'B'

[Bar]
D = "D"

`), &x)
	require.NoError(t, err)
	assert.Equal(t, "A", x.Foo.A)
	assert.Equal(t, "B", x.Foo.B)
	assert.Equal(t, "C", x.Foo.C)
	assert.Equal(t, "D", x.Bar.D)
	assert.Equal(t, "E", x.E)
}
