package toml

import "testing"

func testFlow(t *testing.T, input string, expectedFlow []token) {
	_, ch := lex(input)
	for _, expected := range expectedFlow {
		token := <-ch
		if token != expected {
      t.Log("While testing: ", input)
			t.Log("compared", token, "to", expected)
			t.Log(token.val,  "<->", expected.val)
			t.Log(token.typ,  "<->", expected.typ)
			t.Log(token.line, "<->", expected.line)
			t.Log(token.col,  "<->", expected.col)
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
		token{tokenLeftBracket, "[", 0, 0},
		token{tokenKeyGroup, "hello world", 0, 1},
		token{tokenRightBracket, "]", 0, 12},
		token{tokenEOF, "", 0, 13},
	})
}

func TestUnclosedKeyGroup(t *testing.T) {
	testFlow(t, "[hello world", []token{
		token{tokenLeftBracket, "[", 0, 0},
		token{tokenError, "unclosed key group", 0, 1},
	})
}

func TestComment(t *testing.T) {
	testFlow(t, "# blahblah", []token{
		token{tokenEOF, "", 0, 10},
	})
}

func TestKeyGroupComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah", []token{
		token{tokenLeftBracket, "[", 0, 0},
		token{tokenKeyGroup, "hello world", 0, 1},
		token{tokenRightBracket, "]", 0, 12},
		token{tokenEOF, "", 0, 24},
	})
}

func TestMultipleKeyGroupsComment(t *testing.T) {
	testFlow(t, "[hello world] # blahblah\n[test]", []token{
		token{tokenLeftBracket, "[", 0, 0},
		token{tokenKeyGroup, "hello world", 0, 1},
		token{tokenRightBracket, "]", 0, 12},
		token{tokenLeftBracket, "[", 1, 0},
		token{tokenKeyGroup, "test", 1, 1},
		token{tokenRightBracket, "]", 1, 5},
		token{tokenEOF, "", 1, 6},
	})
}

func TestBasicKey(t *testing.T) {
	testFlow(t, "hello", []token{
		token{tokenKey, "hello", 0, 0},
		token{tokenEOF, "", 0, 5},
	})
}

func TestBasicKeyWithUnderscore(t *testing.T) {
	testFlow(t, "hello_hello", []token{
		token{tokenKey, "hello_hello", 0, 0},
		token{tokenEOF, "", 0, 11},
	})
}

func TestBasicKeyWithDash(t *testing.T) {
	testFlow(t, "hello-world", []token{
		token{tokenKey, "hello-world", 0, 0},
		token{tokenEOF, "", 0, 11},
	})
}

func TestBasicKeyWithUppercaseMix(t *testing.T) {
	testFlow(t, "helloHELLOHello", []token{
		token{tokenKey, "helloHELLOHello", 0, 0},
		token{tokenEOF, "", 0, 15},
	})
}

func TestBasicKeyWithInternationalCharacters(t *testing.T) {
	testFlow(t, "héllÖ", []token{
		token{tokenKey, "héllÖ", 0, 0},
		token{tokenEOF, "", 0, 5},
	})
}

func TestBasicKeyAndEqual(t *testing.T) {
	testFlow(t, "hello =", []token{
		token{tokenKey, "hello", 0, 0},
		token{tokenEqual, "=", 0, 6},
		token{tokenEOF, "", 0, 7},
	})
}

func TestKeyWithSharpAndEqual(t *testing.T) {
	testFlow(t, "key#name = 5", []token{
		token{tokenKey, "key#name", 0, 0},
		token{tokenEqual, "=", 0, 9},
		token{tokenInteger, "5", 0, 11},
		token{tokenEOF, "", 0, 12},
	})
}


func TestKeyWithSymbolsAndEqual(t *testing.T) {
	testFlow(t, "~!@#$^&*()_+-`1234567890[]\\|/?><.,;:' = 5", []token{
		token{tokenKey, "~!@#$^&*()_+-`1234567890[]\\|/?><.,;:'", 0, 0},
		token{tokenEqual, "=", 0, 38},
		token{tokenInteger, "5", 0, 40},
		token{tokenEOF, "", 0, 41},
	})
}

func TestKeyEqualStringEscape(t *testing.T) {
	testFlow(t, `foo = "hello\""`, []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenString, "hello\"" ,0, 7},
		token{tokenEOF, "", 0, 15},
	})
}

