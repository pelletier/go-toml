package toml_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

type unmarshalTextKey struct {
	A string
	B string
}

func (k *unmarshalTextKey) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid text key: %s", text)
	}
	k.A = parts[0]
	k.B = parts[1]
	return nil
}

type unmarshalBadTextKey struct{}

func (k *unmarshalBadTextKey) UnmarshalText([]byte) error {
	return errors.New("error")
}

func ExampleDecoder_DisallowUnknownFields() {
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
	d.DisallowUnknownFields()
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

// pluginConfig demonstrates how to implement dynamic unmarshaling
// based on a "type" field. This pattern is useful for plugin systems
// or polymorphic configuration.
type pluginConfig struct {
	Type   string
	Config any
}

func (p *pluginConfig) UnmarshalTOML(data []byte) error {
	// First, decode just the type field
	var typeOnly struct {
		Type string `toml:"type"`
	}
	if err := toml.Unmarshal(data, &typeOnly); err != nil {
		return err
	}
	p.Type = typeOnly.Type

	// Now decode the config based on the type
	switch typeOnly.Type {
	case "database":
		var cfg struct {
			Type string `toml:"type"`
			Host string `toml:"host"`
			Port int    `toml:"port"`
		}
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return err
		}
		p.Config = map[string]any{"host": cfg.Host, "port": cfg.Port}
	case "cache":
		var cfg struct {
			Type string `toml:"type"`
			TTL  int    `toml:"ttl"`
		}
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return err
		}
		p.Config = map[string]any{"ttl": cfg.TTL}
	}
	return nil
}

// This example demonstrates dynamic unmarshaling based on a discriminator
// field. The pluginConfig type uses UnmarshalTOML to first read the "type"
// field, then decode the rest of the configuration based on that type.
// This pattern is useful for plugin systems or configuration that varies
// by type.
func ExampleDecoder_EnableUnmarshalerInterface_dynamicConfig() {
	doc := `
[[plugins]]
type = "database"
host = "localhost"
port = 5432

[[plugins]]
type = "cache"
ttl = 300
`
	type Config struct {
		Plugins []pluginConfig `toml:"plugins"`
	}

	var cfg Config
	err := toml.NewDecoder(strings.NewReader(doc)).
		EnableUnmarshalerInterface().
		Decode(&cfg)
	if err != nil {
		panic(err)
	}

	for _, p := range cfg.Plugins {
		fmt.Printf("type=%s config=%v\n", p.Type, p.Config)
	}
	// Output:
	// type=database config=map[host:localhost port:5432]
	// type=cache config=map[ttl:300]
}

// This example demonstrates using RawMessage to capture raw TOML bytes
// for later processing. RawMessage is similar to json.RawMessage - it
// delays decoding so you can inspect the raw content or decode it
// differently based on context.
func ExampleDecoder_EnableUnmarshalerInterface_rawMessage() {
	doc := `
[plugin]
name = "example"
version = "1.0"
enabled = true
`

	type Config struct {
		Plugin unstable.RawMessage `toml:"plugin"`
	}

	var cfg Config
	err := toml.NewDecoder(strings.NewReader(doc)).
		EnableUnmarshalerInterface().
		Decode(&cfg)
	if err != nil {
		panic(err)
	}

	// cfg.Plugin contains the raw TOML bytes
	fmt.Printf("Raw TOML captured:\n%s", cfg.Plugin)

	// You can later decode it into a specific type
	var plugin struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
		Enabled bool   `toml:"enabled"`
	}
	if err := toml.Unmarshal(cfg.Plugin, &plugin); err != nil {
		panic(err)
	}
	fmt.Printf("Decoded: name=%s version=%s enabled=%v\n",
		plugin.Name, plugin.Version, plugin.Enabled)

	// Output:
	// Raw TOML captured:
	// name = "example"
	// version = "1.0"
	// enabled = true
	// Decoded: name=example version=1.0 enabled=true
}

type badReader struct{}

func (r *badReader) Read([]byte) (int, error) {
	return 0, errors.New("testing error")
}

