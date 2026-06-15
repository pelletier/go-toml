package toml

import (
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func TestParseIntegerInvalidBasePanics(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = parseInteger([]byte("0z1"))
	})
}

func TestParseLocalDateInvalidYear(t *testing.T) {
	_, err := parseLocalDate([]byte("196f-01-01"))
	assert.Error(t, err)
}

func TestParseLocalDateTimeInvalidSeparator(t *testing.T) {
	_, _, err := parseLocalDateTime([]byte("2021-01-01x00:00:00"))
	assert.Error(t, err)
}

func TestParseDateTimeMissingTimezone(t *testing.T) {
	_, err := parseDateTime([]byte("2021-01-01T00:00:00"))
	assert.Error(t, err)
}
