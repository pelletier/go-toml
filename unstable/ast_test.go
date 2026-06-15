package unstable

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func TestNodeKeyPanicsOnUnsupportedKind(t *testing.T) {
	n := &Node{Kind: Integer}
	assert.Panics(t, func() {
		_ = n.Key()
	})
}

func TestNodeKeyPanicsOnEmptyKeyValue(t *testing.T) {
	n := &Node{Kind: KeyValue}
	assert.Panics(t, func() {
		_ = n.Key()
	})
}
