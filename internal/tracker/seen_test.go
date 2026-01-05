package tracker

import (
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
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