func TestDecodeReaderError(t *testing.T) {
	r := &badReader{}

	dec := toml.NewDecoder(r)
	m := map[string]interface{}{}
	err := dec.Decode(&m)
	assert.Error(t, err)
}

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
			doc := doc{}
			err := toml.Unmarshal([]byte(`A = `+e.input), &doc)
			if e.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, e.expected, doc.A)
			}
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
			desc:     "float exponent zero",
			input:    `0e0`,
			expected: 0.0,
		},
		{
			desc:     "float upper exponent zero",
			input:    `0E0`,
			expected: 0.0,
		},
		{
			desc:     "float zero without decimals",
			input:    `0`,
			expected: 0.0,
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
		{
			desc:  "underscore after integer part",
			input: `1_e2`,
			err:   true,
		},
		{
			desc:  "underscore after integer part",
			input: `1.0_e2`,
			err:   true,
		},
		{
			desc:  "leading zero in positive float",
			input: `+0_0.0`,
			err:   true,
		},
	}

	type doc struct {
		A float64
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			doc := doc{}
			err := toml.Unmarshal([]byte(`A = `+e.input), &doc)
			if e.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if e.testFn != nil {
					e.testFn(t, doc.A)
				} else {
					assert.Equal(t, e.expected, doc.A)
				}
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	type test struct {
		target   interface{}
		expected interface{}
		err      bool
		assert   func(t *testing.T, test test)
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
			desc:  "kv literal string",
			input: `A = 'foo 🙂 '`,
			gen: func() test {
				type doc struct {
					A string
				}

				return test{
					target:   &doc{},
					expected: &doc{A: "foo 🙂 "},
				}
			},
		},
		{
			desc:  "kv text key",
			input: `a-1 = "foo"`,
			gen: func() test {
				type doc = map[unmarshalTextKey]string

				return test{
					target:   &doc{},
					expected: &doc{{A: "a", B: "1"}: "foo"},
				}
			},
		},
		{
			desc: "table text key",
			input: `["a-1"]
foo = "bar"`,
			gen: func() test {
				type doc = map[unmarshalTextKey]map[string]string

				return test{
					target:   &doc{},
					expected: &doc{{A: "a", B: "1"}: map[string]string{"foo": "bar"}},
				}
			},
		},
		{
			desc:  "kv ptr text key",
			input: `a-1 = "foo"`,
			gen: func() test {
				type doc = map[*unmarshalTextKey]string

				return test{
					target:   &doc{},
					expected: &doc{{A: "a", B: "1"}: "foo"},
					assert: func(t *testing.T, test test) {
						t.Helper()
						// Despite the documentation:
						//     Pointer variable equality is determined based on the equality of the
						//		 referenced values (as opposed to the memory addresses).
						// assert.Equal does not work properly with maps with pointer keys
						// https://github.com/stretchr/testify/issues/1143
						expected := make(map[unmarshalTextKey]string)
						for k, v := range *(test.expected.(*doc)) {
							expected[*k] = v
						}
						got := make(map[unmarshalTextKey]string)
						for k, v := range *(test.target.(*doc)) {
							got[*k] = v
						}
						assert.Equal(t, expected, got)
					},
				}
			},
		},
		{
			desc:  "kv bad text key",
			input: `a-1 = "foo"`,
			gen: func() test {
				type doc = map[unmarshalBadTextKey]string

				return test{
					target: &doc{},
					err:    true,
				}
			},
		},
		{
			desc:  "kv bad ptr text key",
			input: `a-1 = "foo"`,
			gen: func() test {
				type doc = map[*unmarshalBadTextKey]string

				return test{
					target: &doc{},
					err:    true,
				}
			},
		},
		{
			desc: "table bad text key",
			input: `["a-1"]
foo = "bar"`,
			gen: func() test {
				type doc = map[unmarshalBadTextKey]map[string]string

				return test{
					target: &doc{},
					err:    true,
				}
			},
		},
		{
			desc:  "time.time with negative zone",
			input: `a = 1979-05-27T00:32:00-07:00 `, // space intentional
			gen: func() test {
				var v map[string]time.Time

				return test{
					target: &v,
					expected: &map[string]time.Time{
						"a": time.Date(1979, 5, 27, 0, 32, 0, 0, time.FixedZone("", -7*3600)),
					},
				}
			},
		},
		{
			desc:  "time.time with positive zone",
			input: `a = 1979-05-27T00:32:00+07:00`,
			gen: func() test {
				var v map[string]time.Time

				return test{
					target: &v,
					expected: &map[string]time.Time{
						"a": time.Date(1979, 5, 27, 0, 32, 0, 0, time.FixedZone("", 7*3600)),
					},
				}
			},
		},
		{
			desc:  "time.time with zone and fractional",
			input: `a = 1979-05-27T00:32:00.999999-07:00`,
			gen: func() test {
				var v map[string]time.Time

				return test{
					target: &v,
					expected: &map[string]time.Time{
						"a": time.Date(1979, 5, 27, 0, 32, 0, 999999000, time.FixedZone("", -7*3600)),
					},
				}
			},
		},
		{
			desc:  "local datetime into time.Time",
			input: `a = 1979-05-27T00:32:00`,
			gen: func() test {
				type doc struct {
					A time.Time
				}

				return test{
					target: &doc{},
					expected: &doc{
						A: time.Date(1979, 5, 27, 0, 32, 0, 0, time.Local),
					},
				}
			},
		},
		{
			desc:  "local datetime into interface",
			input: `a = 1979-05-27T00:32:00`,
			gen: func() test {
				type doc struct {
					A interface{}
				}

				return test{
					target: &doc{},
					expected: &doc{
						A: toml.LocalDateTime{
							toml.LocalDate{1979, 5, 27},
							toml.LocalTime{0, 32, 0, 0, 0},
						},
					},
				}
			},
		},
		{
			desc:  "local date into interface",
			input: `a = 1979-05-27`,
			gen: func() test {
				type doc struct {
					A interface{}
				}

				return test{
					target: &doc{},
					expected: &doc{
						A: toml.LocalDate{1979, 5, 27},
					},
				}
			},
		},
		{
			desc:  "local leap-day date into interface",
			input: `a = 2020-02-29`,
			gen: func() test {
				type doc struct {
					A interface{}
				}

				return test{
					target: &doc{},
					expected: &doc{
						A: toml.LocalDate{2020, 2, 29},
					},
				}
			},
		},
		{
			desc:  "local-time with nano second",
			input: `a = 12:08:05.666666666`,
			gen: func() test {
				var v map[string]interface{}

				return test{
					target: &v,
					expected: &map[string]interface{}{
						"a": toml.LocalTime{Hour: 12, Minute: 8, Second: 5, Nanosecond: 666666666, Precision: 9},
					},
				}
			},
		},
		{
			desc:  "local-time",
			input: `a = 12:08:05`,
			gen: func() test {
				var v map[string]interface{}

				return test{
					target: &v,
					expected: &map[string]interface{}{
						"a": toml.LocalTime{Hour: 12, Minute: 8, Second: 5},
					},
				}
			},
		},
		{
			desc:  "local-time missing digit",
			input: `a = 12:08:0`,
			gen: func() test {
				var v map[string]interface{}

				return test{
					target: &v,
					err:    true,
				}
			},
		},
		{
			desc:  "local-time extra digit",
			input: `a = 12:08:000`,
			gen: func() test {
				var v map[string]interface{}

				return test{
					target: &v,
					err:    true,
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
			desc: "issue 739 - table redefinition",
			input: `
[foo.bar.baz]
wibble = 'wobble'

[foo]

[foo.bar]
huey = 'dewey'
			`,
			gen: func() test {
				m := map[string]interface{}{}

				return test{
					target: &m,
					expected: &map[string]interface{}{
						`foo`: map[string]interface{}{
							"bar": map[string]interface{}{
								"huey": "dewey",
								"baz": map[string]interface{}{
									"wibble": "wobble",
								},
							},
						},
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
			desc:  "multiline literal string with windows newline",
			input: "A = '''\r\nTest'''",
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
			desc:  "multiline basic string with windows newline",
			input: "A = \"\"\"\r\nTe\r\nst\"\"\"",
			gen: func() test {
				type doc struct {
					A string
				}

				return test{
					target:   &doc{},
					expected: &doc{A: "Te\r\nst"},
				}
			},
		},
		{
			desc: "multiline basic string escapes",
			input: `A = """
\\\b\f\n\r\t\uffff\U0001D11E"""`,
			gen: func() test {
				type doc struct {
					A string
				}

				return test{
					target:   &doc{},
					expected: &doc{A: "\\\b\f\n\r\t\uffff\U0001D11E"},
				}
			},
		},
		{
			desc:  "basic string escapes",
			input: `A = "\\\b\f\n\r\t\uffff\U0001D11E"`,
			gen: func() test {
				type doc struct {
					A string
				}

				return test{
					target:   &doc{},
					expected: &doc{A: "\\\b\f\n\r\t\uffff\U0001D11E"},
				}
			},
		},
		{
			desc:  "spaces around dotted keys",
			input: "a . b = 1",
			gen: func() test {
				return test{
					target:   &map[string]map[string]interface{}{},
					expected: &map[string]map[string]interface{}{"a": {"b": int64(1)}},
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
			desc:  "long string array into []string",
			input: `A = ["0","1","2","3","4","5","6","7","8","9","10","11","12","13","14","15","16","17"]`,
			gen: func() test {
				type doc struct {
					A []string
				}

				return test{
					target:   &doc{},
					expected: &doc{A: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17"}},
				}
			},
		},
		{
			desc: "long string array into []interface{}",
			input: `A = ["0","1","2","3","4","5","6","7","8","9","10","11","12","13","14",
"15","16","17"]`,
			gen: func() test {
				type doc struct {
					A []interface{}
				}

				return test{
					target: &doc{},
					expected: &doc{A: []interface{}{
						"0", "1", "2", "3", "4", "5", "6",
						"7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17",
					}},
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
							map[string]interface{}{},
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
			desc: "array table into interface in struct",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo interface{}
				}
				return test{
					target: &doc{},
					expected: &doc{
						Foo: []interface{}{
							map[string]interface{}{
								"bar": "hello",
							},
						},
					},
				}
			},
		},
		{
			desc: "array table into interface in struct already initialized with right type",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo interface{}
				}
				return test{
					target: &doc{
						Foo: []interface{}{},
					},
					expected: &doc{
						Foo: []interface{}{
							map[string]interface{}{
								"bar": "hello",
							},
						},
					},
				}
			},
		},
		{
			desc: "array table into interface in struct already initialized with wrong type",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo interface{}
				}
				return test{
					target: &doc{
						Foo: []string{},
					},
					expected: &doc{
						Foo: []interface{}{
							map[string]interface{}{
								"bar": "hello",
							},
						},
					},
				}
			},
		},
		{
			desc: "array table into maps with pointer on last key",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo **[]interface{}
				}
				x := &[]interface{}{
					map[string]interface{}{
						"bar": "hello",
					},
				}
				return test{
					target: &doc{},
					expected: &doc{
						Foo: &x,
					},
				}
			},
		},
		{
			desc: "array table into maps with pointer on intermediate key",
			input: `[[foo.foo2]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo **map[string]interface{}
				}
				x := &map[string]interface{}{
					"foo2": []interface{}{
						map[string]interface{}{
							"bar": "hello",
						},
					},
				}
				return test{
					target: &doc{},
					expected: &doc{
						Foo: &x,
					},
				}
			},
		},
		{
			desc: "array table into maps with pointer on last key with invalid leaf type",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo **[]map[string]int
				}
				return test{
					target: &doc{},
					err:    true,
				}
			},
		},
		{
			desc:  "unexported struct fields are ignored",
			input: `foo = "bar"`,
			gen: func() test {
				type doc struct {
					foo string //nolint:unused
				}
				return test{
					target:   &doc{},
					expected: &doc{},
				}
			},
		},
		{
			desc: "array table into nil ptr",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo *[]interface{}
				}
				return test{
					target: &doc{},
					expected: &doc{
						Foo: &[]interface{}{
							map[string]interface{}{
								"bar": "hello",
							},
						},
					},
				}
			},
		},
		{
			desc: "array table into nil ptr of invalid type",
			input: `[[foo]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo *string
				}
				return test{
					target: &doc{},
					err:    true,
				}
			},
		},
		{
			desc: "array table with intermediate ptr",
			input: `[[foo.bar]]
			bar = "hello"`,
			gen: func() test {
				type doc struct {
					Foo *map[string]interface{}
				}
				return test{
					target: &doc{},
					expected: &doc{
						Foo: &map[string]interface{}{
							"bar": []interface{}{
								map[string]interface{}{
									"bar": "hello",
								},
							},
						},
					},
				}
			},
		},
		{
			desc:  "unmarshal array into interface that contains a slice",
			input: `a = [1,2,3]`,
			gen: func() test {
				type doc struct {
					A interface{}
				}
				return test{
					target: &doc{
						A: []string{},
					},
					expected: &doc{
						A: []interface{}{
							int64(1),
							int64(2),
							int64(3),
						},
					},
				}
			},
		},
		{
			desc:  "unmarshal array into interface that contains a []interface{}",
			input: `a = [1,2,3]`,
			gen: func() test {
				type doc struct {
					A interface{}
				}
				return test{
					target: &doc{
						A: []interface{}{},
					},
					expected: &doc{
						A: []interface{}{
							int64(1),
							int64(2),
							int64(3),
						},
					},
				}
			},
		},
		{
			desc:  "unmarshal key into map with existing value",
			input: `a = "new"`,
			gen: func() test {
				return test{
					target:   &map[string]interface{}{"a": "old"},
					expected: &map[string]interface{}{"a": "new"},
				}
			},
		},
		{
			desc:  "unmarshal key into map with existing value",
			input: `a.b = "new"`,
			gen: func() test {
				type doc struct {
					A interface{}
				}
				return test{
					target: &doc{},
					expected: &doc{
						A: map[string]interface{}{
							"b": "new",
						},
					},
				}
			},
		},
		{
			desc:  "unmarshal array into struct field with existing array",
			input: `a = [1,2]`,
			gen: func() test {
				type doc struct {
					A []int
				}
				return test{
					target: &doc{},
					expected: &doc{
						A: []int{1, 2},
					},
				}
			},
		},
		{
			desc:  "unmarshal inline table into map",
			input: `a = {b="hello"}`,
			gen: func() test {
				type doc struct {
					A map[string]interface{}
				}
				return test{
					target: &doc{},
					expected: &doc{
						A: map[string]interface{}{
							"b": "hello",
						},
					},
				}
			},
		},
		{
			desc:  "unmarshal inline table into map of incorrect type",
			input: `a = {b="hello"}`,
			gen: func() test {
				type doc struct {
					A map[string]int
				}
				return test{
					target: &doc{},
					err:    true,
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
			desc:  "interface holding a string",
			input: `A = "Hello"`,
			gen: func() test {
				type doc struct {
					A interface{}
				}
				return test{
					target: &doc{},
					expected: &doc{
						A: "Hello",
					},
				}
			},
		},
		{
			desc:  "map of bools",
			input: `A = true`,
			gen: func() test {
				return test{
					target:   &map[string]bool{},
					expected: &map[string]bool{"A": true},
				}
			},
		},
		{
			desc:  "map of int64",
			input: `A = 42`,
			gen: func() test {
				return test{
					target:   &map[string]int64{},
					expected: &map[string]int64{"A": 42},
				}
			},
		},
		{
			desc:  "map of float64",
			input: `A = 4.2`,
			gen: func() test {
				return test{
					target:   &map[string]float64{},
					expected: &map[string]float64{"A": 4.2},
				}
			},
		},
		{
			desc:  "array of int in map",
			input: `A = [1,2,3]`,
			gen: func() test {
				return test{
					target:   &map[string][3]int{},
					expected: &map[string][3]int{"A": {1, 2, 3}},
				}
			},
		},
		{
			desc:  "array of int in map with too many elements",
			input: `A = [1,2,3,4,5]`,
			gen: func() test {
				return test{
					target:   &map[string][3]int{},
					expected: &map[string][3]int{"A": {1, 2, 3}},
				}
			},
		},
		{
			desc:  "array of int in map with invalid element",
			input: `A = [1,2,false]`,
			gen: func() test {
				return test{
					target: &map[string][3]int{},
					err:    true,
				}
			},
		},
		{
			desc: "nested arrays",
			input: `
			[[A]]
			[[A.B]]
			C = 1
			[[A]]
			[[A.B]]
			C = 2`,
			gen: func() test {
				type leaf struct {
					C int
				}
				type inner struct {
					B [2]leaf
				}
				type s struct {
					A [2]inner
				}
				return test{
					target: &s{},
					expected: &s{A: [2]inner{
						{B: [2]leaf{
							{C: 1},
						}},
						{B: [2]leaf{
							{C: 2},
						}},
					}},
				}
			},
		},
		{
			desc: "nested arrays too many",
			input: `
			[[A]]
			[[A.B]]
			C = 1
			[[A.B]]
			C = 2`,
			gen: func() test {
				type leaf struct {
					C int
				}
				type inner struct {
					B [1]leaf
				}
				type s struct {
					A [1]inner
				}
				return test{
					target: &s{},
					err:    true,
				}
			},
		},
		{
			desc:  "empty array table in interface{}",
			input: `[[products]]`,
			gen: func() test {
				return test{
					target: &map[string]interface{}{},
					expected: &map[string]interface{}{
						"products": []interface{}{
							map[string]interface{}{},
						},
					},
				}
			},
		},
		{
			desc:  "into map with invalid key type",
			input: `A = "hello"`,
			gen: func() test {
				return test{
					target: &map[int]string{},
					err:    true,
				}
			},
		},
		{
			desc:  "into map with convertible key type",
			input: `A = "hello"`,
			gen: func() test {
				type foo string
				return test{
					target: &map[foo]string{},
					expected: &map[foo]string{
						"A": "hello",
					},
				}
			},
		},
		{
			desc:  "array of int in struct",
			input: `A = [1,2,3]`,
			gen: func() test {
				type s struct {
					A [3]int
				}
				return test{
					target:   &s{},
					expected: &s{A: [3]int{1, 2, 3}},
				}
			},
		},
		{
			desc: "array of int in struct",
			input: `[A]
			b = 42`,
			gen: func() test {
				type s struct {
					A *map[string]interface{}
				}
				return test{
					target:   &s{},
					expected: &s{A: &map[string]interface{}{"b": int64(42)}},
				}
			},
		},
		{
			desc:  "assign bool to float",
			input: `A = true`,
			gen: func() test {
				return test{
					target: &map[string]float64{},
					err:    true,
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
			input: `A = [42]`,
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
			desc:  "comment with CRLF",
			input: "# foo\r\na=2",
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target:   &doc,
					expected: &map[string]interface{}{"a": int64(2)},
				}
			},
		},
		{
			desc:  "comment that looks like a date",
			input: "a=19#9-",
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target:   &doc,
					expected: &map[string]interface{}{"a": int64(19)},
				}
			},
		},
		{
			desc:  "comment that looks like a date",
			input: "a=199#-",
			gen: func() test {
				doc := map[string]interface{}{}

				return test{
					target:   &doc,
					expected: &map[string]interface{}{"a": int64(199)},
				}
			},
		},
		{
			desc:  "kv that points to a slice",
			input: "a.b.c = 'foo'",
			gen: func() test {
				doc := map[string][]string{}
				return test{
					target: &doc,
					err:    true,
				}
			},
		},
		{
			desc:  "kv that points to a pointer to a slice",
			input: "a.b.c = 'foo'",
			gen: func() test {
				doc := map[string]*[]string{}
				return test{
					target: &doc,
					err:    true,
				}
			},
		},
		{
			desc:  "into map of int to string",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target:   &map[int]string{},
					expected: &map[int]string{1: "a"},
				}
			},
		},
		{
			desc:  "into map of int8 to string",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target:   &map[int8]string{},
					expected: &map[int8]string{1: "a"},
				}
			},
		},
		{
			desc:  "into map of int64 to string",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target:   &map[int64]string{},
					expected: &map[int64]string{1: "a"},
				}
			},
		},
		{
			desc:  "into map of uint to string",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target:   &map[uint]string{},
					expected: &map[uint]string{1: "a"},
				}
			},
		},
		{
			desc:  "into map of uint8 to string",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target:   &map[uint8]string{},
					expected: &map[uint8]string{1: "a"},
				}
			},
		},
		{
			desc:  "into map of uint64 to string",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target:   &map[uint64]string{},
					expected: &map[uint64]string{1: "a"},
				}
			},
		},
		{
			desc:  "into map of uint with invalid key",
			input: `-1 = "a"`,
			gen: func() test {
				return test{
					target: &map[uint]string{},
					err:    true,
				}
			},
		},
		{
			desc:  "into map of float64 to string",
			input: `'1.01' = "a"`,
			gen: func() test {
				return test{
					target:   &map[float64]string{},
					expected: &map[float64]string{1.01: "a"},
				}
			},
		},
		{
			desc:  "into map of float64 with invalid key",
			input: `key = "a"`,
			gen: func() test {
				return test{
					target: &map[float64]string{},
					err:    true,
				}
			},
		},
		{
			desc:  "into map of float32 to string",
			input: `'1.01' = "a"`,
			gen: func() test {
				return test{
					target:   &map[float32]string{},
					expected: &map[float32]string{1.01: "a"},
				}
			},
		},
		{
			desc:  "into map of float32 with invalid key",
			input: `key = "a"`,
			gen: func() test {
				return test{
					target: &map[float32]string{},
					err:    true,
				}
			},
		},
		{
			desc:  "invalid map key type",
			input: `1 = "a"`,
			gen: func() test {
				return test{
					target: &map[struct{ int }]string{},
					err:    true,
				}
			},
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if test.assert != nil {
					test.assert(t, test)
				} else {
					assert.Equal(t, test.expected, test.target)
				}
			}
		})
	}
}

