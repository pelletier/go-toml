package unstable

import (
	"bytes"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func testRoundtrip(t *testing.T, input string) {
	t.Helper()
	p := &Parser{KeepComments: true}
	p.Reset([]byte(input))

	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	for p.NextExpression() {
		err := enc.Encode(p.Expression())
		assert.NoError(t, err)
	}
	assert.NoError(t, p.Error())
	assert.Equal(t, input, buf.String())
}

func TestEncoder_SimpleKeyValue(t *testing.T) {
	testRoundtrip(t, "key = 'value'\n")
}

func TestEncoder_Integer(t *testing.T) {
	testRoundtrip(t, "x = 42\n")
}

func TestEncoder_HexInteger(t *testing.T) {
	testRoundtrip(t, "x = 0xDEADBEEF\n")
}

func TestEncoder_Float(t *testing.T) {
	testRoundtrip(t, "x = 3.14\n")
}

func TestEncoder_FloatInf(t *testing.T) {
	testRoundtrip(t, "x = inf\n")
}

func TestEncoder_Bool(t *testing.T) {
	testRoundtrip(t, "x = true\n")
}

func TestEncoder_LocalDate(t *testing.T) {
	testRoundtrip(t, "d = 2024-01-15\n")
}

func TestEncoder_LocalTime(t *testing.T) {
	testRoundtrip(t, "t = 14:30:00\n")
}

func TestEncoder_LocalDateTime(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00\n")
}

func TestEncoder_DateTime(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00Z\n")
}

func TestEncoder_StringBasic(t *testing.T) {
	testRoundtrip(t, "key = 'hello world'\n")
}

func TestEncoder_StringWithSingleQuote(t *testing.T) {
	// A string containing a single quote needs basic quoting.
	input := `key = "it's"` + "\n"
	testRoundtrip(t, input)
}

func TestEncoder_StringWithNewline(t *testing.T) {
	input := `key = "line1\nline2"` + "\n"
	testRoundtrip(t, input)
}

func TestEncoder_StringWithBackslash(t *testing.T) {
	// Parser decodes "back\\slash" to back\slash in Data.
	// Encoder re-encodes as literal string since \ is valid in literals.
	input := `key = "back\\slash"` + "\n"
	expected := "key = 'back\\slash'\n"

	p := &Parser{KeepComments: true}
	p.Reset([]byte(input))

	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	for p.NextExpression() {
		err := enc.Encode(p.Expression())
		assert.NoError(t, err)
	}
	assert.NoError(t, p.Error())
	assert.Equal(t, expected, buf.String())
}

func TestEncoder_StringEmpty(t *testing.T) {
	testRoundtrip(t, "key = ''\n")
}

func TestEncoder_DottedKey(t *testing.T) {
	testRoundtrip(t, "a.b.c = 1\n")
}

func TestEncoder_QuotedKey(t *testing.T) {
	testRoundtrip(t, "'key with spaces' = 1\n")
}

func TestEncoder_BasicQuotedKey(t *testing.T) {
	input := "\"key'quote\" = 1\n"
	testRoundtrip(t, input)
}

func TestEncoder_EmptyKey(t *testing.T) {
	testRoundtrip(t, "'' = 1\n")
}

func TestEncoder_Table(t *testing.T) {
	testRoundtrip(t, "[table]\n")
}

func TestEncoder_DottedTable(t *testing.T) {
	testRoundtrip(t, "[a.b.c]\n")
}

func TestEncoder_ArrayTable(t *testing.T) {
	testRoundtrip(t, "[[products]]\n")
}

func TestEncoder_SimpleArray(t *testing.T) {
	testRoundtrip(t, "arr = [1, 2, 3]\n")
}

func TestEncoder_NestedArray(t *testing.T) {
	testRoundtrip(t, "arr = [[1, 2], [3, 4]]\n")
}

func TestEncoder_EmptyArray(t *testing.T) {
	testRoundtrip(t, "arr = []\n")
}

func TestEncoder_InlineTable(t *testing.T) {
	testRoundtrip(t, "point = {x = 1, y = 2}\n")
}

func TestEncoder_InlineTableNested(t *testing.T) {
	testRoundtrip(t, "name = {first = 'Tom', last = 'Preston-Werner'}\n")
}

func TestEncoder_StandaloneComment(t *testing.T) {
	testRoundtrip(t, "# This is a comment.\n")
}

func TestEncoder_TableWithComment(t *testing.T) {
	testRoundtrip(t, "[table] # comment\n")
}

func TestEncoder_ArrayTableWithComment(t *testing.T) {
	testRoundtrip(t, "[[products]] # comment\n")
}

func TestEncoder_KeyValueWithComment(t *testing.T) {
	testRoundtrip(t, "key = 'value' # comment\n")
}

func TestEncoder_MultilineArray(t *testing.T) {
	input := `key = [ # header comment
  # before first
  1,
  2,
  3,
]
`
	testRoundtrip(t, input)
}

func TestEncoder_MultilineArrayWithValueComments(t *testing.T) {
	input := `key = [
  1, # first
  2,
  3, # last
]
`
	testRoundtrip(t, input)
}

func TestEncoder_MultilineInlineTable(t *testing.T) {
	input := `point = { # header
  x = 1,
  y = 2,
}
`
	testRoundtrip(t, input)
}

func TestEncoder_FullDocumentRoundtrip(t *testing.T) {
	// Blank lines between expressions are not preserved in the AST,
	// so the canonical output omits them. This is expected behavior.
	input := `# Top of the document comment.
# Optional, any amount of lines.

# Above table.
[table] # Next to table.
# Above simple value.
key = 'value' # Next to simple value.
# Below simple value.

# Some comment alone.

# Multiple comments, on multiple lines.

# Above inline table.
name = { first = "Tom", last = "Preston-Werner" } # Next to inline table.
# Below inline table.

# Above array.
array = [ 1, 2, 3 ] # Next to one-line array.
# Below array.

# Above multi-line array.
key5 = [ # Next to start of inline array.
  # Second line before array content.
  1, # Next to first element.
  # After first element.
  # Before second element.
  2,
  3, # Next to last element
  # After last element.
] # Next to end of array.
# Below multi-line array.

# Before array table.
[[products]] # Next to array table.
# After array table.
`

	expected := `# Top of the document comment.
# Optional, any amount of lines.
# Above table.
[table] # Next to table.
# Above simple value.
key = 'value' # Next to simple value.
# Below simple value.
# Some comment alone.
# Multiple comments, on multiple lines.
# Above inline table.
name = {first = 'Tom', last = 'Preston-Werner'} # Next to inline table.
# Below inline table.
# Above array.
array = [1, 2, 3] # Next to one-line array.
# Below array.
# Above multi-line array.
key5 = [ # Next to start of inline array.
  # Second line before array content.
  1, # Next to first element.
  # After first element.
  # Before second element.
  2,
  3, # Next to last element
  # After last element.
] # Next to end of array.
# Below multi-line array.
# Before array table.
[[products]] # Next to array table.
# After array table.
`

	p := &Parser{KeepComments: true}
	p.Reset([]byte(input))

	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	for p.NextExpression() {
		err := enc.Encode(p.Expression())
		assert.NoError(t, err)
	}
	assert.NoError(t, p.Error())
	assert.Equal(t, expected, buf.String())
}

func TestEncoder_MultipleExpressions(t *testing.T) {
	doc := `[server]
host = 'localhost'
port = 8080
`
	testRoundtrip(t, doc)
}

func TestEncoder_ArrayOfStrings(t *testing.T) {
	testRoundtrip(t, "tags = ['web', 'dev', 'go']\n")
}

func TestEncoder_MixedArrayValues(t *testing.T) {
	testRoundtrip(t, "mixed = [1, 'two', 3.0, true]\n")
}
