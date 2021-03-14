package unmarshaler

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

func TestUnmarshal_Integers(t *testing.T) {
	examples := []struct {
		desc     string
		input    string
		expected int64
		err      bool
	}{
		{
			desc:     "integer just digits",
			input:    `1234`,
			expected: 1234,
		},
		{
			desc:     "integer zero",
			input:    `0`,
			expected: 0,
		},
		{
			desc:     "integer sign",
			input:    `+99`,
			expected: 99,
		},
		{
			desc:     "integer hex uppercase",
			input:    `0xDEADBEEF`,
			expected: 0xDEADBEEF,
		},
		{
			desc:     "integer hex lowercase",
			input:    `0xdead_beef`,
			expected: 0xDEADBEEF,
		},
		{
			desc:     "integer octal",
			input:    `0o01234567`,
			expected: 0o01234567,
		},
		{
			desc:     "integer binary",
			input:    `0b11010110`,
			expected: 0b11010110,
		},
	}

	type doc struct {
		A int64
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			doc := doc{}
			err := Unmarshal([]byte(`A = `+e.input), &doc)
			require.NoError(t, err)
			assert.Equal(t, e.expected, doc.A)
		})
	}
}

func TestUnmarshal_Floats(t *testing.T) {
	examples := []struct {
		desc     string
		input    string
		expected float64
		testFn   func(t *testing.T, v float64)
		err      bool
	}{

		{
			desc:     "float pi",
			input:    `3.1415`,
			expected: 3.1415,
		},
		{
			desc:     "float negative",
			input:    `-0.01`,
			expected: -0.01,
		},
		{
			desc:     "float signed exponent",
			input:    `5e+22`,
			expected: 5e+22,
		},
		{
			desc:     "float exponent lowercase",
			input:    `1e06`,
			expected: 1e06,
		},
		{
			desc:     "float exponent uppercase",
			input:    `-2E-2`,
			expected: -2e-2,
		},
		{
			desc:     "float fractional with exponent",
			input:    `6.626e-34`,
			expected: 6.626e-34,
		},
		{
			desc:     "float underscores",
			input:    `224_617.445_991_228`,
			expected: 224_617.445_991_228,
		},
		{
			desc:     "inf",
			input:    `inf`,
			expected: math.Inf(+1),
		},
		{
			desc:     "inf negative",
			input:    `-inf`,
			expected: math.Inf(-1),
		},
		{
			desc:     "inf positive",
			input:    `+inf`,
			expected: math.Inf(+1),
		},
		{
			desc:  "nan",
			input: `nan`,
			testFn: func(t *testing.T, v float64) {
				assert.True(t, math.IsNaN(v))
			},
		},
		{
			desc:  "nan negative",
			input: `-nan`,
			testFn: func(t *testing.T, v float64) {
				assert.True(t, math.IsNaN(v))
			},
		},
		{
			desc:  "nan positive",
			input: `+nan`,
			testFn: func(t *testing.T, v float64) {
				assert.True(t, math.IsNaN(v))
			},
		},
	}

	type doc struct {
		A float64
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			doc := doc{}
			err := Unmarshal([]byte(`A = `+e.input), &doc)
			require.NoError(t, err)
			if e.testFn != nil {
				e.testFn(t, doc.A)
			} else {
				assert.Equal(t, e.expected, doc.A)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	type test struct {
		target   interface{}
		expected interface{}
	}
	examples := []struct {
		desc  string
		input string
		gen   func() test
	}{
		{
			desc:  "kv string",
			input: `A = "foo"`,
			gen: func() test {
				type doc struct {
					A string
				}
				return test{
					&doc{},
					&doc{A: "foo"},
				}
			},
		},
		{
			desc:  "kv bool true",
			input: `A = true`,
			gen: func() test {
				type doc struct {
					A bool
				}
				return test{
					&doc{},
					&doc{A: true},
				}
			},
		},
		{
			desc:  "kv bool false",
			input: `A = false`,
			gen: func() test {
				type doc struct {
					A bool
				}
				return test{
					&doc{A: true},
					&doc{A: false},
				}
			},
		},
		{
			desc:  "string array",
			input: `A = ["foo", "bar"]`,
			gen: func() test {
				type doc struct {
					A []string
				}
				return test{
					&doc{},
					&doc{A: []string{"foo", "bar"}},
				}
			},
		},
		{
			desc: "standard table",
			input: `[A]
B = "data"`,
			gen: func() test {
				type A struct {
					B string
				}
				type doc struct {
					A A
				}
				return test{
					&doc{},
					&doc{A: A{B: "data"}},
				}
			},
		},
		{
			desc:  "inline table",
			input: `Name = {First = "hello", Last = "world"}`,
			gen: func() test {
				type name struct {
					First string
					Last  string
				}
				type doc struct {
					Name name
				}
				return test{
					&doc{},
					&doc{Name: name{
						First: "hello",
						Last:  "world",
					}},
				}
			},
		},
		{
			desc:  "inline table inside array",
			input: `Names = [{First = "hello", Last = "world"}, {First = "ab", Last = "cd"}]`,
			gen: func() test {
				type name struct {
					First string
					Last  string
				}
				type doc struct {
					Names []name
				}
				return test{
					&doc{},
					&doc{
						Names: []name{
							{
								First: "hello",
								Last:  "world",
							},
							{
								First: "ab",
								Last:  "cd",
							},
						},
					},
				}
			},
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			test := e.gen()
			err := Unmarshal([]byte(e.input), test.target)
			require.NoError(t, err)
			assert.Equal(t, test.expected, test.target)
		})
	}
}

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
	err := fromAst(root, &x)
	require.NoError(t, err)
	assert.Equal(t, Doc{Foo: "hello"}, x)
}

