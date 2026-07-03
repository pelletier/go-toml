package toml

import (
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

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
