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
func (p *parser) raiseError(tok *token, msg string, args ...interface{}) {
	panic(tok.Position.String() + ": " + fmt.Sprintf(msg, args...))
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
		p.raiseError(tok, "was expecting token %s, but token stream is empty", tok)
	}
	if tok.typ != typ {
		p.raiseError(tok, "was expecting token %s, but got %s instead", typ, tok)
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

	// prime position data with root tree instance
	p.tree.position = tok.Position

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
	startToken := p.getToken() // discard the [[
	key := p.getToken()
	if key.typ != tokenKeyGroupArray {
		p.raiseError(key, "unexpected token %s, was expecting a key group array", key)
	}

	// get or create group array element at the indicated part in the path
	keys := strings.Split(key.val, ".")
	p.tree.createSubTree(keys[:len(keys)-1]) // create parent entries
	destTree := p.tree.GetPath(keys)
	var array []*TomlTree
	if destTree == nil {
		array = make([]*TomlTree, 0)
	} else if destTree.([]*TomlTree) != nil {
		array = destTree.([]*TomlTree)
	} else {
		p.raiseError(key, "key %s is already assigned and not of type group array", key)
	}
	p.currentGroup = keys

	// add a new tree to the end of the group array
	newTree := newTomlTree()
	newTree.position = startToken.Position
	array = append(array, newTree)
	p.tree.SetPath(p.currentGroup, array)

	// remove all keys that were children of this group array
	prefix := key.val + "."
	found := false
	for ii := 0; ii < len(p.seenGroupKeys); {
		groupKey := p.seenGroupKeys[ii]
		if strings.HasPrefix(groupKey, prefix) {
			p.seenGroupKeys = append(p.seenGroupKeys[:ii], p.seenGroupKeys[ii+1:]...)
		} else {
			found = (groupKey == key.val)
			ii++
		}
	}

	// keep this key name from use by other kinds of assignments
	if !found {
		p.seenGroupKeys = append(p.seenGroupKeys, key.val)
	}

	// move to next parser state
	p.assume(tokenDoubleRightBracket)
	return parseStart(p)
}

func parseGroup(p *parser) parserStateFn {
	startToken := p.getToken() // discard the [
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
	keys := strings.Split(key.val, ".")
	if err := p.tree.createSubTree(keys); err != nil {
		p.raiseError(key, "%s", err)
	}
	p.assume(tokenRightBracket)
	p.currentGroup = keys
	targetTree := p.tree.GetPath(p.currentGroup).(*TomlTree)
	targetTree.position = startToken.Position
	return parseStart(p)
}

func parseAssign(p *parser) parserStateFn {
	key := p.getToken()
	p.assume(tokenEqual)
	value := parseRvalue(p)
	var groupKey []string
	if len(p.currentGroup) > 0 {
		groupKey = p.currentGroup
	} else {
		groupKey = []string{}
	}

	// find the group to assign, looking out for arrays of groups
	var targetNode *TomlTree
	switch node := p.tree.GetPath(groupKey).(type) {
	case []*TomlTree:
		targetNode = node[len(node)-1]
	case *TomlTree:
		targetNode = node
	default:
		p.raiseError(key, "Unknown group type for path: %s",
			strings.Join(groupKey, "."))
	}

	// assign value to the found group
	localKey := []string{key.val}
	finalKey := append(groupKey, key.val)
	if targetNode.GetPath(localKey) != nil {
		p.raiseError(key, "The following key was defined twice: %s",
			strings.Join(finalKey, "."))
	}
	targetNode.values[key.val] = &tomlValue{value, key.Position}
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
	var array []interface{}
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
	result := newTomlTree()
	parser := &parser{
		flow:          flow,
		tree:          result,
		tokensBuffer:  make([]token, 0),
		currentGroup:  make([]string, 0),
		seenGroupKeys: make([]string, 0),
	}
	parser.run()
	return result
}
