package toml

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"text/tabwriter"
)

func testFlow(t *testing.T, input string, expectedFlow []token) {
	tokens := lexToml([]byte(input))

	if !reflect.DeepEqual(tokens, expectedFlow) {
		diffFlowsColumnsFatal(t, expectedFlow, tokens)
	}
}

func diffFlowsColumnsFatal(t *testing.T, expectedFlow []token, actualFlow []token) {
	max := len(expectedFlow)
	if len(actualFlow) > max {
		max = len(actualFlow)
	}

	b := &bytes.Buffer{}
	w := tabwriter.NewWriter(b, 0, 0, 1, ' ', tabwriter.Debug)

	fmt.Fprintln(w, "expected\tT\tP\tactual\tT\tP\tdiff")

	for i := 0; i < max; i++ {
		expected := ""
		expectedType := ""
		expectedPos := ""
		if i < len(expectedFlow) {
			expected = fmt.Sprintf("%s", expectedFlow[i])
			expectedType = fmt.Sprintf("%s", expectedFlow[i].typ)
			expectedPos = expectedFlow[i].Position.String()
		}
		actual := ""
		actualType := ""
		actualPos := ""
		if i < len(actualFlow) {
			actual = fmt.Sprintf("%s", actualFlow[i])
			actualType = fmt.Sprintf("%s", actualFlow[i].typ)
			actualPos = actualFlow[i].Position.String()
		}
		different := ""
		if i >= len(expectedFlow) {
			different = "+"
		} else if i >= len(actualFlow) {
			different = "-"
		} else if !reflect.DeepEqual(expectedFlow[i], actualFlow[i]) {
			different = "x"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", expected, expectedType, expectedPos, actual, actualType, actualPos, different)
	}
	w.Flush()
	t.Errorf("Different flows:\n%s", b.String())
}

func TestValidKeyGroup(t *testing.T) {
	testFlow(t, "[hello world]", []token{
		{Position{1, 1}, tokenLeftBracket, "["},
		{Position{1, 2}, tokenKeyGroup, "hello world"},
		{Position{1, 13}, tokenRightBracket, "]"},
		{Position{1, 14}, tokenEOF, ""},
	})
}

func TestNestedQuotedUnicodeKeyGroup(t *testing.T) {
	testFlow(t, `[ j . "ʞ" . l . 'ɯ' ]`, []token{
		{Position{1, 1}, tokenLeftBracket, "["},
		{Position{1, 2}, tokenKeyGroup, ` j . "ʞ" . l . 'ɯ' `},
		{Position{1, 21}, tokenRightBracket, "]"},
		{Position{1, 22}, tokenEOF, ""},
	})
}

func TestNestedQuotedUnicodeKeyAssign(t *testing.T) {
	testFlow(t, ` j . "ʞ" . l . 'ɯ' = 3`, []token{
		{Position{1, 2}, tokenKey, `j . "ʞ" . l . 'ɯ'`},
		{Position{1, 20}, tokenEqual, "="},
		{Position{1, 22}, tokenInteger, "3"},
		{Position{1, 23}, tokenEOF, ""},
	})
}

func TestUnclosedKeyGroup(t *testing.T) {
	testFlow(t, "[hello world", []token{
		{Position{1, 1}, tokenLeftBracket, "["},
		{Position{1, 2}, tokenError, "unclosed table key"},
	})
}

func TestComment(t *testing.T) {
	testFlow(t, "# blahblah", []token{
		{Position{1, 11}, tokenEOF, ""},
	})
}

func TestKeyGroupComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah", []token{
		{Position{1, 1}, tokenLeftBracket, "["},
		{Position{1, 2}, tokenKeyGroup, "hello world"},
		{Position{1, 13}, tokenRightBracket, "]"},
		{Position{1, 25}, tokenEOF, ""},
	})
}

func TestMultipleKeyGroupsComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah\n[test]", []token{
		{Position{1, 1}, tokenLeftBracket, "["},
		{Position{1, 2}, tokenKeyGroup, "hello world"},
		{Position{1, 13}, tokenRightBracket, "]"},
		{Position{2, 1}, tokenLeftBracket, "["},
		{Position{2, 2}, tokenKeyGroup, "test"},
		{Position{2, 6}, tokenRightBracket, "]"},
		{Position{2, 7}, tokenEOF, ""},
	})
}

