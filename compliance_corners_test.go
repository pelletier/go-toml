package toml_test

import (
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

// This file pins go-toml's behaviour in TOML spec corners that the upstream
// toml-test conformance suite intentionally leaves undefined or under-tests.
// Without these, a future refactor could silently change behaviour while the
// generated toml_testgen_test.go stays green.
//
// It also keeps native regression coverage for the four TOML 1.1.0 grammar
// changes, independent of the generated suite.

// TOML 1.1.0 grammar features that must be accepted.
func TestCorners_TOML11Accepted(t *testing.T) {
	cases := map[string]string{
		// #894 — seconds are optional in time and datetime.
		"local-time-no-seconds":      "x = 13:37\n",
		"offset-datetime-no-seconds": "x = 1979-05-27T07:32Z\n",
		"offset-datetime-no-sec-num": "x = 1979-05-27T07:32-07:00\n",
		"local-datetime-no-seconds":  "x = 1979-05-27T07:32\n",
		// #790 — \e is the escape character (U+001B).
		"esc-e": "x = \"\\e\"\n",
		// #796 — \xHH is a two-digit hex escape for codepoints <= 0xFF.
		"esc-x-lower": "x = \"\\xff\"\n",
		"esc-x-upper": "x = \"\\xFF\"\n",
		// #904 — newlines and trailing commas in inline tables.
		"inline-trailing-comma": "x = {a = 1,}\n",
		"inline-newline":        "x = {\n\ta = 1,\n}\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			var v map[string]interface{}
			assert.NoError(t, toml.Unmarshal([]byte(input), &v))
		})
	}
}

// Gray-area corners where go-toml deliberately rejects input that the
// conformance suite does not pin. These assertions document the intent.
func TestCorners_DeliberateRejections(t *testing.T) {
	cases := map[string]string{
		// Leap seconds (:60) are grammar-permitted but not representable by
		// time.Time without rolling over, so go-toml rejects them. toml-test
		// only pins :61 as invalid, never :60.
		"leap-second-local":  "x = 23:59:60\n",
		"leap-second-offset": "x = 1979-05-27T23:59:60Z\n",

		// Float overflow: 1e400 is rejected rather than decoded to +Inf.
		// TOML 1.1.0 #1058 makes float size implementation-defined, so either
		// behaviour is conformant; this pins go-toml's choice.
		"float-overflow": "x = 1e400\n",

		// #894 — fractional seconds may only appear when seconds are present.
		"fraction-without-seconds": "x = 07:32.5\n",

		// #796 — \xHH requires exactly two hex digits.
		"esc-x-one-digit": "x = \"\\xf\"\n",
		"esc-x-bad-hex":   "x = \"\\xg0\"\n",

		// DEL (0x7F) is excluded from comments (non-eol = %x09 / %x20-7E /
		// non-ascii) and from every string body.
		"del-in-comment":   "x = 1 # a\x7fb\n",
		"del-in-basic-str": "x = \"a\x7fb\"\n",

		// #904 — a leading comma in an inline table is still invalid.
		"inline-leading-comma": "x = {,a = 1}\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			var v map[string]interface{}
			assert.Error(t, toml.Unmarshal([]byte(input), &v))
		})
	}
}

// Multiline basic string quote-counting: content may end with up to two
// quotation marks before the closing delimiter, but no more.
func TestCorners_MultilineQuoteCounting(t *testing.T) {
	t.Run("two-trailing-quotes", func(t *testing.T) {
		var v struct {
			X string `toml:"x"`
		}
		assert.NoError(t, toml.Unmarshal([]byte("x = \"\"\"a\"\"\"\"\"\n"), &v))
		assert.Equal(t, `a""`, v.X)
	})
	t.Run("too-many-trailing-quotes", func(t *testing.T) {
		var v map[string]interface{}
		assert.Error(t, toml.Unmarshal([]byte("x = \"\"\"a\"\"\"\"\"\"\"\n"), &v))
	})
}
