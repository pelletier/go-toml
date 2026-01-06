package toml_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

// Custom string type with UnmarshalTOML
type CustomString string

func (cs *CustomString) UnmarshalTOML(data []byte) error {
	// Remove quotes if present
	s := string(data)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		s = s[1 : len(s)-1]
	}
	*cs = CustomString("CUSTOM: " + s)
	return nil
}

// Custom int type with UnmarshalTOML
type CustomInt int

func (ci *CustomInt) UnmarshalTOML(data []byte) error {
	val, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	*ci = CustomInt(val * 2) // Double the value
	return nil
}

// Custom float type with UnmarshalTOML
type CustomFloat float64

func (cf *CustomFloat) UnmarshalTOML(data []byte) error {
	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	*cf = CustomFloat(val * 3.0) // Triple the value
	return nil
}

// Custom bool type with UnmarshalTOML
type CustomBool bool

func (cb *CustomBool) UnmarshalTOML(data []byte) error {
	s := string(data)
	*cb = CustomBool(s == "false") // Invert the boolean
	return nil
}

// Custom uint type with UnmarshalTOML
type CustomUint uint64

func (cu *CustomUint) UnmarshalTOML(data []byte) error {
	val, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return err
	}
	*cu = CustomUint(val + 100) // Add 100 to the value
	return nil
}

// Custom type that returns an error
type ErrorUnmarshaler string

func (eu *ErrorUnmarshaler) UnmarshalTOML(data []byte) error {
	return fmt.Errorf("intentional error")
}

// TestUnmarshalTOMLString tests UnmarshalTOML for string types
func TestUnmarshalTOMLString(t *testing.T) {
	type Config struct {
		Name CustomString
	}

	var cfg Config
	doc := `name = "test"`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomString("CUSTOM: test"), cfg.Name)
}

// TestUnmarshalTOMLInt tests UnmarshalTOML for int types
func TestUnmarshalTOMLInt(t *testing.T) {
	type Config struct {
		Value CustomInt
	}

	var cfg Config
	doc := `value = 42`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomInt(84), cfg.Value)
}

// TestUnmarshalTOMLFloat tests UnmarshalTOML for float types
func TestUnmarshalTOMLFloat(t *testing.T) {
	type Config struct {
		Value CustomFloat
	}

	var cfg Config
	doc := `value = 3.14`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomFloat(9.42), cfg.Value)
}

// TestUnmarshalTOMLBool tests UnmarshalTOML for bool types
func TestUnmarshalTOMLBool(t *testing.T) {
	type Config struct {
		Flag CustomBool
	}

	var cfg Config
	doc := `flag = true`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomBool(false), cfg.Flag) // Inverted

	doc = `flag = false`
	err = toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomBool(true), cfg.Flag) // Inverted
}

// TestUnmarshalTOMLUint tests UnmarshalTOML for uint types
func TestUnmarshalTOMLUint(t *testing.T) {
	type Config struct {
		Count CustomUint
	}

	var cfg Config
	doc := `count = 50`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomUint(150), cfg.Count)
}

// TestUnmarshalTOMLError tests error handling
func TestUnmarshalTOMLError(t *testing.T) {
	type Config struct {
		Value ErrorUnmarshaler
	}

	var cfg Config
	doc := `value = "test"`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.Error(t, err)
	if err != nil && err.Error() != "intentional error" {
		t.Errorf("expected error to contain 'intentional error', got: %s", err.Error())
	}
}

// TestUnmarshalTOMLMultipleFields tests with multiple custom fields
func TestUnmarshalTOMLMultipleFields(t *testing.T) {
	type Config struct {
		Name  CustomString
		Value CustomInt
		Score CustomFloat
		Flag  CustomBool
		Count CustomUint
	}

	var cfg Config
	doc := `
name = "test"
value = 10
score = 2.5
flag = true
count = 25
`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, CustomString("CUSTOM: test"), cfg.Name)
	assert.Equal(t, CustomInt(20), cfg.Value)
	assert.Equal(t, CustomFloat(7.5), cfg.Score)
	assert.Equal(t, CustomBool(false), cfg.Flag) // Inverted
	assert.Equal(t, CustomUint(125), cfg.Count)
}

type TestStruct struct {
	Values map[string]any
}

func (ts *TestStruct) UnmarshalTOML(data []byte) error {
	ts.Values = make(map[string]any)
	return toml.Unmarshal(data, &ts.Values)
}

func TestUnmarshalTOMLOnStruct(t *testing.T) {
	type S struct {
		V    TestStruct `toml:"values"`
		Name string
	}
	var cfg S
	doc := `name = "testing"
	
	[values]
	name = "test"
	value = 10
	score = 2.5
	flag = true
	count = 25
	`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "test", cfg.V.Values["name"])
	assert.Equal(t, int64(10), cfg.V.Values["value"].(int64))
}