func TestUnmarshalOverflows(t *testing.T) {
	examples := []struct {
		t      interface{}
		errors []string
	}{
		{
			t:      &map[string]int32{},
			errors: []string{`-2147483649`, `2147483649`},
		},
		{
			t:      &map[string]int16{},
			errors: []string{`-2147483649`, `2147483649`},
		},
		{
			t:      &map[string]int8{},
			errors: []string{`-2147483649`, `2147483649`},
		},
		{
			t:      &map[string]int{},
			errors: []string{`-19223372036854775808`, `9223372036854775808`},
		},
		{
			t:      &map[string]uint64{},
			errors: []string{`-1`, `18446744073709551616`},
		},
		{
			t:      &map[string]uint32{},
			errors: []string{`-1`, `18446744073709551616`},
		},
		{
			t:      &map[string]uint16{},
			errors: []string{`-1`, `18446744073709551616`},
		},
		{
			t:      &map[string]uint8{},
			errors: []string{`-1`, `18446744073709551616`},
		},
		{
			t:      &map[string]uint{},
			errors: []string{`-1`, `18446744073709551616`},
		},
	}

	for _, e := range examples {
		e := e
		for _, v := range e.errors {
			v := v
			t.Run(fmt.Sprintf("%T %s", e.t, v), func(t *testing.T) {
				doc := "A = " + v
				err := toml.Unmarshal([]byte(doc), e.t)
				t.Log("input:", doc)
				assert.Error(t, err)
			})
		}
		t.Run(fmt.Sprintf("%T ok", e.t), func(t *testing.T) {
			doc := "A = 1"
			err := toml.Unmarshal([]byte(doc), e.t)
			t.Log("input:", doc)
			assert.NoError(t, err)
		})
	}
}

func TestUnmarshalErrors(t *testing.T) {
	type mystruct struct {
		Bar string
	}

	data := `bar = 42`

	s := mystruct{}
	err := toml.Unmarshal([]byte(data), &s)
	assert.Error(t, err)

	assert.Equal(t, "toml: cannot decode TOML integer into struct field toml_test.mystruct.Bar of type string", err.Error())
}

func TestUnmarshalStringInvalidStructField(t *testing.T) {
	type Server struct {
		Path string
		Port int
	}

	type Cfg struct {
		Server Server
	}

	var cfg Cfg

	data := `[server]
path = "/my/path"
port = "bad"
`

	file := strings.NewReader(data)
	err := toml.NewDecoder(file).Decode(&cfg)
	assert.Error(t, err)

	var x *toml.DecodeError
	assert.True(t, errors.As(err, &x))
	assert.Equal(t, "toml: cannot decode TOML string into struct field toml_test.Server.Port of type int", x.Error())
	expected := `1| [server]
2| path = "/my/path"
3| port = "bad"
 |        ~~~~~ cannot decode TOML string into struct field toml_test.Server.Port of type int`

	assert.Equal(t, expected, x.String())
}

func TestUnmarshalIntegerInvalidStructField(t *testing.T) {
	type Server struct {
		Path string
		Port int
	}

	type Cfg struct {
		Server Server
	}

	var cfg Cfg

	data := `[server]
path = 100
port = 50
`

	file := strings.NewReader(data)
	err := toml.NewDecoder(file).Decode(&cfg)
	assert.Error(t, err)

	var x *toml.DecodeError
	assert.True(t, errors.As(err, &x))
	assert.Equal(t, "toml: cannot decode TOML integer into struct field toml_test.Server.Path of type string", x.Error())
	expected := `1| [server]
2| path = 100
 |        ~~~ cannot decode TOML integer into struct field toml_test.Server.Path of type string
3| port = 50`

	assert.Equal(t, expected, x.String())
}

func TestUnmarshalInvalidTarget(t *testing.T) {
	x := "foo"
	err := toml.Unmarshal([]byte{}, x)
	assert.Error(t, err)

	var m *map[string]interface{}
	err = toml.Unmarshal([]byte{}, m)
	assert.Error(t, err)
}

func TestUnmarshalFloat32(t *testing.T) {
	t.Run("fits", func(t *testing.T) {
		doc := "A = 1.2"
		err := toml.Unmarshal([]byte(doc), &map[string]float32{})
		assert.NoError(t, err)
	})
	t.Run("overflows", func(t *testing.T) {
		doc := "A = 4.40282346638528859811704183484516925440e+38"
		err := toml.Unmarshal([]byte(doc), &map[string]float32{})
		assert.Error(t, err)
	})
}

func TestDecoderStrict(t *testing.T) {
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
			expected: `2| key1 = "value1"
3| key2 = "missing2"
 | ~~~~ missing field
4| key3 = "missing3"
5| key4 = "value4"
---
2| key1 = "value1"
3| key2 = "missing2"
4| key3 = "missing3"
 | ~~~~ missing field
5| key4 = "value4"`,
			target: &struct {
				Key1 string
				Key4 string
			}{},
		},
		{
			desc:  "multi-part key",
			input: `a.short.key="foo"`,
			expected: `1| a.short.key="foo"
 | ~~~~~~~~~~~ missing field`,
		},
		{
			desc: "missing table",
			input: `
[foo]
bar = 42
`,
			expected: `2| [foo]
 |  ~~~ missing table
3| bar = 42`,
		},

		{
			desc: "missing array table",
			input: `
[[foo]]
bar = 42`,
			expected: `2| [[foo]]
 |   ~~~ missing table
3| bar = 42`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Run("strict", func(t *testing.T) {
				r := strings.NewReader(e.input)
				d := toml.NewDecoder(r)
				d.DisallowUnknownFields()
				x := e.target
				if x == nil {
					x = &struct{}{}
				}
				err := d.Decode(x)

				var tsm *toml.StrictMissingError
				if errors.As(err, &tsm) {
					assert.Equal(t, e.expected, tsm.String())
				} else {
					t.Fatalf("err should have been a *toml.StrictMissingError, but got %s (%T)", err, err)
				}
			})

			t.Run("default", func(t *testing.T) {
				r := strings.NewReader(e.input)
				d := toml.NewDecoder(r)
				x := e.target
				if x == nil {
					x = &struct{}{}
				}
				err := d.Decode(x)
				assert.NoError(t, err)
			})
		})
	}
}

func TestIssue252(t *testing.T) {
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
	assert.NoError(t, err)
	assert.Equal(t, "test2", cfg.Val2)
}

func TestIssue287(t *testing.T) {
	b := `y=[[{}]]`
	v := map[string]interface{}{}
	err := toml.Unmarshal([]byte(b), &v)
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"y": []interface{}{
			[]interface{}{
				map[string]interface{}{},
			},
		},
	}
	assert.Equal(t, expected, v)
}

type (
	Map458   map[string]interface{}
	Slice458 []interface{}
)

func (m Map458) A(s string) Slice458 {
	return m[s].([]interface{})
}

