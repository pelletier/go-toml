package tracker

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

func TestKeyKindString(t *testing.T) {
	expected := map[keyKind]string{
		invalidKind:    "invalid",
		valueKind:      "value",
		kvTableKind:    "kv-table",
		tableKind:      "table",
		arrayTableKind: "array-table",
		anonymousKind:  "anonymous",
	}
	for k, s := range expected {
		assert.Equal(t, s, k.String())
	}
}

func TestKeyKindStringUnknownPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = keyKind(0xFF).String()
	})
}

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

func parseExpr(t *testing.T, doc string) *unstable.Node {
	t.Helper()
	p := &unstable.Parser{}
	p.KeepComments = true
	p.Reset([]byte(doc))
	assert.True(t, p.NextExpression())
	return p.Expression()
}

func TestSeenTrackerFreshReset(t *testing.T) {
	node := parseExpr(t, "a = 1")
	st := SeenTracker{}
	first, err := st.CheckExpression(node)
	assert.NoError(t, err)
	assert.False(t, first)
}

func TestSeenTrackerUnexpectedKind(t *testing.T) {
	node := parseExpr(t, "# just a comment")
	st := SeenTracker{}
	_, err := st.CheckExpression(node)
	assert.Error(t, err)
}
