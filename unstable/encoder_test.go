package unstable

import (
	"bytes"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

// testRoundtrip parses input and encodes it back, asserting the output
// matches the input exactly. Only works when the input is already in
// the encoder's canonical form.
func testRoundtrip(t *testing.T, input string) {
	t.Helper()
	testEncode(t, input, input)
}

// testEncode parses input, encodes it, and asserts the output matches expected.
func testEncode(t *testing.T, input, expected string) {
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
	assert.Equal(t, expected, buf.String())
}

// --- Booleans ---

func TestEncoder_BoolTrue(t *testing.T) {
	testRoundtrip(t, "x = true\n")
}

func TestEncoder_BoolFalse(t *testing.T) {
	testRoundtrip(t, "x = false\n")
}

// --- Integers ---

func TestEncoder_IntegerDecimal(t *testing.T) {
	testRoundtrip(t, "x = 42\n")
}

func TestEncoder_IntegerZero(t *testing.T) {
	testRoundtrip(t, "x = 0\n")
}

func TestEncoder_IntegerPositive(t *testing.T) {
	testRoundtrip(t, "x = +99\n")
}

func TestEncoder_IntegerNegative(t *testing.T) {
	testRoundtrip(t, "x = -17\n")
}

func TestEncoder_IntegerHexUppercase(t *testing.T) {
	testRoundtrip(t, "x = 0xDEADBEEF\n")
}

func TestEncoder_IntegerHexLowercaseUnderscore(t *testing.T) {
	testRoundtrip(t, "x = 0xdead_beef\n")
}

func TestEncoder_IntegerOctal(t *testing.T) {
	testRoundtrip(t, "x = 0o755\n")
}

func TestEncoder_IntegerBinary(t *testing.T) {
	testRoundtrip(t, "x = 0b11010110\n")
}

func TestEncoder_IntegerUnderscore(t *testing.T) {
	testRoundtrip(t, "x = 1_000_000\n")
}

// --- Floats ---

func TestEncoder_FloatRegular(t *testing.T) {
	testRoundtrip(t, "x = 3.14\n")
}

func TestEncoder_FloatPositive(t *testing.T) {
	testRoundtrip(t, "x = +1.0\n")
}

func TestEncoder_FloatNegative(t *testing.T) {
	testRoundtrip(t, "x = -0.01\n")
}

func TestEncoder_FloatExponent(t *testing.T) {
	testRoundtrip(t, "x = 5e+22\n")
}

func TestEncoder_FloatNegativeExponent(t *testing.T) {
	testRoundtrip(t, "x = 1e-06\n")
}

func TestEncoder_FloatCombined(t *testing.T) {
	testRoundtrip(t, "x = 6.626e-34\n")
}

func TestEncoder_FloatInf(t *testing.T) {
	testRoundtrip(t, "x = inf\n")
}

func TestEncoder_FloatPositiveInf(t *testing.T) {
	testRoundtrip(t, "x = +inf\n")
}

func TestEncoder_FloatNegativeInf(t *testing.T) {
	testRoundtrip(t, "x = -inf\n")
}

func TestEncoder_FloatNan(t *testing.T) {
	testRoundtrip(t, "x = nan\n")
}

func TestEncoder_FloatPositiveNan(t *testing.T) {
	testRoundtrip(t, "x = +nan\n")
}

func TestEncoder_FloatNegativeNan(t *testing.T) {
	testRoundtrip(t, "x = -nan\n")
}

func TestEncoder_FloatUnderscore(t *testing.T) {
	testRoundtrip(t, "x = 3_141.5927\n")
}

// --- Strings: literal (canonical for strings without ' \r \n or invalid ASCII) ---

func TestEncoder_StringLiteral(t *testing.T) {
	testRoundtrip(t, "key = 'hello world'\n")
}

func TestEncoder_StringLiteralEmpty(t *testing.T) {
	testRoundtrip(t, "key = ''\n")
}

func TestEncoder_StringLiteralUnicode(t *testing.T) {
	// café has no problematic characters, so it stays literal.
	testRoundtrip(t, "key = 'café'\n")
}

func TestEncoder_StringLiteralWithDoubleQuote(t *testing.T) {
	// Double quotes are fine in literal strings.
	testRoundtrip(t, "key = 'has\"dquote'\n")
}

func TestEncoder_StringLiteralWithTab(t *testing.T) {
	// Tabs are valid in literal strings (0x09 is not invalid ASCII in TOML).
	testRoundtrip(t, "key = 'has\ttab'\n")
}

func TestEncoder_StringLiteralWithBackslash(t *testing.T) {
	// Backslash is a regular character in literal strings.
	testRoundtrip(t, "key = 'back\\slash'\n")
}

// --- Strings: basic (canonical for strings with ' \r \n or invalid ASCII) ---

func TestEncoder_StringBasicSingleQuote(t *testing.T) {
	testRoundtrip(t, "key = \"it's\"\n")
}

func TestEncoder_StringBasicNewline(t *testing.T) {
	testRoundtrip(t, "key = \"line1\\nline2\"\n")
}

func TestEncoder_StringBasicCarriageReturn(t *testing.T) {
	testRoundtrip(t, "key = \"has\\rreturn\"\n")
}

func TestEncoder_StringBasicEscapedBackslash(t *testing.T) {
	testRoundtrip(t, "key = \"has\\\\both\\n\"\n")
}

func TestEncoder_StringBasicEscapedDoubleQuote(t *testing.T) {
	// String contains both ' and ", so basic quoting is needed (due to '),
	// and " must be escaped.
	testRoundtrip(t, "key = \"it's a \\\"quote\\\"\"\n")
}

func TestEncoder_StringBasicBackspace(t *testing.T) {
	testRoundtrip(t, "key = \"has\\bbs\"\n")
}

func TestEncoder_StringBasicFormFeed(t *testing.T) {
	testRoundtrip(t, "key = \"has\\fff\"\n")
}

func TestEncoder_StringBasicControlChar(t *testing.T) {
	// 0x1B (ESC) is an invalid ASCII char, triggers basic quoting with \u escape.
	testRoundtrip(t, "key = \"has\\u001Besc\"\n")
}

// --- Strings: canonical transformations (basic input → literal output) ---

func TestEncoder_StringBasicToLiteral(t *testing.T) {
	// "hello" has no special chars, so canonical form is literal 'hello'.
	testEncode(t, "key = \"hello\"\n", "key = 'hello'\n")
}

func TestEncoder_StringBasicBackslashToLiteral(t *testing.T) {
	// "back\\slash" decodes to back\slash, which is fine in a literal string.
	testEncode(t, "key = \"back\\\\slash\"\n", "key = 'back\\slash'\n")
}

func TestEncoder_StringBasicTabToLiteral(t *testing.T) {
	// "has\ttab" decodes to has<TAB>tab, which is fine in a literal string.
	testEncode(t, "key = \"has\\ttab\"\n", "key = 'has\ttab'\n")
}

func TestEncoder_StringBasicDoubleQuoteToLiteral(t *testing.T) {
	// "has\"dquote" decodes to has"dquote, fine in a literal string.
	testEncode(t, "key = \"has\\\"dquote\"\n", "key = 'has\"dquote'\n")
}

func TestEncoder_StringUnicodeEscapeToLiteral(t *testing.T) {
	// "\u00E9" decodes to é, fine in a literal string.
	testEncode(t, "key = \"caf\\u00E9\"\n", "key = 'café'\n")
}

func TestEncoder_StringMultilineBasicToSingleLine(t *testing.T) {
	// Multiline basic string decodes to content with newlines,
	// re-encoded as single-line basic string.
	testEncode(t,
		"key = \"\"\"line1\nline2\"\"\"\n",
		"key = \"line1\\nline2\"\n")
}

func TestEncoder_StringMultilineLiteralToSingleLine(t *testing.T) {
	// Multiline literal string with newlines becomes basic quoted
	// (because newlines require basic quoting).
	testEncode(t,
		"key = '''line1\nline2'''\n",
		"key = \"line1\\nline2\"\n")
}

func TestEncoder_StringMultilineLiteralSimple(t *testing.T) {
	// Multiline literal with no newline in content becomes literal.
	testEncode(t,
		"key = '''\nhello'''\n",
		"key = 'hello'\n")
}

func TestEncoder_StringMultilineBasicSimple(t *testing.T) {
	// Multiline basic with no special chars becomes literal.
	testEncode(t,
		"key = \"\"\"\nhello\"\"\"\n",
		"key = 'hello'\n")
}

// --- Local Date ---

func TestEncoder_LocalDate(t *testing.T) {
	testRoundtrip(t, "d = 2024-01-15\n")
}

// --- Local Time ---

func TestEncoder_LocalTime(t *testing.T) {
	testRoundtrip(t, "t = 14:30:00\n")
}

func TestEncoder_LocalTimeFractional(t *testing.T) {
	testRoundtrip(t, "t = 14:30:00.123456\n")
}

// --- Local DateTime ---

func TestEncoder_LocalDateTime(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00\n")
}

func TestEncoder_LocalDateTimeFractional(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00.999\n")
}

// --- Offset DateTime ---

func TestEncoder_DateTimeUTC(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00Z\n")
}

func TestEncoder_DateTimePositiveOffset(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00+09:00\n")
}

func TestEncoder_DateTimeNegativeOffset(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00-05:00\n")
}

func TestEncoder_DateTimeFractionalWithOffset(t *testing.T) {
	testRoundtrip(t, "dt = 2024-01-15T14:30:00.123+09:00\n")
}

func TestEncoder_DateTimeSpaceSeparator(t *testing.T) {
	// TOML allows space instead of T between date and time.
	testRoundtrip(t, "dt = 2024-01-15 14:30:00Z\n")
}

// --- Keys ---

func TestEncoder_KeyBare(t *testing.T) {
	testRoundtrip(t, "bare_key-123 = 1\n")
}

func TestEncoder_KeyLiteralQuoted(t *testing.T) {
	testRoundtrip(t, "'key with spaces' = 1\n")
}

func TestEncoder_KeyBasicQuoted(t *testing.T) {
	// Key containing single quote requires basic quoting.
	testRoundtrip(t, "\"key'quote\" = 1\n")
}

func TestEncoder_KeyEmpty(t *testing.T) {
	testRoundtrip(t, "'' = 1\n")
}

func TestEncoder_KeyDotted(t *testing.T) {
	testRoundtrip(t, "a.b.c = 1\n")
}

func TestEncoder_KeyDottedMixedQuoting(t *testing.T) {
	testRoundtrip(t, "a.'b c'.d = 1\n")
}

// --- Tables ---

func TestEncoder_Table(t *testing.T) {
	testRoundtrip(t, "[table]\n")
}

func TestEncoder_TableDotted(t *testing.T) {
	testRoundtrip(t, "[a.b.c]\n")
}

func TestEncoder_TableQuotedKey(t *testing.T) {
	testRoundtrip(t, "['table with spaces']\n")
}

// --- Array Tables ---

func TestEncoder_ArrayTable(t *testing.T) {
	testRoundtrip(t, "[[products]]\n")
}

func TestEncoder_ArrayTableDotted(t *testing.T) {
	testRoundtrip(t, "[[a.b.c]]\n")
}

// --- Arrays ---

func TestEncoder_ArrayEmpty(t *testing.T) {
	testRoundtrip(t, "arr = []\n")
}

func TestEncoder_ArrayIntegers(t *testing.T) {
	testRoundtrip(t, "arr = [1, 2, 3]\n")
}

func TestEncoder_ArrayStrings(t *testing.T) {
	testRoundtrip(t, "arr = ['web', 'dev', 'go']\n")
}

func TestEncoder_ArrayMixed(t *testing.T) {
	testRoundtrip(t, "arr = [1, 'two', 3.0, true, 2024-01-15]\n")
}

func TestEncoder_ArrayNested(t *testing.T) {
	testRoundtrip(t, "arr = [[1, 2], [3, 4]]\n")
}

func TestEncoder_ArraySingleElement(t *testing.T) {
	testRoundtrip(t, "arr = [42]\n")
}

// --- Inline Tables ---

func TestEncoder_InlineTable(t *testing.T) {
	testRoundtrip(t, "point = {x = 1, y = 2}\n")
}

func TestEncoder_InlineTableNested(t *testing.T) {
	testRoundtrip(t, "name = {first = 'Tom', last = 'Doe'}\n")
}

func TestEncoder_InlineTableSingleEntry(t *testing.T) {
	testRoundtrip(t, "x = {val = 1}\n")
}

// --- Comments ---

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

// --- Multiline Arrays (with comments) ---

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

// --- Multiline Inline Tables (with comments) ---

func TestEncoder_MultilineInlineTable(t *testing.T) {
	input := `point = { # header
  x = 1,
  y = 2,
}
`
	testRoundtrip(t, input)
}

func TestEncoder_MultilineInlineTableWithValueComments(t *testing.T) {
	input := `cfg = {
  host = 'localhost', # the host
  port = 8080,
}
`
	testRoundtrip(t, input)
}

// --- Full document roundtrip ---

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

	testEncode(t, input, expected)
}