func TestKeyEqualStringUnfinished(t *testing.T) {
	testFlow(t, `foo = "bar`, []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenError, "unclosed string", 0, 7},
	})
}

func TestKeyEqualString(t *testing.T) {
	testFlow(t, `foo = "bar"`, []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenString, "bar", 0, 7},
		token{tokenEOF, "", 0, 11},
	})
}

func TestKeyEqualTrue(t *testing.T) {
	testFlow(t, "foo = true", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenTrue, "true", 0, 6},
		token{tokenEOF, "", 0, 10},
	})
}

func TestKeyEqualFalse(t *testing.T) {
	testFlow(t, "foo = false", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenFalse, "false", 0, 6},
		token{tokenEOF, "", 0, 11},
	})
}

func TestArrayNestedString(t *testing.T) {
	testFlow(t, `a = [ ["hello", "world"] ]`, []token{
		token{tokenKey, "a", 0, 0},
		token{tokenEqual, "=", 0, 2},
		token{tokenLeftBracket, "[", 0, 4},
		token{tokenLeftBracket, "[", 0, 6},
		token{tokenString, "hello", 0, 8},
		token{tokenComma, ",", 0, 14},
		token{tokenString, "world", 0, 17},
		token{tokenRightBracket, "]", 0, 23},
		token{tokenRightBracket, "]", 0, 25},
		token{tokenEOF, "", 0, 26},
	})
}

func TestArrayNestedInts(t *testing.T) {
	testFlow(t, "a = [ [42, 21], [10] ]", []token{
		token{tokenKey, "a", 0, 0},
		token{tokenEqual, "=", 0, 2},
		token{tokenLeftBracket, "[", 0, 4},
		token{tokenLeftBracket, "[", 0, 6},
		token{tokenInteger, "42", 0, 7},
		token{tokenComma, ",", 0, 9},
		token{tokenInteger, "21", 0, 11},
		token{tokenRightBracket, "]", 0, 13},
		token{tokenComma, ",", 0, 14},
		token{tokenLeftBracket, "[", 0, 16},
		token{tokenInteger, "10", 0, 17},
		token{tokenRightBracket, "]", 0, 19},
		token{tokenRightBracket, "]", 0, 21},
		token{tokenEOF, "", 0, 22},
	})
}

func TestArrayInts(t *testing.T) {
	testFlow(t, "a = [ 42, 21, 10, ]", []token{
		token{tokenKey, "a", 0, 0},
		token{tokenEqual, "=", 0, 2},
		token{tokenLeftBracket, "[", 0, 4},
		token{tokenInteger, "42", 0, 6},
		token{tokenComma, ",", 0, 8},
		token{tokenInteger, "21", 0, 10},
		token{tokenComma, ",", 0, 12},
		token{tokenInteger, "10", 0, 14},
		token{tokenComma, ",", 0, 16},
		token{tokenRightBracket, "]", 0, 18},
		token{tokenEOF, "", 0, 19},
	})
}

func TestMultilineArrayComments(t *testing.T) {
	testFlow(t, "a = [1, # wow\n2, # such items\n3, # so array\n]", []token{
		token{tokenKey, "a", 0, 0},
		token{tokenEqual, "=", 0, 2},
		token{tokenLeftBracket, "[", 0, 4},
		token{tokenInteger, "1", 0, 5},
		token{tokenComma, ",", 0, 6},
		token{tokenInteger, "2", 1, 0},
		token{tokenComma, ",", 1, 1},
		token{tokenInteger, "3", 2, 0},
		token{tokenComma, ",", 2, 1},
		token{tokenRightBracket, "]", 3, 0},
		token{tokenEOF, "", 3, 1},
	})
}

func TestKeyEqualArrayBools(t *testing.T) {
	testFlow(t, "foo = [true, false, true]", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenLeftBracket, "[", 0, 6},
		token{tokenTrue, "true", 0, 7},
		token{tokenComma, ",", 0, 11},
		token{tokenFalse, "false", 0, 13},
		token{tokenComma, ",", 0, 18},
		token{tokenTrue, "true", 0, 20},
		token{tokenRightBracket, "]", 0, 24},
		token{tokenEOF, "", 0, 25},
	})
}