func TestSimpleWindowsCRLF(t *testing.T) {
	testFlow(t, "a=4\r\nb=2", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 2}, tokenEqual, "="},
		{Position{1, 3}, tokenInteger, "4"},
		{Position{2, 1}, tokenKey, "b"},
		{Position{2, 2}, tokenEqual, "="},
		{Position{2, 3}, tokenInteger, "2"},
		{Position{2, 4}, tokenEOF, ""},
	})
}

func TestBasicKey(t *testing.T) {
	testFlow(t, "hello", []token{
		{Position{1, 1}, tokenKey, "hello"},
		{Position{1, 6}, tokenEOF, ""},
	})
}

func TestBasicKeyWithUnderscore(t *testing.T) {
	testFlow(t, "hello_hello", []token{
		{Position{1, 1}, tokenKey, "hello_hello"},
		{Position{1, 12}, tokenEOF, ""},
	})
}

func TestBasicKeyWithDash(t *testing.T) {
	testFlow(t, "hello-world", []token{
		{Position{1, 1}, tokenKey, "hello-world"},
		{Position{1, 12}, tokenEOF, ""},
	})
}

func TestBasicKeyWithUppercaseMix(t *testing.T) {
	testFlow(t, "helloHELLOHello", []token{
		{Position{1, 1}, tokenKey, "helloHELLOHello"},
		{Position{1, 16}, tokenEOF, ""},
	})
}

func TestBasicKeyWithInternationalCharacters(t *testing.T) {
	testFlow(t, "'héllÖ'", []token{
		{Position{1, 1}, tokenKey, "'héllÖ'"},
		{Position{1, 8}, tokenEOF, ""},
	})
}

func TestBasicKeyAndEqual(t *testing.T) {
	testFlow(t, "hello =", []token{
		{Position{1, 1}, tokenKey, "hello"},
		{Position{1, 7}, tokenEqual, "="},
		{Position{1, 8}, tokenEOF, ""},
	})
}

func TestKeyWithSharpAndEqual(t *testing.T) {
	testFlow(t, "key#name = 5", []token{
		{Position{1, 1}, tokenError, "keys cannot contain # character"},
	})
}

func TestKeyWithSymbolsAndEqual(t *testing.T) {
	testFlow(t, "~!@$^&*()_+-`1234567890[]\\|/?><.,;:' = 5", []token{
		{Position{1, 1}, tokenError, "keys cannot contain ~ character"},
	})
}

func TestKeyEqualStringEscape(t *testing.T) {
	testFlow(t, `foo = "hello\""`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "hello\""},
		{Position{1, 16}, tokenEOF, ""},
	})
}

func TestKeyEqualStringUnfinished(t *testing.T) {
	testFlow(t, `foo = "bar`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "unclosed string"},
	})
}

