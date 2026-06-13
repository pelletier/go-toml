package toml

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

func TestKeyLocationEmptyKeyPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = keyLocation(&unstable.Node{Kind: unstable.Table})
	})
}
