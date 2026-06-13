package toml_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

// failingUnmarshaler implements unstable.Unmarshaler and always fails.
type failingUnmarshaler struct{}

func (*failingUnmarshaler) UnmarshalTOML(_ []byte) error {
	return errors.New("always fails")
}

func TestUnmarshalMergeIntoExistingMapValues(t *testing.T) {
	type sub struct{ X int }

	t.Run("nil pointer element", func(t *testing.T) {
		m := map[string]*sub{"a": nil}
		assert.NoError(t, toml.Unmarshal([]byte("[a]\nx = 1"), &m))
		assert.Equal(t, 1, m["a"].X)
	})
	t.Run("interface holding struct replaced", func(t *testing.T) {
		m := map[string]interface{}{"a": sub{X: 9}}
		assert.NoError(t, toml.Unmarshal([]byte("[a]\nx = 1"), &m))
		assert.Equal(t, interface{}(map[string]interface{}{"x": int64(1)}), m["a"])
	})
	t.Run("interface holding scalar replaced", func(t *testing.T) {
		m := map[string]interface{}{"a": 42}
		assert.NoError(t, toml.Unmarshal([]byte("[a]\nx = 1"), &m))
		assert.Equal(t, interface{}(map[string]interface{}{"x": int64(1)}), m["a"])
	})
	t.Run("scalar element errors", func(t *testing.T) {
		m := map[string]int{"a": 1}
		assert.Error(t, toml.Unmarshal([]byte("[a]\nx = 1"), &m))
	})
	t.Run("incompatible interface element type", func(t *testing.T) {
		m := map[string]error{}
		assert.Error(t, toml.Unmarshal([]byte("[a]\nx = 1"), &m))
	})
}

func TestUnmarshalerInterfaceCaptureThroughWrapper(t *testing.T) {
	type wrapper struct {
		X rawRecorder
	}
	type target struct {
		M *wrapper
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, "[m.x]\na = 1", &x))
	assert.True(t, strings.Contains(x.M.X.Raw, "a = 1"))
}

func TestUnmarshalerInterfaceFailures(t *testing.T) {
	t.Run("table target fails", func(t *testing.T) {
		x := struct{ M failingUnmarshaler }{}
		assert.Error(t, decodeRaw(t, "[m]\na = 1", &x))
	})
	t.Run("pointer target fails", func(t *testing.T) {
		x := struct{ M *failingUnmarshaler }{}
		assert.Error(t, decodeRaw(t, "[m]\na = 1", &x))
	})
	t.Run("array table element fails", func(t *testing.T) {
		x := struct{ M []failingUnmarshaler }{}
		assert.Error(t, decodeRaw(t, "[[m]]\na = 1", &x))
	})
	t.Run("map element fails", func(t *testing.T) {
		x := struct{ M map[string]failingUnmarshaler }{}
		assert.Error(t, decodeRaw(t, "[m.x]\na = 1", &x))
	})
	t.Run("nested struct field fails", func(t *testing.T) {
		type wrapper struct {
			X failingUnmarshaler
		}
		x := struct{ M wrapper }{}
		assert.Error(t, decodeRaw(t, "[m.x]\na = 1", &x))
	})
	t.Run("kv pointer target", func(t *testing.T) {
		x := struct{ A *rawRecorder }{}
		assert.NoError(t, decodeRaw(t, "a = 5", &x))
		assert.Equal(t, "5", x.A.Raw)
	})
	t.Run("array elements", func(t *testing.T) {
		x := struct{ A []rawRecorder }{}
		assert.NoError(t, decodeRaw(t, "a = [{x = 1}, {y = 2}]", &x))
		assert.Equal(t, 2, len(x.A))
		// Nested inline containers are delivered best-effort: their raw
		// span is the opening token only (same behavior as v2).
		assert.Equal(t, "{", x.A[0].Raw)
	})
}

func TestUnmarshalDottedKeyPaths(t *testing.T) {
	t.Run("through pointer", func(t *testing.T) {
		type inner struct{ B int }
		x := struct{ A *inner }{}
		assert.NoError(t, toml.Unmarshal([]byte("a.b = 1"), &x))
		assert.Equal(t, 1, x.A.B)
	})
	t.Run("through slice", func(t *testing.T) {
		type inner struct{ B int }
		x := struct{ A []inner }{}
		assert.NoError(t, toml.Unmarshal([]byte("a.b = 1"), &x))
		assert.Equal(t, 1, x.A[0].B)
	})
	t.Run("through fixed array", func(t *testing.T) {
		type inner struct{ B int }
		x := struct{ A [2]inner }{}
		assert.NoError(t, toml.Unmarshal([]byte("a.b = 1"), &x))
		assert.Equal(t, 1, x.A[0].B)
	})
}

func TestUnmarshalScalarConversions(t *testing.T) {
	t.Run("uint", func(t *testing.T) {
		x := struct{ A uint }{}
		assert.NoError(t, toml.Unmarshal([]byte("a = 5"), &x))
		assert.Equal(t, uint(5), x.A)
	})
	t.Run("float32", func(t *testing.T) {
		x := struct{ A float32 }{}
		assert.NoError(t, toml.Unmarshal([]byte("a = 1.5"), &x))
		assert.Equal(t, float32(1.5), x.A)
	})
	t.Run("inf into float32", func(t *testing.T) {
		x := struct{ A float32 }{}
		assert.NoError(t, toml.Unmarshal([]byte("a = inf"), &x))
	})
	t.Run("string into non-empty interface", func(t *testing.T) {
		x := struct{ A error }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = "x"`), &x))
	})
}

func TestUnmarshalValueParseErrors(t *testing.T) {
	t.Run("huge float in array into interface", func(t *testing.T) {
		var x interface{}
		assert.Error(t, toml.Unmarshal([]byte(`a = [1e999]`), &x))
	})
	t.Run("huge float in nested inline table into interface", func(t *testing.T) {
		var x interface{}
		assert.Error(t, toml.Unmarshal([]byte(`a = [{b = 1e999}]`), &x))
	})
	t.Run("huge float in inline table into interface", func(t *testing.T) {
		var x interface{}
		assert.Error(t, toml.Unmarshal([]byte(`a = {b = 1e999}`), &x))
	})
}

func TestUnmarshalInternTableSafetyValve(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 17000; i++ {
		fmt.Fprintf(&sb, "k%d = 1\n", i)
	}
	m := map[string]int{}
	assert.NoError(t, toml.Unmarshal([]byte(sb.String()), &m))
	assert.Equal(t, 17000, len(m))
}

func TestUnmarshalFixedArrayElementError(t *testing.T) {
	type inner struct{ B int }
	x := struct{ A [2]inner }{}
	assert.Error(t, toml.Unmarshal([]byte(`a.b = "x"`), &x))
}

func TestUnmarshalArrayTableClearWithSurvivors(t *testing.T) {
	doc := `
[[a]]
b = 1
[c]
d = 1
[[a]]
e = 1
`
	m := map[string]interface{}{}
	assert.NoError(t, toml.Unmarshal([]byte(doc), &m))
	assert.Equal(t, 1, len(m["a"].([]interface{}))-1)
}