func TestKeyEqualArrayBoolsWithComments(t *testing.T) {
	testFlow(t, "foo = [true, false, true] # YEAH", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenLeftBracket, "[", 0, 6},
		token{tokenTrue, "true", 0, 7},
		token{tokenComma, ",", 0, 11},
		token{tokenFalse, "false", 0, 13},
		token{tokenComma, ",", 0, 18},
		token{tokenTrue, "true", 0, 20},
		token{tokenRightBracket, "]", 0, 24},
		token{tokenEOF, "", 0, 32},
	})
}

func TestDateRegexp(t *testing.T) {
	if dateRegexp.FindString("1979-05-27T07:32:00Z") == "" {
		t.Fail()
	}
}

func TestKeyEqualDate(t *testing.T) {
	testFlow(t, "foo = 1979-05-27T07:32:00Z", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenDate, "1979-05-27T07:32:00Z", 0, 6},
		token{tokenEOF, "", 0, 26},
	})
}

func TestFloatEndingWithDot(t *testing.T) {
	testFlow(t, "foo = 42.", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenError, "float cannot end with a dot", 0, 6},
	})
}

func TestFloatWithTwoDots(t *testing.T) {
	testFlow(t, "foo = 4.2.", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenError, "cannot have two dots in one float", 0, 6},
	})
}

func TestDoubleEqualKey(t *testing.T) {
	testFlow(t, "foo= = 2", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 3},
		token{tokenError, "cannot have multiple equals for the same key", 0, 4},
	})
}

func TestInvalidEsquapeSequence(t *testing.T) {
	testFlow(t, `foo = "\x"`, []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenError, "invalid escape sequence: \\x", 0, 7},
	})
}

func TestNestedArrays(t *testing.T) {
	testFlow(t, "foo = [[[]]]", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenLeftBracket, "[", 0, 6},
		token{tokenLeftBracket, "[", 0, 7},
		token{tokenLeftBracket, "[", 0, 8},
		token{tokenRightBracket, "]", 0, 9},
		token{tokenRightBracket, "]", 0, 10},
		token{tokenRightBracket, "]", 0, 11},
		token{tokenEOF, "", 0, 12},
	})
}

func TestKeyEqualNumber(t *testing.T) {
	testFlow(t, "foo = 42", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenInteger, "42", 0, 6},
		token{tokenEOF, "", 0, 8},
	})

	testFlow(t, "foo = +42", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenInteger, "+42", 0, 6},
		token{tokenEOF, "", 0, 9},
	})

	testFlow(t, "foo = -42", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenInteger, "-42", 0, 6},
		token{tokenEOF, "", 0, 9},
	})

	testFlow(t, "foo = 4.2", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenFloat, "4.2", 0, 6},
		token{tokenEOF, "", 0, 9},
	})

	testFlow(t, "foo = +4.2", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenFloat, "+4.2", 0, 6},
		token{tokenEOF, "", 0, 10},
	})

	testFlow(t, "foo = -4.2", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenFloat, "-4.2", 0, 6},
		token{tokenEOF, "", 0, 10},
	})
}

func TestMultiline(t *testing.T) {
	testFlow(t, "foo = 42\nbar=21", []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenInteger, "42", 0, 6},
		token{tokenKey, "bar", 1, 0},
		token{tokenEqual, "=", 1, 3},
		token{tokenInteger, "21", 1, 4},
		token{tokenEOF, "", 1, 6},
	})
}

func TestKeyEqualStringUnicodeEscape(t *testing.T) {
	testFlow(t, `foo = "hello \u2665"`, []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenString, "hello ♥", 0, 7},
		token{tokenEOF, "", 0, 20},
	})
}

func TestUnicodeString(t *testing.T) {
	testFlow(t, `foo = "hello ♥ world"`, []token{
		token{tokenKey, "foo", 0, 0},
		token{tokenEqual, "=", 0, 4},
		token{tokenString, "hello ♥ world", 0, 7},
		token{tokenEOF, "", 0, 21},
	})
}

func TestKeyGroupArray(t *testing.T) {
	testFlow(t, "[[foo]]", []token{
		token{tokenDoubleLeftBracket, "[[", 0, 0},
		token{tokenKeyGroupArray, "foo", 0, 2},
		token{tokenDoubleRightBracket, "]]", 0, 5},
		token{tokenEOF, "", 0, 7},
	})
}
