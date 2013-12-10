package toml

import "testing"

func testFlow(t *testing.T, input string, expectedFlow []token) {
	_, ch := lex(input)
	for _, expected := range expectedFlow {
		token := <-ch
		if token != expected {
			t.Log("compared", token, "to", expected)
			t.Log(token.val, "<->", expected.val)
			t.Log(token.typ, "<->", expected.typ)
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
	testFlow(t, "[hello world]", []token{
		token{tokenLeftBracket, "["},
		token{tokenKeyGroup, "hello world"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestUnclosedKeyGroup(t *testing.T) {
	testFlow(t, "[hello world", []token{
		token{tokenLeftBracket, "["},
		token{tokenError, "unclosed key group"},
	})
}

func TestComment(t *testing.T) {
	testFlow(t, "# blahblah", []token{
		token{tokenEOF, ""},
	})
}

func TestKeyGroupComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah", []token{
		token{tokenLeftBracket, "["},
		token{tokenKeyGroup, "hello world"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestMultipleKeyGroupsComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah\n[test]", []token{
		token{tokenLeftBracket, "["},
		token{tokenKeyGroup, "hello world"},
		token{tokenRightBracket, "]"},
		token{tokenLeftBracket, "["},
		token{tokenKeyGroup, "test"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestBasicKey(t *testing.T) {
	testFlow(t, "hello", []token{
		token{tokenKey, "hello"},
		token{tokenEOF, ""},
	})
}

func TestBasicKeyWithUnderscore(t *testing.T) {
	testFlow(t, "hello_hello", []token{
		token{tokenKey, "hello_hello"},
		token{tokenEOF, ""},
	})
}

func TestBasicKeyWithDash(t *testing.T) {
	testFlow(t, "hello-world", []token{
		token{tokenKey, "hello-world"},
		token{tokenEOF, ""},
	})
}

func TestBasicKeyWithUppercaseMix(t *testing.T) {
	testFlow(t, "helloHELLOHello", []token{
		token{tokenKey, "helloHELLOHello"},
		token{tokenEOF, ""},
	})
}

func TestBasicKeyWithInternationalCharacters(t *testing.T) {
	testFlow(t, "héllÖ", []token{
		token{tokenKey, "héllÖ"},
		token{tokenEOF, ""},
	})
}

func TestBasicKeyAndEqual(t *testing.T) {
	testFlow(t, "hello =", []token{
		token{tokenKey, "hello"},
		token{tokenEqual, "="},
		token{tokenEOF, ""},
	})
}

func TestKeyWithSharpAndEqual(t *testing.T) {
	testFlow(t, "key#name = 5", []token{
		token{tokenKey, "key#name"},
		token{tokenEqual, "="},
		token{tokenInteger, "5"},
		token{tokenEOF, ""},
	})
}
func TestKeyWithSymbolsAndEqual(t *testing.T) {
	testFlow(t, "~!@#$^&*()_+-`1234567890[]\\|/?><.,;:' = 5", []token{
		token{tokenKey, "~!@#$^&*()_+-`1234567890[]\\|/?><.,;:'"},
		token{tokenEqual, "="},
		token{tokenInteger, "5"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualStringEscape(t *testing.T) {
	testFlow(t, "foo = \"hello\\\"\"", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenString, "hello\""},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualStringUnfinished(t *testing.T) {
	testFlow(t, "foo = \"bar", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenError, "unclosed string"},
	})
}

func TestKeyEqualString(t *testing.T) {
	testFlow(t, "foo = \"bar\"", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenString, "bar"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualTrue(t *testing.T) {
	testFlow(t, "foo = true", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenTrue, "true"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualFalse(t *testing.T) {
	testFlow(t, "foo = false", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenFalse, "false"},
		token{tokenEOF, ""},
	})
}

func TestArrayNestedString(t *testing.T) {
	testFlow(t, "a = [ [\"hello\", \"world\"] ]", []token{
		token{tokenKey, "a"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenLeftBracket, "["},
		token{tokenString, "hello"},
		token{tokenComma, ","},
		token{tokenString, "world"},
		token{tokenRightBracket, "]"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestArrayNestedInts(t *testing.T) {
	testFlow(t, "a = [ [42, 21], [10] ]", []token{
		token{tokenKey, "a"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenLeftBracket, "["},
		token{tokenInteger, "42"},
		token{tokenComma, ","},
		token{tokenInteger, "21"},
		token{tokenRightBracket, "]"},
		token{tokenComma, ","},
		token{tokenLeftBracket, "["},
		token{tokenInteger, "10"},
		token{tokenRightBracket, "]"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestArrayInts(t *testing.T) {
	testFlow(t, "a = [ 42, 21, 10, ]", []token{
		token{tokenKey, "a"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenInteger, "42"},
		token{tokenComma, ","},
		token{tokenInteger, "21"},
		token{tokenComma, ","},
		token{tokenInteger, "10"},
		token{tokenComma, ","},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestMultilineArrayComments(t *testing.T) {
	testFlow(t, "a = [1, # wow\n2, # such items\n3, # so array\n]", []token{
		token{tokenKey, "a"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenInteger, "1"},
		token{tokenComma, ","},
		token{tokenInteger, "2"},
		token{tokenComma, ","},
		token{tokenInteger, "3"},
		token{tokenComma, ","},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualArrayBools(t *testing.T) {
	testFlow(t, "foo = [true, false, true]", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenTrue, "true"},
		token{tokenComma, ","},
		token{tokenFalse, "false"},
		token{tokenComma, ","},
		token{tokenTrue, "true"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualArrayBoolsWithComments(t *testing.T) {
	testFlow(t, "foo = [true, false, true] # YEAH", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenTrue, "true"},
		token{tokenComma, ","},
		token{tokenFalse, "false"},
		token{tokenComma, ","},
		token{tokenTrue, "true"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestDateRegexp(t *testing.T) {
	if dateRegexp.FindString("1979-05-27T07:32:00Z") == "" {
		t.Fail()
	}
}

func TestKeyEqualDate(t *testing.T) {
	testFlow(t, "foo = 1979-05-27T07:32:00Z", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenDate, "1979-05-27T07:32:00Z"},
		token{tokenEOF, ""},
	})
}

func TestFloatEndingWithDot(t *testing.T) {
	testFlow(t, "foo = 42.", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenError, "float cannot end with a dot"},
	})
}

func TestFloatWithTwoDots(t *testing.T) {
	testFlow(t, "foo = 4.2.", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenError, "cannot have two dots in one float"},
	})
}

func TestDoubleEqualKey(t *testing.T) {
	testFlow(t, "foo= = 2", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenError, "cannot have multiple equals for the same key"},
	})
}

func TestInvalidEsquapeSequence(t *testing.T) {
	testFlow(t, "foo = \"\\x\"", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenError, "invalid escape sequence: \\x"},
	})
}

func TestNestedArrays(t *testing.T) {
	testFlow(t, "foo = [[[]]]", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenLeftBracket, "["},
		token{tokenLeftBracket, "["},
		token{tokenLeftBracket, "["},
		token{tokenRightBracket, "]"},
		token{tokenRightBracket, "]"},
		token{tokenRightBracket, "]"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualNumber(t *testing.T) {
	testFlow(t, "foo = 42", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenInteger, "42"},
		token{tokenEOF, ""},
	})

	testFlow(t, "foo = +42", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenInteger, "+42"},
		token{tokenEOF, ""},
	})

	testFlow(t, "foo = -42", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenInteger, "-42"},
		token{tokenEOF, ""},
	})

	testFlow(t, "foo = 4.2", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenFloat, "4.2"},
		token{tokenEOF, ""},
	})

	testFlow(t, "foo = +4.2", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenFloat, "+4.2"},
		token{tokenEOF, ""},
	})

	testFlow(t, "foo = -4.2", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenFloat, "-4.2"},
		token{tokenEOF, ""},
	})
}

func TestMultiline(t *testing.T) {
	testFlow(t, "foo = 42\nbar=21", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenInteger, "42"},
		token{tokenKey, "bar"},
		token{tokenEqual, "="},
		token{tokenInteger, "21"},
		token{tokenEOF, ""},
	})
}

func TestKeyEqualStringUnicodeEscape(t *testing.T) {
	testFlow(t, "foo = \"hello \\u2665\"", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenString, "hello ♥"},
		token{tokenEOF, ""},
	})
}

func TestUnicodeString(t *testing.T) {
	testFlow(t, "foo = \"hello ♥ world\"", []token{
		token{tokenKey, "foo"},
		token{tokenEqual, "="},
		token{tokenString, "hello ♥ world"},
		token{tokenEOF, ""},
	})
}
