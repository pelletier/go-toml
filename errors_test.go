package toml

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/unstable"
)

//nolint:funlen
func TestDecodeError(t *testing.T) {
	examples := []struct {
		desc     string
		doc      [3]string
		msg      string
		expected string
	}{
		{
			desc: "no context",
			doc:  [3]string{"", "morning", ""},
			msg:  "this is wrong",
			expected: `
1| morning
 | ~~~~~~~ this is wrong`,
		},
		{
			desc: "one line",
			doc:  [3]string{"good ", "morning", " everyone"},
			msg:  "this is wrong",
			expected: `
1| good morning everyone
 |      ~~~~~~~ this is wrong`,
		},
		{
			desc: "exactly 3 lines",
			doc: [3]string{`line1
line2
line3
before `, "highlighted", ` after
post line 1
post line 2
post line 3`},
			msg: "this is wrong",
			expected: `
1| line1
2| line2
3| line3
4| before highlighted after
 |        ~~~~~~~~~~~ this is wrong
5| post line 1
6| post line 2
7| post line 3`,
		},
		{
			desc: "more than 3 lines",
			doc: [3]string{`should not be seen1
should not be seen2
line1
line2
line3
before `, "highlighted", ` after
post line 1
post line 2
post line 3
should not be seen3
should not be seen4`},
			msg: "this is wrong",
			expected: `
3| line1
4| line2
5| line3
6| before highlighted after
 |        ~~~~~~~~~~~ this is wrong
7| post line 1
8| post line 2
9| post line 3`,
		},
		{
			desc: "more than 10 total lines",
			doc: [3]string{`should not be seen 0
should not be seen1
should not be seen2
should not be seen3
line1
line2
line3
before `, "highlighted", ` after
post line 1
post line 2
post line 3
should not be seen3
should not be seen4`},
			msg: "this is wrong",
			expected: `
 5| line1
 6| line2
 7| line3
 8| before highlighted after
  |        ~~~~~~~~~~~ this is wrong
 9| post line 1
10| post line 2
11| post line 3`,
		},
		{
			desc: "last line of more than 10",
			doc: [3]string{`should not be seen
should not be seen
should not be seen
should not be seen
should not be seen
should not be seen
should not be seen
line1
line2
line3
before `, "highlighted", ``},
			msg: "this is wrong",
			expected: `
 8| line1
 9| line2
10| line3
11| before highlighted
  |        ~~~~~~~~~~~ this is wrong
`,
		},
		{
			desc: "handle empty lines in the before/after blocks",
			doc: [3]string{
				`line1

line 2
before `, "highlighted", ` after
line 3

line 4
line 5`,
			},
			expected: `1| line1
2|
3| line 2
4| before highlighted after
 |        ~~~~~~~~~~~
5| line 3
6|
7| line 4`,
		},
		{
			desc: "handle remainder of the error line when there is only one line",
			doc:  [3]string{`P=`, `[`, `#`},
			msg:  "array is incomplete",
			expected: `1| P=[#
 |   ~ array is incomplete`,
		},
	}

	for _, e := range examples {
		e := e
		t.Run(e.desc, func(t *testing.T) {
			b := bytes.Buffer{}
			b.WriteString(e.doc[0])
			start := b.Len()
			b.WriteString(e.doc[1])
			end := b.Len()
			b.WriteString(e.doc[2])
			doc := b.Bytes()
			hl := doc[start:end]

			err := wrapDecodeError(doc, &unstable.ParserError{
				Highlight: hl,
				Message:   e.msg,
			})

			var derr *DecodeError
			if !errors.As(err, &derr) {
				t.Errorf("error not in expected format")

				return
			}

			assert.Equal(t, strings.Trim(e.expected, "\n"), derr.String())
		})
	}
}

func TestDecodeError_Accessors(t *testing.T) {
	e := DecodeError{
		message: "foo",
		line:    1,
		column:  2,
		key:     []string{"one", "two"},
		human:   "bar",
	}
	assert.Equal(t, "toml: foo", e.Error())
	r, c := e.Position()
	assert.Equal(t, 1, r)
	assert.Equal(t, 2, c)
	assert.Equal(t, Key{"one", "two"}, e.Key())
	assert.Equal(t, "bar", e.String())
}

func TestStrictErrorUnwrap(t *testing.T) {
	fo := bytes.NewBufferString(`
Missing = 1
OtherMissing = 1
`)
	var out struct{}
	err := NewDecoder(fo).DisallowUnknownFields().Decode(&out)
	assert.Error(t, err)

	strictErr := &StrictMissingError{}
	assert.True(t, errors.As(err, &strictErr))

	assert.Equal(t, 2, len(strictErr.Unwrap()))
}

