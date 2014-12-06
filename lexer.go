// TOML lexer.
//
// Written using the principles developped by Rob Pike in
// http://www.youtube.com/watch?v=HxaD_trXwRE

package toml

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var dateRegexp *regexp.Regexp

// Define state functions
type tomlLexStateFn func() tomlLexStateFn

// Define lexer
type tomlLexer struct {
	input  string
	start  int
	pos    int
	width  int
	tokens chan token
	depth  int
	line   int
	col    int
}

func (l *tomlLexer) run() {
	for state := l.lexVoid; state != nil; {
		state = state()
	}
	close(l.tokens)
}

func (l *tomlLexer) nextStart() {
	// iterate by runes (utf8 characters)
	// search for newlines and advance line/col counts
	for i := l.start; i < l.pos; {
		r, width := utf8.DecodeRuneInString(l.input[i:])
		if r == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		i += width
	}
	// advance start position to next token
	l.start = l.pos
}

func (l *tomlLexer) emit(t tokenType) {
	l.tokens <- token{
		Position: Position{l.line, l.col},
		typ:      t,
		val:      l.input[l.start:l.pos],
	}
	l.nextStart()
}

func (l *tomlLexer) emitWithValue(t tokenType, value string) {
	l.tokens <- token{
		Position: Position{l.line, l.col},
		typ:      t,
		val:      value,
	}
	l.nextStart()
}

func (l *tomlLexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *tomlLexer) ignore() {
	l.nextStart()
}

func (l *tomlLexer) backup() {
	l.pos -= l.width
}

