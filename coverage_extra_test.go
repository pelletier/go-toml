package toml_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

// rawRecorder implements unstable.Unmarshaler and records the raw bytes it
// receives.
type rawRecorder struct {
	Raw string
}

func (r *rawRecorder) UnmarshalTOML(b []byte) error {
	r.Raw = string(b)
	return nil
}

func decodeRaw(t *testing.T, doc string, target interface{}) error {
	t.Helper()
	d := toml.NewDecoder(strings.NewReader(doc)).EnableUnmarshalerInterface()
	return d.Decode(target)
}

func TestUnmarshalerInterfaceTableWithSubTables(t *testing.T) {
	doc := `
[outer]
a = 1
[outer.sub]
b = 2
[[outer.arr]]
c = 3
[other]
d = 4
`
	type target struct {
		Outer rawRecorder
		Other map[string]int
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.True(t, strings.Contains(x.Outer.Raw, "a = 1"))
	assert.True(t, strings.Contains(x.Outer.Raw, "[sub]"))
	assert.True(t, strings.Contains(x.Outer.Raw, "b = 2"))
	assert.True(t, strings.Contains(x.Outer.Raw, "[[arr]]"))
	assert.Equal(t, map[string]int{"d": 4}, x.Other)
}

func TestUnmarshalerInterfaceArrayTableElements(t *testing.T) {
	doc := `
[[item]]
a = 1
[[item]]
a = 2
[item.sub]
b = 3
`
	type target struct {
		Item []rawRecorder
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.Equal(t, 2, len(x.Item))
	assert.True(t, strings.Contains(x.Item[0].Raw, "a = 1"))
	assert.True(t, strings.Contains(x.Item[1].Raw, "a = 2"))
	assert.True(t, strings.Contains(x.Item[1].Raw, "[sub]"))
}

func TestUnmarshalerInterfaceUnderMap(t *testing.T) {
	doc := `
[m.x]
a = 1
`
	x := map[string]rawRecorder{}
	target := struct{ M map[string]rawRecorder }{}
	assert.NoError(t, decodeRaw(t, doc, &target))
	assert.True(t, strings.Contains(target.M["x"].Raw, "a = 1"))
	_ = x
}

func TestUnmarshalerInterfaceBehindPointer(t *testing.T) {
	doc := `
[outer]
a = 1
`
	type target struct {
		Outer *rawRecorder
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.True(t, x.Outer != nil)
	assert.True(t, strings.Contains(x.Outer.Raw, "a = 1"))
}

func TestUnmarshalerInterfaceFixedArrayElements(t *testing.T) {
	doc := `
[[item]]
a = 1
[[item]]
a = 2
`
	type target struct {
		Item [2]rawRecorder
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.True(t, strings.Contains(x.Item[0].Raw, "a = 1"))
	assert.True(t, strings.Contains(x.Item[1].Raw, "a = 2"))
}

func TestUnmarshalerInterfaceKeyValueTargets(t *testing.T) {
	doc := `
inline = {a = 1, b = "x"}
arr = [1, 2, 3]
scalar = 42
`
	type target struct {
		Inline rawRecorder
		Arr    rawRecorder
		Scalar rawRecorder
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.Equal(t, `{a = 1, b = "x"}`, x.Inline.Raw)
	assert.Equal(t, `[1, 2, 3]`, x.Arr.Raw)
	assert.Equal(t, `42`, x.Scalar.Raw)
}

func TestUnmarshalerInterfaceDeepTableKey(t *testing.T) {
	doc := `
[outer.inner]
a = 1
`
	type target struct {
		Outer rawRecorder
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.True(t, strings.Contains(x.Outer.Raw, "[inner]"))
	assert.True(t, strings.Contains(x.Outer.Raw, "a = 1"))
}

func TestUnmarshalerInterfaceDottedKeyValue(t *testing.T) {
	doc := `
[outer]
sub.a = 1
`
	type target struct {
		Outer rawRecorder
	}
	x := target{}
	assert.NoError(t, decodeRaw(t, doc, &x))
	assert.True(t, strings.Contains(x.Outer.Raw, "sub.a = 1"))
}

func TestUnmarshalMapKeyKinds(t *testing.T) {
	t.Run("float key", func(t *testing.T) {
		m := map[float64]int{}
		assert.NoError(t, toml.Unmarshal([]byte(`"1.5" = 3`), &m))
		assert.Equal(t, map[float64]int{1.5: 3}, m)
	})
	t.Run("int key parse error", func(t *testing.T) {
		m := map[int]int{}
		assert.Error(t, toml.Unmarshal([]byte(`abc = 1`), &m))
	})
	t.Run("int key overflow", func(t *testing.T) {
		m := map[int8]int{}
		assert.Error(t, toml.Unmarshal([]byte(`300 = 1`), &m))
	})
	t.Run("uint key parse error", func(t *testing.T) {
		m := map[uint]int{}
		assert.Error(t, toml.Unmarshal([]byte(`abc = 1`), &m))
	})
	t.Run("uint key overflow", func(t *testing.T) {
		m := map[uint8]int{}
		assert.Error(t, toml.Unmarshal([]byte(`300 = 1`), &m))
	})
	t.Run("float key parse error", func(t *testing.T) {
		m := map[float64]int{}
		assert.Error(t, toml.Unmarshal([]byte(`abc = 1`), &m))
	})
	t.Run("unsupported key kind", func(t *testing.T) {
		m := map[bool]int{}
		assert.Error(t, toml.Unmarshal([]byte(`a = 1`), &m))
	})
	t.Run("unsupported key kind in table header", func(t *testing.T) {
		m := map[bool]map[string]int{}
		assert.Error(t, toml.Unmarshal([]byte("[a]\nx = 1"), &m))
	})
	t.Run("int key in table header", func(t *testing.T) {
		m := map[int]map[string]int{}
		assert.NoError(t, toml.Unmarshal([]byte("[12]\nx = 1"), &m))
		assert.Equal(t, 1, m[12]["x"])
	})
	t.Run("invalid int key in table header", func(t *testing.T) {
		m := map[int]map[string]int{}
		assert.Error(t, toml.Unmarshal([]byte("[nope]\nx = 1"), &m))
	})
}

func TestUnmarshalScalarOverflows(t *testing.T) {
	t.Run("int8 overflow", func(t *testing.T) {
		x := struct{ A int8 }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = 300`), &x))
	})
	t.Run("negative into uint", func(t *testing.T) {
		x := struct{ A uint8 }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = -1`), &x))
	})
	t.Run("uint8 overflow", func(t *testing.T) {
		x := struct{ A uint8 }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = 300`), &x))
	})
	t.Run("float32 overflow", func(t *testing.T) {
		x := struct{ A float32 }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = 1e300`), &x))
	})
}

