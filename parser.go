// TOML Parser.

package toml

import (
	"fmt"
	"strconv"
	"time"
)

type parser struct {
	flow         chan token
	tree         *TomlTree
	tokensBuffer []token
	currentGroup string
}

type parserStateFn func(*parser) parserStateFn

func (p *parser) run() {
	for state := parseStart; state != nil; {
		state = state(p)
	}
}

func (p *parser) peek() *token {
	if len(p.tokensBuffer) != 0 {
		return &(p.tokensBuffer[0])
	}

	tok, ok := <-p.flow
	if !ok {
		return nil
	}
	p.tokensBuffer = append(p.tokensBuffer, tok)
	return &tok
}

func (p *parser) assume(typ tokenType) {
	tok := p.getToken()
	if tok == nil {
		panic(fmt.Sprintf("was expecting token %s, but token stream is empty", typ))
	}
	if tok.typ != typ {
		panic(fmt.Sprintf("was expecting token %s, but got %s", typ, tok.typ))
	}
}

func (p *parser) getToken() *token {
	if len(p.tokensBuffer) != 0 {
		tok := p.tokensBuffer[0]
		p.tokensBuffer = p.tokensBuffer[1:]
		return &tok
	}
	tok, ok := <-p.flow
	if !ok {
		return nil
	}
	return &tok
}

func parseStart(p *parser) parserStateFn {
	tok := p.peek()

	// end of stream, parsing is finished
	if tok == nil {
		return nil
	}

	switch tok.typ {
	case tokenLeftBracket:
		return parseGroup
	case tokenKey:
		return parseAssign
	case tokenEOF:
		return nil
	default:
		panic("unexpected token")
	}
	return nil
}

func parseGroup(p *parser) parserStateFn {
	p.getToken() // discard the [
	key := p.getToken()
	if key.typ != tokenKeyGroup {
		panic(fmt.Sprintf("unexpected token %s, was expecting a key group", key))
	}
	p.tree.createSubTree(key.val)
	p.assume(tokenRightBracket)
	p.currentGroup = key.val
	return parseStart(p)
}

func parseAssign(p *parser) parserStateFn {
	key := p.getToken()
	p.assume(tokenEqual)
	value := parseRvalue(p)
	final_key := key.val
	if p.currentGroup != "" {
		final_key = p.currentGroup + "." + key.val
	}
	p.tree.Set(final_key, value)
	return parseStart(p)
}

func parseRvalue(p *parser) interface{} {
	tok := p.getToken()
	if tok == nil || tok.typ == tokenEOF {
		panic("expecting a value")
	}

	switch tok.typ {
	case tokenString:
		return tok.val
	case tokenTrue:
		return true
	case tokenFalse:
		return false
	case tokenInteger:
		val, err := strconv.ParseInt(tok.val, 10, 64)
		if err != nil {
			panic(err)
		}
		return val
	case tokenFloat:
		val, err := strconv.ParseFloat(tok.val, 64)
		if err != nil {
			panic(err)
		}
		return val
	case tokenDate:
		val, err := time.Parse(time.RFC3339, tok.val)
		if err != nil {
			panic(err)
		}
		return val
	case tokenLeftBracket:
		return parseArray(p)
	}

	panic("never reached")

	return nil
}

func parseArray(p *parser) []interface{} {
	array := make([]interface{}, 0)
	for {
		follow := p.peek()
		if follow == nil || follow.typ == tokenEOF {
			panic("unterminated array")
		}
		if follow.typ == tokenRightBracket {
			p.getToken()
			return array
		}
		val := parseRvalue(p)
		array = append(array, val)
		follow = p.peek()
		if follow == nil {
			panic("unterminated array")
		}
		if follow.typ != tokenRightBracket && follow.typ != tokenComma {
			fmt.Println(follow.typ)
			fmt.Println(follow.val)
			panic("missing comma")
		}
		if follow.typ == tokenComma {
			p.getToken()
		}
	}
	return array
}

func parse(flow chan token) *TomlTree {
	result := make(TomlTree)
	parser := &parser{
		flow:         flow,
		tree:         &result,
		tokensBuffer: make([]token, 0),
		currentGroup: "",
	}
	parser.run()
	return parser.tree
}
