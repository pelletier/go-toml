package toml

import (
	"testing"

	"github.com/pelletier/go-toml/token"
)

func testQLFlow(t *testing.T, input string, expectedFlow []token.Token) {
	ch := lexQuery(input)
	for idx, expected := range expectedFlow {
		token := <-ch
		if token != expected {
			t.Log("While testing #", idx, ":", input)
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

func TestLexSpecialChars(t *testing.T) {
	testQLFlow(t, " .$[]..()?*", []token.Token{
		{token.Position{1, 2}, token.Dot, "."},
		{token.Position{1, 3}, token.Dollar, "$"},
		{token.Position{1, 4}, token.LeftBracket, "["},
		{token.Position{1, 5}, token.RightBracket, "]"},
		{token.Position{1, 6}, token.DotDot, ".."},
		{token.Position{1, 8}, token.LeftParen, "("},
		{token.Position{1, 9}, token.RightParen, ")"},
		{token.Position{1, 10}, token.Question, "?"},
		{token.Position{1, 11}, token.Star, "*"},
		{token.Position{1, 12}, token.EOF, ""},
	})
}

func TestLexString(t *testing.T) {
	testQLFlow(t, "'foo\n'", []token.Token{
		{token.Position{1, 2}, token.String, "foo\n"},
		{token.Position{2, 2}, token.EOF, ""},
	})
}

func TestLexDoubleString(t *testing.T) {
	testQLFlow(t, `"bar"`, []token.Token{
		{token.Position{1, 2}, token.String, "bar"},
		{token.Position{1, 6}, token.EOF, ""},
	})
}

func TestLexStringEscapes(t *testing.T) {
	testQLFlow(t, `"foo \" \' \b \f \/ \t \r \\ \u03A9 \U00012345 \n bar"`, []token.Token{
		{token.Position{1, 2}, token.String, "foo \" ' \b \f / \t \r \\ \u03A9 \U00012345 \n bar"},
		{token.Position{1, 55}, token.EOF, ""},
	})
}

func TestLexStringUnfinishedUnicode4(t *testing.T) {
	testQLFlow(t, `"\u000"`, []token.Token{
		{token.Position{1, 2}, token.Error, "unfinished unicode escape"},
	})
}

func TestLexStringUnfinishedUnicode8(t *testing.T) {
	testQLFlow(t, `"\U0000"`, []token.Token{
		{token.Position{1, 2}, token.Error, "unfinished unicode escape"},
	})
}

func TestLexStringInvalidEscape(t *testing.T) {
	testQLFlow(t, `"\x"`, []token.Token{
		{token.Position{1, 2}, token.Error, "invalid escape sequence: \\x"},
	})
}

func TestLexStringUnfinished(t *testing.T) {
	testQLFlow(t, `"bar`, []token.Token{
		{token.Position{1, 2}, token.Error, "unclosed string"},
	})
}

func TestLexKey(t *testing.T) {
	testQLFlow(t, "foo", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 4}, token.EOF, ""},
	})
}

func TestLexRecurse(t *testing.T) {
	testQLFlow(t, "$..*", []token.Token{
		{token.Position{1, 1}, token.Dollar, "$"},
		{token.Position{1, 2}, token.DotDot, ".."},
		{token.Position{1, 4}, token.Star, "*"},
		{token.Position{1, 5}, token.EOF, ""},
	})
}

func TestLexBracketKey(t *testing.T) {
	testQLFlow(t, "$[foo]", []token.Token{
		{token.Position{1, 1}, token.Dollar, "$"},
		{token.Position{1, 2}, token.LeftBracket, "["},
		{token.Position{1, 3}, token.Key, "foo"},
		{token.Position{1, 6}, token.RightBracket, "]"},
		{token.Position{1, 7}, token.EOF, ""},
	})
}

func TestLexSpace(t *testing.T) {
	testQLFlow(t, "foo bar baz", []token.Token{
		{token.Position{1, 1}, token.Key, "foo"},
		{token.Position{1, 5}, token.Key, "bar"},
		{token.Position{1, 9}, token.Key, "baz"},
		{token.Position{1, 12}, token.EOF, ""},
	})
}

func TestLexInteger(t *testing.T) {
	testQLFlow(t, "100 +200 -300", []token.Token{
		{token.Position{1, 1}, token.Integer, "100"},
		{token.Position{1, 5}, token.Integer, "+200"},
		{token.Position{1, 10}, token.Integer, "-300"},
		{token.Position{1, 14}, token.EOF, ""},
	})
}

func TestLexFloat(t *testing.T) {
	testQLFlow(t, "100.0 +200.0 -300.0", []token.Token{
		{token.Position{1, 1}, token.Float, "100.0"},
		{token.Position{1, 7}, token.Float, "+200.0"},
		{token.Position{1, 14}, token.Float, "-300.0"},
		{token.Position{1, 20}, token.EOF, ""},
	})
}

func TestLexFloatWithMultipleDots(t *testing.T) {
	testQLFlow(t, "4.2.", []token.Token{
		{token.Position{1, 1}, token.Error, "cannot have two dots in one float"},
	})
}

func TestLexFloatLeadingDot(t *testing.T) {
	testQLFlow(t, "+.1", []token.Token{
		{token.Position{1, 1}, token.Error, "cannot start float with a dot"},
	})
}

func TestLexFloatWithTrailingDot(t *testing.T) {
	testQLFlow(t, "42.", []token.Token{
		{token.Position{1, 1}, token.Error, "float cannot end with a dot"},
	})
}

func TestLexNumberWithoutDigit(t *testing.T) {
	testQLFlow(t, "+", []token.Token{
		{token.Position{1, 1}, token.Error, "no digit in that number"},
	})
}

func TestLexUnknown(t *testing.T) {
	testQLFlow(t, "^", []token.Token{
		{token.Position{1, 1}, token.Error, "unexpected char: '94'"},
	})
}
