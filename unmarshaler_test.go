package toml_test

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal_Integers(t *testing.T) {
	t.Parallel()

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
			desc:     "integer decimal underscore",
			input:    `123_456`,
			expected: 123456,
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
		{
			desc:  "double underscore",
			input: "12__3",
			err:   true,
		},
		{
			desc:  "starts with underscore",
			input: "_1",
			err:   true,
		},
		{
			desc:  "ends with underscore",
			input: "1_",
			err:   true,
		},
	}

	type doc struct {
		A int64
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			doc := doc{}
			err := toml.Unmarshal([]byte(`A = `+e.input), &doc)
			if e.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, e.expected, doc.A)
			}
		})
	}
}

//nolint:funlen
func TestUnmarshal_Floats(t *testing.T) {
	t.Parallel()

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
				t.Helper()
				assert.True(t, math.IsNaN(v))
			},
		},
		{
			desc:  "nan negative",
			input: `-nan`,
			testFn: func(t *testing.T, v float64) {
				t.Helper()
				assert.True(t, math.IsNaN(v))
			},
		},
		{
			desc:  "nan positive",
			input: `+nan`,
			testFn: func(t *testing.T, v float64) {
				t.Helper()
				assert.True(t, math.IsNaN(v))
			},
		},
	}

	type doc struct {
		A float64
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			doc := doc{}
			err := toml.Unmarshal([]byte(`A = `+e.input), &doc)
			require.NoError(t, err)
			if e.testFn != nil {
				e.testFn(t, doc.A)
			} else {
				assert.Equal(t, e.expected, doc.A)
			}
		})
	}
}

