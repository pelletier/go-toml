package toml

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructTarget_Ensure(t *testing.T) {
	examples := []struct {
		desc  string
		input reflect.Value
		name  string
		test  func(v reflect.Value, err error)
	}{
		{
			desc:  "handle a nil slice of string",
			input: reflect.ValueOf(&struct{ A []string }{}).Elem(),
			name:  "A",
			test: func(v reflect.Value, err error) {
				assert.NoError(t, err)
				assert.False(t, v.IsNil())
			},
		},
		{
			desc:  "handle an existing slice of string",
			input: reflect.ValueOf(&struct{ A []string }{A: []string{"foo"}}).Elem(),
			name:  "A",
			test: func(v reflect.Value, err error) {
				assert.NoError(t, err)
				require.False(t, v.IsNil())
				s := v.Interface().([]string)
				assert.Equal(t, []string{"foo"}, s)
			},
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			target, err := scope(e.input, e.name)
			require.NoError(t, err)
			err = target.ensureSlice()
			v := target.get()
			e.test(v, err)
		})
	}
}

func TestStructTarget_SetString(t *testing.T) {
	str := "value"

	examples := []struct {
		desc  string
		input reflect.Value
		name  string
		test  func(v reflect.Value, err error)
	}{
		{
			desc:  "sets a string",
			input: reflect.ValueOf(&struct{ A string }{}).Elem(),
			name:  "A",
			test: func(v reflect.Value, err error) {
				assert.NoError(t, err)
				assert.Equal(t, str, v.String())
			},
		},
		{
			desc:  "fails on a float",
			input: reflect.ValueOf(&struct{ A float64 }{}).Elem(),
			name:  "A",
			test: func(v reflect.Value, err error) {
				assert.Error(t, err)
			},
		},
		{
			desc:  "fails on a slice",
			input: reflect.ValueOf(&struct{ A []string }{}).Elem(),
			name:  "A",
			test: func(v reflect.Value, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			target, err := scope(e.input, e.name)
			require.NoError(t, err)
			err = target.setString(str)
			v := target.get()
			e.test(v, err)
		})
	}
}

func TestPushNew(t *testing.T) {
	t.Run("slice of strings", func(t *testing.T) {
		type Doc struct {
			A []string
		}
		d := Doc{}

		x, err := scope(reflect.ValueOf(&d).Elem(), "A")
		require.NoError(t, err)

		n, err := x.pushNew()
		require.NoError(t, err)
		require.NoError(t, n.setString("hello"))
		require.Equal(t, []string{"hello"}, d.A)

		n, err = x.pushNew()
		require.NoError(t, err)
		require.NoError(t, n.setString("world"))
		require.Equal(t, []string{"hello", "world"}, d.A)
	})

	t.Run("slice of interfaces", func(t *testing.T) {
		type Doc struct {
			A []interface{}
		}
		d := Doc{}

		x, err := scope(reflect.ValueOf(&d).Elem(), "A")
		require.NoError(t, err)

		n, err := x.pushNew()
		require.NoError(t, err)
		require.NoError(t, n.setString("hello"))
		require.Equal(t, []interface{}{"hello"}, d.A)

		n, err = x.pushNew()
		require.NoError(t, err)
		require.NoError(t, n.setString("world"))
		require.Equal(t, []interface{}{"hello", "world"}, d.A)
	})
}

func TestScope_Struct(t *testing.T) {
	examples := []struct {
		desc  string
		input reflect.Value
		name  string
		err   bool
		idx   []int
	}{
		{
			desc:  "simple field",
			input: reflect.ValueOf(&struct{ A string }{}).Elem(),
			name:  "A",
			idx:   []int{0},
		},
		{
			desc:  "fails not-exported field",
			input: reflect.ValueOf(&struct{ a string }{}).Elem(),
			name:  "a",
			err:   true,
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			x, err := scope(e.input, e.name)
			if e.err {
				require.Error(t, err)
			} else {
				x2, ok := x.(valueTarget)
				require.True(t, ok)
				x2.get()
			}
		})
	}
}