func TestKeyEqualString(t *testing.T) {
	testFlow(t, `foo = "bar"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "bar"},
		{Position{1, 12}, tokenEOF, ""},
	})
}

func TestKeyEqualTrue(t *testing.T) {
	testFlow(t, "foo = true", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenTrue, "true"},
		{Position{1, 11}, tokenEOF, ""},
	})
}

func TestKeyEqualFalse(t *testing.T) {
	testFlow(t, "foo = false", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenFalse, "false"},
		{Position{1, 12}, tokenEOF, ""},
	})
}

func TestArrayNestedString(t *testing.T) {
	testFlow(t, `a = [ ["hello", "world"] ]`, []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenLeftBracket, "["},
		{Position{1, 7}, tokenLeftBracket, "["},
		{Position{1, 9}, tokenString, "hello"},
		{Position{1, 15}, tokenComma, ","},
		{Position{1, 18}, tokenString, "world"},
		{Position{1, 24}, tokenRightBracket, "]"},
		{Position{1, 26}, tokenRightBracket, "]"},
		{Position{1, 27}, tokenEOF, ""},
	})
}

func TestArrayNestedInts(t *testing.T) {
	testFlow(t, "a = [ [42, 21], [10] ]", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenLeftBracket, "["},
		{Position{1, 7}, tokenLeftBracket, "["},
		{Position{1, 8}, tokenInteger, "42"},
		{Position{1, 10}, tokenComma, ","},
		{Position{1, 12}, tokenInteger, "21"},
		{Position{1, 14}, tokenRightBracket, "]"},
		{Position{1, 15}, tokenComma, ","},
		{Position{1, 17}, tokenLeftBracket, "["},
		{Position{1, 18}, tokenInteger, "10"},
		{Position{1, 20}, tokenRightBracket, "]"},
		{Position{1, 22}, tokenRightBracket, "]"},
		{Position{1, 23}, tokenEOF, ""},
	})
}

func TestArrayInts(t *testing.T) {
	testFlow(t, "a = [ 42, 21, 10, ]", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenLeftBracket, "["},
		{Position{1, 7}, tokenInteger, "42"},
		{Position{1, 9}, tokenComma, ","},
		{Position{1, 11}, tokenInteger, "21"},
		{Position{1, 13}, tokenComma, ","},
		{Position{1, 15}, tokenInteger, "10"},
		{Position{1, 17}, tokenComma, ","},
		{Position{1, 19}, tokenRightBracket, "]"},
		{Position{1, 20}, tokenEOF, ""},
	})
}

func TestMultilineArrayComments(t *testing.T) {
	testFlow(t, "a = [1, # wow\n2, # such items\n3, # so array\n]", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenLeftBracket, "["},
		{Position{1, 6}, tokenInteger, "1"},
		{Position{1, 7}, tokenComma, ","},
		{Position{2, 1}, tokenInteger, "2"},
		{Position{2, 2}, tokenComma, ","},
		{Position{3, 1}, tokenInteger, "3"},
		{Position{3, 2}, tokenComma, ","},
		{Position{4, 1}, tokenRightBracket, "]"},
		{Position{4, 2}, tokenEOF, ""},
	})
}

func TestNestedArraysComment(t *testing.T) {
	toml := `
someArray = [
# does not work
["entry1"]
]`
	testFlow(t, toml, []token{
		{Position{2, 1}, tokenKey, "someArray"},
		{Position{2, 11}, tokenEqual, "="},
		{Position{2, 13}, tokenLeftBracket, "["},
		{Position{4, 1}, tokenLeftBracket, "["},
		{Position{4, 3}, tokenString, "entry1"},
		{Position{4, 10}, tokenRightBracket, "]"},
		{Position{5, 1}, tokenRightBracket, "]"},
		{Position{5, 2}, tokenEOF, ""},
	})
}

func TestKeyEqualArrayBools(t *testing.T) {
	testFlow(t, "foo = [true, false, true]", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftBracket, "["},
		{Position{1, 8}, tokenTrue, "true"},
		{Position{1, 12}, tokenComma, ","},
		{Position{1, 14}, tokenFalse, "false"},
		{Position{1, 19}, tokenComma, ","},
		{Position{1, 21}, tokenTrue, "true"},
		{Position{1, 25}, tokenRightBracket, "]"},
		{Position{1, 26}, tokenEOF, ""},
	})
}

func TestKeyEqualArrayBoolsWithComments(t *testing.T) {
	testFlow(t, "foo = [true, false, true] # YEAH", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftBracket, "["},
		{Position{1, 8}, tokenTrue, "true"},
		{Position{1, 12}, tokenComma, ","},
		{Position{1, 14}, tokenFalse, "false"},
		{Position{1, 19}, tokenComma, ","},
		{Position{1, 21}, tokenTrue, "true"},
		{Position{1, 25}, tokenRightBracket, "]"},
		{Position{1, 33}, tokenEOF, ""},
	})
}

func TestKeyEqualDate(t *testing.T) {
	t.Run("local date time", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T07:32:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "07:32:00"},
			{Position{1, 26}, tokenEOF, ""},
		})
	})

	t.Run("local date time space", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 07:32:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "07:32:00"},
			{Position{1, 26}, tokenEOF, ""},
		})
	})

	t.Run("local date time fraction", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T00:32:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00.999999"},
			{Position{1, 33}, tokenEOF, ""},
		})
	})

	t.Run("local date time fraction space", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:32:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00.999999"},
			{Position{1, 33}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time utc", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T07:32:00Z", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "07:32:00"},
			{Position{1, 26}, tokenTimeOffset, "Z"},
			{Position{1, 27}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time -07:00", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T00:32:00-07:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00"},
			{Position{1, 26}, tokenTimeOffset, "-07:00"},
			{Position{1, 32}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time fractions -07:00", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T00:32:00.999999-07:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00.999999"},
			{Position{1, 33}, tokenTimeOffset, "-07:00"},
			{Position{1, 39}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time space separated utc", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 07:32:00Z", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "07:32:00"},
			{Position{1, 26}, tokenTimeOffset, "Z"},
			{Position{1, 27}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time space separated offset", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:32:00-07:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00"},
			{Position{1, 26}, tokenTimeOffset, "-07:00"},
			{Position{1, 32}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time space separated fraction offset", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:32:00.999999-07:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00.999999"},
			{Position{1, 33}, tokenTimeOffset, "-07:00"},
			{Position{1, 39}, tokenEOF, ""},
		})
	})

	t.Run("local date", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 17}, tokenEOF, ""},
		})
	})

	t.Run("local time", func(t *testing.T) {
		testFlow(t, "foo = 07:32:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalTime, "07:32:00"},
			{Position{1, 15}, tokenEOF, ""},
		})
	})

	t.Run("local time fraction", func(t *testing.T) {
		testFlow(t, "foo = 00:32:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalTime, "00:32:00.999999"},
			{Position{1, 22}, tokenEOF, ""},
		})
	})

	t.Run("local time invalid minute digit", func(t *testing.T) {
		testFlow(t, "foo = 00:3x:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenError, "invalid minute digit in time: x"},
		})
	})

	t.Run("local time invalid minute/second digit", func(t *testing.T) {
		testFlow(t, "foo = 00:30x00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenError, "time minute/second separator should be :, not x"},
		})
	})

	t.Run("local time invalid second digit", func(t *testing.T) {
		testFlow(t, "foo = 00:30:x0.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenError, "invalid second digit in time: x"},
		})
	})

	t.Run("local time invalid second digit", func(t *testing.T) {
		testFlow(t, "foo = 00:30:00.F", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenError, "expected at least one digit in time's fraction, not F"},
		})
	})

	t.Run("local date-time invalid minute digit", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:3x:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenError, "invalid minute digit in time: x"},
		})
	})

	t.Run("local date-time invalid hour digit", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T0x:30:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenError, "invalid hour digit in time: x"},
		})
	})

	t.Run("local date-time invalid hour digit", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27T00x30:00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenError, "time hour/minute separator should be :, not x"},
		})
	})

	t.Run("local date-time invalid minute/second digit", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:30x00.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenError, "time minute/second separator should be :, not x"},
		})
	})

	t.Run("local date-time invalid second digit", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:30:x0.999999", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenError, "invalid second digit in time: x"},
		})
	})

	t.Run("local date-time invalid fraction", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:30:00.F", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenError, "expected at least one digit in time's fraction, not F"},
		})
	})

	t.Run("local date-time invalid month-date separator", func(t *testing.T) {
		testFlow(t, "foo = 1979-05X27 00:30:00.F", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenError, "expected - to separate month of a date, not X"},
		})
	})

	t.Run("local date-time extra whitespace", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27  ", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 19}, tokenEOF, ""},
		})
	})

	t.Run("local date-time extra whitespace", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27     ", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 22}, tokenEOF, ""},
		})
	})

	t.Run("offset date-time space separated offset", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:32:00-0x:00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00"},
			{Position{1, 26}, tokenError, "invalid hour digit in time offset: x"},
		})
	})

	t.Run("offset date-time space separated offset", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:32:00-07x00", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00"},
			{Position{1, 26}, tokenError, "time offset hour/minute separator should be :, not x"},
		})
	})

	t.Run("offset date-time space separated offset", func(t *testing.T) {
		testFlow(t, "foo = 1979-05-27 00:32:00-07:x0", []token{
			{Position{1, 1}, tokenKey, "foo"},
			{Position{1, 5}, tokenEqual, "="},
			{Position{1, 7}, tokenLocalDate, "1979-05-27"},
			{Position{1, 18}, tokenLocalTime, "00:32:00"},
			{Position{1, 26}, tokenError, "invalid minute digit in time offset: x"},
		})
	})
}

func TestFloatEndingWithDot(t *testing.T) {
	testFlow(t, "foo = 42.", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenError, "float cannot end with a dot"},
	})
}

func TestFloatWithTwoDots(t *testing.T) {
	testFlow(t, "foo = 4.2.", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenError, "cannot have two dots in one float"},
	})
}

func TestFloatWithExponent1(t *testing.T) {
	testFlow(t, "a = 5e+22", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenFloat, "5e+22"},
		{Position{1, 10}, tokenEOF, ""},
	})
}

func TestFloatWithExponent2(t *testing.T) {
	testFlow(t, "a = 5E+22", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenFloat, "5E+22"},
		{Position{1, 10}, tokenEOF, ""},
	})
}

func TestFloatWithExponent3(t *testing.T) {
	testFlow(t, "a = -5e+22", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenFloat, "-5e+22"},
		{Position{1, 11}, tokenEOF, ""},
	})
}

func TestFloatWithExponent4(t *testing.T) {
	testFlow(t, "a = -5e-22", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenFloat, "-5e-22"},
		{Position{1, 11}, tokenEOF, ""},
	})
}

func TestFloatWithExponent5(t *testing.T) {
	testFlow(t, "a = 6.626e-34", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenFloat, "6.626e-34"},
		{Position{1, 14}, tokenEOF, ""},
	})
}

func TestInvalidEsquapeSequence(t *testing.T) {
	testFlow(t, `foo = "\x"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "invalid escape sequence: \\x"},
	})
}