func TestUnmarshalTableStructureErrors(t *testing.T) {
	t.Run("array table into scalar", func(t *testing.T) {
		x := struct{ A int }{}
		assert.Error(t, toml.Unmarshal([]byte("[[a]]\nx = 1"), &x))
	})
	t.Run("table walk through scalar", func(t *testing.T) {
		x := struct{ A int }{}
		assert.Error(t, toml.Unmarshal([]byte("[a.b]\nx = 1"), &x))
	})
	t.Run("table into scalar map element", func(t *testing.T) {
		m := map[string]int{}
		assert.Error(t, toml.Unmarshal([]byte("[m.a]\nx = 1"), &struct{ M map[string]int }{m}))
	})
	t.Run("table into incompatible interface", func(t *testing.T) {
		x := struct{ A error }{}
		assert.Error(t, toml.Unmarshal([]byte("[a]\nx = 1"), &x))
	})
	t.Run("walk through incompatible interface", func(t *testing.T) {
		x := struct{ A error }{}
		assert.Error(t, toml.Unmarshal([]byte("[a.b]\nx = 1"), &x))
	})
	t.Run("array table overflow fixed array", func(t *testing.T) {
		x := struct{ A [1]map[string]int }{}
		assert.Error(t, toml.Unmarshal([]byte("[[a]]\nx = 1\n[[a]]\ny = 2"), &x))
	})
	t.Run("table header through empty fixed array", func(t *testing.T) {
		x := struct{ A [0]map[string]int }{}
		assert.Error(t, toml.Unmarshal([]byte("[a.b]\nx = 1"), &x))
	})
	t.Run("dotted key through empty fixed array", func(t *testing.T) {
		x := struct{ A [0]map[string]int }{}
		assert.Error(t, toml.Unmarshal([]byte("a.b = 1"), &x))
	})
}

func TestUnmarshalMapStructValueWriteBack(t *testing.T) {
	type inner struct {
		X int
	}
	type elem struct {
		F     int
		Inner inner
	}
	doc := `
[m.a.inner]
x = 1
[m.a]
f = 2
`
	target := struct{ M map[string]elem }{}
	assert.NoError(t, toml.Unmarshal([]byte(doc), &target))
	assert.Equal(t, 1, target.M["a"].Inner.X)
	assert.Equal(t, 2, target.M["a"].F)
}

func TestUnmarshalMapPointerValues(t *testing.T) {
	type elem struct {
		F int
	}
	doc := `
[m.a]
f = 1
[m.a.sub]
`
	target := struct {
		M map[string]*elem
		// sub does not exist in elem: strict-less skip
	}{}
	assert.NoError(t, toml.Unmarshal([]byte(doc), &target))
	assert.Equal(t, 1, target.M["a"].F)
}

func TestUnmarshalLocalTimeIntoTime(t *testing.T) {
	x := struct{ A time.Time }{}
	assert.NoError(t, toml.Unmarshal([]byte(`a = 04:05:06`), &x))
	assert.Equal(t, 4, x.A.Hour())
	assert.Equal(t, 5, x.A.Minute())
	assert.Equal(t, 6, x.A.Second())

	y := struct{ A int }{}
	assert.Error(t, toml.Unmarshal([]byte(`a = 04:05:06`), &y))
}

func TestUnmarshalArrayErrors(t *testing.T) {
	t.Run("bad integer element into interface", func(t *testing.T) {
		var x interface{}
		assert.Error(t, toml.Unmarshal([]byte(`a = [9223372036854775808]`), &x))
	})
	t.Run("bad float element into interface", func(t *testing.T) {
		var x interface{}
		assert.Error(t, toml.Unmarshal([]byte(`a = [1.e1]`), &x))
	})
	t.Run("bad element into slice", func(t *testing.T) {
		x := struct{ A []int }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = ["x"]`), &x))
	})
	t.Run("bad element into fixed array", func(t *testing.T) {
		x := struct{ A [2]int }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = ["x"]`), &x))
	})
	t.Run("inline table into int", func(t *testing.T) {
		x := struct{ A int }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = {b = 1}`), &x))
	})
	t.Run("array into int", func(t *testing.T) {
		x := struct{ A int }{}
		assert.Error(t, toml.Unmarshal([]byte(`a = [1]`), &x))
	})
	t.Run("nested array element error in interface", func(t *testing.T) {
		var x interface{}
		assert.Error(t, toml.Unmarshal([]byte(`a = [{b = 1.e1}]`), &x))
	})
}

func TestDecoderReadError(t *testing.T) {
	d := toml.NewDecoder(failingReader{})
	x := map[string]interface{}{}
	assert.Error(t, d.Decode(&x))
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, bytes.ErrTooLarge
}