//nolint:funlen
func TestUnmarshal(t *testing.T) {
	t.Parallel()

	type test struct {
		target   interface{}
		expected interface{}
		err      bool
	}
	examples := []struct {
		skip  bool
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
					target:   &doc{},
					expected: &doc{A: "foo"},
				}
			},
		},
		{
			desc: "issue 475 - space between dots in key",
			input: `fruit. color = "yellow"
					fruit . flavor = "banana"`,
			gen: func() test {
				m := map[string]interface{}{}

				return test{
					target: &m,
					expected: &map[string]interface{}{
						"fruit": map[string]interface{}{
							"color":  "yellow",
							"flavor": "banana",
						},
					},
				}
			},
		},
		{
			desc: "issue 427 - quotation marks in key",
			input: `'"a"' = 1
					"\"b\"" = 2`,
			gen: func() test {
				m := map[string]interface{}{}

				return test{
					target: &m,
					expected: &map[string]interface{}{
						`"a"`: int64(1),
						`"b"`: int64(2),
					},
				}
			},
		},
		{
			desc: "multiline basic string",
			input: `A = """\
					Test"""`,
			gen: func() test {
				type doc struct {
					A string
				}

				return test{
					target:   &doc{},
					expected: &doc{A: "Test"},
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
					target:   &doc{},
					expected: &doc{A: true},
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
					target:   &doc{A: true},
					expected: &doc{A: false},
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
					target:   &doc{},
					expected: &doc{A: []string{"foo", "bar"}},
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
					target:   &doc{},
					expected: &doc{A: A{B: "data"}},
				}
			},
		},
		{
			desc:  "standard empty table",
			input: `[A]`,
			gen: func() test {
				var v map[string]interface{}

				return test{
					target:   &v,
					expected: &map[string]interface{}{`A`: map[string]interface{}{}},
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
					target: &doc{},
					expected: &doc{Name: name{
						First: "hello",
						Last:  "world",
					}},
				}
			},
		},
		{
			desc:  "inline empty table",
			input: `A = {}`,
			gen: func() test {
				var v map[string]interface{}

				return test{
					target:   &v,
					expected: &map[string]interface{}{`A`: map[string]interface{}{}},
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
					target: &doc{},
					expected: &doc{
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
		{
			desc:  "into map[string]interface{}",
			input: `A = "foo"`,
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target: &doc,
					expected: &map[string]interface{}{
						"A": "foo",
					},
				}
			},
		},
		{
			desc: "multi keys of different types into map[string]interface{}",
			input: `A = "foo"
					B = 42`,
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target: &doc,
					expected: &map[string]interface{}{
						"A": "foo",
						"B": int64(42),
					},
				}
			},
		},
		{
			desc:  "slice in a map[string]interface{}",
			input: `A = ["foo", "bar"]`,
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target: &doc,
					expected: &map[string]interface{}{
						"A": []interface{}{"foo", "bar"},
					},
				}
			},
		},
		{
			desc:  "string into map[string]string",
			input: `A = "foo"`,
			gen: func() test {
				doc := map[string]string{}

				return test{
					target: &doc,
					expected: &map[string]string{
						"A": "foo",
					},
				}
			},
		},
		{
			desc:  "float64 into map[string]string",
			input: `A = 42.0`,
			gen: func() test {
				doc := map[string]string{}

				return test{
					target: &doc,
					err:    true,
				}
			},
		},
		{
			desc: "one-level one-element array table",
			input: `[[First]]
					Second = "hello"`,
			gen: func() test {
				type First struct {
					Second string
				}
				type Doc struct {
					First []First
				}

				return test{
					target: &Doc{},
					expected: &Doc{
						First: []First{
							{
								Second: "hello",
							},
						},
					},
				}
			},
		},
		{
			desc: "one-level multi-element array table",
			input: `[[Products]]
					Name = "Hammer"
					Sku = 738594937

					[[Products]]  # empty table within the array

					[[Products]]
					Name = "Nail"
					Sku = 284758393

					Color = "gray"`,
			gen: func() test {
				type Product struct {
					Name  string
					Sku   int64
					Color string
				}
				type Doc struct {
					Products []Product
				}

				return test{
					target: &Doc{},
					expected: &Doc{
						Products: []Product{
							{Name: "Hammer", Sku: 738594937},
							{},
							{Name: "Nail", Sku: 284758393, Color: "gray"},
						},
					},
				}
			},
		},
		{
			desc: "one-level multi-element array table to map",
			input: `[[Products]]
					Name = "Hammer"
					Sku = 738594937

					[[Products]]  # empty table within the array

					[[Products]]
					Name = "Nail"
					Sku = 284758393

					Color = "gray"`,
			gen: func() test {
				return test{
					target: &map[string]interface{}{},
					expected: &map[string]interface{}{
						"Products": []interface{}{
							map[string]interface{}{
								"Name": "Hammer",
								"Sku":  int64(738594937),
							},
							nil,
							map[string]interface{}{
								"Name":  "Nail",
								"Sku":   int64(284758393),
								"Color": "gray",
							},
						},
					},
				}
			},
		},
		{
			desc: "sub-table in array table",
			input: `[[Fruits]]
					Name = "apple"

					[Fruits.Physical]  # subtable
					Color = "red"
					Shape = "round"`,
			gen: func() test {
				return test{
					target: &map[string]interface{}{},
					expected: &map[string]interface{}{
						"Fruits": []interface{}{
							map[string]interface{}{
								"Name": "apple",
								"Physical": map[string]interface{}{
									"Color": "red",
									"Shape": "round",
								},
							},
						},
					},
				}
			},
		},
		{
			desc: "multiple sub-table in array tables",
			input: `[[Fruits]]
					Name = "apple"

					[[Fruits.Varieties]]  # nested array of tables
					Name = "red delicious"

					[[Fruits.Varieties]]
					Name = "granny smith"

					[[Fruits]]
					Name = "banana"

					[[Fruits.Varieties]]
					Name = "plantain"`,
			gen: func() test {
				return test{
					target: &map[string]interface{}{},
					expected: &map[string]interface{}{
						"Fruits": []interface{}{
							map[string]interface{}{
								"Name": "apple",
								"Varieties": []interface{}{
									map[string]interface{}{
										"Name": "red delicious",
									},
									map[string]interface{}{
										"Name": "granny smith",
									},
								},
							},
							map[string]interface{}{
								"Name": "banana",
								"Varieties": []interface{}{
									map[string]interface{}{
										"Name": "plantain",
									},
								},
							},
						},
					},
				}
			},
		},
		{
			desc: "multiple sub-table in array tables into structs",
			input: `[[Fruits]]
					Name = "apple"

					[[Fruits.Varieties]]  # nested array of tables
					Name = "red delicious"

					[[Fruits.Varieties]]
					Name = "granny smith"

					[[Fruits]]
					Name = "banana"

					[[Fruits.Varieties]]
					Name = "plantain"`,
			gen: func() test {
				type Variety struct {
					Name string
				}
				type Fruit struct {
					Name      string
					Varieties []Variety
				}
				type doc struct {
					Fruits []Fruit
				}

				return test{
					target: &doc{},
					expected: &doc{
						Fruits: []Fruit{
							{
								Name: "apple",
								Varieties: []Variety{
									{Name: "red delicious"},
									{Name: "granny smith"},
								},
							},
							{
								Name: "banana",
								Varieties: []Variety{
									{Name: "plantain"},
								},
							},
						},
					},
				}
			},
		},
		{
			desc:  "slice pointer in slice pointer",
			input: `A = ["Hello"]`,
			gen: func() test {
				type doc struct {
					A *[]*string
				}
				hello := "Hello"

				return test{
					target: &doc{},
					expected: &doc{
						A: &[]*string{&hello},
					},
				}
			},
		},
		{
			desc: "interface holding a struct",
			input: `[A]
					B = "After"`,
			gen: func() test {
				type inner struct {
					B interface{}
				}
				type doc struct {
					A interface{}
				}

				return test{
					target: &doc{
						A: inner{
							B: "Before",
						},
					},
					expected: &doc{
						A: map[string]interface{}{
							"B": "After",
						},
					},
				}
			},
		},
		{
			desc: "array of structs with table arrays",
			input: `[[A]]
			B = "one"
			[[A]]
			B = "two"`,
			gen: func() test {
				type inner struct {
					B string
				}
				type doc struct {
					A [4]inner
				}

				return test{
					target: &doc{},
					expected: &doc{
						A: [4]inner{
							{B: "one"},
							{B: "two"},
						},
					},
				}
			},
		},
		{
			desc:  "windows line endings",
			input: "A = 1\r\n\r\nB = 2",
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target: &doc,
					expected: &map[string]interface{}{
						"A": int64(1),
						"B": int64(2),
					},
				}
			},
		},
		{
			desc:  "dangling CR",
			input: "A = 1\r",
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target: &doc,
					err:    true,
				}
			},
		},
		{
			desc:  "missing NL after CR",
			input: "A = 1\rB = 2",
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target: &doc,
					err:    true,
				}
			},
		},
		{
			desc:  "no newline (#526)",
			input: `a = 1z = 2`,
			gen: func() test {
				m := map[string]interface{}{}

				return test{
					target: &m,
					err:    true,
				}
			},
		},
		{
			desc:  "mismatch types int to string",
			input: `A = 42`,
			gen: func() test {
				type S struct {
					A string
				}
				return test{
					target: &S{},
					err:    true,
				}
			},
		},
		{
			desc:  "mismatch types array of int to interface with non-slice",
			input: `A = [[42]]`,
			skip:  true,
			gen: func() test {
				type S struct {
					A *string
				}
				return test{
					target:   &S{},
					expected: &S{},
				}
			},
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			if e.skip {
				t.Skip()
			}
			test := e.gen()
			if test.err && test.expected != nil {
				panic("invalid test: cannot expect both an error and a value")
			}
			err := toml.Unmarshal([]byte(e.input), test.target)
			if test.err {
				if err == nil {
					t.Log("=>", test.target)
				}
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.target)
			}
		})
	}
}

