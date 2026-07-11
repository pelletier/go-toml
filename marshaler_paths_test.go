package toml

import (
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
