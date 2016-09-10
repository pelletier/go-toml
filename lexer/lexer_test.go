package lexer

import (
	"strings"
	"testing"

	"github.com/pelletier/go-toml/token"
)

func testFlow(t *testing.T, input string, expectedFlow []token.Token) {
	ch := New(strings.NewReader(input))
	for _, expected := range expectedFlow {
		token := <-ch
		if token != expected {
			t.Log("While testing: ", input)
			t.Log("compared (got)", token, "to (expected)", expected)
			t.Log("\tvalue:", token.Val, "<->", expected.Val)
			t.Log("\tvalue as bytes:", []byte(token.Val), "<->", []byte(expected.Val))
			t.Log("\ttype:", token.Typ.String(), "<->", expected.Typ.String())
			t.Log("\tline:", token.Line, "<->", expected.Line)
			t.Log("\tcolumn:", token.Col, "<->", expected.Col)
			t.Log("compared", token, "to", expected)
			t.FailNow()
		}
	}

	tok, ok := <-ch
	if ok {
		t.Log("channel is not closed!")
		t.Log(len(ch)+1, "tokens remaining:")

		t.Log("token ->", tok)
		for token := range ch {
			t.Log("token ->", token)
		}
		t.FailNow()
	}
}

func TestValidKeyGroup(t *testing.T) {
	testFlow(t, "[hello world]", []token.Token{
		{token.Position{1, 1}, token.LeftBracket, "["},
		{token.Position{1, 2}, token.KeyGroup, "hello world"},
		{token.Position{1, 13}, token.RightBracket, "]"},
		{token.Position{1, 14}, token.EOF, ""},
	})
}

func TestNestedQuotedUnicodeKeyGroup(t *testing.T) {
	testFlow(t, `[ j . "ʞ" . l ]`, []token.Token{
		{token.Position{1, 1}, token.LeftBracket, "["},
		{token.Position{1, 2}, token.KeyGroup, ` j . "ʞ" . l `},
		{token.Position{1, 15}, token.RightBracket, "]"},
		{token.Position{1, 16}, token.EOF, ""},
	})
}

func TestUnclosedKeyGroup(t *testing.T) {
	testFlow(t, "[hello world", []token.Token{
		{token.Position{1, 1}, token.LeftBracket, "["},
		{token.Position{1, 2}, token.Error, "unclosed key group"},
	})
}

func TestComment(t *testing.T) {
	testFlow(t, "# blahblah", []token.Token{
		{token.Position{1, 11}, token.EOF, ""},
	})
}

func TestKeyGroupComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah", []token.Token{
		{token.Position{1, 1}, token.LeftBracket, "["},
		{token.Position{1, 2}, token.KeyGroup, "hello world"},
		{token.Position{1, 13}, token.RightBracket, "]"},
		{token.Position{1, 25}, token.EOF, ""},
	})
}

func TestMultipleKeyGroupsComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah\n[test]", []token.Token{
		{token.Position{1, 1}, token.LeftBracket, "["},
		{token.Position{1, 2}, token.KeyGroup, "hello world"},
		{token.Position{1, 13}, token.RightBracket, "]"},
		{token.Position{2, 1}, token.LeftBracket, "["},
		{token.Position{2, 2}, token.KeyGroup, "test"},
		{token.Position{2, 6}, token.RightBracket, "]"},
		{token.Position{2, 7}, token.EOF, ""},
	})
}

func TestSimpleWindowsCRLF(t *testing.T) {
	testFlow(t, "a=4\r\nb=2", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 2}, token.Equal, "="},
		{token.Position{1, 3}, token.Integer, "4"},
		{token.Position{2, 1}, token.Key, "b"},
		{token.Position{2, 2}, token.Equal, "="},
		{token.Position{2, 3}, token.Integer, "2"},
		{token.Position{2, 4}, token.EOF, ""},
	})
}

func TestBasicKey(t *testing.T) {
	testFlow(t, "hello", []token.Token{
		{token.Position{1, 1}, token.Key, "hello"},
		{token.Position{1, 6}, token.EOF, ""},
	})
}

func TestBasicKeyWithUnderscore(t *testing.T) {
	testFlow(t, "hello_hello", []token.Token{
		{token.Position{1, 1}, token.Key, "hello_hello"},
		{token.Position{1, 12}, token.EOF, ""},
	})
}

func TestBasicKeyWithDash(t *testing.T) {
	testFlow(t, "hello-world", []token.Token{
		{token.Position{1, 1}, token.Key, "hello-world"},
		{token.Position{1, 12}, token.EOF, ""},
	})
}