type Integer484 struct {
	Value int
}

func (i Integer484) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(i.Value)), nil
}

func (i *Integer484) UnmarshalText(data []byte) error {
	conv, err := strconv.Atoi(string(data))
	if err != nil {
		return fmt.Errorf("UnmarshalText: %w", err)
	}
	i.Value = conv

	return nil
}

type Config484 struct {
	Integers []Integer484 `toml:"integers"`
}

func TestIssue484(t *testing.T) {
	t.Parallel()

	raw := []byte(`integers = ["1","2","3","100"]`)

	var cfg Config484
	err := toml.Unmarshal(raw, &cfg)
	require.NoError(t, err)
	assert.Equal(t, Config484{
		Integers: []Integer484{{1}, {2}, {3}, {100}},
	}, cfg)
}

type (
	Map458   map[string]interface{}
	Slice458 []interface{}
)

func (m Map458) A(s string) Slice458 {
	return m[s].([]interface{})
}

func TestIssue458(t *testing.T) {
	t.Parallel()

	s := []byte(`[[package]]
dependencies = ["regex"]
name = "decode"
version = "0.1.0"`)
	m := Map458{}
	err := toml.Unmarshal(s, &m)
	require.NoError(t, err)
	a := m.A("package")
	expected := Slice458{
		map[string]interface{}{
			"dependencies": []interface{}{"regex"},
			"name":         "decode",
			"version":      "0.1.0",
		},
	}
	assert.Equal(t, expected, a)
}

