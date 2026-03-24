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

func TestDecodeError_DuplicateContent(t *testing.T) {
	// This test verifies that when the same content appears multiple times
	// in the document, the error correctly points to the actual location
	// of the error, not the first occurrence of the content.
	//
	// The document has "1__2" on line 1 and "3__4" on line 2.
	// Both have "__" which is invalid, but we want to ensure errors
	// on line 2 report line 2, not line 1.

	doc := `a = 1
b = 3__4`

	var v map[string]int
	err := Unmarshal([]byte(doc), &v)

	var derr *DecodeError
	if !errors.As(err, &derr) {
		t.Fatal("error not in expected format")
	}

	row, col := derr.Position()
	// The error should be on line 2 where "3__4" is
	if row != 2 {
		t.Errorf("expected error on row 2, got row %d", row)
	}
	// Column should point to the "__" part (after "3")
	if col < 5 {
		t.Errorf("expected error at column >= 5, got column %d", col)
	}
}

func TestDecodeError_Position(t *testing.T) {
	// Test that error positions are correctly reported for various error locations
	examples := []struct {
		name        string
		doc         string
		expectedRow int
		minCol      int
	}{
		{
			name:        "error on first line",
			doc:         `a = 1__2`,
			expectedRow: 1,
			minCol:      5,
		},
		{
			name:        "error on second line",
			doc:         "a = 1\nb = 2__3",
			expectedRow: 2,
			minCol:      5,
		},
		{
			name:        "error on third line",
			doc:         "a = 1\nb = 2\nc = 3__4",
			expectedRow: 3,
			minCol:      5,
		},
		{
			name:        "missing equals on last line without trailing newline",
			doc:         "a = 1\nb = 2\nc",
			expectedRow: 3,
			minCol:      1,
		},
	}

	for _, e := range examples {
		t.Run(e.name, func(t *testing.T) {
			var v map[string]int
			err := Unmarshal([]byte(e.doc), &v)

			var derr *DecodeError
			if !errors.As(err, &derr) {
				t.Fatal("error not in expected format")
			}

			row, col := derr.Position()
			assert.Equal(t, e.expectedRow, row)
			if col < e.minCol {
				t.Errorf("expected column >= %d, got %d", e.minCol, col)
			}
		})
	}
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