func TestEncoder_ComprehensiveTypesRoundtrip(t *testing.T) {
	// All TOML native types in canonical form, verifying exact roundtrip.
	doc := `# Booleans
bool_true = true
bool_false = false
# Integers: decimal, signed, hex, octal, binary, underscores
int_dec = 42
int_zero = 0
int_pos = +99
int_neg = -17
int_hex = 0xDEADBEEF
int_hex_lower = 0xdead_beef
int_oct = 0o755
int_bin = 0b11010110
int_underscore = 1_000_000
# Floats: regular, signed, exponent, combined, special values
flt = 3.14
flt_pos = +1.0
flt_neg = -0.01
flt_exp = 5e+22
flt_neg_exp = 1e-06
flt_combined = 6.626e-34
flt_underscore = 9_224_617.445_991_228_313
flt_inf = inf
flt_pos_inf = +inf
flt_neg_inf = -inf
flt_nan = nan
flt_pos_nan = +nan
flt_neg_nan = -nan
# Strings: literal
str_simple = 'hello world'
str_empty = ''
str_unicode = 'café'
str_tab = 'has	tab'
str_backslash = 'back\slash'
str_dquote = 'has"dquote'
str_special_chars = 'path/to/file.txt'
# Strings: basic (required when value contains ' \r \n or control chars)
str_squote = "it's"
str_newline = "line1\nline2"
str_cr = "has\rreturn"
str_both_escapes = "has\\both\n"
str_all_escapes = "bs\b ff\f cr\r lf\n tab\t"
str_control = "esc\u001B"
# Local dates
ld = 2024-01-15
# Local times
lt = 14:30:00
lt_frac = 14:30:00.123456
# Local datetimes
ldt = 2024-01-15T14:30:00
ldt_frac = 2024-01-15T14:30:00.999
# Offset datetimes: Z, positive offset, negative offset, fractional
odt_z = 2024-01-15T14:30:00Z
odt_pos = 2024-01-15T14:30:00+09:00
odt_neg = 2024-01-15T14:30:00-05:00
odt_frac = 2024-01-15T14:30:00.123+09:00
odt_space = 2024-01-15 14:30:00Z
# Keys: bare, literal-quoted, basic-quoted, empty, dotted, mixed
bare_key = 1
'key with spaces' = 2
"key'quote" = 3
'' = 4
a.b.c = 5
a.'b c'.d = 6
# Tables
[simple]
[dotted.table.key]
['table with spaces']
# Array tables
[[array-table]]
[[dotted.array.table]]
# Arrays: empty, single, inline, nested, mixed types
empty_arr = []
single_arr = [42]
int_arr = [1, 2, 3]
str_arr = ['web', 'dev', 'go']
nested_arr = [[1, 2], [3, 4]]
mixed_arr = [1, 'two', 3.0, true, 2024-01-15]
# Inline tables: single entry, multiple entries
single_tbl = {val = 1}
point_tbl = {x = 1, y = 2}
name_tbl = {first = 'Tom', last = 'Doe'}
# Multiline array with comments
multiline_arr = [ # array header
  # before first
  1, # first
  # between
  2,
  3, # last
  # after last
]
# Multiline inline table with comments
multiline_tbl = { # table header
  host = 'localhost', # the host
  port = 8080,
}
`
	testRoundtrip(t, doc)
}