func TestNestedArrays(t *testing.T) {
	testFlow(t, "foo = [[[]]]", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftBracket, "["},
		{Position{1, 8}, tokenLeftBracket, "["},
		{Position{1, 9}, tokenLeftBracket, "["},
		{Position{1, 10}, tokenRightBracket, "]"},
		{Position{1, 11}, tokenRightBracket, "]"},
		{Position{1, 12}, tokenRightBracket, "]"},
		{Position{1, 13}, tokenEOF, ""},
	})
}

func TestKeyEqualNumber(t *testing.T) {
	testFlow(t, "foo = 42", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "42"},
		{Position{1, 9}, tokenEOF, ""},
	})

	testFlow(t, "foo = +42", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "+42"},
		{Position{1, 10}, tokenEOF, ""},
	})

	testFlow(t, "foo = -42", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "-42"},
		{Position{1, 10}, tokenEOF, ""},
	})

	testFlow(t, "foo = 4.2", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenFloat, "4.2"},
		{Position{1, 10}, tokenEOF, ""},
	})

	testFlow(t, "foo = +4.2", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenFloat, "+4.2"},
		{Position{1, 11}, tokenEOF, ""},
	})

	testFlow(t, "foo = -4.2", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenFloat, "-4.2"},
		{Position{1, 11}, tokenEOF, ""},
	})

	testFlow(t, "foo = 1_000", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "1_000"},
		{Position{1, 12}, tokenEOF, ""},
	})

	testFlow(t, "foo = 5_349_221", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "5_349_221"},
		{Position{1, 16}, tokenEOF, ""},
	})

	testFlow(t, "foo = 1_2_3_4_5", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "1_2_3_4_5"},
		{Position{1, 16}, tokenEOF, ""},
	})

	testFlow(t, "flt8 = 9_224_617.445_991_228_313", []token{
		{Position{1, 1}, tokenKey, "flt8"},
		{Position{1, 6}, tokenEqual, "="},
		{Position{1, 8}, tokenFloat, "9_224_617.445_991_228_313"},
		{Position{1, 33}, tokenEOF, ""},
	})

	testFlow(t, "foo = +", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenError, "no digit in that number"},
	})
}

