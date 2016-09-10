package token

import (
	"fmt"
	"strconv"
)

// Define tokens
type Type int

const (
	eof = -(iota + 1)
)

const (
	Error Type = iota
	EOF
	Comment
	Key
	String
	Integer
	True
	False
	Float
	Equal
	LeftBracket
	RightBracket
	LeftCurlyBrace
	RightCurlyBrace
	LeftParen
	RightParen
	DoubleLeftBracket
	DoubleRightBracket
	Date
	KeyGroup
	KeyGroupArray
	Comma
	Colon
	Dollar
	Star
	Question
	Dot
	DotDot
	EOL
)

var tokenTypeNames = []string{
	"Error",
	"EOF",
	"Comment",
	"Key",
	"String",
	"Integer",
	"True",
	"False",
	"Float",
	"=",
	"[",
	"]",
	"{",
	"}",
	"(",
	")",
	"]]",
	"[[",
	"Date",
	"KeyGroup",
	"KeyGroupArray",
	",",
	":",
	"$",
	"*",
	"?",
	".",
	"..",
	"EOL",
}

type Token struct {
	Position
	Typ Type
	Val string
}

func (t Type) String() string {
	idx := int(t)
	if idx < len(tokenTypeNames) {
		return tokenTypeNames[idx]
	}
	return "Unknown"
}

func (t Token) Int() int {
	if result, err := strconv.Atoi(t.Val); err != nil {
		panic(err)
	} else {
		return result
	}
}

func (t Token) String() string {
	switch t.Typ {
	case EOF:
		return "EOF"
	case Error:
		return t.Val
	}

	return fmt.Sprintf("%q", t.Val)
}

func IsComma(t *Token) bool {
	return t != nil && t.Typ == Comma
}