func (l *tomlLexer) errorf(format string, args ...interface{}) tomlLexStateFn {
	l.tokens <- token{
		Position: Position{l.line, l.col},
		typ:      tokenError,
		val:      fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *tomlLexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *tomlLexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *tomlLexer) follow(next string) bool {
	return strings.HasPrefix(l.input[l.pos:], next)
}

func (l *tomlLexer) lexVoid() tomlLexStateFn {
	for {
		next := l.peek()
		switch next {
		case '[':
			return l.lexKeyGroup
		case '#':
			return l.lexComment
		case '=':
			return l.lexEqual
		}

		if isSpace(next) {
			l.ignore()
		}

		if l.depth > 0 {
			return l.lexRvalue
		}

		if isKeyStartChar(next) {
			return l.lexKey
		}

		if l.next() == eof {
			break
		}
	}

	l.emit(tokenEOF)
	return nil
}

func (l *tomlLexer) lexRvalue() tomlLexStateFn {
	for {
		next := l.peek()
		switch next {
		case '.':
			return l.errorf("cannot start float with a dot")
		case '=':
			return l.errorf("cannot have multiple equals for the same key")
		case '[':
			l.depth++
			return l.lexLeftBracket
		case ']':
			l.depth--
			return l.lexRightBracket
		case '#':
			return l.lexComment
		case '"':
			return l.lexString
		case '\'':
			return l.lexLiteralString
		case ',':
			return l.lexComma
		case '\n':
			l.ignore()
			l.pos++
			if l.depth == 0 {
				return l.lexVoid
			}
			return l.lexRvalue
		}

		if l.follow("true") {
			return l.lexTrue
		}

		if l.follow("false") {
			return l.lexFalse
		}

		if isAlphanumeric(next) {
			return l.lexKey
		}

		dateMatch := dateRegexp.FindString(l.input[l.pos:])
		if dateMatch != "" {
			l.ignore()
			l.pos += len(dateMatch)
			return l.lexDate
		}

		if next == '+' || next == '-' || isDigit(next) {
			return l.lexNumber
		}

		if isSpace(next) {
			l.ignore()
		}

		if l.next() == eof {
			break
		}
	}

	l.emit(tokenEOF)
	return nil
}

func (l *tomlLexer) lexDate() tomlLexStateFn {
	l.emit(tokenDate)
	return l.lexRvalue
}

func (l *tomlLexer) lexTrue() tomlLexStateFn {
	l.ignore()
	l.pos += 4
	l.emit(tokenTrue)
	return l.lexRvalue
}

func (l *tomlLexer) lexFalse() tomlLexStateFn {
	l.ignore()
	l.pos += 5
	l.emit(tokenFalse)
	return l.lexRvalue
}

func (l *tomlLexer) lexEqual() tomlLexStateFn {
	l.ignore()
	l.accept("=")
	l.emit(tokenEqual)
	return l.lexRvalue
}

func (l *tomlLexer) lexComma() tomlLexStateFn {
	l.ignore()
	l.accept(",")
	l.emit(tokenComma)
	return l.lexRvalue
}

func (l *tomlLexer) lexKey() tomlLexStateFn {
	l.ignore()
	for r := l.next(); isKeyChar(r); r = l.next() {
		if (r == '#') {
			return l.errorf("keys cannot contain # character")
		}
	}
	l.backup()
	l.emit(tokenKey)
	return l.lexVoid
}

func (l *tomlLexer) lexComment() tomlLexStateFn {
	for {
		next := l.next()
		if next == '\n' || next == eof {
			break
		}
	}
	l.ignore()
	return l.lexVoid
}

func (l *tomlLexer) lexLeftBracket() tomlLexStateFn {
	l.ignore()
	l.pos++
	l.emit(tokenLeftBracket)
	return l.lexRvalue
}

func (l *tomlLexer) lexLiteralString() tomlLexStateFn {
	l.pos++
	l.ignore()
	growingString := ""

	for {
		if l.peek() == '\'' {
			l.emitWithValue(tokenString, growingString)
			l.pos++
			l.ignore()
			return l.lexRvalue
		}

		growingString += string(l.peek())

		if l.next() == eof {
			break
		}
	}

	return l.errorf("unclosed string")
}

func (l *tomlLexer) lexString() tomlLexStateFn {
	l.pos++
	l.ignore()
	growingString := ""

	for {
		if l.peek() == '"' {
			l.emitWithValue(tokenString, growingString)
			l.pos++
			l.ignore()
			return l.lexRvalue
		}

		if l.follow("\\\"") {
			l.pos++
			growingString += "\""
		} else if l.follow("\\n") {
			l.pos++
			growingString += "\n"
		} else if l.follow("\\b") {
			l.pos++
			growingString += "\b"
		} else if l.follow("\\f") {
			l.pos++
			growingString += "\f"
		} else if l.follow("\\/") {
			l.pos++
			growingString += "/"
		} else if l.follow("\\t") {
			l.pos++
			growingString += "\t"
		} else if l.follow("\\r") {
			l.pos++
			growingString += "\r"
		} else if l.follow("\\\\") {
			l.pos++
			growingString += "\\"
		} else if l.follow("\\u") {
			l.pos += 2
			code := ""
			for i := 0; i < 4; i++ {
				c := l.peek()
				l.pos++
				if !isHexDigit(c) {
					return l.errorf("unfinished unicode escape")
				}
				code = code + string(c)
			}
			l.pos--
			intcode, err := strconv.ParseInt(code, 16, 32)
			if err != nil {
				return l.errorf("invalid unicode escape: \\u" + code)
			}
			growingString += string(rune(intcode))
		} else if l.follow("\\") {
			l.pos++
			return l.errorf("invalid escape sequence: \\" + string(l.peek()))
		} else {
			growingString += string(l.peek())
		}

		if l.next() == eof {
			break
		}
	}

	return l.errorf("unclosed string")
}

func (l *tomlLexer) lexKeyGroup() tomlLexStateFn {
	l.ignore()
	l.pos++

	if l.peek() == '[' {
		// token '[[' signifies an array of anonymous key groups
		l.pos++
		l.emit(tokenDoubleLeftBracket)
		return l.lexInsideKeyGroupArray
	}
	// vanilla key group
	l.emit(tokenLeftBracket)
	return l.lexInsideKeyGroup
}

func (l *tomlLexer) lexInsideKeyGroupArray() tomlLexStateFn {
	for {
		if l.peek() == ']' {
			if l.pos > l.start {
				l.emit(tokenKeyGroupArray)
			}
			l.ignore()
			l.pos++
			if l.peek() != ']' {
				break // error
			}
			l.pos++
			l.emit(tokenDoubleRightBracket)
			return l.lexVoid
		} else if l.peek() == '[' {
			return l.errorf("group name cannot contain ']'")
		}

		if l.next() == eof {
			break
		}
	}
	return l.errorf("unclosed key group array")
}

func (l *tomlLexer) lexInsideKeyGroup() tomlLexStateFn {
	for {
		if l.peek() == ']' {
			if l.pos > l.start {
				l.emit(tokenKeyGroup)
			}
			l.ignore()
			l.pos++
			l.emit(tokenRightBracket)
			return l.lexVoid
		} else if l.peek() == '[' {
			return l.errorf("group name cannot contain ']'")
		}

		if l.next() == eof {
			break
		}
	}
	return l.errorf("unclosed key group")
}

func (l *tomlLexer) lexRightBracket() tomlLexStateFn {
	l.ignore()
	l.pos++
	l.emit(tokenRightBracket)
	return l.lexRvalue
}

func (l *tomlLexer) lexNumber() tomlLexStateFn {
	l.ignore()
	if !l.accept("+") {
		l.accept("-")
	}
	pointSeen := false
	expSeen := false
	digitSeen := false
	for {
		next := l.next()
		if next == '.' {
			if pointSeen {
				return l.errorf("cannot have two dots in one float")
			}
			if !isDigit(l.peek()) {
				return l.errorf("float cannot end with a dot")
			}
			pointSeen = true
		} else if next == 'e' || next == 'E' {
			expSeen = true
			if !l.accept("+") {
				l.accept("-")
			}
		} else if isDigit(next) {
			digitSeen = true
		} else {
			l.backup()
			break
		}
		if pointSeen && !digitSeen {
			return l.errorf("cannot start float with a dot")
		}
	}

	if !digitSeen {
		return l.errorf("no digit in that number")
	}
	if pointSeen || expSeen {
		l.emit(tokenFloat)
	} else {
		l.emit(tokenInteger)
	}
	return l.lexRvalue
}

func init() {
	dateRegexp = regexp.MustCompile("^\\d{1,4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\.\\d{1,9})?(Z|[+-]\\d{2}:\\d{2})")
}

// Entry point
func lexToml(input string) chan token {
	l := &tomlLexer{
		input:  input,
		tokens: make(chan token),
		line:   1,
		col:    1,
	}
	go l.run()
	return l.tokens
}
