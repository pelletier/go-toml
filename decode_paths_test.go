package toml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

// TestUnmarshalManyKeysTable exercises the seen-tracker's spill to its hash
// index (tables with more keys than fit the sibling chains), for both the
// fused generic path and the reflection path, including duplicate detection
// after the spill.
func TestUnmarshalManyKeysTable(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&sb, "key%d = %d\n", i, i)
	}
	doc := sb.String()

	t.Run("map", func(t *testing.T) {
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(doc), &m))
		assert.Equal(t, 100, len(m))
		assert.Equal(t, interface{}(int64(99)), m["key99"])
	})

	t.Run("struct", func(t *testing.T) {
		var s struct {
			Key0  int64
			Key42 int64
			Key99 int64
		}
		assert.NoError(t, Unmarshal([]byte(doc), &s))
		assert.Equal(t, int64(42), s.Key42)
		assert.Equal(t, int64(99), s.Key99)
	})

	t.Run("duplicate after spill", func(t *testing.T) {
		m := map[string]interface{}{}
		err := Unmarshal([]byte(doc+"key12 = 1\n"), &m)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "already"), "unexpected error: %v", err)
	})

	t.Run("array table refresh under root", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 40; i++ {
			fmt.Fprintf(&sb, "key%d = %d\n", i, i)
		}
		for i := 0; i < 3; i++ {
			sb.WriteString("[[elem]]\nname = 'x'\n")
		}
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(sb.String()), &m))
		assert.Equal(t, 3, len(m["elem"].([]interface{})))
	})
}

// TestUnmarshalGenericValueShapes covers the generic decoding of every scalar
// kind (through interface{} struct fields, which use the AST path), long

// TestArrayTableRefreshUnderSpilledParent covers refreshing an array table
// whose parent's children have spilled to the tracker's hash index, and the
// sibling-chain swap when they have not.
func TestArrayTableRefreshUnderSpilledParent(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 70; i++ {
		fmt.Fprintf(&sb, "key%d = %d\n", i, i)
	}
	sb.WriteString("[[t]]\nx = 1\n[[t]]\nx = 2\n")
	m := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(sb.String()), &m))
	assert.Equal(t, 2, len(m["t"].([]interface{})))

	// Unspilled parent, refresh target not at the chain head.
	doc := "a = 1\n[[t]]\nx = 1\nb = 2\n[[t]]\nx = 2\n"
	m2 := map[string]interface{}{}
	assert.NoError(t, Unmarshal([]byte(doc), &m2))
	assert.Equal(t, 2, len(m2["t"].([]interface{})))
}
