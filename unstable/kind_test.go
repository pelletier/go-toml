package unstable

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func TestKindStringAll(t *testing.T) {
	expected := map[Kind]string{
		Invalid:       "Invalid",
		Comment:       "Comment",
		Key:           "Key",
		Table:         "Table",
		ArrayTable:    "ArrayTable",
		KeyValue:      "KeyValue",
		Array:         "Array",
		InlineTable:   "InlineTable",
		String:        "String",
		Bool:          "Bool",
		Float:         "Float",
		Integer:       "Integer",
		LocalDate:     "LocalDate",
		LocalTime:     "LocalTime",
		LocalDateTime: "LocalDateTime",
		DateTime:      "DateTime",
	}
	for k, s := range expected {
		assert.Equal(t, s, k.String())
	}
}

func TestKindStringUnknownPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = Kind(0xFF).String()
	})
}
