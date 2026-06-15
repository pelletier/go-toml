package tracker

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

func TestKeyTrackerArrayTable(t *testing.T) {
	p := &unstable.Parser{}
	p.Reset([]byte("[[a.b]]"))
	assert.True(t, p.NextExpression())
	node := p.Expression()
	assert.Equal(t, unstable.ArrayTable, node.Kind)

	kt := KeyTracker{}
	kt.UpdateArrayTable(node)
	assert.Equal(t, []string{"a", "b"}, kt.Key())
}
