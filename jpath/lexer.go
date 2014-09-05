// TOML JSONPath lexer.
//
// Written using the principles developped by Rob Pike in
// http://www.youtube.com/watch?v=HxaD_trXwRE

package jpath

import (
	"fmt"
	. "github.com/pelletier/go-toml"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var dateRegexp *regexp.Regexp

// Define tokens
type tokenType int

const (
	eof = -(iota + 1)
)

const (
	tokenError tokenType = iota
	tokenEOF
	tokenKey
	tokenString
	tokenFloat
	tokenInteger
	tokenAtCost
	tokenDollar
	tokenLBracket
	tokenRBracket
	tokenDot
	tokenDotDot
	tokenStar
	tokenComma
	tokenColon
	tokenQuestion
	tokenLParen
	tokenRParen
)

var tokenTypeNames = []string{
	"EOF",
	"Key",
	"String",
	"Float",
	"Integer",
	"@",
	"$",
	"[",
	"]",
	".",
	"..",
	"*",
	",",
	":",
	"?",
	"(",
	")",
}

type token struct {
	Position
	typ tokenType
	val string
}

func (tt tokenType) String() string {
	idx := int(tt)
	if idx < len(tokenTypeNames) {
		return tokenTypeNames[idx]
	}
	return "Unknown"
}

func (t token) Int() int {
  if result, err := strconv.Atoi(t.val); err != nil {
    panic(err)
  } else {
    return result
  }
}

func (t token) String() string {
	switch t.typ {
	case tokenEOF:
		return "EOF"
	case tokenError:
		return t.val
	}

	if len(t.val) > 10 {
		return fmt.Sprintf("%.10q...", t.val)
	}
	return fmt.Sprintf("%q", t.val)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isKeyChar(r rune) bool {
	// "Keys start with the first non-whitespace character and end with the last
	// non-whitespace character before the equals sign."
	return !(isSpace(r) || r == '\r' || r == '\n' || r == eof || r == '=')
}

func isDigit(r rune) bool {
	return unicode.IsNumber(r)
}

func isHexDigit(r rune) bool {
	return isDigit(r) ||
		r == 'A' || r == 'B' || r == 'C' || r == 'D' || r == 'E' || r == 'F'
}

// Define lexer
type lexer struct {
	input      string
	start      int
	pos        int
	width      int
	tokens     chan token
	depth      int
	line       int
	col        int
	stringTerm string
}

func (l *lexer) run() {
	for state := lexVoid; state != nil; {
		state = state(l)
	}
	close(l.tokens)
}

func (l *lexer) nextStart() {
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

func (l *lexer) emit(t tokenType) {
	l.tokens <- token{
		Position: Position{l.line, l.col},
		typ:      t,
		val:      l.input[l.start:l.pos],
	}
	l.nextStart()
}

func (l *lexer) emitWithValue(t tokenType, value string) {
	l.tokens <- token{
		Position: Position{l.line, l.col},
		typ:      t,
		val:      value,
	}
	l.nextStart()
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) ignore() {
	l.nextStart()
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{
		Position: Position{l.line, l.col},
		typ:      tokenError,
		val:      fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) follow(next string) bool {
	return strings.HasPrefix(l.input[l.pos:], next)
}

// Define state functions
type stateFn func(*lexer) stateFn

func lexVoid(l *lexer) stateFn {
	for {
		next := l.peek()
		switch next {
		case '$':
			l.pos++
			l.emit(tokenDollar)
			continue
		case '.':
			if l.follow("..") {
				l.pos += 2
				l.emit(tokenDotDot)
			} else {
				l.pos++
				l.emit(tokenDot)
			}
			continue
		case '@':
			l.pos++
			l.emit(tokenAtCost)
			continue
		case '[':
			l.pos++
			l.emit(tokenLBracket)
			continue
		case ']':
			l.pos++
			l.emit(tokenRBracket)
			continue
		case ',':
			l.pos++
			l.emit(tokenComma)
			continue
		case '*':
			l.pos++
			l.emit(tokenStar)
			continue
		case '(':
			l.pos++
			l.emit(tokenLParen)
			continue
		case ')':
			l.pos++
			l.emit(tokenRParen)
			continue
		case '?':
			l.pos++
			l.emit(tokenQuestion)
			continue
		case ':':
			l.pos++
			l.emit(tokenColon)
			continue
		case '\'':
			l.ignore()
			l.stringTerm = string(next)
			return lexString
		case '"':
			l.ignore()
			l.stringTerm = string(next)
			return lexString
		}

    if isSpace(next) {
			l.next()
      l.ignore()
      continue
		}

		if isAlphanumeric(next) {
			return lexKey
		}

		if next == '+' || next == '-' || isDigit(next) {
			return lexNumber
		}

		if l.next() == eof {
			break
		}

		return l.errorf("unexpected char: '%v'", next)
	}
	l.emit(tokenEOF)
	return nil
}

func lexKey(l *lexer) stateFn {
	for {
		next := l.peek()
		if !isAlphanumeric(next) {
			l.emit(tokenKey)
			return lexVoid
		}

		if l.next() == eof {
			break
		}
	}
	l.emit(tokenEOF)
	return nil
}

func lexString(l *lexer) stateFn {
	l.pos++
	l.ignore()
	growingString := ""

	for {
		if l.follow(l.stringTerm) {
			l.emitWithValue(tokenString, growingString)
			l.pos++
			l.ignore()
			return lexVoid
		}

		if l.follow("\\\"") {
			l.pos++
			growingString += "\""
		} else if l.follow("\\'") {
			l.pos++
			growingString += "'"
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

func lexNumber(l *lexer) stateFn {
	l.ignore()
	if !l.accept("+") {
		l.accept("-")
	}
	pointSeen := false
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
	if pointSeen {
		l.emit(tokenFloat)
	} else {
		l.emit(tokenInteger)
	}
	return lexVoid
}

// Entry point
func lex(input string) (*lexer, chan token) {
	l := &lexer{
		input:  input,
		tokens: make(chan token),
		line:   1,
		col:    1,
	}
	go l.run()
	return l, l.tokens
}
