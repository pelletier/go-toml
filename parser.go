// TOML Parser.

package toml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type parser struct {
	flow          chan token
	tree          *TomlTree
	tokensBuffer  []token
	currentGroup  []string
	seenGroupKeys []string
}

type parserStateFn func(*parser) parserStateFn

// Formats and panics an error message based on a token
func (p *parser) raiseError(tok *token, msg string, args... interface{}) {
  panic(tok.Pos() + ": " + fmt.Sprintf(msg, args...))
}

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
		p.raiseError(tok, "was expecting token %s, but token stream is empty", tok.typ)
	}
	if tok.typ != typ {
		p.raiseError(tok, "was expecting token %s, but got %s", typ, tok.typ)
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
	case tokenDoubleLeftBracket:
		return parseGroupArray
	case tokenLeftBracket:
		return parseGroup
	case tokenKey:
		return parseAssign
	case tokenEOF:
		return nil
	default:
		p.raiseError(tok, "unexpected token")
	}
	return nil
}

func parseGroupArray(p *parser) parserStateFn {
	p.getToken() // discard the [[
	key := p.getToken()
	if key.typ != tokenKeyGroupArray {
		p.raiseError(key, "unexpected token %s, was expecting a key group array", key)
	}

	// get or create group array element at the indicated part in the path
	p.currentGroup = strings.Split(key.val, ".")
	dest_tree := p.tree.GetPath(p.currentGroup)
	var array []*TomlTree
	if dest_tree == nil {
		array = make([]*TomlTree, 0)
	} else if dest_tree.([]*TomlTree) != nil {
		array = dest_tree.([]*TomlTree)
	} else {
		p.raiseError(key, "key %s is already assigned and not of type group array", key)
	}

	// add a new tree to the end of the group array
	new_tree := make(TomlTree)
	array = append(array, &new_tree)
	p.tree.SetPath(p.currentGroup, array)

	// keep this key name from use by other kinds of assignments
	p.seenGroupKeys = append(p.seenGroupKeys, key.val)

	// move to next parser state
	p.assume(tokenDoubleRightBracket)
	return parseStart(p)
}

func parseGroup(p *parser) parserStateFn {
	p.getToken() // discard the [
	key := p.getToken()
	if key.typ != tokenKeyGroup {
		p.raiseError(key, "unexpected token %s, was expecting a key group", key)
	}
	for _, item := range p.seenGroupKeys {
		if item == key.val {
			p.raiseError(key, "duplicated tables")
		}
	}
	p.seenGroupKeys = append(p.seenGroupKeys, key.val)
	if err := p.tree.createSubTree(key.val); err != nil {
    p.raiseError(key, "%s", err)
  }
	p.assume(tokenRightBracket)
	p.currentGroup = strings.Split(key.val, ".")
	return parseStart(p)
}

func parseAssign(p *parser) parserStateFn {
	key := p.getToken()
	p.assume(tokenEqual)
	value := parseRvalue(p)
	var group_key []string
	if len(p.currentGroup) > 0 {
		group_key = p.currentGroup
	} else {
		group_key = make([]string, 0)
	}

	// find the group to assign, looking out for arrays of groups
	var target_node *TomlTree
	switch node := p.tree.GetPath(group_key).(type) {
	case []*TomlTree:
		target_node = node[len(node)-1]
	case *TomlTree:
		target_node = node
	default:
		p.raiseError(key, "Unknown group type for path %s", group_key)
	}

	// assign value to the found group
	local_key := []string{key.val}
	final_key := append(group_key, key.val)
	if target_node.GetPath(local_key) != nil {
		p.raiseError(key, "the following key was defined twice: %s", strings.Join(final_key, "."))
	}
	target_node.SetPath(local_key, value)
	return parseStart(p)
}

func parseRvalue(p *parser) interface{} {
	tok := p.getToken()
	if tok == nil || tok.typ == tokenEOF {
		p.raiseError(tok, "expecting a value")
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
			p.raiseError(tok, "%s", err)
		}
		return val
	case tokenFloat:
		val, err := strconv.ParseFloat(tok.val, 64)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		return val
	case tokenDate:
		val, err := time.Parse(time.RFC3339, tok.val)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		return val
	case tokenLeftBracket:
		return parseArray(p)
	case tokenError:
		p.raiseError(tok, "%s", tok)
	}

  p.raiseError(tok, "never reached")

	return nil
}

func parseArray(p *parser) []interface{} {
	array := make([]interface{}, 0)
	arrayType := reflect.TypeOf(nil)
	for {
		follow := p.peek()
		if follow == nil || follow.typ == tokenEOF {
			p.raiseError(follow, "unterminated array")
		}
		if follow.typ == tokenRightBracket {
			p.getToken()
			return array
		}
		val := parseRvalue(p)
		if arrayType == nil {
			arrayType = reflect.TypeOf(val)
		}
		if reflect.TypeOf(val) != arrayType {
			p.raiseError(follow, "mixed types in array")
		}
		array = append(array, val)
		follow = p.peek()
		if follow == nil {
			p.raiseError(follow, "unterminated array")
		}
		if follow.typ != tokenRightBracket && follow.typ != tokenComma {
			p.raiseError(follow, "missing comma")
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
		flow:          flow,
		tree:          &result,
		tokensBuffer:  make([]token, 0),
		currentGroup:  make([]string, 0),
		seenGroupKeys: make([]string, 0),
	}
	parser.run()
	return parser.tree
}
