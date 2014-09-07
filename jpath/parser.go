/*
  Based on the "jsonpath" spec/concept.

  http://goessner.net/articles/JsonPath/
  https://code.google.com/p/json-path/
*/

package jpath

import (
	"fmt"
	"math"
)

type parser struct {
	flow         chan token
	tokensBuffer []token
	path         *QueryPath
  union        []PathFn
}

type parserStateFn func(*parser) parserStateFn

// Formats and panics an error message based on a token
func (p *parser) raiseError(tok *token, msg string, args ...interface{}) {
	panic(tok.Position.String() + ": " + fmt.Sprintf(msg, args...))
}

func (p *parser) run() {
	for state := parseStart; state != nil; {
		state = state(p)
	}
}

func (p *parser) backup(tok *token) {
	p.tokensBuffer = append(p.tokensBuffer, *tok)
}

func (p *parser) peek() *token {
	if len(p.tokensBuffer) != 0 {
		return &(p.tokensBuffer[0])
	}

	tok, ok := <-p.flow
	if !ok {
		return nil
	}
	p.backup(&tok)
	return &tok
}

func (p *parser) lookahead(types... tokenType) bool {
  result := true
  buffer := []token{}

  for _, typ := range types {
    tok := p.getToken()
    if tok == nil {
      result = false
      break
    }
    buffer = append(buffer, *tok)
    if tok.typ != typ {
      result = false
      break
    }
  }
  // add the tokens back to the buffer, and return
  p.tokensBuffer = append(p.tokensBuffer, buffer...)
  return result
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
	tok := p.getToken()

	if tok == nil || tok.typ == tokenEOF {
		return nil
	}

	if tok.typ != tokenDollar {
		p.raiseError(tok, "Expected '$' at start of expression")
	}

	return parseMatchExpr
}

// handle '.' prefix, '[]', and '..'
func parseMatchExpr(p *parser) parserStateFn {
	tok := p.getToken()
	switch tok.typ {
	case tokenDotDot:
    p.path.Append(&matchRecursiveFn{})
    // nested parse for '..'
    tok := p.getToken()
    switch tok.typ {
    case tokenKey:
      p.path.Append(newMatchKeyFn(tok.val))
      return parseMatchExpr
    case tokenLBracket:
      return parseBracketExpr
    case tokenStar:
      // do nothing - the recursive predicate is enough
      return parseMatchExpr
    }

	case tokenDot:
    // nested parse for '.'
    tok := p.getToken()
    switch tok.typ {
    case tokenKey:
      p.path.Append(newMatchKeyFn(tok.val))
      return parseMatchExpr
    case tokenStar:
      p.path.Append(&matchAnyFn{})
      return parseMatchExpr
    }

	case tokenLBracket:
		return parseBracketExpr

	case tokenEOF:
		return nil // allow EOF at this stage
	}
	p.raiseError(tok, "expected match expression")
	return nil
}

func parseBracketExpr(p *parser) parserStateFn {
  if p.lookahead(tokenInteger, tokenColon) {
    return parseSliceExpr
  }
  if p.peek().typ == tokenColon {
    return parseSliceExpr
  }
	return parseUnionExpr
}

func parseUnionExpr(p *parser) parserStateFn {
  // this state can be traversed after some sub-expressions
  // so be careful when setting up state in the parser
	if p.union == nil {
    p.union = []PathFn{}
  }

loop: // labeled loop for easy breaking
  for {
		// parse sub expression
		tok := p.getToken()
		switch tok.typ {
		case tokenInteger:
			p.union = append(p.union, newMatchIndexFn(tok.Int()))
		case tokenKey:
			p.union = append(p.union, newMatchKeyFn(tok.val))
		case tokenString:
			p.union = append(p.union, newMatchKeyFn(tok.val))
		case tokenQuestion:
			return parseFilterExpr
		case tokenLParen:
			return parseScriptExpr
		default:
			p.raiseError(tok, "expected union sub expression, not '%s'", tok.val)
		}
		// parse delimiter or terminator
		tok = p.getToken()
		switch tok.typ {
		case tokenComma:
			continue
		case tokenRBracket:
			break loop
		default:
			p.raiseError(tok, "expected ',' or ']'")
		}
	}

  // if there is only one sub-expression, use that instead
  if len(p.union) == 1 {
    p.path.Append(p.union[0])
  }else {
    p.path.Append(&matchUnionFn{p.union})
  }

  p.union = nil // clear out state
	return parseMatchExpr
}

func parseSliceExpr(p *parser) parserStateFn {
	// init slice to grab all elements
	start, end, step := 0, math.MaxInt64, 1

	// parse optional start
	tok := p.getToken()
	if tok.typ == tokenInteger {
		start = tok.Int()
		tok = p.getToken()
	}
	if tok.typ != tokenColon {
		p.raiseError(tok, "expected ':'")
	}

	// parse optional end
	tok = p.getToken()
	if tok.typ == tokenInteger {
		end = tok.Int()
		tok = p.getToken()
	}
  if tok.typ == tokenRBracket {
	  p.path.Append(newMatchSliceFn(start, end, step))
    return parseMatchExpr
  }
  if tok.typ != tokenColon {
		p.raiseError(tok, "expected ']' or ':'")
	}

	// parse optional step
	tok = p.getToken()
	if tok.typ == tokenInteger {
		step = tok.Int()
		if step < 0 {
			p.raiseError(tok, "step must be a positive value")
		}
		tok = p.getToken()
	}
	if tok.typ != tokenRBracket {
		p.raiseError(tok, "expected ']'")
	}

	p.path.Append(newMatchSliceFn(start, end, step))
	return parseMatchExpr
}

func parseFilterExpr(p *parser) parserStateFn {
	p.raiseError(p.peek(), "filter expressions are unsupported")
	return nil
}

func parseScriptExpr(p *parser) parserStateFn {
	p.raiseError(p.peek(), "script expressions are unsupported")
	return nil
}

func parse(flow chan token) *QueryPath {
	parser := &parser{
		flow:         flow,
		tokensBuffer: []token{},
		path:         newQueryPath(),
	}
	parser.run()
	return parser.path
}
