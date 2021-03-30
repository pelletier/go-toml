package unsafe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pelletier/go-toml/v2/internal/unsafe"
)

func TestUnsafeSubsliceOffsetValid(t *testing.T) {
	examples := []struct {
		desc   string
		test   func() ([]byte, []byte)
		offset int
	}{
		{
			desc: "simple",
			test: func() ([]byte, []byte) {
				data := []byte("hello")
				return data, data[1:]
			},
			offset: 1,
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			d, s := e.test()
			offset := unsafe.SubsliceOffset(d, s)
			assert.Equal(t, e.offset, offset)
		})
	}
}

func TestUnsafeSubsliceOffsetInvalid(t *testing.T) {
	examples := []struct {
		desc string
		test func() ([]byte, []byte)
	}{
		{
			desc: "unrelated arrays",
			test: func() ([]byte, []byte) {
				return []byte("one"), []byte("two")
			},
		},
		{
			desc: "slice starts before data",
			test: func() ([]byte, []byte) {
				full := []byte("hello world")
				return full[5:], full[1:]
			},
		},
		{
			desc: "slice starts after data",
			test: func() ([]byte, []byte) {
				full := []byte("hello world")
				return full[:3], full[5:]
			},
		},
		{
			desc: "slice ends after data",
			test: func() ([]byte, []byte) {
				full := []byte("hello world")
				return full[:5], full[3:8]
			},
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			d, s := e.test()
			require.Panics(t, func() {
				unsafe.SubsliceOffset(d, s)
			})
		})
	}
}
