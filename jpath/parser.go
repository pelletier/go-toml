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
	path         *Query
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
    p.path.appendPath(&matchRecursiveFn{})
    // nested parse for '..'
    tok := p.getToken()
    switch tok.typ {
    case tokenKey:
      p.path.appendPath(newMatchKeyFn(tok.val))
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
      p.path.appendPath(newMatchKeyFn(tok.val))
      return parseMatchExpr
    case tokenStar:
      p.path.appendPath(&matchAnyFn{})
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
  var tok *token

  // this state can be traversed after some sub-expressions
  // so be careful when setting up state in the parser
	if p.union == nil {
    p.union = []PathFn{}
  }

loop: // labeled loop for easy breaking
  for {
    if len(p.union) > 0 {
      // parse delimiter or terminator
      tok = p.getToken()
      switch tok.typ {
      case tokenComma:
        // do nothing
      case tokenRBracket:
        break loop
      default:
        p.raiseError(tok, "expected ',' or ']', not '%s'", tok.val)
      }
    }

		// parse sub expression
		tok = p.getToken()
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
			p.raiseError(tok, "expected union sub expression, not '%s', %d", tok.val, len(p.union))
		}
	}

  // if there is only one sub-expression, use that instead
  if len(p.union) == 1 {
    p.path.appendPath(p.union[0])
  }else {
    p.path.appendPath(&matchUnionFn{p.union})
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
	  p.path.appendPath(newMatchSliceFn(start, end, step))
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

	p.path.appendPath(newMatchSliceFn(start, end, step))
	return parseMatchExpr
}

func parseFilterExpr(p *parser) parserStateFn {
  tok := p.getToken()
  if tok.typ != tokenLParen {
    p.raiseError(tok, "expected left-parenthesis for filter expression")
  }
  tok = p.getToken()
  if tok.typ != tokenKey && tok.typ != tokenString {
    p.raiseError(tok, "expected key or string for filter funciton name")
  }
  name := tok.val
  tok = p.getToken()
  if tok.typ != tokenRParen {
    p.raiseError(tok, "expected right-parenthesis for filter expression")
  }
	p.union = append(p.union, newMatchFilterFn(name, tok.Position))
	return parseUnionExpr
}

func parseScriptExpr(p *parser) parserStateFn {
  tok := p.getToken()
  if tok.typ != tokenKey && tok.typ != tokenString {
    p.raiseError(tok, "expected key or string for script funciton name")
  }
  name := tok.val
  tok = p.getToken()
  if tok.typ != tokenRParen {
    p.raiseError(tok, "expected right-parenthesis for script expression")
  }
	p.union = append(p.union, newMatchScriptFn(name, tok.Position))
	return parseUnionExpr
}

func parse(flow chan token) *Query {
	parser := &parser{
		flow:         flow,
		tokensBuffer: []token{},
		path:         newQuery(),
	}
	parser.run()
	return parser.path
}
