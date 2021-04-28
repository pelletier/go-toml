package toml

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructTarget_Ensure(t *testing.T) {
	t.Parallel()

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

				s, ok := v.Interface().([]string)
				if !ok {
					panic(fmt.Sprintf("interface %v should be castable into []string", s))
				}

				assert.Equal(t, []string{"foo"}, s)
			},
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			d := decoder{}
			target, _, err := d.scopeTableTarget(false, valueTarget(e.input), e.name)
			require.NoError(t, err)
			err = ensureValueIndexable(target)
			v := target.get()
			e.test(v, err)
		})
	}
}

func TestStructTarget_SetString(t *testing.T) {
	t.Parallel()

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
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			d := decoder{}
			target, _, err := d.scopeTableTarget(false, valueTarget(e.input), e.name)
			require.NoError(t, err)
			err = setString(target, str)
			v := target.get()
			e.test(v, err)
		})
	}
}

func TestPushNew(t *testing.T) {
	t.Parallel()

	t.Run("slice of strings", func(t *testing.T) {
		t.Parallel()

		type Doc struct {
			A []string
		}
		d := Doc{}

		dec := decoder{}
		x, _, err := dec.scopeTableTarget(false, valueTarget(reflect.ValueOf(&d).Elem()), "A")
		require.NoError(t, err)

		n, err := elementAt(x, 0)
		require.NoError(t, err)
		require.NoError(t, n.setString("hello"))
		require.Equal(t, []string{"hello"}, d.A)

		n, err = elementAt(x, 1)
		require.NoError(t, err)
		require.NoError(t, n.setString("world"))
		require.Equal(t, []string{"hello", "world"}, d.A)
	})

	t.Run("slice of interfaces", func(t *testing.T) {
		t.Parallel()

		type Doc struct {
			A []interface{}
		}
		d := Doc{}

		dec := decoder{}
		x, _, err := dec.scopeTableTarget(false, valueTarget(reflect.ValueOf(&d).Elem()), "A")
		require.NoError(t, err)

		n, err := elementAt(x, 0)
		require.NoError(t, err)
		require.NoError(t, setString(n, "hello"))
		require.Equal(t, []interface{}{"hello"}, d.A)

		n, err = elementAt(x, 1)
		require.NoError(t, err)
		require.NoError(t, setString(n, "world"))
		require.Equal(t, []interface{}{"hello", "world"}, d.A)
	})
}

func TestScope_Struct(t *testing.T) {
	t.Parallel()

	examples := []struct {
		desc  string
		input reflect.Value
		name  string
		err   bool
		found bool
		idx   []int
	}{
		{
			desc:  "simple field",
			input: reflect.ValueOf(&struct{ A string }{}).Elem(),
			name:  "A",
			idx:   []int{0},
			found: true,
		},
		{
			desc:  "fails not-exported field",
			input: reflect.ValueOf(&struct{ a string }{}).Elem(),
			name:  "a",
			err:   false,
			found: false,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			dec := decoder{}
			x, found, err := dec.scopeTableTarget(false, valueTarget(e.input), e.name)
			assert.Equal(t, e.found, found)
			if e.err {
				assert.Error(t, err)
			}
			if found {
				x2, ok := x.(valueTarget)
				require.True(t, ok)
				x2.get()
			}
		})
	}
}