func TestIssue458(t *testing.T) {
	s := []byte(`[[package]]
dependencies = ["regex"]
name = "decode"
version = "0.1.0"`)
	m := Map458{}
	err := toml.Unmarshal(s, &m)
	assert.NoError(t, err)
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
	raw := []byte(`integers = ["1","2","3","100"]`)

	var cfg Config484
	err := toml.Unmarshal(raw, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, Config484{
		Integers: []Integer484{{1}, {2}, {3}, {100}},
	}, cfg)
}

func TestIssue494(t *testing.T) {
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
	assert.NoError(t, err)
}

func TestIssue508(t *testing.T) {
	type head struct {
		Title string `toml:"title"`
	}

	type text struct {
		head
	}

	b := []byte(`title = "This is a title"`)

	t1 := text{}
	err := toml.Unmarshal(b, &t1)
	assert.NoError(t, err)
	assert.Equal(t, "This is a title", t1.Title)
}

func TestIssue507(t *testing.T) {
	data := []byte{'0', '=', '\n', '0', 'a', 'm', 'e'}
	m := map[string]interface{}{}
	err := toml.Unmarshal(data, &m)
	assert.Error(t, err)
}

type uuid [16]byte

func (u *uuid) UnmarshalText([]byte) (err error) {
	// Note: the original reported issue had a more complex implementation
	// of this function. But the important part is to verify that a
	// non-struct type implementing UnmarshalText works with the unmarshal
	// process.
	placeholder := bytes.Repeat([]byte{0xAA}, 16)
	copy(u[:], placeholder)
	return nil
}

func TestIssue564(t *testing.T) {
	type Config struct {
		ID uuid
	}

	var config Config

	err := toml.Unmarshal([]byte(`id = "0818a52b97b94768941ba1172c76cf6c"`), &config)
	assert.NoError(t, err)
	assert.Equal(t, uuid{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}, config.ID)
}

func TestIssue575(t *testing.T) {
	b := []byte(`
[pkg.cargo]
version = "0.55.0 (5ae8d74b3 2021-06-22)"
git_commit_hash = "a178d0322ce20e33eac124758e837cbd80a6f633"
[pkg.cargo.target.aarch64-apple-darwin]
available = true
url = "https://static.rust-lang.org/dist/2021-07-29/cargo-1.54.0-aarch64-apple-darwin.tar.gz"
hash = "7bac3901d8eb6a4191ffeebe75b29c78bcb270158ec901addb31f588d965d35d"
xz_url = "https://static.rust-lang.org/dist/2021-07-29/cargo-1.54.0-aarch64-apple-darwin.tar.xz"
xz_hash = "5207644fd6379f3e5b8ae60016b854efa55a381b0c363bff7f9b2f25bfccc430"

[pkg.cargo.target.aarch64-pc-windows-msvc]
available = true
url = "https://static.rust-lang.org/dist/2021-07-29/cargo-1.54.0-aarch64-pc-windows-msvc.tar.gz"
hash = "eb8ccd9b1f6312b06dc749c17896fa4e9c163661c273dcb61cd7a48376227f6d"
xz_url = "https://static.rust-lang.org/dist/2021-07-29/cargo-1.54.0-aarch64-pc-windows-msvc.tar.xz"
xz_hash = "1a48f723fea1f17d786ce6eadd9d00914d38062d28fd9c455ed3c3801905b388"
`)

	type target struct {
		XZ_URL string //revive:disable:var-naming
	}

	type pkg struct {
		Target map[string]target
	}

	type doc struct {
		Pkg map[string]pkg
	}

	var dist doc
	err := toml.Unmarshal(b, &dist)
	assert.NoError(t, err)

	expected := doc{
		Pkg: map[string]pkg{
			"cargo": {
				Target: map[string]target{
					"aarch64-apple-darwin": {
						XZ_URL: "https://static.rust-lang.org/dist/2021-07-29/cargo-1.54.0-aarch64-apple-darwin.tar.xz",
					},
					"aarch64-pc-windows-msvc": {
						XZ_URL: "https://static.rust-lang.org/dist/2021-07-29/cargo-1.54.0-aarch64-pc-windows-msvc.tar.xz",
					},
				},
			},
		},
	}

	assert.Equal(t, expected, dist)
}

func TestIssue579(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`[foo`), &v)
	assert.Error(t, err)
}

func TestIssue581(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`P=[#`), &v)
	assert.Error(t, err)
}

func TestIssue585(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`a=1979-05127T 0`), &v)
	assert.Error(t, err)
}

func TestIssue586(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`a={ `), &v)
	assert.Error(t, err)
}

func TestIssue588(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`a=[1#`), &v)
	assert.Error(t, err)
}

// Support lowercase 'T' and 'Z'
func TestIssue600(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`a=1979-05-27t00:32:00z`), &v)
	assert.NoError(t, err)
}

func TestIssue596(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(`a=1979-05-27T90:+2:99`), &v)
	assert.Error(t, err)
}

func TestIssue602(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte(""), &v)
	assert.NoError(t, err)

	var expected interface{} = map[string]interface{}{}

	assert.Equal(t, expected, v)
}

func TestIssue623(t *testing.T) {
	definition := struct {
		Things []string
	}{}

	values := `[things]
foo = "bar"`

	err := toml.Unmarshal([]byte(values), &definition)
	assert.Error(t, err)
}

func TestIssue631(t *testing.T) {
	v := map[string]interface{}{}
	err := toml.Unmarshal([]byte("\"\\b\u007f\"= 2"), &v)
	assert.Error(t, err)
}

func TestIssue658(t *testing.T) {
	var v map[string]interface{}
	err := toml.Unmarshal([]byte("e={b=1,b=4}"), &v)
	assert.Error(t, err)
}

func TestIssue662(t *testing.T) {
	var v map[string]interface{}
	err := toml.Unmarshal([]byte("a=[{b=1,b=2}]"), &v)
	assert.Error(t, err)
}

func TestIssue666(t *testing.T) {
	var v map[string]interface{}
	err := toml.Unmarshal([]byte("a={}\na={}"), &v)
	assert.Error(t, err)
}

func TestIssue677(t *testing.T) {
	doc := `
[Build]
Name = "publication build"

[[Build.Dependencies]]
Name = "command"
Program = "hugo"
`

	type _tomlJob struct {
		Dependencies []map[string]interface{}
	}

	type tomlParser struct {
		Build *_tomlJob
	}

	p := tomlParser{}

	err := toml.Unmarshal([]byte(doc), &p)
	assert.NoError(t, err)

	expected := tomlParser{
		Build: &_tomlJob{
			Dependencies: []map[string]interface{}{
				{
					"Name":    "command",
					"Program": "hugo",
				},
			},
		},
	}
	assert.Equal(t, expected, p)
}

func TestIssue701(t *testing.T) {
	// Expected behavior:
	// Return an error since a cannot be modified. From the TOML spec:
	//
	// > Inline tables are fully self-contained and define all
	// keys and sub-tables within them. Keys and sub-tables cannot
	// be added outside the braces.

	docs := []string{
		`
a={}
[a.b]
z=0
`,
		`
a={}
[[a.b]]
z=0
`,
	}

	for _, doc := range docs {
		var v interface{}
		err := toml.Unmarshal([]byte(doc), &v)
		assert.Error(t, err)
	}
}

func TestIssue703(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte("[a]\nx.y=0\n[a.x]"), &v)
	assert.Error(t, err)
}

func TestIssue708(t *testing.T) {
	v := map[string]string{}
	err := toml.Unmarshal([]byte("0=\"\"\"\\\r\n\"\"\""), &v)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"0": ""}, v)
}

func TestIssue710(t *testing.T) {
	v := map[string]toml.LocalTime{}
	err := toml.Unmarshal([]byte(`0=00:00:00.0000000000`), &v)
	assert.NoError(t, err)
	assert.Equal(t, map[string]toml.LocalTime{"0": {Precision: 9}}, v)
	v1 := map[string]toml.LocalTime{}
	err = toml.Unmarshal([]byte(`0=00:00:00.0000000001`), &v1)
	assert.NoError(t, err)
	assert.Equal(t, map[string]toml.LocalTime{"0": {Precision: 9}}, v1)
	v2 := map[string]toml.LocalTime{}
	err = toml.Unmarshal([]byte(`0=00:00:00.1111111119`), &v2)
	assert.NoError(t, err)
	assert.Equal(t, map[string]toml.LocalTime{"0": {Nanosecond: 111111111, Precision: 9}}, v2)
}

func TestIssue715(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte("0=+"), &v)
	assert.Error(t, err)

	err = toml.Unmarshal([]byte("0=-"), &v)
	assert.Error(t, err)

	err = toml.Unmarshal([]byte("0=+A"), &v)
	assert.Error(t, err)
}

func TestIssue714(t *testing.T) {
	var v interface{}
	err := toml.Unmarshal([]byte("0."), &v)
	assert.Error(t, err)

	err = toml.Unmarshal([]byte("0={0=0,"), &v)
	assert.Error(t, err)
}

func TestIssue772(t *testing.T) {
	type FileHandling struct {
		FilePattern string `toml:"pattern"`
	}

	type Config struct {
		FileHandling `toml:"filehandling"`
	}

	defaultConfigFile := []byte(`
		[filehandling]
		pattern = "reach-masterdev-"`)

	config := Config{}
	err := toml.Unmarshal(defaultConfigFile, &config)
	assert.NoError(t, err)
	assert.Equal(t, "reach-masterdev-", config.FilePattern)
}

func TestIssue774(t *testing.T) {
	type ScpData struct {
		Host string `json:"host"`
	}

	type GenConfig struct {
		SCP []ScpData `toml:"scp" comment:"Array of Secure Copy Configurations"`
	}

	c := &GenConfig{}
	c.SCP = []ScpData{{Host: "main.domain.com"}}

	b, err := toml.Marshal(c)
	assert.NoError(t, err)

	expected := `# Array of Secure Copy Configurations
[[scp]]
Host = 'main.domain.com'
`

	assert.Equal(t, expected, string(b))
}

func TestIssue799(t *testing.T) {
	const testTOML = `
# notice the double brackets
[[test]]
answer = 42
`

	var s struct {
		// should be []map[string]int
		Test map[string]int `toml:"test"`
	}

	err := toml.Unmarshal([]byte(testTOML), &s)
	assert.Error(t, err)
}

func TestIssue807(t *testing.T) {
	type A struct {
		Name string `toml:"name"`
	}

	type M struct {
		*A
	}

	var m M
	err := toml.Unmarshal([]byte(`name = 'foo'`), &m)
	assert.NoError(t, err)
	assert.Equal(t, "foo", m.Name)
}

func TestIssue850(t *testing.T) {
	data := make(map[string]string)
	err := toml.Unmarshal([]byte("foo = {}"), &data)
	assert.Error(t, err)
}

func TestIssue851(t *testing.T) {
	type Target struct {
		Params map[string]string `toml:"params"`
	}

	content := "params = {a=\"1\",b=\"2\"}"
	var target Target
	err := toml.Unmarshal([]byte(content), &target)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, target.Params)
	err = toml.Unmarshal([]byte(content), &target)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, target.Params)
}