//nolint:funlen
func TestDecodeError_Messages(t *testing.T) {
	// Comprehensive error reporting test: verifies that Unmarshal produces
	// correct positions and human-readable error strings for a wide range
	// of parse errors.
	examples := []struct {
		desc string
		doc  string
		row  int
		col  int
		str  string
	}{
		// Invalid key start after various leading content.
		{
			desc: "invalid key after comment",
			doc:  "# comment\n= \"value\"",
			row:  2, col: 1,
			str: "1| # comment\n2| = \"value\"\n | ~ invalid character at start of key: U+003D '='",
		},
		{
			desc: "invalid key after two comments",
			doc:  "# one\n# two\n= \"value\"",
			row:  3, col: 1,
			str: "1| # one\n2| # two\n3| = \"value\"\n | ~ invalid character at start of key: U+003D '='",
		},
		{
			desc: "invalid key after key-value pair",
			doc:  "a = 1\n= 2",
			row:  2, col: 1,
			str: "1| a = 1\n2| = 2\n | ~ invalid character at start of key: U+003D '='",
		},
		{
			desc: "invalid key after blank line",
			doc:  "a = 1\n\n= 2",
			row:  3, col: 1,
			str: "1| a = 1\n2|\n3| = 2\n | ~ invalid character at start of key: U+003D '='",
		},
		{
			desc: "invalid key no context",
			doc:  "= \"value\"",
			row:  1, col: 1,
			str: "1| = \"value\"\n | ~ invalid character at start of key: U+003D '='",
		},
		{
			desc: "invalid key after blank lines",
			doc:  "\n\n= \"val\"",
			row:  3, col: 1,
			str: "3| = \"val\"\n | ~ invalid character at start of key: U+003D '='",
		},

		// Expected newline.
		{
			desc: "expected newline",
			doc:  "a = 1 b = 2",
			row:  1, col: 7,
			str: "1| a = 1 b = 2\n |       ~ expected newline but got U+0062 'b'",
		},

		// Unterminated strings.
		{
			desc: "unterminated basic string",
			doc:  "a = \"hello",
			row:  1, col: 10,
			str: "1| a = \"hello\n |          ~ unterminated basic string",
		},
		{
			desc: "unterminated literal string",
			doc:  "a = 'hello",
			row:  1, col: 10,
			str: "1| a = 'hello\n |          ~ unterminated literal string",
		},
		{
			desc: "unterminated multiline basic string",
			doc:  "a = \"\"\"hello",
			row:  1, col: 12,
			str: "1| a = \"\"\"hello\n |            ~ multiline basic string not terminated by \"\"\"",
		},
		{
			desc: "unterminated multiline literal string",
			doc:  "a = '''hello",
			row:  1, col: 12,
			str: "1| a = '''hello\n |            ~ multiline literal string not terminated by '''",
		},

		// Incomplete containers.
		{
			desc: "incomplete inline table",
			doc:  "a = {b = 1,",
			row:  1, col: 11,
			str: "1| a = {b = 1,\n |           ~ inline table is incomplete",
		},
		{
			desc: "incomplete array",
			doc:  "a = [1, 2,",
			row:  1, col: 10,
			str: "1| a = [1, 2,\n |          ~ array is incomplete",
		},

		// End-of-input errors.
		{
			desc: "expected value eof",
			doc:  "a = ",
			row:  1, col: 4,
			str: "1| a = \n |    ~ expected value, not end of input",
		},
		{
			desc: "missing value second line",
			doc:  "a = 1\nb = ",
			row:  2, col: 4,
			str: "1| a = 1\n2| b = \n |    ~ expected value, not end of input",
		},
		{
			desc: "expected equals after key",
			doc:  "a",
			row:  1, col: 1,
			str: "1| a\n | ~ expected '=' after key",
		},
		{
			desc: "expected equals after key second line",
			doc:  "x = 1\na",
			row:  2, col: 1,
			str: "1| x = 1\n2| a\n | ~ expected '=' after key",
		},

		// Invalid values.
		{
			desc: "invalid number underscore",
			doc:  "a = 1__2",
			row:  1, col: 6,
			str: "1| a = 1__2\n |      ~~ number must have at least one digit between underscores",
		},
		{
			desc: "invalid number underscore second line",
			doc:  "a = 1\nb = 3__4",
			row:  2, col: 6,
			str: "1| a = 1\n2| b = 3__4\n |      ~~ number must have at least one digit between underscores",
		},
		{
			desc: "invalid bool",
			doc:  "a = tru",
			row:  1, col: 5,
			str: "1| a = tru\n |     ~~~ expected keyword \"true\"",
		},
		{
			desc: "newline in basic string",
			doc:  "a = \"hello\nworld\"",
			row:  1, col: 11,
			str: "1| a = \"hello\n |           ~ basic strings cannot have new lines\n2| world\"",
		},
		{
			desc: "array unexpected comma",
			doc:  "a = [,1]",
			row:  1, col: 6,
			str: "1| a = [,1]\n |      ~ expected value but got U+002C ','",
		},

		// Missing equals on last line without trailing newline.
		{
			desc: "missing equals on last line",
			doc:  "a = 1\nb = 2\nc",
			row:  3, col: 1,
			str: "1| a = 1\n2| b = 2\n3| c\n | ~ expected '=' after key",
		},

		// Error position on later lines.
		{
			desc: "error on third line",
			doc:  "a = 1\nb = 2\nc = 3__4",
			row:  3, col: 6,
			str: "1| a = 1\n2| b = 2\n3| c = 3__4\n |      ~~ number must have at least one digit between underscores",
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			var v interface{}
			err := Unmarshal([]byte(e.doc), &v)
			if err == nil {
				t.Fatal("expected an error")
			}

			var derr *DecodeError
			if !errors.As(err, &derr) {
				t.Fatalf("expected *DecodeError, got %T: %v", err, err)
			}

			row, col := derr.Position()
			if row != e.row {
				t.Errorf("row: got %d, want %d", row, e.row)
			}
			if col != e.col {
				t.Errorf("col: got %d, want %d", col, e.col)
			}

			assert.Equal(t, e.str, derr.String())
		})
	}
}

