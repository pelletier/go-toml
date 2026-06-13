package tracker

import (
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

func TestEntrySize(t *testing.T) {
	// Validate no regression on the size of entry{}. This is a critical bit for
	// performance of unmarshaling documents. Should only be increased with care
	// and a very good reason.
	maxExpectedEntrySize := 48
	entrySize := int(reflect.TypeOf(entry{}).Size())
	assert.True(t,
		entrySize <= maxExpectedEntrySize,
		"Expected entry to be less than or equal to %d, got: %d",
		maxExpectedEntrySize, entrySize,
	)
}

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

func parseExpr(t *testing.T, doc string) *unstable.Node {
	t.Helper()
	p := &unstable.Parser{}
	p.KeepComments = true
	p.Reset([]byte(doc))
	assert.True(t, p.NextExpression())
	return p.Expression()
}

func TestSeenTrackerFreshReset(t *testing.T) {
	doc := "a = 1"
	node := parseExpr(t, doc)
	st := SeenTracker{}
	first, err := st.CheckExpression([]byte(doc), node)
	assert.NoError(t, err)
	assert.False(t, first)
}

func TestSeenTrackerUnexpectedKind(t *testing.T) {
	doc := "# just a comment"
	node := parseExpr(t, doc)
	st := SeenTracker{}
	_, err := st.CheckExpression([]byte(doc), node)
	assert.Error(t, err)
}
