package token

import "testing"

func TestTokenStringer(t *testing.T) {
	var tests = []struct {
		tt     Type
		expect string
	}{
		{Error, "Error"},
		{EOF, "EOF"},
		{Comment, "Comment"},
		{Key, "Key"},
		{String, "String"},
		{Integer, "Integer"},
		{True, "True"},
		{False, "False"},
		{Float, "Float"},
		{Equal, "="},
		{LeftBracket, "["},
		{RightBracket, "]"},
		{LeftCurlyBrace, "{"},
		{RightCurlyBrace, "}"},
		{LeftParen, "("},
		{RightParen, ")"},
		{DoubleLeftBracket, "]]"},
		{DoubleRightBracket, "[["},
		{Date, "Date"},
		{KeyGroup, "KeyGroup"},
		{KeyGroupArray, "KeyGroupArray"},
		{Comma, ","},
		{Colon, ":"},
		{Dollar, "$"},
		{Star, "*"},
		{Question, "?"},
		{Dot, "."},
		{DotDot, ".."},
		{EOL, "EOL"},
		{EOL + 1, "Unknown"},
	}

	for i, test := range tests {
		got := test.tt.String()
		if got != test.expect {
			t.Errorf("[%d] invalid string of token type; got %q, expected %q", i, got, test.expect)
		}
	}
}

func TestTokenString(t *testing.T) {
	var tests = []struct {
		tok    Token
		expect string
	}{
		{Token{Position{1, 1}, EOF, ""}, "EOF"},
		{Token{Position{1, 1}, Error, "Δt"}, "Δt"},
		{Token{Position{1, 1}, String, "bar"}, `"bar"`},
		{Token{Position{1, 1}, String, "123456789012345"}, `"123456789012345"`},
	}

	for i, test := range tests {
		got := test.tok.String()
		if got != test.expect {
			t.Errorf("[%d] invalid of string token; got %q, expected %q", i, got, test.expect)
		}
	}
}