func TestBasicKeyWithUppercaseMix(t *testing.T) {
	testFlow(t, "helloHELLOHello", []token.Token{
		{token.Position{1, 1}, token.Key, "helloHELLOHello"},
		{token.Position{1, 16}, token.EOF, ""},
	})
}

func TestBasicKeyWithInternationalCharacters(t *testing.T) {
	testFlow(t, "héllÖ", []token.Token{
		{token.Position{1, 1}, token.Key, "héllÖ"},
		{token.Position{1, 6}, token.EOF, ""},
	})
}

func TestBasicKeyAndEqual(t *testing.T) {
	testFlow(t, "hello =", []token.Token{
		{token.Position{1, 1}, token.Key, "hello"},
		{token.Position{1, 7}, token.Equal, "="},
		{token.Position{1, 8}, token.EOF, ""},
	})
}

func TestKeyWithSharpAndEqual(t *testing.T) {
	testFlow(t, "key#name = 5", []token.Token{
		{token.Position{1, 1}, token.Error, "keys cannot contain # character"},
	})
}

func TestKeyWithSymbolsAndEqual(t *testing.T) {
	testFlow(t, "~!@$^&*()_+-`1234567890[]\\|/?><.,;:' = 5", []token.Token{
		{token.Position{1, 1}, token.Error, "keys cannot contain ~ character"},
	})
}

func TestKeyEqualStringEscape(t *testing.T) {
	testFlow(t, `foo = "hello\""`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, "hello\""},
		{token.Position{1, 16}, token.EOF, ""},
	})
}

func TestKeyEqualStringUnfinished(t *testing.T) {
	testFlow(t, `foo = "bar`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "unclosed string"},
	})
}

