package toml

import (
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func TestArrayCountsFreshDecoder(t *testing.T) {
	// Whether a decode starts with a nil arrayCounts map depends on decoder
	// pooling: a reused decoder keeps the map allocated across documents.
	// Exercise the nil-map paths on a guaranteed-fresh decoder so their
	// coverage does not depend on sync.Pool behavior.
	d := &decoder{}
	key := []byte("a")
	assert.Equal(t, 0, d.arrayCount(key))
	d.resetChildArrayCounts(key)
	d.setArrayCount(key, 1)
	assert.Equal(t, 1, d.arrayCount(key))
	d.setArrayCount(key, 2)
	assert.Equal(t, 2, d.arrayCount(key))
	assert.Equal(t, 0, d.arrayCount([]byte("b")))
	// Child paths are joined with a NUL separator; resetting the parent
	// zeroes the child count but leaves the parent untouched.
	child := []byte("a\x00b")
	d.setArrayCount(child, 3)
	d.resetChildArrayCounts(key)
	assert.Equal(t, 0, d.arrayCount(child))
	assert.Equal(t, 2, d.arrayCount(key))
}

type unexpCaptureInner struct{ C int }

type unexpCaptureOuter struct {
	*unexpCaptureInner
}

func TestResolveCaptureUnexportedEmbeddedPointer(t *testing.T) {
	// resolveCapture walks the same field paths as walkTable, which has
	// either allocated the intermediate embedded pointers or already failed
	// by the time captures are resolved. Exercise its defensive branch
	// directly: resolving a capture through a nil embedded pointer of
	// unexported type must surface fieldByIndexAlloc's error, not panic.
	d := &decoder{}
	c := rawCapture{
		names:   []string{"C"},
		indexes: []int{-1, -1},
		buf:     []byte("x = 1\n"),
	}
	root := reflect.ValueOf(&unexpCaptureOuter{}).Elem()
	_, err := d.resolveCapture(root, &c, 0, false)
	assert.Error(t, err)
	assert.Equal(t, "toml: cannot set embedded pointer to unexported struct: toml.unexpCaptureInner", err.Error())
}
