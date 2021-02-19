package reflectbuild_test

import (
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/reflectbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilderSuccess(t *testing.T) {
	x := struct{}{}
	_, err := reflectbuild.NewBuilder("", &x)
	assert.NoError(t, err)
}

func TestNewBuilderNil(t *testing.T) {
	_, err := reflectbuild.NewBuilder("", nil)
	assert.Error(t, err)
}

func TestNewBuilderNonPtr(t *testing.T) {
	_, err := reflectbuild.NewBuilder("", struct{}{})
	assert.Error(t, err)
}

func TestDigField(t *testing.T) {
	x := struct {
		Field string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	assert.Error(t, b.DigField("oops"))
	assert.NoError(t, b.DigField("Field"))
	assert.Error(t, b.DigField("does not work on strings"))
}

func TestBack(t *testing.T) {
	x := struct {
		A string
		B string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	b.Save()
	assert.NoError(t, b.DigField("A"))
	assert.NoError(t, b.SetString("A"))
	b.Load()
	b.Save()
	assert.NoError(t, b.DigField("B"))
	assert.NoError(t, b.SetString("B"))
	assert.Equal(t, "A", x.A)
	assert.Equal(t, "B", x.B)
	b.Load() // back to root
	assert.Panics(t, func() {
		b.Load() // oops
	})
}

func TestReset(t *testing.T) {
	x := struct {
		A []string
		B string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	require.NoError(t, b.DigField("A"))
	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SetString("hello"))
	b.Reset()
	require.NoError(t, b.DigField("B"))
	require.NoError(t, b.SetString("world"))

	assert.Equal(t, []string{"hello"}, x.A)
	assert.Equal(t, "world", x.B)
}

func TestSetString(t *testing.T) {
	x := struct {
		Field string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	assert.Error(t, b.SetString("oops"))
	require.NoError(t, b.DigField("Field"))
	require.NoError(t, b.SetString("hello!"))
	assert.Equal(t, "hello!", x.Field)
}

func TestSliceNewElem(t *testing.T) {
	x := struct {
		Field []string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	require.NoError(t, b.DigField("Field"))
	b.Save()

	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SetString("Val1"))
	b.Load()
	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SetString("Val2"))

	require.Error(t, b.SliceNewElem())

	assert.Equal(t, []string{"Val1", "Val2"}, x.Field)
}

func TestSliceNewElemNested(t *testing.T) {
	x := struct {
		Field [][]string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	require.NoError(t, b.DigField("Field"))

	b.Save()

	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SetString("Val1.1"))
	b.Load()
	b.Save()

	require.NoError(t, b.SliceNewElem())
	b.Save()
	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SetString("Val2.1"))
	b.Load()
	require.NoError(t, b.SliceNewElem())
	require.NoError(t, b.SetString("Val2.2"))
	b.Load()
	require.NoError(t, b.SliceNewElem())

	assert.Equal(t, [][]string{{"Val1.1"}, {"Val2.1", "Val2.2"}, nil}, x.Field)
}

func TestIncorrectKindError(t *testing.T) {
	err := reflectbuild.IncorrectKindError{
		Actual:   reflect.String,
		Expected: reflect.Struct,
	}
	assert.NotEmpty(t, err.Error())
}

func TestFieldNotFoundError(t *testing.T) {
	err := reflectbuild.FieldNotFoundError{
		Struct: reflect.ValueOf(struct {
			Blah string
		}{}),
		FieldName: "Foo",
	}
	assert.NotEmpty(t, err.Error())
}

func TestCursor(t *testing.T) {
	x := struct {
		Field string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	assert.Equal(t, b.Cursor().Kind(), reflect.Struct)
	require.NoError(t, b.DigField("Field"))
	assert.Equal(t, b.Cursor().Kind(), reflect.String)
}

func TestStringPtr(t *testing.T) {
	x := struct {
		Field *string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	assert.Equal(t, b.Cursor().Kind(), reflect.Struct)
	require.NoError(t, b.DigField("Field"))
	assert.NoError(t, b.SetString("A"))
	assert.Equal(t, "A", *x.Field)
}

func TestAppendSlicePtr(t *testing.T) {
	x := struct {
		Field *[]string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	assert.Equal(t, b.Cursor().Kind(), reflect.Struct)
	require.NoError(t, b.DigField("Field"))
	v := "A"
	assert.NoError(t, b.SliceAppend(reflect.ValueOf(&v)))
	assert.Equal(t, []string{"A"}, *x.Field)
}

func TestAppendPtrSlicePtr(t *testing.T) {
	x := struct {
		Field *[]*string
	}{}
	b, err := reflectbuild.NewBuilder("", &x)
	require.NoError(t, err)
	assert.Equal(t, b.Cursor().Kind(), reflect.Struct)
	require.NoError(t, b.DigField("Field"))
	v := "A"
	assert.NoError(t, b.SliceAppend(reflect.ValueOf(&v)))
	assert.Equal(t, "A", *(*x.Field)[0])
}
