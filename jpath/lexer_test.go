
package jpath

import (
  . "github.com/pelletier/go-toml"
  "testing"
)

func testFlow(t *testing.T, input string, expectedFlow []token) {
	_, ch := lex(input)
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
	testFlow(t, "@.$[]..()?*", []token{
		token{Position{1, 1}, tokenAtCost, "@"},
		token{Position{1, 2}, tokenDot, "."},
		token{Position{1, 3}, tokenDollar, "$"},
		token{Position{1, 4}, tokenLBracket, "["},
		token{Position{1, 5}, tokenRBracket, "]"},
		token{Position{1, 6}, tokenDotDot, ".."},
		token{Position{1, 8}, tokenLParen, "("},
		token{Position{1, 9}, tokenRParen, ")"},
		token{Position{1, 10}, tokenQuestion, "?"},
		token{Position{1, 11}, tokenStar, "*"},
		token{Position{1, 12}, tokenEOF, ""},
	})
}

func TestLexString(t *testing.T) {
	testFlow(t, "'foo'", []token{
		token{Position{1, 2}, tokenString, "foo"},
		token{Position{1, 6}, tokenEOF, ""},
	})

	testFlow(t, `"bar"`, []token{
		token{Position{1, 2}, tokenString, "bar"},
		token{Position{1, 6}, tokenEOF, ""},
	})
}

func TestLexKey(t *testing.T) {
	testFlow(t, "foo", []token{
		token{Position{1, 1}, tokenKey, "foo"},
		token{Position{1, 4}, tokenEOF, ""},
	})
}