func TestMultiline(t *testing.T) {
	testFlow(t, "foo = 42\nbar=21", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenInteger, "42"},
		{Position{2, 1}, tokenKey, "bar"},
		{Position{2, 4}, tokenEqual, "="},
		{Position{2, 5}, tokenInteger, "21"},
		{Position{2, 7}, tokenEOF, ""},
	})
}

func TestKeyEqualStringUnicodeEscape(t *testing.T) {
	testFlow(t, `foo = "hello \u2665"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "hello ♥"},
		{Position{1, 21}, tokenEOF, ""},
	})
	testFlow(t, `foo = "hello \U000003B4"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "hello δ"},
		{Position{1, 25}, tokenEOF, ""},
	})
	testFlow(t, `foo = "\uabcd"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "\uabcd"},
		{Position{1, 15}, tokenEOF, ""},
	})
	testFlow(t, `foo = "\uABCD"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "\uABCD"},
		{Position{1, 15}, tokenEOF, ""},
	})
	testFlow(t, `foo = "\U000bcdef"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "\U000bcdef"},
		{Position{1, 19}, tokenEOF, ""},
	})
	testFlow(t, `foo = "\U000BCDEF"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "\U000BCDEF"},
		{Position{1, 19}, tokenEOF, ""},
	})
	testFlow(t, `foo = "\u2"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "unfinished unicode escape"},
	})
	testFlow(t, `foo = "\U2"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "unfinished unicode escape"},
	})
}

func TestKeyEqualStringNoEscape(t *testing.T) {
	testFlow(t, "foo = \"hello \u0002\"", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "unescaped control character U+0002"},
	})
	testFlow(t, "foo = \"hello \u001F\"", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "unescaped control character U+001F"},
	})
}