func TestKeyEqualString(t *testing.T) {
	testFlow(t, `foo = "bar"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, "bar"},
		{token.Position{1, 12}, token.EOF, ""},
	})
}

func TestKeyEqualTrue(t *testing.T) {
	testFlow(t, "foo = true", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.True, "true"},
		{token.Position{1, 11}, token.EOF, ""},
	})
}

func TestKeyEqualFalse(t *testing.T) {
	testFlow(t, "foo = false", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.False, "false"},
		{token.Position{1, 12}, token.EOF, ""},
	})
}

func TestArrayNestedString(t *testing.T) {
	testFlow(t, `a = [ ["hello", "world"] ]`, []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.LeftBracket, "["},
		{token.Position{1, 7}, token.LeftBracket, "["},
		{token.Position{1, 9}, token.String, "hello"},
		{token.Position{1, 15}, token.Comma, ","},
		{token.Position{1, 18}, token.String, "world"},
		{token.Position{1, 24}, token.RightBracket, "]"},
		{token.Position{1, 26}, token.RightBracket, "]"},
		{token.Position{1, 27}, token.EOF, ""},
	})
}

func TestArrayNestedInts(t *testing.T) {
	testFlow(t, "a = [ [42, 21], [10] ]", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.LeftBracket, "["},
		{token.Position{1, 7}, token.LeftBracket, "["},
		{token.Position{1, 8}, token.Integer, "42"},
		{token.Position{1, 10}, token.Comma, ","},
		{token.Position{1, 12}, token.Integer, "21"},
		{token.Position{1, 14}, token.RightBracket, "]"},
		{token.Position{1, 15}, token.Comma, ","},
		{token.Position{1, 17}, token.LeftBracket, "["},
		{token.Position{1, 18}, token.Integer, "10"},
		{token.Position{1, 20}, token.RightBracket, "]"},
		{token.Position{1, 22}, token.RightBracket, "]"},
		{token.Position{1, 23}, token.EOF, ""},
	})
}

func TestArrayInts(t *testing.T) {
	testFlow(t, "a = [ 42, 21, 10, ]", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.LeftBracket, "["},
		{token.Position{1, 7}, token.Integer, "42"},
		{token.Position{1, 9}, token.Comma, ","},
		{token.Position{1, 11}, token.Integer, "21"},
		{token.Position{1, 13}, token.Comma, ","},
		{token.Position{1, 15}, token.Integer, "10"},
		{token.Position{1, 17}, token.Comma, ","},
		{token.Position{1, 19}, token.RightBracket, "]"},
		{token.Position{1, 20}, token.EOF, ""},
	})
}

func TestMultilineArrayComments(t *testing.T) {
	testFlow(t, "a = [1, # wow\n2, # such items\n3, # so array\n]", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.LeftBracket, "["},
		{token.Position{1, 6}, token.Integer, "1"},
		{token.Position{1, 7}, token.Comma, ","},
		{token.Position{2, 1}, token.Integer, "2"},
		{token.Position{2, 2}, token.Comma, ","},
		{token.Position{3, 1}, token.Integer, "3"},
		{token.Position{3, 2}, token.Comma, ","},
		{token.Position{4, 1}, token.RightBracket, "]"},
		{token.Position{4, 2}, token.EOF, ""},
	})
}

func TestKeyEqualArrayBools(t *testing.T) {
	testFlow(t, "foo = [true, false, true]", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.LeftBracket, "["},
		{token.Position{1, 8}, token.True, "true"},
		{token.Position{1, 12}, token.Comma, ","},
		{token.Position{1, 14}, token.False, "false"},
		{token.Position{1, 19}, token.Comma, ","},
		{token.Position{1, 21}, token.True, "true"},
		{token.Position{1, 25}, token.RightBracket, "]"},
		{token.Position{1, 26}, token.EOF, ""},
	})
}

func TestKeyEqualArrayBoolsWithComments(t *testing.T) {
	testFlow(t, "foo = [true, false, true] # YEAH", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.LeftBracket, "["},
		{token.Position{1, 8}, token.True, "true"},
		{token.Position{1, 12}, token.Comma, ","},
		{token.Position{1, 14}, token.False, "false"},
		{token.Position{1, 19}, token.Comma, ","},
		{token.Position{1, 21}, token.True, "true"},
		{token.Position{1, 25}, token.RightBracket, "]"},
		{token.Position{1, 33}, token.EOF, ""},
	})
}

func TestDateRegexp(t *testing.T) {
	if dateRegexp.FindString("1979-05-27T07:32:00Z") == "" {
		t.Error("basic lexing")
	}
	if dateRegexp.FindString("1979-05-27T00:32:00-07:00") == "" {
		t.Error("offset lexing")
	}
	if dateRegexp.FindString("1979-05-27T00:32:00.999999-07:00") == "" {
		t.Error("nano precision lexing")
	}
}

func TestKeyEqualDate(t *testing.T) {
	testFlow(t, "foo = 1979-05-27T07:32:00Z", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Date, "1979-05-27T07:32:00Z"},
		{token.Position{1, 27}, token.EOF, ""},
	})
	testFlow(t, "foo = 1979-05-27T00:32:00-07:00", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Date, "1979-05-27T00:32:00-07:00"},
		{token.Position{1, 32}, token.EOF, ""},
	})
	testFlow(t, "foo = 1979-05-27T00:32:00.999999-07:00", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Date, "1979-05-27T00:32:00.999999-07:00"},
		{token.Position{1, 39}, token.EOF, ""},
	})
}

func TestFloatEndingWithDot(t *testing.T) {
	testFlow(t, "foo = 42.", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Error, "float cannot end with a dot"},
	})
}

func TestFloatWithTwoDots(t *testing.T) {
	testFlow(t, "foo = 4.2.", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Error, "cannot have two dots in one float"},
	})
}

func TestFloatWithExponent1(t *testing.T) {
	testFlow(t, "a = 5e+22", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Float, "5e+22"},
		{token.Position{1, 10}, token.EOF, ""},
	})
}

func TestFloatWithExponent2(t *testing.T) {
	testFlow(t, "a = 5E+22", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Float, "5E+22"},
		{token.Position{1, 10}, token.EOF, ""},
	})
}

func TestFloatWithExponent3(t *testing.T) {
	testFlow(t, "a = -5e+22", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Float, "-5e+22"},
		{token.Position{1, 11}, token.EOF, ""},
	})
}

func TestFloatWithExponent4(t *testing.T) {
	testFlow(t, "a = -5e-22", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Float, "-5e-22"},
		{token.Position{1, 11}, token.EOF, ""},
	})
}

func TestFloatWithExponent5(t *testing.T) {
	testFlow(t, "a = 6.626e-34", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Float, "6.626e-34"},
		{token.Position{1, 14}, token.EOF, ""},
	})
}

func TestInvalidEsquapeSequence(t *testing.T) {
	testFlow(t, `foo = "\x"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "invalid escape sequence: \\x"},
	})
}

func TestNestedArrays(t *testing.T) {
	testFlow(t, "foo = [[[]]]", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.LeftBracket, "["},
		{token.Position{1, 8}, token.LeftBracket, "["},
		{token.Position{1, 9}, token.LeftBracket, "["},
		{token.Position{1, 10}, token.RightBracket, "]"},
		{token.Position{1, 11}, token.RightBracket, "]"},
		{token.Position{1, 12}, token.RightBracket, "]"},
		{token.Position{1, 13}, token.EOF, ""},
	})
}

