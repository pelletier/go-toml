// TOML JSONPath lexer.
//
// Written using the principles developed by Rob Pike in
// http://www.youtube.com/watch?v=HxaD_trXwRE

package toml

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pelletier/go-toml/lexer"
	"github.com/pelletier/go-toml/token"
)

const (
	eof = -(iota + 1)
)

// Lexer state function
type queryLexStateFn func() queryLexStateFn

// Lexer definition
type queryLexer struct {
	input      string
	start      int
	pos        int
	width      int
	tokens     chan token.Token
	depth      int
	line       int
	col        int
	stringTerm string
}

func (l *queryLexer) run() {
	for state := l.lexVoid; state != nil; {
		state = state()
	}
	close(l.tokens)
}

func (l *queryLexer) nextStart() {
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

func (l *queryLexer) emit(t token.Type) {
	l.tokens <- token.Token{
		Position: token.Position{l.line, l.col},
		Typ:      t,
		Val:      l.input[l.start:l.pos],
	}
	l.nextStart()
}

func (l *queryLexer) emitWithValue(t token.Type, value string) {
	l.tokens <- token.Token{
		Position: token.Position{l.line, l.col},
		Typ:      t,
		Val:      value,
	}
	l.nextStart()
}

func (l *queryLexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *queryLexer) ignore() {
	l.nextStart()
}

func (l *queryLexer) backup() {
	l.pos -= l.width
}

func (l *queryLexer) errorf(format string, args ...interface{}) queryLexStateFn {
	l.tokens <- token.Token{
		Position: token.Position{l.line, l.col},
		Typ:      token.Error,
		Val:      fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *queryLexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *queryLexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *queryLexer) follow(next string) bool {
	return strings.HasPrefix(l.input[l.pos:], next)
}

func (l *queryLexer) lexVoid() queryLexStateFn {
	for {
		next := l.peek()
		switch next {
		case '$':
			l.pos++
			l.emit(token.Dollar)
			continue
		case '.':
			if l.follow("..") {
				l.pos += 2
				l.emit(token.DotDot)
			} else {
				l.pos++
				l.emit(token.Dot)
			}
			continue
		case '[':
			l.pos++
			l.emit(token.LeftBracket)
			continue
		case ']':
			l.pos++
			l.emit(token.RightBracket)
			continue
		case ',':
			l.pos++
			l.emit(token.Comma)
			continue
		case '*':
			l.pos++
			l.emit(token.Star)
			continue
		case '(':
			l.pos++
			l.emit(token.LeftParen)
			continue
		case ')':
			l.pos++
			l.emit(token.RightParen)
			continue
		case '?':
			l.pos++
			l.emit(token.Question)
			continue
		case ':':
			l.pos++
			l.emit(token.Colon)
			continue
		case '\'':
			l.ignore()
			l.stringTerm = string(next)
			return l.lexString
		case '"':
			l.ignore()
			l.stringTerm = string(next)
			return l.lexString
		}

		if lexer.IsSpace(next) {
			l.next()
			l.ignore()
			continue
		}

		if lexer.IsAlphanumeric(next) {
			return l.lexKey
		}

		if next == '+' || next == '-' || lexer.IsDigit(next) {
			return l.lexNumber
		}

		if l.next() == eof {
			break
		}

		return l.errorf("unexpected char: '%v'", next)
	}
	l.emit(token.EOF)
	return nil
}

func (l *queryLexer) lexKey() queryLexStateFn {
	for {
		next := l.peek()
		if !lexer.IsAlphanumeric(next) {
			l.emit(token.Key)
			return l.lexVoid
		}

		if l.next() == eof {
			break
		}
	}
	l.emit(token.EOF)
	return nil
}

func (l *queryLexer) lexString() queryLexStateFn {
	l.pos++
	l.ignore()
	growingString := ""

	for {
		if l.follow(l.stringTerm) {
			l.emitWithValue(token.String, growingString)
			l.pos++
			l.ignore()
			return l.lexVoid
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
				if !lexer.IsHexDigit(c) {
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
		} else if l.follow("\\U") {
			l.pos += 2
			code := ""
			for i := 0; i < 8; i++ {
				c := l.peek()
				l.pos++
				if !lexer.IsHexDigit(c) {
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

func (l *queryLexer) lexNumber() queryLexStateFn {
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
			if !lexer.IsDigit(l.peek()) {
				return l.errorf("float cannot end with a dot")
			}
			pointSeen = true
		} else if lexer.IsDigit(next) {
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
		l.emit(token.Float)
	} else {
		l.emit(token.Integer)
	}
	return l.lexVoid
}

// Entry point
func lexQuery(input string) chan token.Token {
	l := &queryLexer{
		input:  input,
		tokens: make(chan token.Token),
		line:   1,
		col:    1,
	}
	go l.run()
	return l.tokens
}