func TestIssue866(t *testing.T) {
	type Pipeline struct {
		Mapping map[string]struct {
			Req [][]string `toml:"req"`
			Res [][]string `toml:"res"`
		} `toml:"mapping"`
	}

	type Pipelines struct {
		PipelineMapping map[string]*Pipeline `toml:"pipelines"`
	}

	badToml := `
[pipelines.register]
mapping.inst.req = [
    ["param1", "value1"],
]
mapping.inst.res = [
    ["param2", "value2"],
]
`

	pipelines := new(Pipelines)
	if err := toml.NewDecoder(bytes.NewBufferString(badToml)).DisallowUnknownFields().Decode(pipelines); err != nil {
		t.Fatal(err)
	}
	if pipelines.PipelineMapping["register"].Mapping["inst"].Req[0][0] != "param1" {
		t.Fatal("unmarshal failed with mismatch value")
	}

	goodTooToml := `
[pipelines.register]
mapping.inst.req = [
    ["param1", "value1"],
]
`

	pipelines = new(Pipelines)
	if err := toml.NewDecoder(bytes.NewBufferString(goodTooToml)).DisallowUnknownFields().Decode(pipelines); err != nil {
		t.Fatal(err)
	}
	if pipelines.PipelineMapping["register"].Mapping["inst"].Req[0][0] != "param1" {
		t.Fatal("unmarshal failed with mismatch value")
	}

	goodToml := `
[pipelines.register.mapping.inst]
req = [
    ["param1", "value1"],
]
res = [
    ["param2", "value2"],
]
`

	pipelines = new(Pipelines)
	if err := toml.NewDecoder(bytes.NewBufferString(goodToml)).DisallowUnknownFields().Decode(pipelines); err != nil {
		t.Fatal(err)
	}
	if pipelines.PipelineMapping["register"].Mapping["inst"].Req[0][0] != "param1" {
		t.Fatal("unmarshal failed with mismatch value")
	}
}

func TestIssue915(t *testing.T) {
	type blah struct {
		A string `toml:"a"`
	}

	type config struct {
		Fizz string `toml:"fizz"`
		blah `toml:"blah"`
	}

	b := []byte(`
fizz = "abc"
blah.a = "def"`)
	var cfg config
	err := toml.Unmarshal(b, &cfg)
	assert.NoError(t, err)

	assert.Equal(t, "abc", cfg.Fizz)
	assert.Equal(t, "def", cfg.A)
	assert.Equal(t, "def", cfg.A)
}

func TestIssue931(t *testing.T) {
	type item struct {
		Name string
	}

	type items struct {
		Slice []item
	}

	its := items{[]item{{"a"}, {"b"}}}

	b := []byte(`
	[[Slice]]
  Name = 'c'

[[Slice]]
  Name = 'd'
	`)

	err := toml.Unmarshal(b, &its)
	assert.NoError(t, err)
	assert.Equal(t, items{[]item{{"c"}, {"d"}}}, its)
}

func TestIssue931Interface(t *testing.T) {
	type items struct {
		Slice interface{}
	}

	type item = map[string]interface{}

	its := items{[]interface{}{item{"Name": "a"}, item{"Name": "b"}}}

	b := []byte(`
	[[Slice]]
  Name = 'c'

[[Slice]]
  Name = 'd'
	`)

	err := toml.Unmarshal(b, &its)
	assert.NoError(t, err)
	assert.Equal(t, items{[]interface{}{item{"Name": "c"}, item{"Name": "d"}}}, its)
}

func TestIssue931SliceInterface(t *testing.T) {
	type items struct {
		Slice []interface{}
	}

	type item = map[string]interface{}

	its := items{
		[]interface{}{
			item{"Name": "a"},
			item{"Name": "b"},
		},
	}

	b := []byte(`
	[[Slice]]
  Name = 'c'

[[Slice]]
  Name = 'd'
	`)

	err := toml.Unmarshal(b, &its)
	assert.NoError(t, err)
	assert.Equal(t, items{[]interface{}{item{"Name": "c"}, item{"Name": "d"}}}, its)
}

func TestUnmarshalDecodeErrors(t *testing.T) {
	examples := []struct {
		desc string
		data string
		msg  string
	}{
		{
			desc: "local date with invalid digit",
			data: `a = 20x1-05-21`,
		},
		{
			desc: "local time with fractional",
			data: `a = 11:22:33.x`,
		},
		{
			desc: "wrong time offset separator",
			data: `a = 1979-05-27T00:32:00.-07:00`,
		},
		{
			desc: "wrong time offset separator",
			data: `a = 1979-05-27T00:32:00Z07:00`,
		},
		{
			desc: "float with double _",
			data: `flt8 = 224_617.445_991__228`,
		},
		{
			desc: "float with double .",
			data: `flt8 = 1..2`,
		},
		{
			desc: "number with plus sign and leading underscore",
			data: `a = +_0`,
		},
		{
			desc: "number with negative sign and leading underscore",
			data: `a = -_0`,
		},
		{
			desc: "exponent with plus sign and leading underscore",
			data: `a = 0e+_0`,
		},
		{
			desc: "exponent with negative sign and leading underscore",
			data: `a = 0e-_0`,
		},
		{
			desc: "int with wrong base",
			data: `a = 0f2`,
		},
		{
			desc: "int hex with double underscore",
			data: `a = 0xFFF__FFF`,
		},
		{
			desc: "int hex very large",
			data: `a = 0xFFFFFFFFFFFFFFFFF`,
		},
		{
			desc: "int oct with double underscore",
			data: `a = 0o777__77`,
		},
		{
			desc: "int oct very large",
			data: `a = 0o77777777777777777777777`,
		},
		{
			desc: "int bin with double underscore",
			data: `a = 0b111__111`,
		},
		{
			desc: "int bin very large",
			data: `a = 0b11111111111111111111111111111111111111111111111111111111111111111111111111111`,
		},
		{
			desc: "int dec very large",
			data: `a = 999999999999999999999999`,
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
		{
			desc: "invalid hour value",
			data: `a=1979-05-27T90:+2:99`,
			msg:  `hour cannot be greater 23`,
		},
		{
			desc: "invalid minutes value",
			data: `a=1979-05-27T23:+2:99`,
			msg:  `expected digit (0-9)`,
		},
		{
			desc: "invalid seconds value",
			data: `a=1979-05-27T12:45:99`,
			msg:  `seconds cannot be greater than 59`,
		},
		{
			desc: "leap second not supported",
			data: `a=1979-05-27T12:45:60`,
			msg:  `seconds cannot be greater than 59`,
		},
		{
			desc: "leap second with max date causes overflow",
			data: `s=9999-12-31 23:59:60z`,
			msg:  `seconds cannot be greater than 59`,
		},
		{
			desc: `binary with invalid digit`,
			data: `a = 0bf`,
		},
		{
			desc: `invalid i in dec`,
			data: `a = 0i`,
		},
		{
			desc: `invalid n in dec`,
			data: `a = 0n`,
		},
		{
			desc: `invalid unquoted key`,
			data: `a`,
		},
		{
			desc: "dt with tz has no time",
			data: `a = 2021-03-30TZ`,
		},
		{
			desc: "invalid end of array table",
			data: `[[a}`,
		},
		{
			desc: "invalid end of array table two",
			data: `[[a]}`,
		},
		{
			desc: "eof after equal",
			data: `a =`,
		},
		{
			desc: "invalid true boolean",
			data: `a = trois`,
		},
		{
			desc: "invalid false boolean",
			data: `a = faux`,
		},
		{
			desc: "inline table with incorrect separator",
			data: `a = {b=1;}`,
		},
		{
			desc: "inline table with invalid value",
			data: `a = {b=faux}`,
		},
		{
			desc: `incomplete array after whitespace`,
			data: `a = [ `,
		},
		{
			desc: `array with comma first`,
			data: `a = [ ,]`,
		},
		{
			desc: `array staring with incomplete newline`,
			data: "a = [\r]",
		},
		{
			desc: `array with incomplete newline after comma`,
			data: "a = [1,\r]",
		},
		{
			desc: `array with incomplete newline after value`,
			data: "a = [1\r]",
		},
		{
			desc: `invalid unicode in basic multiline string`,
			data: `A = """\u123"""`,
		},
		{
			desc: `invalid long unicode in basic multiline string`,
			data: `A = """\U0001D11"""`,
		},
		{
			desc: `invalid unicode in basic string`,
			data: `A = "\u123"`,
		},
		{
			desc: `invalid long unicode in basic string`,
			data: `A = "\U0001D11"`,
		},
		{
			desc: `invalid escape char basic multiline string`,
			data: `A = """\z"""`,
		},
		{
			desc: `invalid inf`,
			data: `A = ick`,
		},
		{
			desc: `invalid nan`,
			data: `A = non`,
		},
		{
			desc: `invalid character in comment in array`,
			data: "A = [#\x00\n]",
		},
		{
			desc: "invalid utf8 character in long string with no escape sequence",
			data: "a = \"aaaa\x80aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"",
		},
		{
			desc: "invalid ascii character in long string with no escape sequence",
			data: "a = \"aaaa\x00aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"",
		},
		{
			desc: "unfinished 2-byte utf8 character in string with no escape sequence",
			data: "a = \"aaaa\xC2\"",
		},
		{
			desc: "unfinished 3-byte utf8 character in string with no escape sequence",
			data: "a = \"aaaa\xE2\x00\x00\"",
		},
		{
			desc: "invalid 3rd byte of 3-byte utf8 character in string with no escape sequence",
			data: "a = \"aaaa\xE2\x80\x00\"",
		},
		{
			desc: "invalid 4th byte of 4-byte utf8 character in string with no escape sequence",
			data: "a = \"aaaa\xF2\x81\x81\x00\"",
		},
		{
			desc: "unfinished 2-byte utf8 character in literal string",
			data: "a = 'aaa\xC2'",
		},
		{
			desc: "unfinished 3-byte utf8 character in literal string",
			data: "a = 'aaaa\xE2\x00\x00'",
		},
		{
			desc: "invalid 3rd byte of 3-byte utf8 character in literal string",
			data: "a = 'aaaa\xE2\x80\x00'",
		},
		{
			desc: "invalid 4th byte of 4-byte utf8 character in literal string",
			data: "a = 'aaaa\xF2\x81\x81\x00'",
		},
		{
			desc: "invalid start utf8 character in literal string",
			data: "a = '\x80'",
		},
		{
			desc: "utf8 character with not enough bytes before end in literal string",
			data: "a = '\xEF'",
		},
		{
			desc: "basic string with newline after the first escape code",
			data: "a = \"\\t\n\"",
		},
		{
			desc: "basic string with unfinished escape sequence after the first escape code",
			data: "a = \"\\t\\",
		},
		{
			desc: "basic string with unfinished after the first escape code",
			data: "a = \"\\t",
		},
		{
			desc: "multiline basic string with unfinished escape sequence after the first escape code",
			data: "a = \"\"\"\\t\\",
		},
		{
			desc: `impossible date-day`,
			data: `A = 2021-03-40T23:59:00`,
			msg:  `impossible date`,
		},
		{
			desc: `leap day in non-leap year`,
			data: `A = 2021-02-29T23:59:00`,
			msg:  `impossible date`,
		},
		{
			desc: `missing minute digit`,
			data: `a=17:4::01`,
		},
		{
			desc: `invalid space in year`,
			data: `i=19 7-12-21T10:32:00`,
		},
		{
			desc: `missing nanoseconds digits`,
			data: `a=17:45:56.`,
		},
		{
			desc: `minutes over 60`,
			data: `a=17:99:00`,
		},
		{
			desc: `invalid second`,
			data: `a=17:00::0`,
		},
		{
			desc: `invalid hour`,
			data: `a=1::00:00`,
		},
		{
			desc: `invalid month`,
			data: `a=2021-0--29`,
		},
		{
			desc: `zero is an invalid day`,
			data: `a=2021-11-00`,
		},
		{
			desc: `zero is an invalid month`,
			data: `a=2021-00-11`,
		},
		{
			desc: `invalid number of seconds digits with trailing digit`,
			data: `a=0000-01-01 00:00:000000Z3`,
		},
		{
			desc: `invalid zone offset hours`,
			data: `a=0000-01-01 00:00:00+24:00`,
		},
		{
			desc: `invalid zone offset minutes`,
			data: `a=0000-01-01 00:00:00+00:60`,
		},
		{
			desc: `invalid character in zone offset hours`,
			data: `a=0000-01-01 00:00:00+0Z:00`,
		},
		{
			desc: `invalid character in zone offset minutes`,
			data: `a=0000-01-01 00:00:00+00:0Z`,
		},
		{
			desc: `invalid number of seconds`,
			data: `a=0000-01-01 00:00:00+27000`,
		},
		{
			desc: `carriage return inside basic key`,
			data: "\"\r\"=42",
		},
		{
			desc: `carriage return inside literal key`,
			data: "'\r'=42",
		},
		{
			desc: `carriage return inside basic string`,
			data: "A = \"\r\"",
		},
		{
			desc: `carriage return inside basic multiline string`,
			data: "a=\"\"\"\r\"\"\"",
		},
		{
			desc: `carriage return at the trail of basic multiline string`,
			data: "a=\"\"\"\r",
		},
		{
			desc: `carriage return inside literal string`,
			data: "A = '\r'",
		},
		{
			desc: `carriage return inside multiline literal string`,
			data: "a='''\r'''",
		},
		{
			desc: `carriage return at trail of multiline literal string`,
			data: "a='''\r",
		},
		{
			desc: `carriage return in comment`,
			data: "# this is a test\ra=1",
		},
		{
			desc: `backspace in comment`,
			data: "# this is a test\ba=1",
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			m := map[string]interface{}{}
			err := toml.Unmarshal([]byte(e.data), &m)
			assert.Error(t, err)

			var de *toml.DecodeError
			if !errors.As(err, &de) {
				t.Fatalf("err should have been a *toml.DecodeError, but got %s (%T)", err, err)
			}

			if e.msg != "" {
				t.Log("\n" + de.String())
				assert.Equal(t, "toml: "+e.msg, de.Error())
			}
		})
	}
}