// TestDecodeErrorRedefinition checks that errors raised by the duplicate-key
// tracker are reported as DecodeError, carrying the key path and a position
// pointing at the offending key (see issue #668).
func TestDecodeErrorRedefinition(t *testing.T) {
	examples := []struct {
		desc  string
		doc   string
		msg   string
		key   Key
		row   int
		col   int
		human string
	}{
		{
			desc: "duplicate key",
			doc:  "a = 1\nb = 2\nb = 3\n",
			msg:  "toml: key b is already defined",
			key:  Key{"b"},
			row:  3,
			col:  1,
			human: `
1| a = 1
2| b = 2
3| b = 3
 | ~ key b is already defined`,
		},
		{
			desc: "duplicate dotted key",
			doc:  "foo.bar = 1\nfoo.bar = 2\n",
			msg:  "toml: key bar is already defined",
			key:  Key{"foo", "bar"},
			row:  2,
			col:  1,
			human: `
1| foo.bar = 1
2| foo.bar = 2
 | ~~~~~~~ key bar is already defined`,
		},
		{
			desc: "redefined table",
			doc:  "[a]\nx = 1\n[a]\ny = 2\n",
			msg:  "toml: table a already exists",
			key:  Key{"a"},
			row:  3,
			col:  2,
		},
		{
			desc: "duplicate key in table body",
			doc:  "[a]\nx = 1\nx = 2\n",
			msg:  "toml: key x is already defined",
			key:  Key{"x"},
			row:  3,
			col:  1,
		},
		{
			desc: "redefined nested table",
			doc:  "[a.b]\n[a.b]\n",
			msg:  "toml: table b already exists",
			key:  Key{"a", "b"},
			row:  2,
			col:  2,
		},
		{
			desc: "table over value",
			doc:  "a = 1\n[a]\n",
			msg:  "toml: key a should be a table, not a value",
			key:  Key{"a"},
			row:  2,
			col:  2,
		},
		{
			desc: "array table over table",
			doc:  "[t]\n[[t]]\n",
			msg:  "toml: key t already exists as a table, but should be an array table",
			key:  Key{"t"},
			row:  2,
			col:  3,
		},
		{
			desc: "duplicate key in inline table",
			doc:  "a = { b = 1, b = 2 }\n",
			msg:  "toml: key b is already defined",
			key:  Key{"a"},
			row:  1,
			col:  1,
		},
	}

	for _, e := range examples {
		t.Run(e.desc, func(t *testing.T) {
			m := map[string]interface{}{}
			err := Unmarshal([]byte(e.doc), &m)

			var de *DecodeError
			if !errors.As(err, &de) {
				t.Fatalf("expected *DecodeError, got %T (%v)", err, err)
			}

			assert.Equal(t, e.msg, de.Error())
			assert.Equal(t, e.key, de.Key())

			row, col := de.Position()
			if row != e.row || col != e.col {
				t.Errorf("position = (%d, %d), want (%d, %d)", row, col, e.row, e.col)
			}

			if e.human != "" {
				assert.Equal(t, e.human[1:], de.String())
			}
		})
	}
}

func ExampleDecodeError() {
	doc := `name = 123__456`

	s := map[string]interface{}{}
	err := Unmarshal([]byte(doc), &s)

	fmt.Println(err)

	var derr *DecodeError
	if errors.As(err, &derr) {
		fmt.Println(derr.String())
		row, col := derr.Position()
		fmt.Println("error occurred at row", row, "column", col)
	}
	// Output:
	// toml: number must have at least one digit between underscores
	// 1| name = 123__456
	//  |           ~~ number must have at least one digit between underscores
	// error occurred at row 1 column 11
}

func TestWrapDecodeErrorNil(t *testing.T) {
	assert.True(t, wrapDecodeError([]byte("a = 1"), nil) == nil)
}

func TestSubsliceOffsetPastEndPanics(t *testing.T) {
	data := []byte("0123456789")
	document := data[:5]
	highlight := data[8:10]
	assert.Panics(t, func() {
		_ = subsliceOffset(document, highlight)
	})
}