// TestUnmarshalTOMLOnStructWithBool tests the slow path with boolean values (no Raw fields)
func TestUnmarshalTOMLOnStructWithBool(t *testing.T) {
	type S struct {
		V TestStruct `toml:"settings"`
	}
	var cfg S
	doc := `
[settings]
enabled = true
disabled = false
active = true
`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, true, cfg.V.Values["enabled"])
	assert.Equal(t, false, cfg.V.Values["disabled"])
	assert.Equal(t, true, cfg.V.Values["active"])
}

// TestUnmarshalTOMLOnStructWithArrays tests the slow path with array values (no Raw fields)
func TestUnmarshalTOMLOnStructWithArrays(t *testing.T) {
	type S struct {
		V TestStruct `toml:"config"`
	}
	var cfg S
	doc := `
[config]
numbers = [1, 2, 3, 4, 5]
strings = ["hello", "world"]
mixed = [1, "two", 3.0]
`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)

	numbers := cfg.V.Values["numbers"].([]interface{})
	assert.Equal(t, 5, len(numbers))
	assert.Equal(t, int64(1), numbers[0].(int64))
	assert.Equal(t, int64(5), numbers[4].(int64))

	strings := cfg.V.Values["strings"].([]interface{})
	assert.Equal(t, 2, len(strings))
	assert.Equal(t, "hello", strings[0])
	assert.Equal(t, "world", strings[1])
}

// TestUnmarshalTOMLOnStructWithDateTime tests the slow path with datetime values (no Raw fields)
func TestUnmarshalTOMLOnStructWithDateTime(t *testing.T) {
	type S struct {
		V TestStruct `toml:"timestamps"`
	}
	var cfg S
	doc := `
[timestamps]
created = 1979-05-27T07:32:00Z
modified = 1979-05-27T00:32:00-07:00
local_datetime = 1979-05-27T07:32:00
local_date = 1979-05-27
local_time = 07:32:00
`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)

	// Verify datetime fields exist and are parsed
	assert.NotZero(t, cfg.V.Values["created"])
	assert.NotZero(t, cfg.V.Values["modified"])
	assert.NotZero(t, cfg.V.Values["local_datetime"])
	assert.NotZero(t, cfg.V.Values["local_date"])
	assert.NotZero(t, cfg.V.Values["local_time"])
}

// TestUnmarshalTOMLOnStructMixed tests mixed value types (triggers slow path)
func TestUnmarshalTOMLOnStructMixed(t *testing.T) {
	type S struct {
		V TestStruct `toml:"data"`
	}
	var cfg S
	doc := `
[data]
name = "test"
count = 42
score = 3.14
enabled = true
tags = ["go", "toml", "test"]
created = 2024-01-15T10:30:00Z
`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)

	// Verify all types are correctly unmarshaled through slow path
	assert.Equal(t, "test", cfg.V.Values["name"])
	assert.Equal(t, int64(42), cfg.V.Values["count"].(int64))
	assert.Equal(t, 3.14, cfg.V.Values["score"])
	assert.Equal(t, true, cfg.V.Values["enabled"])

	tags := cfg.V.Values["tags"].([]interface{})
	assert.Equal(t, 3, len(tags))
	assert.Equal(t, "go", tags[0])

	assert.NotZero(t, cfg.V.Values["created"])
}

// TestUnmarshalTOMLOnStructFastPath tests that fast path is used for strings/numbers only
func TestUnmarshalTOMLOnStructFastPath(t *testing.T) {
	type S struct {
		V TestStruct `toml:"metrics"`
	}
	var cfg S
	doc := `
[metrics]
name = "performance"
requests = 1000
latency = 25.5
throughput = 99.9
status = "healthy"
`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)

	// Verify fast path works correctly (all values have Raw fields)
	assert.Equal(t, "performance", cfg.V.Values["name"])
	assert.Equal(t, int64(1000), cfg.V.Values["requests"].(int64))
	assert.Equal(t, 25.5, cfg.V.Values["latency"])
	assert.Equal(t, 99.9, cfg.V.Values["throughput"])
	assert.Equal(t, "healthy", cfg.V.Values["status"])
}

// CustomArray is a custom array type with UnmarshalTOML
type CustomArray []string

func (ca *CustomArray) UnmarshalTOML(data []byte) error {
	// Parse as TOML array - wrap in document since data is just the value
	doc := []byte(fmt.Sprintf("value = %s", data))
	var wrapper struct {
		Value []string
	}
	if err := toml.Unmarshal(doc, &wrapper); err != nil {
		return err
	}
	*ca = make(CustomArray, len(wrapper.Value))
	for i, v := range wrapper.Value {
		(*ca)[i] = "custom-" + v
	}
	return nil
}

// TestUnmarshalTOMLCustomArrayType tests custom array type with UnmarshalTOML
func TestUnmarshalTOMLCustomArrayType(t *testing.T) {
	type Config struct {
		Tags CustomArray `toml:"tags"`
	}
	var cfg Config
	doc := `tags = ["go", "toml", "test"]`
	err := toml.Unmarshal([]byte(doc), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(cfg.Tags))
	assert.Equal(t, "custom-go", cfg.Tags[0])
	assert.Equal(t, "custom-toml", cfg.Tags[1])
	assert.Equal(t, "custom-test", cfg.Tags[2])
}
