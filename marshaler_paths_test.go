package toml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

// tmText has a pointer-receiver TextMarshaler; tmRawVal/tmRawPtr implement
// unstable.Marshaler with value and pointer receivers.
type tmText struct{ s string }

func (t *tmText) MarshalText() ([]byte, error) { return []byte(t.s), nil }

type tmRawVal struct{}

func (tmRawVal) MarshalTOML() ([]byte, error) { return []byte("1"), nil }

type tmRawPtr struct{}

func (*tmRawPtr) MarshalTOML() ([]byte, error) { return []byte("2"), nil }

// TestEncPropsReceiverVariants covers the pointer-receiver classification
// branches of the encoder's type properties.
func TestEncPropsReceiverVariants(t *testing.T) {
	out, err := Marshal(map[string]tmText{"a": {s: "x"}})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), "'x'"), "got %q", out)

	var buf strings.Builder
	enc := NewEncoder(&buf)
	enc.EnableMarshalerInterface()
	assert.NoError(t, enc.Encode(map[string]interface{}{
		"v": tmRawVal{},
		"p": &tmRawPtr{},
	}))
	back := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(buf.String()), &back))
	assert.Equal(t, interface{}(int64(1)), back["v"])
	assert.Equal(t, interface{}(int64(2)), back["p"])
}

// TestMarshalLargeTypedMap covers the value-slab path of map entry
// collection (maps with at least eight entries).
func TestMarshalLargeTypedMap(t *testing.T) {
	m := map[string]int64{}
	for i := 0; i < 12; i++ {
		m[fmt.Sprintf("key%02d", i)] = int64(i)
	}
	out, err := Marshal(m)
	assert.NoError(t, err)
	back := map[string]int64{}
	assert.NoError(t, Unmarshal(out, &back))
	assert.Equal(t, m, back)
}