func TestIssue252(t *testing.T) {
	t.Parallel()

	type config struct {
		Val1 string `toml:"val1"`
		Val2 string `toml:"val2"`
	}

	configFile := []byte(
		`
val1 = "test1"
`)

	cfg := &config{
		Val2: "test2",
	}

	err := toml.Unmarshal(configFile, cfg)
	require.NoError(t, err)
	require.Equal(t, "test2", cfg.Val2)
}

func TestIssue494(t *testing.T) {
	t.Parallel()

	data := `
foo = 2021-04-08
bar = 2021-04-08
`

	type s struct {
		Foo time.Time `toml:"foo"`
		Bar time.Time `toml:"bar"`
	}
	ss := new(s)
	err := toml.Unmarshal([]byte(data), ss)
	require.NoError(t, err)
}

func TestIssue507(t *testing.T) {
	t.Parallel()

	data := []byte{'0', '=', '\n', '0', 'a', 'm', 'e'}
	m := map[string]interface{}{}
	err := toml.Unmarshal(data, &m)
	require.Error(t, err)
}

//nolint:funlen
func TestUnmarshalDecodeErrors(t *testing.T) {
	t.Parallel()

	examples := []struct {
		desc string
		data string
		msg  string
	}{
		{
			desc: "int with wrong base",
			data: `a = 0f2`,
		},
		{
			desc: "literal string with new lines",
			data: `a = 'hello
world'`,
			msg: `literal strings cannot have new lines`,
		},
		{
			desc: "unterminated literal string",
			data: `a = 'hello`,
			msg:  `unterminated literal string`,
		},
		{
			desc: "unterminated multiline literal string",
			data: `a = '''hello`,
			msg:  `multiline literal string not terminated by '''`,
		},
		{
			desc: "basic string with new lines",
			data: `a = "hello
"`,
			msg: `basic strings cannot have new lines`,
		},
		{
			desc: "basic string with unfinished escape",
			data: `a = "hello \`,
			msg:  `need a character after \`,
		},
		{
			desc: "basic unfinished multiline string",
			data: `a = """hello`,
			msg:  `multiline basic string not terminated by """`,
		},
		{
			desc: "basic unfinished escape in multiline string",
			data: `a = """hello \`,
			msg:  `need a character after \`,
		},
		{
			desc: "malformed local date",
			data: `a = 2021-033-0`,
			msg:  `dates are expected to have the format YYYY-MM-DD`,
		},
		{
			desc: "malformed tz",
			data: `a = 2021-03-30 21:31:00+1`,
			msg:  `invalid date-time timezone`,
		},
		{
			desc: "malformed tz first char",
			data: `a = 2021-03-30 21:31:00:1`,
			msg:  `extra characters at the end of a local date time`,
		},
		{
			desc: "bad char between hours and minutes",
			data: `a = 2021-03-30 213:1:00`,
			msg:  `expecting colon between hours and minutes`,
		},
		{
			desc: "bad char between minutes and seconds",
			data: `a = 2021-03-30 21:312:0`,
			msg:  `expecting colon between minutes and seconds`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			m := map[string]interface{}{}
			err := toml.Unmarshal([]byte(e.data), &m)
			require.Error(t, err)

			var de *toml.DecodeError
			if !errors.As(err, &de) {
				t.Fatalf("err should have been a *toml.DecodeError, but got %s (%T)", err, err)
			}

			if e.msg != "" {
				t.Log("\n" + de.String())
				require.Equal(t, "toml: "+e.msg, de.Error())
			}
		})
	}
}

//nolint:funlen
func TestLocalDateTime(t *testing.T) {
	t.Parallel()

	examples := []struct {
		desc  string
		input string
	}{
		{
			desc:  "9 digits",
			input: "2006-01-02T15:04:05.123456789",
		},
		{
			desc:  "8 digits",
			input: "2006-01-02T15:04:05.12345678",
		},
		{
			desc:  "7 digits",
			input: "2006-01-02T15:04:05.1234567",
		},
		{
			desc:  "6 digits",
			input: "2006-01-02T15:04:05.123456",
		},
		{
			desc:  "5 digits",
			input: "2006-01-02T15:04:05.12345",
		},
		{
			desc:  "4 digits",
			input: "2006-01-02T15:04:05.1234",
		},
		{
			desc:  "3 digits",
			input: "2006-01-02T15:04:05.123",
		},
		{
			desc:  "2 digits",
			input: "2006-01-02T15:04:05.12",
		},
		{
			desc:  "1 digit",
			input: "2006-01-02T15:04:05.1",
		},
		{
			desc:  "0 digit",
			input: "2006-01-02T15:04:05",
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()
			t.Log("input:", e.input)
			doc := `a = ` + e.input
			m := map[string]toml.LocalDateTime{}
			err := toml.Unmarshal([]byte(doc), &m)
			require.NoError(t, err)
			actual := m["a"]
			golang, err := time.Parse("2006-01-02T15:04:05.999999999", e.input)
			require.NoError(t, err)
			expected := toml.LocalDateTimeOf(golang)
			require.Equal(t, expected, actual)
		})
	}
}

func TestIssue287(t *testing.T) {
	t.Parallel()

	b := `y=[[{}]]`
	v := map[string]interface{}{}
	err := toml.Unmarshal([]byte(b), &v)
	require.NoError(t, err)

	expected := map[string]interface{}{
		"y": []interface{}{
			[]interface{}{
				map[string]interface{}{},
			},
		},
	}
	require.Equal(t, expected, v)
}

func TestIssue508(t *testing.T) {
	t.Parallel()

	type head struct {
		Title string `toml:"title"`
	}

	type text struct {
		head
	}

	b := []byte(`title = "This is a title"`)

	t1 := text{}
	err := toml.Unmarshal(b, &t1)
	require.NoError(t, err)
	require.Equal(t, "This is a title", t1.head.Title)
}

//nolint:funlen
func TestDecoderStrict(t *testing.T) {
	t.Parallel()

	examples := []struct {
		desc     string
		input    string
		expected string
		target   interface{}
	}{
		{
			desc: "multiple missing root keys",
			input: `
key1 = "value1"
key2 = "missing2"
key3 = "missing3"
key4 = "value4"
`,
			expected: `
2| key1 = "value1"
3| key2 = "missing2"
 | ~~~~ missing field
4| key3 = "missing3"
5| key4 = "value4"
---
2| key1 = "value1"
3| key2 = "missing2"
4| key3 = "missing3"
 | ~~~~ missing field
5| key4 = "value4"
`,
			target: &struct {
				Key1 string
				Key4 string
			}{},
		},
		{
			desc:  "multi-part key",
			input: `a.short.key="foo"`,
			expected: `
1| a.short.key="foo"
 | ~~~~~~~~~~~ missing field
`,
		},
		{
			desc: "missing table",
			input: `
[foo]
bar = 42
`,
			expected: `
2| [foo]
 |  ~~~ missing table
3| bar = 42
`,
		},

		{
			desc: "missing array table",
			input: `
[[foo]]
bar = 42
`,
			expected: `
2| [[foo]]
 |   ~~~ missing table
3| bar = 42
`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Parallel()

			r := strings.NewReader(e.input)
			d := toml.NewDecoder(r)
			d.SetStrict(true)
			x := e.target
			if x == nil {
				x = &struct{}{}
			}
			err := d.Decode(x)

			var tsm *toml.StrictMissingError
			if errors.As(err, &tsm) {
				equalStringsIgnoreNewlines(t, e.expected, tsm.String())
			} else {
				t.Fatalf("err should have been a *toml.StrictMissingError, but got %s (%T)", err, err)
			}
		})
	}
}

func ExampleDecoder_SetStrict() {
	type S struct {
		Key1 string
		Key3 string
	}
	doc := `
key1 = "value1"
key2 = "value2"
key3 = "value3"
`
	r := strings.NewReader(doc)
	d := toml.NewDecoder(r)
	d.SetStrict(true)
	s := S{}
	err := d.Decode(&s)

	fmt.Println(err.Error())

	var details *toml.StrictMissingError
	if !errors.As(err, &details) {
		panic(fmt.Sprintf("err should have been a *toml.StrictMissingError, but got %s (%T)", err, err))
	}

	fmt.Println(details.String())
	// Output:
	// strict mode: fields in the document are missing in the target struct
	// 2| key1 = "value1"
	// 3| key2 = "value2"
	//  | ~~~~ missing field
	// 4| key3 = "value3"
}

func ExampleUnmarshal() {
	type MyConfig struct {
		Version int
		Name    string
		Tags    []string
	}

	doc := `
	version = 2
	name = "go-toml"
	tags = ["go", "toml"]
	`

	var cfg MyConfig
	err := toml.Unmarshal([]byte(doc), &cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("version:", cfg.Version)
	fmt.Println("name:", cfg.Name)
	fmt.Println("tags:", cfg.Tags)

	// Output:
	// version: 2
	// name: go-toml
	// tags: [go toml]
}
