package unstable

import (
	"bytes"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

// testFormatRoundtrip parses input and prints it back, asserting the output matches
// the input exactly. Only works when the input is already in the printer's
// canonical form.
func testFormatRoundtrip(t *testing.T, input string) {
	t.Helper()
	testFormat(t, input, input)
}

// testFormat parses input, prints it, and asserts the output matches expected.
func testFormat(t *testing.T, input, expected string) {
	t.Helper()
	p := &Parser{KeepComments: true}
	p.Reset([]byte(input))

	var buf bytes.Buffer
	printer := NewPrinter(&buf)

	for p.NextExpression() {
		err := printer.Format(p.Expression())
		assert.NoError(t, err)
	}
	assert.NoError(t, p.Error())
	assert.Equal(t, expected, buf.String())
}

// --- Booleans ---

func TestFormat_BoolTrue(t *testing.T) {
	testFormatRoundtrip(t, "x = true\n")
}

func TestFormat_BoolFalse(t *testing.T) {
	testFormatRoundtrip(t, "x = false\n")
}

// --- Integers ---

func TestFormat_IntegerDecimal(t *testing.T) {
	testFormatRoundtrip(t, "x = 42\n")
}

func TestFormat_IntegerZero(t *testing.T) {
	testFormatRoundtrip(t, "x = 0\n")
}

func TestFormat_IntegerPositive(t *testing.T) {
	testFormatRoundtrip(t, "x = +99\n")
}

func TestFormat_IntegerNegative(t *testing.T) {
	testFormatRoundtrip(t, "x = -17\n")
}

func TestFormat_IntegerHexUppercase(t *testing.T) {
	testFormatRoundtrip(t, "x = 0xDEADBEEF\n")
}

func TestFormat_IntegerHexLowercaseUnderscore(t *testing.T) {
	testFormatRoundtrip(t, "x = 0xdead_beef\n")
}

func TestFormat_IntegerOctal(t *testing.T) {
	testFormatRoundtrip(t, "x = 0o755\n")
}

func TestFormat_IntegerBinary(t *testing.T) {
	testFormatRoundtrip(t, "x = 0b11010110\n")
}

func TestFormat_IntegerUnderscore(t *testing.T) {
	testFormatRoundtrip(t, "x = 1_000_000\n")
}

// --- Floats ---

func TestFormat_FloatRegular(t *testing.T) {
	testFormatRoundtrip(t, "x = 3.14\n")
}

func TestFormat_FloatPositive(t *testing.T) {
	testFormatRoundtrip(t, "x = +1.0\n")
}

func TestFormat_FloatNegative(t *testing.T) {
	testFormatRoundtrip(t, "x = -0.01\n")
}

func TestFormat_FloatExponent(t *testing.T) {
	testFormatRoundtrip(t, "x = 5e+22\n")
}

func TestFormat_FloatNegativeExponent(t *testing.T) {
	testFormatRoundtrip(t, "x = 1e-06\n")
}

func TestFormat_FloatCombined(t *testing.T) {
	testFormatRoundtrip(t, "x = 6.626e-34\n")
}

func TestFormat_FloatInf(t *testing.T) {
	testFormatRoundtrip(t, "x = inf\n")
}

func TestFormat_FloatPositiveInf(t *testing.T) {
	testFormatRoundtrip(t, "x = +inf\n")
}

func TestFormat_FloatNegativeInf(t *testing.T) {
	testFormatRoundtrip(t, "x = -inf\n")
}

func TestFormat_FloatNan(t *testing.T) {
	testFormatRoundtrip(t, "x = nan\n")
}

func TestFormat_FloatPositiveNan(t *testing.T) {
	testFormatRoundtrip(t, "x = +nan\n")
}

func TestFormat_FloatNegativeNan(t *testing.T) {
	testFormatRoundtrip(t, "x = -nan\n")
}

func TestFormat_FloatUnderscore(t *testing.T) {
	testFormatRoundtrip(t, "x = 3_141.5927\n")
}

// --- Strings: literal (canonical for strings without ' \r \n or invalid ASCII) ---

func TestFormat_StringLiteral(t *testing.T) {
	testFormatRoundtrip(t, "key = 'hello world'\n")
}

func TestFormat_StringLiteralEmpty(t *testing.T) {
	testFormatRoundtrip(t, "key = ''\n")
}

func TestFormat_StringLiteralUnicode(t *testing.T) {
	// café has no problematic characters, so it stays literal.
	testFormatRoundtrip(t, "key = 'café'\n")
}

func TestFormat_StringLiteralWithDoubleQuote(t *testing.T) {
	// Double quotes are fine in literal strings.
	testFormatRoundtrip(t, "key = 'has\"dquote'\n")
}

func TestFormat_StringLiteralWithTab(t *testing.T) {
	// Tabs are valid in literal strings (0x09 is not invalid ASCII in TOML).
	testFormatRoundtrip(t, "key = 'has\ttab'\n")
}

func TestFormat_StringLiteralWithBackslash(t *testing.T) {
	// Backslash is a regular character in literal strings.
	testFormatRoundtrip(t, "key = 'back\\slash'\n")
}

// --- Strings: basic (canonical for strings with ' \r \n or invalid ASCII) ---

func TestFormat_StringBasicSingleQuote(t *testing.T) {
	testFormatRoundtrip(t, "key = \"it's\"\n")
}

func TestFormat_StringBasicNewline(t *testing.T) {
	testFormatRoundtrip(t, "key = \"line1\\nline2\"\n")
}

func TestFormat_StringBasicCarriageReturn(t *testing.T) {
	testFormatRoundtrip(t, "key = \"has\\rreturn\"\n")
}

func TestFormat_StringBasicEscapedBackslash(t *testing.T) {
	testFormatRoundtrip(t, "key = \"has\\\\both\\n\"\n")
}

func TestFormat_StringBasicEscapedDoubleQuote(t *testing.T) {
	// String contains both ' and ", so basic quoting is needed (due to '),
	// and " must be escaped.
	testFormatRoundtrip(t, "key = \"it's a \\\"quote\\\"\"\n")
}

func TestFormat_StringBasicBackspace(t *testing.T) {
	testFormatRoundtrip(t, "key = \"has\\bbs\"\n")
}

func TestFormat_StringBasicFormFeed(t *testing.T) {
	testFormatRoundtrip(t, "key = \"has\\fff\"\n")
}

func TestFormat_StringBasicControlChar(t *testing.T) {
	// 0x1B (ESC) is an invalid ASCII char, triggers basic quoting with \u escape.
	testFormatRoundtrip(t, "key = \"has\\u001Besc\"\n")
}

// --- Strings: canonical transformations (basic input -> literal output) ---

func TestFormat_StringBasicToLiteral(t *testing.T) {
	// "hello" has no special chars, so canonical form is literal 'hello'.
	testFormat(t, "key = \"hello\"\n", "key = 'hello'\n")
}

func TestFormat_StringBasicBackslashToLiteral(t *testing.T) {
	// "back\\slash" decodes to back\slash, which is fine in a literal string.
	testFormat(t, "key = \"back\\\\slash\"\n", "key = 'back\\slash'\n")
}

func TestFormat_StringBasicTabToLiteral(t *testing.T) {
	// "has\ttab" decodes to has<TAB>tab, which is fine in a literal string.
	testFormat(t, "key = \"has\\ttab\"\n", "key = 'has\ttab'\n")
}

func TestFormat_StringBasicDoubleQuoteToLiteral(t *testing.T) {
	// "has\"dquote" decodes to has"dquote, fine in a literal string.
	testFormat(t, "key = \"has\\\"dquote\"\n", "key = 'has\"dquote'\n")
}

func TestFormat_StringUnicodeEscapeToLiteral(t *testing.T) {
	// "\u00E9" decodes to é, fine in a literal string.
	testFormat(t, "key = \"caf\\u00E9\"\n", "key = 'café'\n")
}

func TestFormat_StringMultilineBasicToSingleLine(t *testing.T) {
	// Multiline basic string decodes to content with newlines,
	// re-encoded as single-line basic string.
	testFormat(t,
		"key = \"\"\"line1\nline2\"\"\"\n",
		"key = \"line1\\nline2\"\n")
}

func TestFormat_StringMultilineLiteralToSingleLine(t *testing.T) {
	// Multiline literal string with newlines becomes basic quoted
	// (because newlines require basic quoting).
	testFormat(t,
		"key = '''line1\nline2'''\n",
		"key = \"line1\\nline2\"\n")
}

func TestFormat_StringMultilineLiteralSimple(t *testing.T) {
	// Multiline literal with no newline in content becomes literal.
	testFormat(t,
		"key = '''\nhello'''\n",
		"key = 'hello'\n")
}

func TestFormat_StringMultilineBasicSimple(t *testing.T) {
	// Multiline basic with no special chars becomes literal.
	testFormat(t,
		"key = \"\"\"\nhello\"\"\"\n",
		"key = 'hello'\n")
}

// --- Local Date ---

func TestFormat_LocalDate(t *testing.T) {
	testFormatRoundtrip(t, "d = 2024-01-15\n")
}

// --- Local Time ---

func TestFormat_LocalTime(t *testing.T) {
	testFormatRoundtrip(t, "t = 14:30:00\n")
}

func TestFormat_LocalTimeFractional(t *testing.T) {
	testFormatRoundtrip(t, "t = 14:30:00.123456\n")
}

// --- Local DateTime ---

func TestFormat_LocalDateTime(t *testing.T) {
	testFormatRoundtrip(t, "dt = 2024-01-15T14:30:00\n")
}

func TestFormat_LocalDateTimeFractional(t *testing.T) {
	testFormatRoundtrip(t, "dt = 2024-01-15T14:30:00.999\n")
}

// --- Offset DateTime ---

func TestFormat_DateTimeUTC(t *testing.T) {
	testFormatRoundtrip(t, "dt = 2024-01-15T14:30:00Z\n")
}

func TestFormat_DateTimePositiveOffset(t *testing.T) {
	testFormatRoundtrip(t, "dt = 2024-01-15T14:30:00+09:00\n")
}

func TestFormat_DateTimeNegativeOffset(t *testing.T) {
	testFormatRoundtrip(t, "dt = 2024-01-15T14:30:00-05:00\n")
}

func TestFormat_DateTimeFractionalWithOffset(t *testing.T) {
	testFormatRoundtrip(t, "dt = 2024-01-15T14:30:00.123+09:00\n")
}

func TestFormat_DateTimeSpaceSeparator(t *testing.T) {
	// TOML allows space instead of T between date and time.
	testFormatRoundtrip(t, "dt = 2024-01-15 14:30:00Z\n")
}

// --- Keys ---

func TestFormat_KeyBare(t *testing.T) {
	testFormatRoundtrip(t, "bare_key-123 = 1\n")
}

func TestFormat_KeyLiteralQuoted(t *testing.T) {
	testFormatRoundtrip(t, "'key with spaces' = 1\n")
}

func TestFormat_KeyBasicQuoted(t *testing.T) {
	// Key containing single quote requires basic quoting.
	testFormatRoundtrip(t, "\"key'quote\" = 1\n")
}

func TestFormat_KeyEmpty(t *testing.T) {
	testFormatRoundtrip(t, "'' = 1\n")
}

func TestFormat_KeyDotted(t *testing.T) {
	testFormatRoundtrip(t, "a.b.c = 1\n")
}

func TestFormat_KeyDottedMixedQuoting(t *testing.T) {
	testFormatRoundtrip(t, "a.'b c'.d = 1\n")
}

// --- Tables ---

func TestFormat_Table(t *testing.T) {
	testFormatRoundtrip(t, "[table]\n")
}

func TestFormat_TableDotted(t *testing.T) {
	testFormatRoundtrip(t, "[a.b.c]\n")
}

func TestFormat_TableQuotedKey(t *testing.T) {
	testFormatRoundtrip(t, "['table with spaces']\n")
}

// --- Array Tables ---

func TestFormat_ArrayTable(t *testing.T) {
	testFormatRoundtrip(t, "[[products]]\n")
}

func TestFormat_ArrayTableDotted(t *testing.T) {
	testFormatRoundtrip(t, "[[a.b.c]]\n")
}

// --- Arrays ---

func TestFormat_ArrayEmpty(t *testing.T) {
	testFormatRoundtrip(t, "arr = []\n")
}

func TestFormat_ArrayIntegers(t *testing.T) {
	testFormatRoundtrip(t, "arr = [1, 2, 3]\n")
}

func TestFormat_ArrayStrings(t *testing.T) {
	testFormatRoundtrip(t, "arr = ['web', 'dev', 'go']\n")
}

func TestFormat_ArrayMixed(t *testing.T) {
	testFormatRoundtrip(t, "arr = [1, 'two', 3.0, true, 2024-01-15]\n")
}

func TestFormat_ArrayNested(t *testing.T) {
	testFormatRoundtrip(t, "arr = [[1, 2], [3, 4]]\n")
}

func TestFormat_ArraySingleElement(t *testing.T) {
	testFormatRoundtrip(t, "arr = [42]\n")
}

// --- Inline Tables ---

func TestFormat_InlineTable(t *testing.T) {
	testFormatRoundtrip(t, "point = {x = 1, y = 2}\n")
}

func TestFormat_InlineTableNested(t *testing.T) {
	testFormatRoundtrip(t, "name = {first = 'Tom', last = 'Doe'}\n")
}

func TestFormat_InlineTableSingleEntry(t *testing.T) {
	testFormatRoundtrip(t, "x = {val = 1}\n")
}

// --- Comments ---

func TestFormat_StandaloneComment(t *testing.T) {
	testFormatRoundtrip(t, "# This is a comment.\n")
}

func TestFormat_TableWithComment(t *testing.T) {
	testFormatRoundtrip(t, "[table] # comment\n")
}

func TestFormat_ArrayTableWithComment(t *testing.T) {
	testFormatRoundtrip(t, "[[products]] # comment\n")
}

func TestFormat_KeyValueWithComment(t *testing.T) {
	testFormatRoundtrip(t, "key = 'value' # comment\n")
}

// --- Multiline Arrays (with comments) ---

func TestFormat_MultilineArray(t *testing.T) {
	input := `key = [ # header comment
  # before first
  1,
  2,
  3,
]
`
	testFormatRoundtrip(t, input)
}

func TestFormat_MultilineArrayWithValueComments(t *testing.T) {
	input := `key = [
  1, # first
  2,
  3, # last
]
`
	testFormatRoundtrip(t, input)
}

// --- Multiline Inline Tables (with comments) ---

func TestFormat_MultilineInlineTable(t *testing.T) {
	input := `point = { # header
  x = 1,
  y = 2,
}
`
	testFormatRoundtrip(t, input)
}

func TestFormat_MultilineInlineTableWithValueComments(t *testing.T) {
	input := `cfg = {
  host = 'localhost', # the host
  port = 8080,
}
`
	testFormatRoundtrip(t, input)
}

// --- Full document roundtrip ---

func TestFormat_FullDocumentRoundtrip(t *testing.T) {
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

	testFormat(t, input, expected)
}

func TestFormat_ComprehensiveTypesRoundtrip(t *testing.T) {
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
	testFormatRoundtrip(t, doc)
}