func TestOmitEmpty(t *testing.T) {
	type inner struct {
		private string //nolint:unused
		Skip    string `toml:"-"`
		V       string
	}

	type elem struct {
		Foo   string `toml:",omitempty"`
		Bar   string `toml:",omitempty"`
		Inner inner  `toml:",omitempty"`
	}

	type doc struct {
		X []elem `toml:",inline"`
	}

	d := doc{X: []elem{{
		Foo: "test",
		Inner: inner{
			V: "alue",
		},
	}}}

	b, err := toml.Marshal(d)
	assert.NoError(t, err)

	assert.Equal(t, "X = [{Foo = 'test', Inner = {V = 'alue'}}]\n", string(b))
}

func TestUnmarshalTags(t *testing.T) {
	type doc struct {
		Dash   string `toml:"-,"`
		Ignore string `toml:"-"`
		A      string `toml:"hello"`
		B      string `toml:"comma,omitempty"`
	}

	data := `
'-' = "dash"
Ignore = 'me'
hello = 'content'
comma = 'ok'
`

	d := doc{}
	expected := doc{
		Dash:   "dash",
		Ignore: "",
		A:      "content",
		B:      "ok",
	}

	err := toml.Unmarshal([]byte(data), &d)
	assert.NoError(t, err)
	assert.Equal(t, expected, d)
}

func TestASCIIControlCharacters(t *testing.T) {
	invalidCharacters := []byte{0x7F}
	for c := byte(0x0); c <= 0x08; c++ {
		invalidCharacters = append(invalidCharacters, c)
	}
	for c := byte(0x0B); c <= 0x0C; c++ {
		invalidCharacters = append(invalidCharacters, c)
	}
	for c := byte(0x0E); c <= 0x1F; c++ {
		invalidCharacters = append(invalidCharacters, c)
	}

	type stringType struct {
		Delimiter string
		CanEscape bool
	}

	stringTypes := map[string]stringType{
		"basic":            {Delimiter: "\"", CanEscape: true},
		"basicMultiline":   {Delimiter: "\"\"\"", CanEscape: true},
		"literal":          {Delimiter: "'", CanEscape: false},
		"literalMultiline": {Delimiter: "'''", CanEscape: false},
	}

	checkError := func(t *testing.T, input []byte) {
		t.Helper()
		m := map[string]interface{}{}
		err := toml.Unmarshal(input, &m)
		assert.Error(t, err)

		var de *toml.DecodeError
		if !errors.As(err, &de) {
			t.Fatalf("err should have been a *toml.DecodeError, but got %s (%T)", err, err)
		}
	}

	for name, st := range stringTypes {
		t.Run(name, func(t *testing.T) {
			for _, c := range invalidCharacters {
				name := fmt.Sprintf("%2X", c)
				t.Run(name, func(t *testing.T) {
					data := []byte("A = " + st.Delimiter + string(c) + st.Delimiter)
					checkError(t, data)

					if st.CanEscape {
						t.Run("withEscapeBefore", func(t *testing.T) {
							data := []byte("A = " + st.Delimiter + "\\t" + string(c) + st.Delimiter)
							checkError(t, data)
						})
						t.Run("withEscapeAfter", func(t *testing.T) {
							data := []byte("A = " + st.Delimiter + string(c) + "\\t" + st.Delimiter)
							checkError(t, data)
						})
					}
				})
			}
		})
	}
}

func TestLocalDateTime(t *testing.T) {
	examples := []struct {
		desc  string
		input string
		prec  int
	}{
		{
			desc:  "9 digits zero nanoseconds",
			input: "2006-01-02T15:04:05.000000000",
			prec:  9,
		},
		{
			desc:  "9 digits",
			input: "2006-01-02T15:04:05.123456789",
			prec:  9,
		},
		{
			desc:  "8 digits",
			input: "2006-01-02T15:04:05.12345678",
			prec:  8,
		},
		{
			desc:  "7 digits",
			input: "2006-01-02T15:04:05.1234567",
			prec:  7,
		},
		{
			desc:  "6 digits",
			input: "2006-01-02T15:04:05.123456",
			prec:  6,
		},
		{
			desc:  "5 digits",
			input: "2006-01-02T15:04:05.12345",
			prec:  5,
		},
		{
			desc:  "4 digits",
			input: "2006-01-02T15:04:05.1234",
			prec:  4,
		},
		{
			desc:  "3 digits",
			input: "2006-01-02T15:04:05.123",
			prec:  3,
		},
		{
			desc:  "2 digits",
			input: "2006-01-02T15:04:05.12",
			prec:  2,
		},
		{
			desc:  "1 digit",
			input: "2006-01-02T15:04:05.1",
			prec:  1,
		},
		{
			desc:  "0 digit",
			input: "2006-01-02T15:04:05",
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			t.Log("input:", e.input)
			doc := `a = ` + e.input
			m := map[string]toml.LocalDateTime{}
			err := toml.Unmarshal([]byte(doc), &m)
			assert.NoError(t, err)
			actual := m["a"]
			golang, err := time.Parse("2006-01-02T15:04:05.999999999", e.input)
			assert.NoError(t, err)
			expected := toml.LocalDateTime{
				toml.LocalDate{golang.Year(), int(golang.Month()), golang.Day()},
				toml.LocalTime{golang.Hour(), golang.Minute(), golang.Second(), golang.Nanosecond(), e.prec},
			}
			assert.Equal(t, expected, actual)
		})
	}
}

func TestUnmarshal_RecursiveTable(t *testing.T) {
	type Foo struct {
		I int
		F *Foo
	}

	examples := []struct {
		desc     string
		input    string
		expected string
		err      bool
	}{
		{
			desc: "simplest",
			input: `
				I=1
			`,
			expected: `{"I":1,"F":null}`,
		},
		{
			desc: "depth 1",
			input: `
				I=1
				[F]
				I=2
			`,
			expected: `{"I":1,"F":{"I":2,"F":null}}`,
		},
		{
			desc: "depth 3",
			input: `
				I=1
				[F]
				I=2
				[F.F]
				I=3
			`,
			expected: `{"I":1,"F":{"I":2,"F":{"I":3,"F":null}}}`,
		},
		{
			desc: "depth 4",
			input: `
				I=1
				[F]
				I=2
				[F.F]
				I=3
				[F.F.F]
				I=4
			`,
			expected: `{"I":1,"F":{"I":2,"F":{"I":3,"F":{"I":4,"F":null}}}}`,
		},
		{
			desc: "skip mid step",
			input: `
				I=1
				[F.F]
				I=7
			`,
			expected: `{"I":1,"F":{"I":0,"F":{"I":7,"F":null}}}`,
		},
	}

	for _, ex := range examples {
		e := ex
		t.Run(e.desc, func(t *testing.T) {
			foo := Foo{}
			err := toml.Unmarshal([]byte(e.input), &foo)
			if e.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				j, err := json.Marshal(foo)
				assert.NoError(t, err)
				assert.Equal(t, e.expected, string(j))
			}
		})
	}
}

