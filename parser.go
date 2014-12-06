// TOML Parser.

package toml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type tomlParser struct {
	flow          chan token
	tree          *TomlTree
	tokensBuffer  []token
	currentGroup  []string
	seenGroupKeys []string
}

type tomlParserStateFn func() tomlParserStateFn

// Formats and panics an error message based on a token
func (p *tomlParser) raiseError(tok *token, msg string, args ...interface{}) {
	panic(tok.Position.String() + ": " + fmt.Sprintf(msg, args...))
}

func (p *tomlParser) run() {
	for state := p.parseStart; state != nil; {
		state = state()
	}
}

func (p *tomlParser) peek() *token {
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

func (p *tomlParser) assume(typ tokenType) {
	tok := p.getToken()
	if tok == nil {
		p.raiseError(tok, "was expecting token %s, but token stream is empty", tok)
	}
	if tok.typ != typ {
		p.raiseError(tok, "was expecting token %s, but got %s instead", typ, tok)
	}
}

func (p *tomlParser) getToken() *token {
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

func (p *tomlParser) parseStart() tomlParserStateFn {
	tok := p.peek()

	// end of stream, parsing is finished
	if tok == nil {
		return nil
	}

	switch tok.typ {
	case tokenDoubleLeftBracket:
		return p.parseGroupArray
	case tokenLeftBracket:
		return p.parseGroup
	case tokenKey:
		return p.parseAssign
	case tokenEOF:
		return nil
	default:
		p.raiseError(tok, "unexpected token")
	}
	return nil
}

func (p *tomlParser) parseGroupArray() tomlParserStateFn {
	startToken := p.getToken() // discard the [[
	key := p.getToken()
	if key.typ != tokenKeyGroupArray {
		p.raiseError(key, "unexpected token %s, was expecting a key group array", key)
	}

	// get or create group array element at the indicated part in the path
	keys := strings.Split(key.val, ".")
	p.tree.createSubTree(keys[:len(keys)-1], startToken.Position) // create parent entries
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
	return p.parseStart
}

func (p *tomlParser) parseGroup() tomlParserStateFn {
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
	if err := p.tree.createSubTree(keys, startToken.Position); err != nil {
		p.raiseError(key, "%s", err)
	}
	p.assume(tokenRightBracket)
	p.currentGroup = keys
	return p.parseStart
}

func (p *tomlParser) parseAssign() tomlParserStateFn {
	key := p.getToken()
	p.assume(tokenEqual)
	value := p.parseRvalue()
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
	return p.parseStart
}

func (p *tomlParser) parseRvalue() interface{} {
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
		val, err := time.Parse(time.RFC3339Nano, tok.val)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		return val
	case tokenLeftBracket:
		return p.parseArray()
	case tokenError:
		p.raiseError(tok, "%s", tok)
	}

	p.raiseError(tok, "never reached")

	return nil
}

func (p *tomlParser) parseArray() []interface{} {
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
		val := p.parseRvalue()
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

func parseToml(flow chan token) *TomlTree {
	result := newTomlTree()
	result.position = Position{1, 1}
	parser := &tomlParser{
		flow:          flow,
		tree:          result,
		tokensBuffer:  make([]token, 0),
		currentGroup:  make([]string, 0),
		seenGroupKeys: make([]string, 0),
	}
	parser.run()
	return result
}
