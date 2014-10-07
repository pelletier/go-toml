package toml

import (
	"testing"
)

func testQLFlow(t *testing.T, input string, expectedFlow []token) {
	ch := lexQuery(input)
	for idx, expected := range expectedFlow {
		token := <-ch
		if token != expected {
			t.Log("While testing #", idx, ":", input)
			t.Log("compared", token, "to", expected)
			t.Log(token.val, "<->", expected.val)
			t.Log(token.typ, "<->", expected.typ)
			t.Log(token.Line, "<->", expected.Line)
			t.Log(token.Col, "<->", expected.Col)
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
	testQLFlow(t, " .$[]..()?*", []token{
		token{Position{1, 2}, tokenDot, "."},
		token{Position{1, 3}, tokenDollar, "$"},
		token{Position{1, 4}, tokenLeftBracket, "["},
		token{Position{1, 5}, tokenRightBracket, "]"},
		token{Position{1, 6}, tokenDotDot, ".."},
		token{Position{1, 8}, tokenLeftParen, "("},
		token{Position{1, 9}, tokenRightParen, ")"},
		token{Position{1, 10}, tokenQuestion, "?"},
		token{Position{1, 11}, tokenStar, "*"},
		token{Position{1, 12}, tokenEOF, ""},
	})
}

func TestLexString(t *testing.T) {
	testQLFlow(t, "'foo'", []token{
		token{Position{1, 2}, tokenString, "foo"},
		token{Position{1, 6}, tokenEOF, ""},
	})
}

func TestLexDoubleString(t *testing.T) {
	testQLFlow(t, `"bar"`, []token{
		token{Position{1, 2}, tokenString, "bar"},
		token{Position{1, 6}, tokenEOF, ""},
	})
}

func TestLexKey(t *testing.T) {
	testQLFlow(t, "foo", []token{
		token{Position{1, 1}, tokenKey, "foo"},
		token{Position{1, 4}, tokenEOF, ""},
	})
}

func TestLexRecurse(t *testing.T) {
	testQLFlow(t, "$..*", []token{
		token{Position{1, 1}, tokenDollar, "$"},
		token{Position{1, 2}, tokenDotDot, ".."},
		token{Position{1, 4}, tokenStar, "*"},
		token{Position{1, 5}, tokenEOF, ""},
	})
}

func TestLexBracketKey(t *testing.T) {
	testQLFlow(t, "$[foo]", []token{
		token{Position{1, 1}, tokenDollar, "$"},
		token{Position{1, 2}, tokenLeftBracket, "["},
		token{Position{1, 3}, tokenKey, "foo"},
		token{Position{1, 6}, tokenRightBracket, "]"},
		token{Position{1, 7}, tokenEOF, ""},
	})
}

func TestLexSpace(t *testing.T) {
	testQLFlow(t, "foo bar baz", []token{
		token{Position{1, 1}, tokenKey, "foo"},
		token{Position{1, 5}, tokenKey, "bar"},
		token{Position{1, 9}, tokenKey, "baz"},
		token{Position{1, 12}, tokenEOF, ""},
	})
}