func TestUnmarshal_RecursiveTableArray(t *testing.T) {
	type Foo struct {
		I int
		F []*Foo
	}

	examples := []struct {
		desc     string
		input    string
		expected string
		err      bool
	}{
		{
			desc: "simplest",
			input: `
				I=1
				F=[]
			`,
			expected: `{"I":1,"F":[]}`,
		},
		{
			desc: "depth 1",
			input: `
				I=1
				[[F]]
				I=2
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[]}]}`,
		},
		{
			desc: "depth 2",
			input: `
				I=1
				[[F]]
				I=2
				[[F.F]]
				I=3
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[{"I":3,"F":[]}]}]}`,
		},
		{
			desc: "depth 3",
			input: `
				I=1
				[[F]]
				I=2
				[[F.F]]
				I=3
				[[F.F.F]]
				I=4
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[{"I":3,"F":[{"I":4,"F":[]}]}]}]}`,
		},
		{
			desc: "depth 4",
			input: `
				I=1
				[[F]]
				I=2
				[[F.F]]
				I=3
				[[F.F.F]]
				I=4
				[[F.F.F.F]]
				I=5
				F=[]
			`,
			expected: `{"I":1,"F":[{"I":2,"F":[{"I":3,"F":[{"I":4,"F":[{"I":5,"F":[]}]}]}]}]}`,
		},
	}

	for _, ex := range examples {
		e := ex
		t.Run(e.desc, func(t *testing.T) {
			foo := Foo{}
			err := toml.Unmarshal([]byte(e.input), &foo)
			if e.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				j, err := json.Marshal(foo)
				assert.NoError(t, err)
				assert.Equal(t, e.expected, string(j))
			}
		})
	}
}

func TestUnmarshalEmbedNonString(t *testing.T) {
	type Foo []byte
	type doc struct {
		Foo
	}

	d := doc{}

	err := toml.Unmarshal([]byte(`foo = 'bar'`), &d)
	assert.NoError(t, err)
	assert.Equal(t, d.Foo, nil)
}

func TestUnmarshal_Nil(t *testing.T) {
	type Foo struct {
		Foo *Foo `toml:"foo,omitempty"`
		Bar *Foo `toml:"bar,omitempty"`
	}

	examples := []struct {
		desc     string
		input    string
		expected string
		err      bool
	}{
		{
			desc:     "empty",
			input:    ``,
			expected: ``,
		},
		{
			desc: "simplest",
			input: `
	    [foo]
	    [foo.foo]
	    `,
			expected: "[foo]\n[foo.foo]\n",
		},
	}

	for _, ex := range examples {
		e := ex
		t.Run(e.desc, func(t *testing.T) {
			foo := Foo{}
			err := toml.Unmarshal([]byte(e.input), &foo)
			if e.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				j, err := toml.Marshal(foo)
				assert.NoError(t, err)
				assert.Equal(t, e.expected, string(j))
			}
		})
	}
}

type CustomUnmarshalerKey struct {
	A int64
}

func (k *CustomUnmarshalerKey) UnmarshalTOML(data []byte) error {
	item, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return fmt.Errorf("error converting to int64, %w", err)
	}
	k.A = item
	return nil
}

func TestUnmarshal_CustomUnmarshaler(t *testing.T) {
	type MyConfig struct {
		Unmarshalers []CustomUnmarshalerKey `toml:"unmarshalers"`
		Foo          *string                `toml:"foo,omitempty"`
	}

	examples := []struct {
		desc                        string
		disableUnmarshalerInterface bool
		input                       string
		expected                    MyConfig
		err                         bool
	}{
		{
			desc:     "empty",
			input:    ``,
			expected: MyConfig{Unmarshalers: []CustomUnmarshalerKey{}, Foo: nil},
		},
		{
			desc:  "simple",
			input: `unmarshalers = [1,2,3]`,
			expected: MyConfig{
				Unmarshalers: []CustomUnmarshalerKey{
					{A: 1},
					{A: 2},
					{A: 3},
				},
				Foo: nil,
			},
		},
		{
			desc: "unmarshal string and custom unmarshaler",
			input: `unmarshalers = [1,2,3]
foo = "bar"`,
			expected: MyConfig{
				Unmarshalers: []CustomUnmarshalerKey{
					{A: 1},
					{A: 2},
					{A: 3},
				},
				Foo: func(v string) *string {
					return &v
				}("bar"),
			},
		},
		{
			desc:                        "simple example, but unmarshaler interface disabled",
			disableUnmarshalerInterface: true,
			input:                       `unmarshalers = [1,2,3]`,
			err:                         true,
		},
	}

	for _, ex := range examples {
		e := ex
		t.Run(e.desc, func(t *testing.T) {
			foo := MyConfig{}

			decoder := toml.NewDecoder(bytes.NewReader([]byte(e.input)))
			if !ex.disableUnmarshalerInterface {
				decoder.EnableUnmarshalerInterface()
			}
			err := decoder.Decode(&foo)

			if e.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(foo.Unmarshalers), len(e.expected.Unmarshalers))
				for i := 0; i < len(foo.Unmarshalers); i++ {
					assert.Equal(t, foo.Unmarshalers[i], e.expected.Unmarshalers[i])
				}
				assert.Equal(t, foo.Foo, e.expected.Foo)
			}
		})
	}
}

type doc994 struct{}

func (d *doc994) UnmarshalTOML([]byte) error {
	return errors.New("expected-error")
}

func TestIssue994(t *testing.T) {
	var _ unstable.Unmarshaler = (*doc994)(nil)
	tomlBytes := []byte(`foo = "bar"`)
	var d doc994
	err := toml.NewDecoder(bytes.NewReader(tomlBytes)).
		EnableUnmarshalerInterface().
		Decode(&d)

	assert.Error(t, err)

	if err.Error() != "expected-error" {
		t.Fatalf("expected error 'expected-error', got '%s'", err.Error())
	}
}

type doc994ok struct {
	S string
}

func (d *doc994ok) UnmarshalTOML(data []byte) error {
	d.S = string(data) + " from unmarshaler"
	return nil
}

func TestIssue994_OK(t *testing.T) {
	var _ unstable.Unmarshaler = (*doc994ok)(nil)
	tomlBytes := []byte(`foo = "bar"`)
	var d doc994ok
	err := toml.NewDecoder(bytes.NewReader(tomlBytes)).
		EnableUnmarshalerInterface().
		Decode(&d)

	assert.NoError(t, err)
	// With bytes-based interface, raw TOML bytes are passed including quotes
	assert.Equal(t, "\"bar\" from unmarshaler", d.S)
}

func TestIssue995(t *testing.T) {
	type AllowList struct {
		Description string
		Condition   string
		Commits     []string
		Paths       []string
		RegexTarget string
		Regexes     []string
		StopWords   []string
	}

	type Rule struct {
		ID          string
		Description string
		Regex       string
		SecretGroup int
		Entropy     interface{}
		Keywords    []string
		Path        string
		Tags        []string
		AllowList   *AllowList
		Allowlists  []AllowList
	}

	type GitleaksConfig struct {
		Description string
		Rules       []Rule
		Allowlist   struct {
			Commits     []string
			Paths       []string
			RegexTarget string
			Regexes     []string
			StopWords   []string
		}
	}

	doc := `
[[allowlists]]
  description = "Exception for File "
  files = [ '''app/src''']

[[rules.allowlists]]
  description = "policies"
  regexes = [
    '''abc'''
  ]
`

	var cfg GitleaksConfig
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)

	// Ensure no panic and that nested array table was created.
	if len(cfg.Rules) == 0 {
		t.Fatalf("expected Rules to contain at least one element after unmarshaling nested array table")
	}
	if len(cfg.Rules[0].Allowlists) != 1 {
		t.Fatalf("expected first Rule to have exactly one allowlists entry, got %d", len(cfg.Rules[0].Allowlists))
	}
	assert.Equal(t, "policies", cfg.Rules[0].Allowlists[0].Description)
	assert.Equal(t, []string{"abc"}, cfg.Rules[0].Allowlists[0].Regexes)
}

func TestIssue995_InterfaceSlice_MultiNested(t *testing.T) {
	type Root struct {
		Rules []interface{}
	}

	doc := `
[[rules.allowlists]]
  description = "a"

[[rules.allowlists]]
  description = "b"
`

	var r Root
	err := toml.Unmarshal([]byte(doc), &r)
	assert.NoError(t, err)

	if len(r.Rules) != 1 {
		t.Fatalf("expected one element in Rules, got %d", len(r.Rules))
	}

	m, ok := r.Rules[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Rules[0] to be a map[string]any, got %T", r.Rules[0])
	}

	als, ok := m["allowlists"].([]interface{})
	if !ok {
		t.Fatalf("expected allowlists to be []any, got %T", m["allowlists"])
	}
	if len(als) != 2 {
		t.Fatalf("expected 2 allowlists entries, got %d", len(als))
	}

	a0, ok := als[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected allowlists[0] to be map[string]any, got %T", als[0])
	}
	a1, ok := als[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected allowlists[1] to be map[string]any, got %T", als[1])
	}
	assert.Equal(t, "a", a0["description"])
	assert.Equal(t, "b", a1["description"])
}

func TestIssue995_MultiNestedConcrete(t *testing.T) {
	type AllowList struct {
		Description string
	}
	type Rule struct {
		Allowlists []AllowList
	}
	type Root struct {
		Rules []Rule
	}

	doc := `
[[rules.allowlists]]
  description = "a"

[[rules.allowlists]]
  description = "b"
`

	var r Root
	err := toml.Unmarshal([]byte(doc), &r)
	assert.NoError(t, err)

	if len(r.Rules) != 1 {
		t.Fatalf("expected one element in Rules, got %d", len(r.Rules))
	}
	assert.Equal(t, 2, len(r.Rules[0].Allowlists))
	assert.Equal(t, "a", r.Rules[0].Allowlists[0].Description)
	assert.Equal(t, "b", r.Rules[0].Allowlists[1].Description)
}

func TestIssue995_PointerToSlice_Rules(t *testing.T) {
	type AllowList struct{ Description string }
	type Rule struct{ Allowlists []AllowList }
	type Root struct{ Rules *[]Rule }

	doc := `
[[rules.allowlists]]
  description = "a"

[[rules.allowlists]]
  description = "b"
`

	var r Root
	err := toml.Unmarshal([]byte(doc), &r)
	assert.NoError(t, err)
	if r.Rules == nil {
		t.Fatalf("expected Rules pointer to be initialized")
	}
	if len(*r.Rules) != 1 {
		t.Fatalf("expected one element in Rules, got %d", len(*r.Rules))
	}
	rule := (*r.Rules)[0]
	assert.Equal(t, 2, len(rule.Allowlists))
	assert.Equal(t, "a", rule.Allowlists[0].Description)
	assert.Equal(t, "b", rule.Allowlists[1].Description)
}

func TestIssue995_SliceNonEmpty_UsesLastElement(t *testing.T) {
	type AllowList struct{ Description string }
	type Rule struct{ Allowlists []AllowList }
	type Root struct{ Rules []Rule }

	// Pre-initialize with one Rule; nested array table should populate
	// the last element, not create a new one at this level.
	var r Root
	r.Rules = []Rule{{}}

	doc := `
[[rules.allowlists]]
  description = "a"

[[rules.allowlists]]
  description = "b"
`

	err := toml.Unmarshal([]byte(doc), &r)
	assert.NoError(t, err)
	if len(r.Rules) != 1 {
		t.Fatalf("expected one element in Rules, got %d", len(r.Rules))
	}
	assert.Equal(t, 2, len(r.Rules[0].Allowlists))
	// Values presence check
	got := [...]string{r.Rules[0].Allowlists[0].Description, r.Rules[0].Allowlists[1].Description}
	if got != [2]string{"a", "b"} && got != [2]string{"b", "a"} {
		t.Fatalf("unexpected values in allowlists: %v", got)
	}
}

