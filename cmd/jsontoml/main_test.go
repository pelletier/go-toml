package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	examples := []struct {
		name      string
		input     string
		expected  string
		errors    bool
		useNumber bool
	}{
		{
			name: "valid json",
			input: `
{
  "mytoml": {
    "a": 42
  }
}`,
			expected: `[mytoml]
a = 42.0
`,
		},
		{
			name:      "use json number",
			useNumber: true,
			input: `
{
  "mytoml": {
    "a": 42
  }
}`,
			expected: `[mytoml]
a = 42
`,
		},
		{
			name:   "invalid json",
			input:  `{ foo`,
			errors: true,
		},
	}

	trueValue := true

	for _, e := range examples {
		b := new(bytes.Buffer)
		if e.useNumber {
			useNumber = &trueValue
		}
		err := convert(strings.NewReader(e.input), b)
		if e.errors {
			require.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, e.expected, b.String())
		}
	}
}
