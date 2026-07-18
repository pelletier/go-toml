package toml

import (
	"errors"
	"io"
	"reflect"
	"strings"
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

// lyingLenReader reports a Len smaller than the content it delivers,
// exercising readDocument's fallback to io.ReadAll when a reader outgrows
// its announced size.
type lyingLenReader struct {
	r io.Reader
}

func (l *lyingLenReader) Read(p []byte) (int, error) { return l.r.Read(p) }
func (l *lyingLenReader) Len() int                   { return 2 }

// TestReadDocumentGrowingReader covers readers whose Len underestimates the
// actual content.
func TestReadDocumentGrowingReader(t *testing.T) {
	var m map[string]interface{}
	d := NewDecoder(&lyingLenReader{r: strings.NewReader("key = 'longer than two bytes'")})
	assert.NoError(t, d.Decode(&m))
	assert.Equal(t, interface{}("longer than two bytes"), m["key"])
}

type errAfterReader struct{ n int }

func (e *errAfterReader) Read(p []byte) (int, error) {
	if e.n == 0 {
		return 0, errors.New("boom")
	}
	p[0] = 'a'
	e.n--
	return 1, nil
}

func (e *errAfterReader) Len() int { return 8 }

// noLenReader hides the Len method of its underlying reader, exercising
// readDocument's io.ReadAll fallback for readers of unknown size.
type noLenReader struct {
	r io.Reader
}

func (n *noLenReader) Read(p []byte) (int, error) { return n.r.Read(p) }

func TestReadDocumentReadError(t *testing.T) {
	var m map[string]interface{}
	d := NewDecoder(&errAfterReader{n: 1})
	assert.Error(t, d.Decode(&m))

	d = NewDecoder(&noLenReader{r: &errAfterReader{n: 1}})
	assert.Error(t, d.Decode(&m))

	// Error surfaced by the io.ReadAll finish after the buffer filled up.
	d = NewDecoder(&lyingLenReader{r: &errAfterReader{n: 3}})
	assert.Error(t, d.Decode(&m))
}

func TestReadDocumentNoLen(t *testing.T) {
	var m map[string]interface{}
	d := NewDecoder(&noLenReader{r: strings.NewReader("a = 1")})
	assert.NoError(t, d.Decode(&m))
	assert.Equal(t, interface{}(int64(1)), m["a"])
}
