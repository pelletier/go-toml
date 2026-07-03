package toml

import (
	"math"
	"math/rand"
	"strconv"
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

// TestParseFloatWideSignificand exercises the exact 128-bit path used when the
// significand does not fit in 53 bits or the exponent is outside the exact
// float64 powers of ten, comparing bit patterns against strconv.
func TestParseFloatWideSignificand(t *testing.T) {
	cases := []string{
		// 17 significant digits (coordinate-style data): > 2^53.
		"45.302308000000002", "-65.613616999999977", "43.420273000000009",
		// 19 digits, the maximum accumulated.
		"9999999999999999999.0", "1234567890123456789.0",
		"-9223372036854775809.5",
		// Exponents just outside the Clinger range on both sides.
		"1e23", "-1e23", "1e27", "1e-23", "1e-27", "7e26", "7e-26",
		"123456789012345678e-27", "999999999999999999e27",
		// Half-way and boundary cases around the 53-bit mantissa limit.
		"9007199254740993.0", "9007199254740994.0", "9007199254740995.0",
		"18014398509481985.0", "4503599627370497.5",
		// Wide fractions with trailing digits forcing sticky-bit rounding.
		"2.00000000000000011", "1.00000000000000033",
		// Zero mantissa through the wide path.
		"0.00000000000000000000000",
	}
	for _, s := range cases {
		want, err := strconv.ParseFloat(s, 64)
		assert.NoError(t, err)
		got, err := parseFloat([]byte(s))
		assert.NoError(t, err)
		assert.Equal(t, math.Float64bits(want), math.Float64bits(got),
			"parseFloat(%q) = %v, strconv gives %v", s, got, want)
	}
}

// TestParseFloatWideRandomized cross-checks a deterministic sample of random
// wide-significand floats against strconv.
func TestParseFloatWideRandomized(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		mantissa := rng.Uint64()>>uint(rng.Intn(11)) | 1<<53 // always > 2^53
		exp := rng.Intn(55) - 27                             // [-27, 27]
		s := strconv.FormatUint(mantissa, 10) + "e" + strconv.Itoa(exp)
		want, err := strconv.ParseFloat(s, 64)
		assert.NoError(t, err)
		got, err := parseFloat([]byte(s))
		assert.NoError(t, err)
		assert.Equal(t, math.Float64bits(want), math.Float64bits(got),
			"parseFloat(%q) = %v, strconv gives %v", s, got, want)
	}
}

func TestWideParseFloatOutOfRange(t *testing.T) {
	// Exponents beyond the exact uint64 powers of five defer to strconv.
	_, ok := wideParseFloat(1<<54, 28)
	assert.False(t, ok)
	_, ok = wideParseFloat(1<<54, -28)
	assert.False(t, ok)
	// In-range sanity, both signs of the exponent and the zero mantissa.
	f, ok := wideParseFloat(0, 5)
	assert.True(t, ok)
	assert.Equal(t, 0.0, f)
	f, ok = wideParseFloat(9007199254740993, 0)
	assert.True(t, ok)
	assert.Equal(t, 9007199254740992.0, f)
}

func TestUnmarshalInlineTableErrors(t *testing.T) {
	cases := []string{
		"a = {x 1}",          // missing '='
		"a = {x =",           // EOF after '='
		"a = {x = 1",         // unterminated
		"a = {, x = 1}",      // leading comma
		"a = {x = 1 y = 2}",  // missing comma
		"a = [1,, 2]",        // double comma
		"a = [1 2]",          // missing comma in array
		"a = [",              // unterminated array
		"a = {x = 1, x = 2}", // duplicate key
	}
	for _, doc := range cases {
		m := map[string]interface{}{}
		err := Unmarshal([]byte(doc), &m)
		assert.Error(t, err, "expected error for %q", doc)
	}
}