func TestLiteralString(t *testing.T) {
	testFlow(t, `foo = 'C:\Users\nodejs\templates'`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, `C:\Users\nodejs\templates`},
		{Position{1, 34}, tokenEOF, ""},
	})
	testFlow(t, `foo = '\\ServerX\admin$\system32\'`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, `\\ServerX\admin$\system32\`},
		{Position{1, 35}, tokenEOF, ""},
	})
	testFlow(t, `foo = 'Tom "Dubs" Preston-Werner'`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, `Tom "Dubs" Preston-Werner`},
		{Position{1, 34}, tokenEOF, ""},
	})
	testFlow(t, `foo = '<\i\c*\s*>'`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, `<\i\c*\s*>`},
		{Position{1, 19}, tokenEOF, ""},
	})
	testFlow(t, `foo = 'C:\Users\nodejs\unfinis`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenError, "unclosed string"},
	})
}

func TestMultilineLiteralString(t *testing.T) {
	testFlow(t, `foo = '''hello 'literal' world'''`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 10}, tokenString, `hello 'literal' world`},
		{Position{1, 34}, tokenEOF, ""},
	})

	testFlow(t, "foo = '''\nhello\n'literal'\nworld'''", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{2, 1}, tokenString, "hello\n'literal'\nworld"},
		{Position{4, 9}, tokenEOF, ""},
	})
	testFlow(t, "foo = '''\r\nhello\r\n'literal'\r\nworld'''", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{2, 1}, tokenString, "hello\r\n'literal'\r\nworld"},
		{Position{4, 9}, tokenEOF, ""},
	})
}

func TestMultilineString(t *testing.T) {
	testFlow(t, `foo = """hello "literal" world"""`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 10}, tokenString, `hello "literal" world`},
		{Position{1, 34}, tokenEOF, ""},
	})

	testFlow(t, "foo = \"\"\"\r\nhello\\\r\n\"literal\"\\\nworld\"\"\"", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{2, 1}, tokenString, "hello\"literal\"world"},
		{Position{4, 9}, tokenEOF, ""},
	})

	testFlow(t, "foo = \"\"\"\\\n    \\\n    \\\n    hello\\\nmultiline\\\nworld\"\"\"", []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 10}, tokenString, "hellomultilineworld"},
		{Position{6, 9}, tokenEOF, ""},
	})

	testFlow(t, `foo = """hello	world"""`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 10}, tokenString, "hello\tworld"},
		{Position{1, 24}, tokenEOF, ""},
	})

	testFlow(t, "key2 = \"\"\"\nThe quick brown \\\n\n\n  fox jumps over \\\n    the lazy dog.\"\"\"", []token{
		{Position{1, 1}, tokenKey, "key2"},
		{Position{1, 6}, tokenEqual, "="},
		{Position{2, 1}, tokenString, "The quick brown fox jumps over the lazy dog."},
		{Position{6, 21}, tokenEOF, ""},
	})

	testFlow(t, "key2 = \"\"\"\\\n       The quick brown \\\n       fox jumps over \\\n       the lazy dog.\\\n       \"\"\"", []token{
		{Position{1, 1}, tokenKey, "key2"},
		{Position{1, 6}, tokenEqual, "="},
		{Position{1, 11}, tokenString, "The quick brown fox jumps over the lazy dog."},
		{Position{5, 11}, tokenEOF, ""},
	})

	testFlow(t, `key2 = "Roses are red\nViolets are blue"`, []token{
		{Position{1, 1}, tokenKey, "key2"},
		{Position{1, 6}, tokenEqual, "="},
		{Position{1, 9}, tokenString, "Roses are red\nViolets are blue"},
		{Position{1, 41}, tokenEOF, ""},
	})

	testFlow(t, "key2 = \"\"\"\nRoses are red\nViolets are blue\"\"\"", []token{
		{Position{1, 1}, tokenKey, "key2"},
		{Position{1, 6}, tokenEqual, "="},
		{Position{2, 1}, tokenString, "Roses are red\nViolets are blue"},
		{Position{3, 20}, tokenEOF, ""},
	})
}

func TestUnicodeString(t *testing.T) {
	testFlow(t, `foo = "hello ♥ world"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "hello ♥ world"},
		{Position{1, 22}, tokenEOF, ""},
	})
}

func TestEscapeInString(t *testing.T) {
	testFlow(t, `foo = "\b\f\/"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "\b\f/"},
		{Position{1, 15}, tokenEOF, ""},
	})
}

