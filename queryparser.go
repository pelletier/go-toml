/*
  Based on the "jsonpath" spec/concept.

  http://goessner.net/articles/JsonPath/
  https://code.google.com/p/json-path/
*/

package toml

import (
	"fmt"

	"github.com/pelletier/go-toml/token"
)

const maxInt = int(^uint(0) >> 1)

type queryParser struct {
	flow         chan token.Token
	tokensBuffer []token.Token
	query        *Query
	union        []pathFn
	err          error
}

type queryParserStateFn func() queryParserStateFn

// Formats and panics an error message based on a token
func (p *queryParser) parseError(tok *token.Token, msg string, args ...interface{}) queryParserStateFn {
	p.err = fmt.Errorf(tok.Position.String()+": "+msg, args...)
	return nil // trigger parse to end
}

func (p *queryParser) run() {
	for state := p.parseStart; state != nil; {
		state = state()
	}
}

func (p *queryParser) backup(tok *token.Token) {
	p.tokensBuffer = append(p.tokensBuffer, *tok)
}

func (p *queryParser) peek() *token.Token {
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

func (p *queryParser) lookahead(types ...token.Type) bool {
	result := true
	buffer := []token.Token{}

	for _, typ := range types {
		tok := p.getToken()
		if tok == nil {
			result = false
			break
		}
		buffer = append(buffer, *tok)
		if tok.Typ != typ {
			result = false
			break
		}
	}
	// add the tokens back to the buffer, and return
	p.tokensBuffer = append(p.tokensBuffer, buffer...)
	return result
}

func (p *queryParser) getToken() *token.Token {
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

func (p *queryParser) parseStart() queryParserStateFn {
	tok := p.getToken()

	if tok == nil || tok.Typ == token.EOF {
		return nil
	}

	if tok.Typ != token.Dollar {
		return p.parseError(tok, "Expected '$' at start of expression")
	}

	return p.parseMatchExpr
}

// handle '.' prefix, '[]', and '..'
func (p *queryParser) parseMatchExpr() queryParserStateFn {
	tok := p.getToken()
	switch tok.Typ {
	case token.DotDot:
		p.query.appendPath(&matchRecursiveFn{})
		// nested parse for '..'
		tok := p.getToken()
		switch tok.Typ {
		case token.Key:
			p.query.appendPath(newMatchKeyFn(tok.Val))
			return p.parseMatchExpr
		case token.LeftBracket:
			return p.parseBracketExpr
		case token.Star:
			// do nothing - the recursive predicate is enough
			return p.parseMatchExpr
		}

	case token.Dot:
		// nested parse for '.'
		tok := p.getToken()
		switch tok.Typ {
		case token.Key:
			p.query.appendPath(newMatchKeyFn(tok.Val))
			return p.parseMatchExpr
		case token.Star:
			p.query.appendPath(&matchAnyFn{})
			return p.parseMatchExpr
		}

	case token.LeftBracket:
		return p.parseBracketExpr

	case token.EOF:
		return nil // allow EOF at this stage
	}
	return p.parseError(tok, "expected match expression")
}

func (p *queryParser) parseBracketExpr() queryParserStateFn {
	if p.lookahead(token.Integer, token.Colon) {
		return p.parseSliceExpr
	}
	if p.peek().Typ == token.Colon {
		return p.parseSliceExpr
	}
	return p.parseUnionExpr
}

func (p *queryParser) parseUnionExpr() queryParserStateFn {
	var tok *token.Token

	// this state can be traversed after some sub-expressions
	// so be careful when setting up state in the parser
	if p.union == nil {
		p.union = []pathFn{}
	}

loop: // labeled loop for easy breaking
	for {
		if len(p.union) > 0 {
			// parse delimiter or terminator
			tok = p.getToken()
			switch tok.Typ {
			case token.Comma:
				// do nothing
			case token.RightBracket:
				break loop
			default:
				return p.parseError(tok, "expected ',' or ']', not '%s'", tok.Val)
			}
		}

		// parse sub expression
		tok = p.getToken()
		switch tok.Typ {
		case token.Integer:
			p.union = append(p.union, newMatchIndexFn(tok.Int()))
		case token.Key:
			p.union = append(p.union, newMatchKeyFn(tok.Val))
		case token.String:
			p.union = append(p.union, newMatchKeyFn(tok.Val))
		case token.Question:
			return p.parseFilterExpr
		default:
			return p.parseError(tok, "expected union sub expression, not '%s', %d", tok.Val, len(p.union))
		}
	}

	// if there is only one sub-expression, use that instead
	if len(p.union) == 1 {
		p.query.appendPath(p.union[0])
	} else {
		p.query.appendPath(&matchUnionFn{p.union})
	}

	p.union = nil // clear out state
	return p.parseMatchExpr
}

func (p *queryParser) parseSliceExpr() queryParserStateFn {
	// init slice to grab all elements
	start, end, step := 0, maxInt, 1

	// parse optional start
	tok := p.getToken()
	if tok.Typ == token.Integer {
		start = tok.Int()
		tok = p.getToken()
	}
	if tok.Typ != token.Colon {
		return p.parseError(tok, "expected ':'")
	}

	// parse optional end
	tok = p.getToken()
	if tok.Typ == token.Integer {
		end = tok.Int()
		tok = p.getToken()
	}
	if tok.Typ == token.RightBracket {
		p.query.appendPath(newMatchSliceFn(start, end, step))
		return p.parseMatchExpr
	}
	if tok.Typ != token.Colon {
		return p.parseError(tok, "expected ']' or ':'")
	}

	// parse optional step
	tok = p.getToken()
	if tok.Typ == token.Integer {
		step = tok.Int()
		if step < 0 {
			return p.parseError(tok, "step must be a positive value")
		}
		tok = p.getToken()
	}
	if tok.Typ != token.RightBracket {
		return p.parseError(tok, "expected ']'")
	}

	p.query.appendPath(newMatchSliceFn(start, end, step))
	return p.parseMatchExpr
}

func (p *queryParser) parseFilterExpr() queryParserStateFn {
	tok := p.getToken()
	if tok.Typ != token.LeftParen {
		return p.parseError(tok, "expected left-parenthesis for filter expression")
	}
	tok = p.getToken()
	if tok.Typ != token.Key && tok.Typ != token.String {
		return p.parseError(tok, "expected key or string for filter funciton name")
	}
	name := tok.Val
	tok = p.getToken()
	if tok.Typ != token.RightParen {
		return p.parseError(tok, "expected right-parenthesis for filter expression")
	}
	p.union = append(p.union, newMatchFilterFn(name, tok.Position))
	return p.parseUnionExpr
}

func parseQuery(flow chan token.Token) (*Query, error) {
	parser := &queryParser{
		flow:         flow,
		tokensBuffer: []token.Token{},
		query:        newQuery(),
	}
	parser.run()
	return parser.query, parser.err
}