func TestFromAst_Table(t *testing.T) {
	t.Run("one level table on struct", func(t *testing.T) {
		root := ast.Root{
			ast.Node{
				Kind: ast.Table,
				Children: []ast.Node{
					{Kind: ast.Key, Data: []byte(`Level1`)},
				},
			},
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
			ast.Node{
				Kind: ast.KeyValue,
				Children: []ast.Node{
					{
						Kind: ast.Key,
						Data: []byte(`B`),
					},
					{
						Kind: ast.String,
						Data: []byte(`world`),
					},
				},
			},
		}

		type Level1 struct {
			A string
			B string
		}

		type Doc struct {
			Level1 Level1
		}

		x := Doc{}
		err := fromAst(root, &x)
		require.NoError(t, err)
		assert.Equal(t, Doc{
			Level1: Level1{
				A: "hello",
				B: "world",
			},
		}, x)
	})
	t.Run("one level table on struct", func(t *testing.T) {
		root := ast.Root{
			ast.Node{
				Kind: ast.Table,
				Children: []ast.Node{
					{Kind: ast.Key, Data: []byte(`A`)},
					{Kind: ast.Key, Data: []byte(`B`)},
				},
			},
			ast.Node{
				Kind: ast.KeyValue,
				Children: []ast.Node{
					{
						Kind: ast.Key,
						Data: []byte(`C`),
					},
					{
						Kind: ast.String,
						Data: []byte(`value`),
					},
				},
			},
		}

		type B struct {
			C string
		}

		type A struct {
			B B
		}

		type Doc struct {
			A A
		}

		x := Doc{}
		err := fromAst(root, &x)
		require.NoError(t, err)
		assert.Equal(t, Doc{
			A: A{B: B{C: "value"}},
		}, x)
	})
}

func TestFromAst_InlineTable(t *testing.T) {
	t.Run("one level of strings", func(t *testing.T) {
		//		name = { first = "Tom", last = "Preston-Werner" }

		root := ast.Root{
			ast.Node{
				Kind: ast.KeyValue,
				Children: []ast.Node{
					{
						Kind: ast.Key,
						Data: []byte(`Name`)},
					{
						Kind: ast.InlineTable,
						Children: []ast.Node{
							{
								Kind: ast.KeyValue,
								Children: []ast.Node{
									{Kind: ast.Key, Data: []byte(`First`)},
									{Kind: ast.String, Data: []byte(`Tom`)},
								},
							},
							{
								Kind: ast.KeyValue,
								Children: []ast.Node{
									{Kind: ast.Key, Data: []byte(`Last`)},
									{Kind: ast.String, Data: []byte(`Preston-Werner`)},
								},
							},
						},
					},
				},
			},
		}

		type Name struct {
			First string
			Last  string
		}

		type Doc struct {
			Name Name
		}

		x := Doc{}
		err := fromAst(root, &x)
		require.NoError(t, err)
		assert.Equal(t, Doc{
			Name: Name{
				First: "Tom",
				Last:  "Preston-Werner",
			},
		}, x)

	})
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
		err := fromAst(root, &x)
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
		err := fromAst(root, &x)
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
		err := fromAst(root, &x)
		require.NoError(t, err)
		assert.Equal(t, Doc{Foo: []interface{}{"hello", []interface{}{"inner1", "inner2"}}}, x)
	})
}