func TestTabInString(t *testing.T) {
	testFlow(t, `foo = "hello	world"`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 8}, tokenString, "hello\tworld"},
		{Position{1, 20}, tokenEOF, ""},
	})
}

func TestKeyGroupArray(t *testing.T) {
	testFlow(t, "[[foo]]", []token{
		{Position{1, 1}, tokenDoubleLeftBracket, "[["},
		{Position{1, 3}, tokenKeyGroupArray, "foo"},
		{Position{1, 6}, tokenDoubleRightBracket, "]]"},
		{Position{1, 8}, tokenEOF, ""},
	})
}

func TestQuotedKey(t *testing.T) {
	testFlow(t, "\"a b\" = 42", []token{
		{Position{1, 1}, tokenKey, "\"a b\""},
		{Position{1, 7}, tokenEqual, "="},
		{Position{1, 9}, tokenInteger, "42"},
		{Position{1, 11}, tokenEOF, ""},
	})
}

func TestQuotedKeyTab(t *testing.T) {
	testFlow(t, "\"num\tber\" = 123", []token{
		{Position{1, 1}, tokenKey, "\"num\tber\""},
		{Position{1, 11}, tokenEqual, "="},
		{Position{1, 13}, tokenInteger, "123"},
		{Position{1, 16}, tokenEOF, ""},
	})
}

func TestKeyNewline(t *testing.T) {
	testFlow(t, "a\n= 4", []token{
		{Position{1, 1}, tokenError, "keys cannot contain new lines"},
	})
}

func TestInvalidFloat(t *testing.T) {
	testFlow(t, "a=7e1_", []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 2}, tokenEqual, "="},
		{Position{1, 3}, tokenFloat, "7e1_"},
		{Position{1, 7}, tokenEOF, ""},
	})
}

func TestLexUnknownRvalue(t *testing.T) {
	testFlow(t, `a = !b`, []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenError, "no value can start with !"},
	})

	testFlow(t, `a = \b`, []token{
		{Position{1, 1}, tokenKey, "a"},
		{Position{1, 3}, tokenEqual, "="},
		{Position{1, 5}, tokenError, `no value can start with \`},
	})
}

func TestLexInlineTableEmpty(t *testing.T) {
	testFlow(t, `foo = {}`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 8}, tokenRightCurlyBrace, "}"},
		{Position{1, 9}, tokenEOF, ""},
	})
}

func TestLexInlineTableBareKey(t *testing.T) {
	testFlow(t, `foo = { bar = "baz" }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "bar"},
		{Position{1, 13}, tokenEqual, "="},
		{Position{1, 16}, tokenString, "baz"},
		{Position{1, 21}, tokenRightCurlyBrace, "}"},
		{Position{1, 22}, tokenEOF, ""},
	})
}

func TestLexInlineTableBareKeyDash(t *testing.T) {
	testFlow(t, `foo = { -bar = "baz" }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "-bar"},
		{Position{1, 14}, tokenEqual, "="},
		{Position{1, 17}, tokenString, "baz"},
		{Position{1, 22}, tokenRightCurlyBrace, "}"},
		{Position{1, 23}, tokenEOF, ""},
	})
}

func TestLexInlineTableBareKeyInArray(t *testing.T) {
	testFlow(t, `foo = [{ -bar_ = "baz" }]`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftBracket, "["},
		{Position{1, 8}, tokenLeftCurlyBrace, "{"},
		{Position{1, 10}, tokenKey, "-bar_"},
		{Position{1, 16}, tokenEqual, "="},
		{Position{1, 19}, tokenString, "baz"},
		{Position{1, 24}, tokenRightCurlyBrace, "}"},
		{Position{1, 25}, tokenRightBracket, "]"},
		{Position{1, 26}, tokenEOF, ""},
	})
}

func TestLexInlineTableError1(t *testing.T) {
	testFlow(t, `foo = { 123 = 0 ]`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "123"},
		{Position{1, 13}, tokenEqual, "="},
		{Position{1, 15}, tokenInteger, "0"},
		{Position{1, 17}, tokenRightBracket, "]"},
		{Position{1, 18}, tokenError, "cannot have ']' here"},
	})
}

func TestLexInlineTableError2(t *testing.T) {
	testFlow(t, `foo = { 123 = 0 }}`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "123"},
		{Position{1, 13}, tokenEqual, "="},
		{Position{1, 15}, tokenInteger, "0"},
		{Position{1, 17}, tokenRightCurlyBrace, "}"},
		{Position{1, 18}, tokenRightCurlyBrace, "}"},
		{Position{1, 19}, tokenError, "cannot have '}' here"},
	})
}

func TestLexInlineTableDottedKey1(t *testing.T) {
	testFlow(t, `foo = { a = 0, 123.45abc = 0 }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "a"},
		{Position{1, 11}, tokenEqual, "="},
		{Position{1, 13}, tokenInteger, "0"},
		{Position{1, 14}, tokenComma, ","},
		{Position{1, 16}, tokenKey, "123.45abc"},
		{Position{1, 26}, tokenEqual, "="},
		{Position{1, 28}, tokenInteger, "0"},
		{Position{1, 30}, tokenRightCurlyBrace, "}"},
		{Position{1, 31}, tokenEOF, ""},
	})
}

