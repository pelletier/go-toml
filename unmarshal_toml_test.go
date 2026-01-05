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
