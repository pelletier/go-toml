package jpath

import (
	"fmt"
  "math"
	"strconv"
)

type parser struct {
	flow          chan token
	tokensBuffer  []token
  path          []PathFn
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


func (p *parser) appendPath(fn PathFn) {
  p.path = append(p.path, fn)
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

func parseMatchExpr(p *parser) parserStateFn {
	tok := p.getToken()
	switch tok.typ {
	case tokenDot:
    p.appendPath(matchKeyFn(tok.val))
    return parseMatchExpr
  case tokenDotDot:
    p.appendPath(matchRecurseFn())
		return parseSimpleMatchExpr
	case tokenLBracket:
		return parseBracketExpr
  case tokenStar:
    p.appendPath(matchAnyFn())
    return parseMatchExpr
  case tokenEOF:
    return nil  // allow EOF at this stage
	}
	p.raiseError(tok, "expected match expression")
	return nil
}

func parseSimpleMatchExpr(p *parser) parserStateFn {
	tok := p.getToken()
	switch tok.typ {
	case tokenLBracket:
		return parseBracketExpr
	case tokenKey:
    p.appendPath(matchKeyFn(tok.val))
    return parseMatchExpr
  case tokenStar:
    p.appendPath(matchAnyFn())
    return parseMatchExpr
	}
	p.raiseError(tok, "expected match expression")
	return nil
}

func parseBracketExpr(p *parser) parserStateFn {
  tok := p.peek()
  switch tok.typ {
  case tokenInteger:
    // look ahead for a ':'
    p.getToken()
    next := p.peek()
    p.backup(tok)
    if next.typ == tokenColon {
      return parseSliceExpr
    }
    return parseUnionExpr
  case tokenColon:
    return parseSliceExpr
	}
	return parseUnionExpr
}

func parseUnionExpr(p *parser) parserStateFn {
  union := []PathFn{}
  for {
    // parse sub expression
    tok := p.getToken()
    switch tok.typ {
    case tokenInteger:
      idx, _ := strconv.Atoi(tok.val)
      union = append(union, matchIndexFn(idx))
    case tokenKey:
      union = append(union, matchKeyFn(tok.val))
    case tokenQuestion:
      return parseFilterExpr
    case tokenLParen:
      return parseScriptExpr
    default:
      p.raiseError(tok, "expected union sub expression")
    }
    // parse delimiter or terminator
    tok = p.getToken()
    switch tok.typ {
    case tokenComma:
      continue
    case tokenRBracket:
      break
    default:
      p.raiseError(tok, "expected ',' or ']'")
    }
  }
  p.appendPath(matchUnionFn(union))
  return parseMatchExpr
}

func parseSliceExpr(p *parser) parserStateFn {
  // init slice to grab all elements
  start, end, step := 0, math.MaxInt64, 1

  // parse optional start
  tok := p.getToken()
  if tok.typ == tokenInteger {
    start, _ = strconv.Atoi(tok.val)
    tok = p.getToken()
  }
  if tok.typ != tokenColon {
    p.raiseError(tok, "expected ':'")
  }

  // parse optional end
  tok = p.getToken()
  if tok.typ == tokenInteger {
    end, _ = strconv.Atoi(tok.val)
    tok = p.getToken()
  }
  if tok.typ != tokenColon || tok.typ != tokenRBracket {
    p.raiseError(tok, "expected ']' or ':'")
  }

  // parse optional step
  tok = p.getToken()
  if tok.typ == tokenInteger {
    step, _ = strconv.Atoi(tok.val)
    if step < 0 {
      p.raiseError(tok, "step must be a positive value")
    }
    tok = p.getToken()
  }
  if tok.typ != tokenRBracket {
    p.raiseError(tok, "expected ']'")
  }

  p.appendPath(matchSliceFn(start, end, step))
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

func parse(flow chan token) []PathFn {
	result := []PathFn{}
	parser := &parser{
		flow:          flow,
		tokensBuffer:  []token{},
    path:          result,
	}
	parser.run()
	return result
}