func TestLexInlineTableDottedKey2(t *testing.T) {
	testFlow(t, `foo = { a = 0, '123'.'45abc' = 0 }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "a"},
		{Position{1, 11}, tokenEqual, "="},
		{Position{1, 13}, tokenInteger, "0"},
		{Position{1, 14}, tokenComma, ","},
		{Position{1, 16}, tokenKey, "'123'.'45abc'"},
		{Position{1, 30}, tokenEqual, "="},
		{Position{1, 32}, tokenInteger, "0"},
		{Position{1, 34}, tokenRightCurlyBrace, "}"},
		{Position{1, 35}, tokenEOF, ""},
	})
}

func TestLexInlineTableDottedKey3(t *testing.T) {
	testFlow(t, `foo = { a = 0, "123"."45ʎǝʞ" = 0 }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "a"},
		{Position{1, 11}, tokenEqual, "="},
		{Position{1, 13}, tokenInteger, "0"},
		{Position{1, 14}, tokenComma, ","},
		{Position{1, 16}, tokenKey, `"123"."45ʎǝʞ"`},
		{Position{1, 30}, tokenEqual, "="},
		{Position{1, 32}, tokenInteger, "0"},
		{Position{1, 34}, tokenRightCurlyBrace, "}"},
		{Position{1, 35}, tokenEOF, ""},
	})
}

func TestLexInlineTableBareKeyWithComma(t *testing.T) {
	testFlow(t, `foo = { -bar1 = "baz", -bar_ = "baz" }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "-bar1"},
		{Position{1, 15}, tokenEqual, "="},
		{Position{1, 18}, tokenString, "baz"},
		{Position{1, 22}, tokenComma, ","},
		{Position{1, 24}, tokenKey, "-bar_"},
		{Position{1, 30}, tokenEqual, "="},
		{Position{1, 33}, tokenString, "baz"},
		{Position{1, 38}, tokenRightCurlyBrace, "}"},
		{Position{1, 39}, tokenEOF, ""},
	})
}

func TestLexInlineTableBareKeyUnderscore(t *testing.T) {
	testFlow(t, `foo = { _bar = "baz" }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "_bar"},
		{Position{1, 14}, tokenEqual, "="},
		{Position{1, 17}, tokenString, "baz"},
		{Position{1, 22}, tokenRightCurlyBrace, "}"},
		{Position{1, 23}, tokenEOF, ""},
	})
}

func TestLexInlineTableQuotedKey(t *testing.T) {
	testFlow(t, `foo = { "bar" = "baz" }`, []token{
		{Position{1, 1}, tokenKey, "foo"},
		{Position{1, 5}, tokenEqual, "="},
		{Position{1, 7}, tokenLeftCurlyBrace, "{"},
		{Position{1, 9}, tokenKey, "\"bar\""},
		{Position{1, 15}, tokenEqual, "="},
		{Position{1, 18}, tokenString, "baz"},
		{Position{1, 23}, tokenRightCurlyBrace, "}"},
		{Position{1, 24}, tokenEOF, ""},
	})
}

func BenchmarkLexer(b *testing.B) {
	sample := `title = "Hugo: A Fast and Flexible Website Generator"
baseurl = "http://gohugo.io/"
MetaDataFormat = "yaml"
pluralizeListTitles = false

[params]
  description = "Documentation of Hugo, a fast and flexible static site generator built with love by spf13, bep and friends in Go"
  author = "Steve Francia (spf13) and friends"
  release = "0.22-DEV"

[[menu.main]]
	name = "Download Hugo"
	pre = "<i class='fa fa-download'></i>"
	url = "https://github.com/spf13/hugo/releases"
	weight = -200
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexToml([]byte(sample))
	}
}