func TestKeyEqualNumber(t *testing.T) {
	testFlow(t, "foo = 42", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "42"},
		{token.Position{1, 9}, token.EOF, ""},
	})

	testFlow(t, "foo = +42", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "+42"},
		{token.Position{1, 10}, token.EOF, ""},
	})

	testFlow(t, "foo = -42", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "-42"},
		{token.Position{1, 10}, token.EOF, ""},
	})

	testFlow(t, "foo = 4.2", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Float, "4.2"},
		{token.Position{1, 10}, token.EOF, ""},
	})

	testFlow(t, "foo = +4.2", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Float, "+4.2"},
		{token.Position{1, 11}, token.EOF, ""},
	})

	testFlow(t, "foo = -4.2", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Float, "-4.2"},
		{token.Position{1, 11}, token.EOF, ""},
	})

	testFlow(t, "foo = 1_000", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "1_000"},
		{token.Position{1, 12}, token.EOF, ""},
	})

	testFlow(t, "foo = 5_349_221", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "5_349_221"},
		{token.Position{1, 16}, token.EOF, ""},
	})

	testFlow(t, "foo = 1_2_3_4_5", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "1_2_3_4_5"},
		{token.Position{1, 16}, token.EOF, ""},
	})

	testFlow(t, "flt8 = 9_224_617.445_991_228_313", []token.Token{
		{token.Position{1, 1}, token.Key, "flt8"},
		{token.Position{1, 6}, token.Equal, "="},
		{token.Position{1, 8}, token.Float, "9_224_617.445_991_228_313"},
		{token.Position{1, 33}, token.EOF, ""},
	})

	testFlow(t, "foo = +", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Error, "no digit in that number"},
	})
}

func TestMultiline(t *testing.T) {
	testFlow(t, "foo = 42\nbar=21", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 7}, token.Integer, "42"},
		{token.Position{2, 1}, token.Key, "bar"},
		{token.Position{2, 4}, token.Equal, "="},
		{token.Position{2, 5}, token.Integer, "21"},
		{token.Position{2, 7}, token.EOF, ""},
	})
}

func TestKeyEqualStringUnicodeEscape(t *testing.T) {
	testFlow(t, `foo = "hello \u2665"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, "hello ♥"},
		{token.Position{1, 21}, token.EOF, ""},
	})
	testFlow(t, `foo = "hello \U000003B4"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, "hello δ"},
		{token.Position{1, 25}, token.EOF, ""},
	})
	testFlow(t, `foo = "\u2"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "unfinished unicode escape"},
	})
	testFlow(t, `foo = "\U2"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "unfinished unicode escape"},
	})
}

func TestKeyEqualStringNoEscape(t *testing.T) {
	testFlow(t, "foo = \"hello \u0002\"", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "unescaped control character U+0002"},
	})
	testFlow(t, "foo = \"hello \u001F\"", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "unescaped control character U+001F"},
	})
}

func TestLiteralString(t *testing.T) {
	testFlow(t, `foo = 'C:\Users\nodejs\templates'`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, `C:\Users\nodejs\templates`},
		{token.Position{1, 34}, token.EOF, ""},
	})
	testFlow(t, `foo = '\\ServerX\admin$\system32\'`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, `\\ServerX\admin$\system32\`},
		{token.Position{1, 35}, token.EOF, ""},
	})
	testFlow(t, `foo = 'Tom "Dubs" Preston-Werner'`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, `Tom "Dubs" Preston-Werner`},
		{token.Position{1, 34}, token.EOF, ""},
	})
	testFlow(t, `foo = '<\i\c*\s*>'`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, `<\i\c*\s*>`},
		{token.Position{1, 19}, token.EOF, ""},
	})
	testFlow(t, `foo = 'C:\Users\nodejs\unfinis`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.Error, "unclosed string"},
	})
}

func TestMultilineLiteralString(t *testing.T) {
	testFlow(t, `foo = '''hello 'literal' world'''`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 10}, token.String, `hello 'literal' world`},
		{token.Position{1, 34}, token.EOF, ""},
	})

	testFlow(t, "foo = '''\nhello\n'literal'\nworld'''", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{2, 1}, token.String, "hello\n'literal'\nworld"},
		{token.Position{4, 9}, token.EOF, ""},
	})
	testFlow(t, "foo = '''\r\nhello\r\n'literal'\r\nworld'''", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{2, 1}, token.String, "hello\r\n'literal'\r\nworld"},
		{token.Position{4, 9}, token.EOF, ""},
	})
}