// fooConfig974 is a struct that implements UnmarshalText for simple string
// parsing, but can also be populated field-by-field from a TOML table.
type fooConfig974 struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

func (f *fooConfig974) UnmarshalText(text []byte) error {
	s := string(text)
	f.Name = s
	f.Value = s
	return nil
}

type config974 struct {
	Foo []fooConfig974 `toml:"foo"`
}

func TestIssue974_UnmarshalTextFallbackToStructForInlineTable(t *testing.T) {
	// When the TOML value is an inline table, the unmarshaler should skip
	// UnmarshalText and populate the struct fields directly.
	doc := `foo = [{name = "a", value = "a"}, {name = "b", value = "b"}]`

	var cfg config974
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, config974{
		Foo: []fooConfig974{
			{Name: "a", Value: "a"},
			{Name: "b", Value: "b"},
		},
	}, cfg)
}

func TestIssue974_UnmarshalTextStillWorksForStrings(t *testing.T) {
	// When the TOML value is a string, UnmarshalText should still be used.
	doc := `foo = ["a", "b"]`

	var cfg config974
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, config974{
		Foo: []fooConfig974{
			{Name: "a", Value: "a"},
			{Name: "b", Value: "b"},
		},
	}, cfg)
}

// singleFooConfig974 tests the inline table case for a single value (not array)
type singleConfig974 struct {
	Foo fooConfig974 `toml:"foo"`
}

func TestIssue974_SingleInlineTable(t *testing.T) {
	// A single inline table should also skip UnmarshalText
	doc := `foo = {name = "hello", value = "world"}`

	var cfg singleConfig974
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, singleConfig974{
		Foo: fooConfig974{Name: "hello", Value: "world"},
	}, cfg)
}

func TestIssue974_SingleString(t *testing.T) {
	// A single string should use UnmarshalText
	doc := `foo = "hello"`

	var cfg singleConfig974
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, singleConfig974{
		Foo: fooConfig974{Name: "hello", Value: "hello"},
	}, cfg)
}

func TestIssue974_TableSyntax(t *testing.T) {
	// Regular table syntax should also work (uses struct unmarshaling)
	doc := `
[foo]
name = "hello"
value = "world"
`

	var cfg singleConfig974
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, singleConfig974{
		Foo: fooConfig974{Name: "hello", Value: "world"},
	}, cfg)
}

func TestIssue974_ArrayTableSyntax(t *testing.T) {
	// Array of tables syntax should also work
	doc := `
[[foo]]
name = "a"
value = "a"

[[foo]]
name = "b"
value = "b"
`

	var cfg config974
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, config974{
		Foo: []fooConfig974{
			{Name: "a", Value: "a"},
			{Name: "b", Value: "b"},
		},
	}, cfg)
}

func TestIssue1028(t *testing.T) {
	// Datetime values assigned to incompatible types should return an error,
	// not panic.

	type Item struct {
		Name string `toml:"name"`
	}

	type Config struct {
		Items map[string]Item `toml:"items"`
	}

	// Error: "cannot decode TOML datetime into struct field Config.Items of type map[string]Item"
	t.Run("OffsetDateTime", func(t *testing.T) {
		var c Config
		err := toml.Unmarshal([]byte(`items = 2023-01-01T10:20:30Z`), &c)
		assert.Error(t, err)
	})

	// Error: "cannot decode TOML local datetime into struct field Config.Items of type map[string]Item"
	t.Run("LocalDateTime", func(t *testing.T) {
		var c Config
		err := toml.Unmarshal([]byte(`items = 2023-01-01T10:20:30`), &c)
		assert.Error(t, err)
	})

	// Error: "cannot decode TOML local date into struct field Config.Items of type map[string]Item"
	t.Run("LocalDate", func(t *testing.T) {
		var c Config
		err := toml.Unmarshal([]byte(`items = 2023-01-01`), &c)
		assert.Error(t, err)
	})

	// Error: "cannot decode TOML local time into struct field Config.Items of type map[string]Item"
	t.Run("LocalTime", func(t *testing.T) {
		var c Config
		err := toml.Unmarshal([]byte(`items = 10:20:30`), &c)
		assert.Error(t, err)
	})
}

// Tests for issue #873 - Bring back toml.Unmarshaler for tables and arrays

type customTable873 struct {
	Keys   []string
	Values map[string]string
}

func (c *customTable873) UnmarshalTOML(data []byte) error {
	c.Keys = []string{}
	c.Values = make(map[string]string)

	// Parse the raw TOML bytes into a map to extract keys in order
	// For this test, we use a simple line-by-line parser to preserve order
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		// Skip table headers
		if line[0] == '[' {
			continue
		}
		// Parse key = value
		eqIdx := bytes.Index(line, []byte{'='})
		if eqIdx < 0 {
			continue
		}
		key := string(bytes.TrimSpace(line[:eqIdx]))
		// Remove quotes from quoted keys
		if len(key) >= 2 && key[0] == '"' && key[len(key)-1] == '"' {
			key = key[1 : len(key)-1]
		}
		valueBytes := bytes.TrimSpace(line[eqIdx+1:])
		// Remove quotes from string values
		if len(valueBytes) >= 2 && valueBytes[0] == '"' && valueBytes[len(valueBytes)-1] == '"' {
			valueBytes = valueBytes[1 : len(valueBytes)-1]
		}
		c.Keys = append(c.Keys, key)
		c.Values[key] = string(valueBytes)
	}

	return nil
}

// Test for split tables - when the same parent table is defined in multiple places
// This is a key requirement for issue #873: if type A implements Unmarshaler,
// and [a.b] and [a.d] are defined with another table [x] in between,
// A should receive content for both b and d, but not x.
func TestIssue873_SplitTables(t *testing.T) {
	// For this test, we expect each sub-table to be handled separately
	// The parent doesn't receive the sub-tables directly - each sub-table
	// (b and d) gets its own call to handleKeyValues
	type Config struct {
		A struct {
			B customTable873 `toml:"b"`
			D customTable873 `toml:"d"`
		} `toml:"a"`
		X customTable873 `toml:"x"`
	}

	doc := `
[a.b]
C = "1"

[x]
Y = "100"

[a.d]
E = "2"
`

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.NoError(t, err)
	// Each sub-table should have received its own key-values
	assert.Equal(t, []string{"C"}, cfg.A.B.Keys)
	assert.Equal(t, "1", cfg.A.B.Values["C"])
	assert.Equal(t, []string{"E"}, cfg.A.D.Keys)
	assert.Equal(t, "2", cfg.A.D.Values["E"])
	assert.Equal(t, []string{"Y"}, cfg.X.Keys)
	assert.Equal(t, "100", cfg.X.Values["Y"])
}

// Test using RawMessage to capture raw TOML bytes
func TestIssue873_RawMessage(t *testing.T) {
	type Config struct {
		Plugin unstable.RawMessage `toml:"plugin"`
	}

	doc := `
[plugin]
name = "example"
version = "1.0"
`

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.NoError(t, err)
	// RawMessage should contain the raw key-value bytes
	expected := "name = \"example\"\nversion = \"1.0\"\n"
	assert.Equal(t, expected, string(cfg.Plugin))
}

// Test keys that need quoting (contain special characters)
func TestIssue873_QuotedKeys(t *testing.T) {
	type Config struct {
		Section customTable873 `toml:"section"`
	}

	doc := `
[section]
"key with spaces" = "value1"
"key.with.dots" = "value2"
`

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(cfg.Section.Keys))
	assert.Equal(t, "value1", cfg.Section.Values["key with spaces"])
	assert.Equal(t, "value2", cfg.Section.Values["key.with.dots"])
}

// errorUnmarshaler873 is used to test error propagation from UnmarshalTOML
type errorUnmarshaler873 struct{}

func (e *errorUnmarshaler873) UnmarshalTOML([]byte) error {
	return errors.New("intentional error")
}

// Test error propagation from UnmarshalTOML
func TestIssue873_UnmarshalerError(t *testing.T) {
	doc := `
[section]
key = "value"
`

	type Config struct {
		Section errorUnmarshaler873 `toml:"section"`
	}

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "intentional error"))
}

// Test dotted keys in a table (e.g., a.b = value)
func TestIssue873_DottedKeys(t *testing.T) {
	type Config struct {
		Section customTable873 `toml:"section"`
	}

	doc := `
[section]
sub.key = "value1"
another.nested.key = "value2"
`

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(cfg.Section.Keys))
	// The dotted keys should be preserved in the raw output
	assert.Equal(t, "value1", cfg.Section.Values["sub.key"])
	assert.Equal(t, "value2", cfg.Section.Values["another.nested.key"])
}

// Test pointer to pointer to Unmarshaler (covers pointer dereferencing loop)
func TestIssue873_DoublePointerUnmarshaler(t *testing.T) {
	type Config struct {
		Section **customTable873 `toml:"section"`
	}

	doc := `
[section]
key = "value"
`

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.NoError(t, err)
	assert.True(t, cfg.Section != nil)
	assert.True(t, *cfg.Section != nil)
	assert.Equal(t, []string{"key"}, (*cfg.Section).Keys)
	assert.Equal(t, "value", (*cfg.Section).Values["key"])
}

// formattingCapture captures the raw TOML bytes to verify formatting preservation
type formattingCapture struct {
	RawBytes string
}

func (f *formattingCapture) UnmarshalTOML(data []byte) error {
	f.RawBytes = string(data)
	return nil
}

func TestIssue873_FormattingPreservation(t *testing.T) {
	type Config struct {
		Section *formattingCapture `toml:"section"`
	}

	// Test that various formatting styles are preserved:
	// - Extra spaces around '='
	// - Literal strings (single quotes)
	// - Hex numbers
	// - Inline tables
	doc := `[section]
key1   =   "value with spaces"
key2 = 'literal string'
hex_val = 0xDEADBEEF
inline = { a = 1, b = 2 }
`

	var cfg Config
	err := toml.NewDecoder(bytes.NewReader([]byte(doc))).
		EnableUnmarshalerInterface().
		Decode(&cfg)

	assert.NoError(t, err)
	assert.True(t, cfg.Section != nil)

	// The raw bytes should preserve original formatting
	raw := cfg.Section.RawBytes

	// Check that extra spaces around '=' are preserved
	assert.True(t, strings.Contains(raw, "key1   =   \"value with spaces\""),
		"Expected spacing to be preserved, got: %s", raw)

	// Check that literal string style is preserved
	assert.True(t, strings.Contains(raw, "key2 = 'literal string'"),
		"Expected literal string to be preserved, got: %s", raw)

	// Check that hex format is preserved
	assert.True(t, strings.Contains(raw, "hex_val = 0xDEADBEEF"),
		"Expected hex format to be preserved, got: %s", raw)

	// Check that inline table is preserved
	assert.True(t, strings.Contains(raw, "inline = { a = 1, b = 2 }"),
		"Expected inline table to be preserved, got: %s", raw)
}
