package unstable

import (
	"bytes"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

// testPrintRoundtrip parses input and prints it back, asserting the output matches
// the input exactly. Only works when the input is already in the printer's
// canonical form.
func testPrintRoundtrip(t *testing.T, input string) {
	t.Helper()
	testPrint(t, input, input)
}

// testPrint parses input, prints it, and asserts the output matches expected.
func testPrint(t *testing.T, input, expected string) {
	t.Helper()
	p := &Parser{KeepComments: true}
	p.Reset([]byte(input))

	var buf bytes.Buffer
	printer := NewPrinter(&buf)

	for p.NextExpression() {
		err := printer.Print(p.Expression())
		assert.NoError(t, err)
	}
	assert.NoError(t, p.Error())
	assert.Equal(t, expected, buf.String())
}

// --- Booleans ---

func TestPrint_BoolTrue(t *testing.T) {
	testPrintRoundtrip(t, "x = true\n")
}

func TestPrint_BoolFalse(t *testing.T) {
	testPrintRoundtrip(t, "x = false\n")
}

// --- Integers ---

func TestPrint_IntegerDecimal(t *testing.T) {
	testPrintRoundtrip(t, "x = 42\n")
}

func TestPrint_IntegerZero(t *testing.T) {
	testPrintRoundtrip(t, "x = 0\n")
}

func TestPrint_IntegerPositive(t *testing.T) {
	testPrintRoundtrip(t, "x = +99\n")
}

func TestPrint_IntegerNegative(t *testing.T) {
	testPrintRoundtrip(t, "x = -17\n")
}

func TestPrint_IntegerHexUppercase(t *testing.T) {
	testPrintRoundtrip(t, "x = 0xDEADBEEF\n")
}

func TestPrint_IntegerHexLowercaseUnderscore(t *testing.T) {
	testPrintRoundtrip(t, "x = 0xdead_beef\n")
}

func TestPrint_IntegerOctal(t *testing.T) {
	testPrintRoundtrip(t, "x = 0o755\n")
}

func TestPrint_IntegerBinary(t *testing.T) {
	testPrintRoundtrip(t, "x = 0b11010110\n")
}

func TestPrint_IntegerUnderscore(t *testing.T) {
	testPrintRoundtrip(t, "x = 1_000_000\n")
}

// --- Floats ---

func TestPrint_FloatRegular(t *testing.T) {
	testPrintRoundtrip(t, "x = 3.14\n")
}

func TestPrint_FloatPositive(t *testing.T) {
	testPrintRoundtrip(t, "x = +1.0\n")
}

func TestPrint_FloatNegative(t *testing.T) {
	testPrintRoundtrip(t, "x = -0.01\n")
}

func TestPrint_FloatExponent(t *testing.T) {
	testPrintRoundtrip(t, "x = 5e+22\n")
}

func TestPrint_FloatNegativeExponent(t *testing.T) {
	testPrintRoundtrip(t, "x = 1e-06\n")
}

func TestPrint_FloatCombined(t *testing.T) {
	testPrintRoundtrip(t, "x = 6.626e-34\n")
}

func TestPrint_FloatInf(t *testing.T) {
	testPrintRoundtrip(t, "x = inf\n")
}

func TestPrint_FloatPositiveInf(t *testing.T) {
	testPrintRoundtrip(t, "x = +inf\n")
}

func TestPrint_FloatNegativeInf(t *testing.T) {
	testPrintRoundtrip(t, "x = -inf\n")
}

func TestPrint_FloatNan(t *testing.T) {
	testPrintRoundtrip(t, "x = nan\n")
}

func TestPrint_FloatPositiveNan(t *testing.T) {
	testPrintRoundtrip(t, "x = +nan\n")
}

func TestPrint_FloatNegativeNan(t *testing.T) {
	testPrintRoundtrip(t, "x = -nan\n")
}

func TestPrint_FloatUnderscore(t *testing.T) {
	testPrintRoundtrip(t, "x = 3_141.5927\n")
}

// --- Strings: literal (canonical for strings without ' \r \n or invalid ASCII) ---

func TestPrint_StringLiteral(t *testing.T) {
	testPrintRoundtrip(t, "key = 'hello world'\n")
}

func TestPrint_StringLiteralEmpty(t *testing.T) {
	testPrintRoundtrip(t, "key = ''\n")
}

func TestPrint_StringLiteralUnicode(t *testing.T) {
	// café has no problematic characters, so it stays literal.
	testPrintRoundtrip(t, "key = 'café'\n")
}

func TestPrint_StringLiteralWithDoubleQuote(t *testing.T) {
	// Double quotes are fine in literal strings.
	testPrintRoundtrip(t, "key = 'has\"dquote'\n")
}

func TestPrint_StringLiteralWithTab(t *testing.T) {
	// Tabs are valid in literal strings (0x09 is not invalid ASCII in TOML).
	testPrintRoundtrip(t, "key = 'has\ttab'\n")
}

func TestPrint_StringLiteralWithBackslash(t *testing.T) {
	// Backslash is a regular character in literal strings.
	testPrintRoundtrip(t, "key = 'back\\slash'\n")
}

// --- Strings: basic (canonical for strings with ' \r \n or invalid ASCII) ---

func TestPrint_StringBasicSingleQuote(t *testing.T) {
	testPrintRoundtrip(t, "key = \"it's\"\n")
}

func TestPrint_StringBasicNewline(t *testing.T) {
	testPrintRoundtrip(t, "key = \"line1\\nline2\"\n")
}

func TestPrint_StringBasicCarriageReturn(t *testing.T) {
	testPrintRoundtrip(t, "key = \"has\\rreturn\"\n")
}

func TestPrint_StringBasicEscapedBackslash(t *testing.T) {
	testPrintRoundtrip(t, "key = \"has\\\\both\\n\"\n")
}

func TestPrint_StringBasicEscapedDoubleQuote(t *testing.T) {
	// String contains both ' and ", so basic quoting is needed (due to '),
	// and " must be escaped.
	testPrintRoundtrip(t, "key = \"it's a \\\"quote\\\"\"\n")
}

func TestPrint_StringBasicBackspace(t *testing.T) {
	testPrintRoundtrip(t, "key = \"has\\bbs\"\n")
}

func TestPrint_StringBasicFormFeed(t *testing.T) {
	testPrintRoundtrip(t, "key = \"has\\fff\"\n")
}

func TestPrint_StringBasicControlChar(t *testing.T) {
	// 0x1B (ESC) is an invalid ASCII char, triggers basic quoting with \u escape.
	testPrintRoundtrip(t, "key = \"has\\u001Besc\"\n")
}

// --- Strings: canonical transformations (basic input -> literal output) ---

func TestPrint_StringBasicToLiteral(t *testing.T) {
	// "hello" has no special chars, so canonical form is literal 'hello'.
	testPrint(t, "key = \"hello\"\n", "key = 'hello'\n")
}

func TestPrint_StringBasicBackslashToLiteral(t *testing.T) {
	// "back\\slash" decodes to back\slash, which is fine in a literal string.
	testPrint(t, "key = \"back\\\\slash\"\n", "key = 'back\\slash'\n")
}

func TestPrint_StringBasicTabToLiteral(t *testing.T) {
	// "has\ttab" decodes to has<TAB>tab, which is fine in a literal string.
	testPrint(t, "key = \"has\\ttab\"\n", "key = 'has\ttab'\n")
}

func TestPrint_StringBasicDoubleQuoteToLiteral(t *testing.T) {
	// "has\"dquote" decodes to has"dquote, fine in a literal string.
	testPrint(t, "key = \"has\\\"dquote\"\n", "key = 'has\"dquote'\n")
}

func TestPrint_StringUnicodeEscapeToLiteral(t *testing.T) {
	// "\u00E9" decodes to é, fine in a literal string.
	testPrint(t, "key = \"caf\\u00E9\"\n", "key = 'café'\n")
}

func TestPrint_StringMultilineBasicToSingleLine(t *testing.T) {
	// Multiline basic string decodes to content with newlines,
	// re-encoded as single-line basic string.
	testPrint(t,
		"key = \"\"\"line1\nline2\"\"\"\n",
		"key = \"line1\\nline2\"\n")
}

func TestPrint_StringMultilineLiteralToSingleLine(t *testing.T) {
	// Multiline literal string with newlines becomes basic quoted
	// (because newlines require basic quoting).
	testPrint(t,
		"key = '''line1\nline2'''\n",
		"key = \"line1\\nline2\"\n")
}

func TestPrint_StringMultilineLiteralSimple(t *testing.T) {
	// Multiline literal with no newline in content becomes literal.
	testPrint(t,
		"key = '''\nhello'''\n",
		"key = 'hello'\n")
}

func TestPrint_StringMultilineBasicSimple(t *testing.T) {
	// Multiline basic with no special chars becomes literal.
	testPrint(t,
		"key = \"\"\"\nhello\"\"\"\n",
		"key = 'hello'\n")
}

// --- Local Date ---

func TestPrint_LocalDate(t *testing.T) {
	testPrintRoundtrip(t, "d = 2024-01-15\n")
}

// --- Local Time ---

func TestPrint_LocalTime(t *testing.T) {
	testPrintRoundtrip(t, "t = 14:30:00\n")
}

func TestPrint_LocalTimeFractional(t *testing.T) {
	testPrintRoundtrip(t, "t = 14:30:00.123456\n")
}

// --- Local DateTime ---

func TestPrint_LocalDateTime(t *testing.T) {
	testPrintRoundtrip(t, "dt = 2024-01-15T14:30:00\n")
}

func TestPrint_LocalDateTimeFractional(t *testing.T) {
	testPrintRoundtrip(t, "dt = 2024-01-15T14:30:00.999\n")
}

// --- Offset DateTime ---

func TestPrint_DateTimeUTC(t *testing.T) {
	testPrintRoundtrip(t, "dt = 2024-01-15T14:30:00Z\n")
}

func TestPrint_DateTimePositiveOffset(t *testing.T) {
	testPrintRoundtrip(t, "dt = 2024-01-15T14:30:00+09:00\n")
}

func TestPrint_DateTimeNegativeOffset(t *testing.T) {
	testPrintRoundtrip(t, "dt = 2024-01-15T14:30:00-05:00\n")
}

func TestPrint_DateTimeFractionalWithOffset(t *testing.T) {
	testPrintRoundtrip(t, "dt = 2024-01-15T14:30:00.123+09:00\n")
}

func TestPrint_DateTimeSpaceSeparator(t *testing.T) {
	// TOML allows space instead of T between date and time.
	testPrintRoundtrip(t, "dt = 2024-01-15 14:30:00Z\n")
}

// --- Keys ---

func TestPrint_KeyBare(t *testing.T) {
	testPrintRoundtrip(t, "bare_key-123 = 1\n")
}

func TestPrint_KeyLiteralQuoted(t *testing.T) {
	testPrintRoundtrip(t, "'key with spaces' = 1\n")
}

func TestPrint_KeyBasicQuoted(t *testing.T) {
	// Key containing single quote requires basic quoting.
	testPrintRoundtrip(t, "\"key'quote\" = 1\n")
}

func TestPrint_KeyEmpty(t *testing.T) {
	testPrintRoundtrip(t, "'' = 1\n")
}

func TestPrint_KeyDotted(t *testing.T) {
	testPrintRoundtrip(t, "a.b.c = 1\n")
}

func TestPrint_KeyDottedMixedQuoting(t *testing.T) {
	testPrintRoundtrip(t, "a.'b c'.d = 1\n")
}

// --- Tables ---

func TestPrint_Table(t *testing.T) {
	testPrintRoundtrip(t, "[table]\n")
}

func TestPrint_TableDotted(t *testing.T) {
	testPrintRoundtrip(t, "[a.b.c]\n")
}

func TestPrint_TableQuotedKey(t *testing.T) {
	testPrintRoundtrip(t, "['table with spaces']\n")
}

// --- Array Tables ---

func TestPrint_ArrayTable(t *testing.T) {
	testPrintRoundtrip(t, "[[products]]\n")
}

func TestPrint_ArrayTableDotted(t *testing.T) {
	testPrintRoundtrip(t, "[[a.b.c]]\n")
}

// --- Arrays ---

func TestPrint_ArrayEmpty(t *testing.T) {
	testPrintRoundtrip(t, "arr = []\n")
}

func TestPrint_ArrayIntegers(t *testing.T) {
	testPrintRoundtrip(t, "arr = [1, 2, 3]\n")
}

func TestPrint_ArrayStrings(t *testing.T) {
	testPrintRoundtrip(t, "arr = ['web', 'dev', 'go']\n")
}

func TestPrint_ArrayMixed(t *testing.T) {
	testPrintRoundtrip(t, "arr = [1, 'two', 3.0, true, 2024-01-15]\n")
}

func TestPrint_ArrayNested(t *testing.T) {
	testPrintRoundtrip(t, "arr = [[1, 2], [3, 4]]\n")
}

func TestPrint_ArraySingleElement(t *testing.T) {
	testPrintRoundtrip(t, "arr = [42]\n")
}

// --- Inline Tables ---

func TestPrint_InlineTable(t *testing.T) {
	testPrintRoundtrip(t, "point = {x = 1, y = 2}\n")
}

func TestPrint_InlineTableNested(t *testing.T) {
	testPrintRoundtrip(t, "name = {first = 'Tom', last = 'Doe'}\n")
}

func TestPrint_InlineTableSingleEntry(t *testing.T) {
	testPrintRoundtrip(t, "x = {val = 1}\n")
}

// --- Comments ---

func TestPrint_StandaloneComment(t *testing.T) {
	testPrintRoundtrip(t, "# This is a comment.\n")
}

func TestPrint_TableWithComment(t *testing.T) {
	testPrintRoundtrip(t, "[table] # comment\n")
}

func TestPrint_ArrayTableWithComment(t *testing.T) {
	testPrintRoundtrip(t, "[[products]] # comment\n")
}

func TestPrint_KeyValueWithComment(t *testing.T) {
	testPrintRoundtrip(t, "key = 'value' # comment\n")
}

// --- Multiline Arrays (with comments) ---

func TestPrint_MultilineArray(t *testing.T) {
	input := `key = [ # header comment
  # before first
  1,
  2,
  3,
]
`
	testPrintRoundtrip(t, input)
}

func TestPrint_MultilineArrayWithValueComments(t *testing.T) {
	input := `key = [
  1, # first
  2,
  3, # last
]
`
	testPrintRoundtrip(t, input)
}

// --- Multiline Inline Tables (with comments) ---

func TestPrint_MultilineInlineTable(t *testing.T) {
	input := `point = { # header
  x = 1,
  y = 2,
}
`
	testPrintRoundtrip(t, input)
}

func TestPrint_MultilineInlineTableWithValueComments(t *testing.T) {
	input := `cfg = {
  host = 'localhost', # the host
  port = 8080,
}
`
	testPrintRoundtrip(t, input)
}

// --- Full document roundtrip ---

func TestPrint_FullDocumentRoundtrip(t *testing.T) {
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

	testPrint(t, input, expected)
}

func TestPrint_ComprehensiveTypesRoundtrip(t *testing.T) {
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
	testPrintRoundtrip(t, doc)
}