func TestMultilineString(t *testing.T) {
	testFlow(t, `foo = """hello "literal" world"""`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 10}, token.String, `hello "literal" world`},
		{token.Position{1, 34}, token.EOF, ""},
	})

	testFlow(t, "foo = \"\"\"\r\nhello\\\r\n\"literal\"\\\nworld\"\"\"", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{2, 1}, token.String, "hello\"literal\"world"},
		{token.Position{4, 9}, token.EOF, ""},
	})

	testFlow(t, "foo = \"\"\"\\\n    \\\n    \\\n    hello\\\nmultiline\\\nworld\"\"\"", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 10}, token.String, "hellomultilineworld"},
		{token.Position{6, 9}, token.EOF, ""},
	})

	testFlow(t, "key2 = \"\"\"\nThe quick brown \\\n\n\n  fox jumps over \\\n    the lazy dog.\"\"\"", []token.Token{
		{token.Position{1, 1}, token.Key, "key2"},
		{token.Position{1, 6}, token.Equal, "="},
		{token.Position{2, 1}, token.String, "The quick brown fox jumps over the lazy dog."},
		{token.Position{6, 21}, token.EOF, ""},
	})

	testFlow(t, "key2 = \"\"\"\\\n       The quick brown \\\n       fox jumps over \\\n       the lazy dog.\\\n       \"\"\"", []token.Token{
		{token.Position{1, 1}, token.Key, "key2"},
		{token.Position{1, 6}, token.Equal, "="},
		{token.Position{1, 11}, token.String, "The quick brown fox jumps over the lazy dog."},
		{token.Position{5, 11}, token.EOF, ""},
	})

	testFlow(t, `key2 = "Roses are red\nViolets are blue"`, []token.Token{
		{token.Position{1, 1}, token.Key, "key2"},
		{token.Position{1, 6}, token.Equal, "="},
		{token.Position{1, 9}, token.String, "Roses are red\nViolets are blue"},
		{token.Position{1, 41}, token.EOF, ""},
	})

	testFlow(t, "key2 = \"\"\"\nRoses are red\nViolets are blue\"\"\"", []token.Token{
		{token.Position{1, 1}, token.Key, "key2"},
		{token.Position{1, 6}, token.Equal, "="},
		{token.Position{2, 1}, token.String, "Roses are red\nViolets are blue"},
		{token.Position{3, 20}, token.EOF, ""},
	})
}

func TestUnicodeString(t *testing.T) {
	testFlow(t, `foo = "hello ♥ world"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, "hello ♥ world"},
		{token.Position{1, 22}, token.EOF, ""},
	})
}
func TestEscapeInString(t *testing.T) {
	testFlow(t, `foo = "\b\f\/"`, []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Equal, "="},
		{token.Position{1, 8}, token.String, "\b\f/"},
		{token.Position{1, 15}, token.EOF, ""},
	})
}

func TestKeyGroupArray(t *testing.T) {
	testFlow(t, "[[foo]]", []token.Token{
		{token.Position{1, 1}, token.DoubleLeftBracket, "[["},
		{token.Position{1, 3}, token.KeyGroupArray, "foo"},
		{token.Position{1, 6}, token.DoubleRightBracket, "]]"},
		{token.Position{1, 8}, token.EOF, ""},
	})
}

func TestQuotedKey(t *testing.T) {
	testFlow(t, "\"a b\" = 42", []token.Token{
		{token.Position{1, 1}, token.Key, "\"a b\""},
		{token.Position{1, 7}, token.Equal, "="},
		{token.Position{1, 9}, token.Integer, "42"},
		{token.Position{1, 11}, token.EOF, ""},
	})
}

func TestKeyNewline(t *testing.T) {
	testFlow(t, "a\n= 4", []token.Token{
		{token.Position{1, 1}, token.Error, "keys cannot contain new lines"},
	})
}

func TestInvalidFloat(t *testing.T) {
	testFlow(t, "a=7e1_", []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 2}, token.Equal, "="},
		{token.Position{1, 3}, token.Float, "7e1_"},
		{token.Position{1, 7}, token.EOF, ""},
	})
}

func TestLexUnknownRvalue(t *testing.T) {
	testFlow(t, `a = !b`, []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Error, "no value can start with !"},
	})

	testFlow(t, `a = \b`, []token.Token{
		{token.Position{1, 1}, token.Key, "a"},
		{token.Position{1, 3}, token.Equal, "="},
		{token.Position{1, 5}, token.Error, `no value can start with \`},
	})
}
