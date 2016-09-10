// TOML Parser.

package toml

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/token"
)

type tomlParser struct {
	flow          chan token.Token
	tree          *TomlTree
	tokensBuffer  []token.Token
	currentGroup  []string
	seenGroupKeys []string
}

type tomlParserStateFn func() tomlParserStateFn

// Formats and panics an error message based on a token
func (p *tomlParser) raiseError(tok *token.Token, msg string, args ...interface{}) {
	panic(tok.Position.String() + ": " + fmt.Sprintf(msg, args...))
}

func (p *tomlParser) run() {
	for state := p.parseStart; state != nil; {
		state = state()
	}
}

func (p *tomlParser) peek() *token.Token {
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

func (p *tomlParser) assume(typ token.Type) {
	tok := p.getToken()
	if tok == nil {
		p.raiseError(tok, "was expecting token %s, but token stream is empty", tok)
	}
	if tok.Typ != typ {
		p.raiseError(tok, "was expecting token %s, but got %s instead", typ, tok)
	}
}

func (p *tomlParser) getToken() *token.Token {
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

	switch tok.Typ {
	case token.DoubleLeftBracket:
		return p.parseGroupArray
	case token.LeftBracket:
		return p.parseGroup
	case token.Key:
		return p.parseAssign
	case token.EOF:
		return nil
	default:
		p.raiseError(tok, "unexpected token")
	}
	return nil
}

func (p *tomlParser) parseGroupArray() tomlParserStateFn {
	startToken := p.getToken() // discard the [[
	key := p.getToken()
	if key.Typ != token.KeyGroupArray {
		p.raiseError(key, "unexpected token %s, was expecting a key group array", key)
	}

	// get or create group array element at the indicated part in the path
	keys, err := parseKey(key.Val)
	if err != nil {
		p.raiseError(key, "invalid group array key: %s", err)
	}
	p.tree.createSubTree(keys[:len(keys)-1], startToken.Position) // create parent entries
	destTree := p.tree.GetPath(keys)
	var array []*TomlTree
	if destTree == nil {
		array = make([]*TomlTree, 0)
	} else if target, ok := destTree.([]*TomlTree); ok && target != nil {
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
	prefix := key.Val + "."
	found := false
	for ii := 0; ii < len(p.seenGroupKeys); {
		groupKey := p.seenGroupKeys[ii]
		if strings.HasPrefix(groupKey, prefix) {
			p.seenGroupKeys = append(p.seenGroupKeys[:ii], p.seenGroupKeys[ii+1:]...)
		} else {
			found = (groupKey == key.Val)
			ii++
		}
	}

	// keep this key name from use by other kinds of assignments
	if !found {
		p.seenGroupKeys = append(p.seenGroupKeys, key.Val)
	}

	// move to next parser state
	p.assume(token.DoubleRightBracket)
	return p.parseStart
}

func (p *tomlParser) parseGroup() tomlParserStateFn {
	startToken := p.getToken() // discard the [
	key := p.getToken()
	if key.Typ != token.KeyGroup {
		p.raiseError(key, "unexpected token %s, was expecting a key group", key)
	}
	for _, item := range p.seenGroupKeys {
		if item == key.Val {
			p.raiseError(key, "duplicated tables")
		}
	}

	p.seenGroupKeys = append(p.seenGroupKeys, key.Val)
	keys, err := parseKey(key.Val)
	if err != nil {
		p.raiseError(key, "invalid group array key: %s", err)
	}
	if err := p.tree.createSubTree(keys, startToken.Position); err != nil {
		p.raiseError(key, "%s", err)
	}
	p.assume(token.RightBracket)
	p.currentGroup = keys
	return p.parseStart
}

func (p *tomlParser) parseAssign() tomlParserStateFn {
	key := p.getToken()
	p.assume(token.Equal)

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
	keyVals, err := parseKey(key.Val)
	if err != nil {
		p.raiseError(key, "%s", err)
	}
	if len(keyVals) != 1 {
		p.raiseError(key, "Invalid key")
	}
	keyVal := keyVals[0]
	localKey := []string{keyVal}
	finalKey := append(groupKey, keyVal)
	if targetNode.GetPath(localKey) != nil {
		p.raiseError(key, "The following key was defined twice: %s",
			strings.Join(finalKey, "."))
	}
	var toInsert interface{}

	switch value.(type) {
	case *TomlTree:
		toInsert = value
	default:
		toInsert = &tomlValue{value, key.Position}
	}
	targetNode.values[keyVal] = toInsert
	return p.parseStart
}

var numberUnderscoreInvalidRegexp *regexp.Regexp

func cleanupNumberToken(value string) (string, error) {
	if numberUnderscoreInvalidRegexp.MatchString(value) {
		return "", fmt.Errorf("invalid use of _ in number")
	}
	cleanedVal := strings.Replace(value, "_", "", -1)
	return cleanedVal, nil
}

func (p *tomlParser) parseRvalue() interface{} {
	tok := p.getToken()
	if tok == nil || tok.Typ == token.EOF {
		p.raiseError(tok, "expecting a value")
	}

	switch tok.Typ {
	case token.String:
		return tok.Val
	case token.True:
		return true
	case token.False:
		return false
	case token.Integer:
		cleanedVal, err := cleanupNumberToken(tok.Val)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		val, err := strconv.ParseInt(cleanedVal, 10, 64)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		return val
	case token.Float:
		cleanedVal, err := cleanupNumberToken(tok.Val)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		val, err := strconv.ParseFloat(cleanedVal, 64)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		return val
	case token.Date:
		val, err := time.ParseInLocation(time.RFC3339Nano, tok.Val, time.UTC)
		if err != nil {
			p.raiseError(tok, "%s", err)
		}
		return val
	case token.LeftBracket:
		return p.parseArray()
	case token.LeftCurlyBrace:
		return p.parseInlineTable()
	case token.Equal:
		p.raiseError(tok, "cannot have multiple equals for the same key")
	case token.Error:
		p.raiseError(tok, "%s", tok)
	}

	p.raiseError(tok, "never reached")

	return nil
}

func (p *tomlParser) parseInlineTable() *TomlTree {
	tree := newTomlTree()
	var previous *token.Token
Loop:
	for {
		follow := p.peek()
		if follow == nil || follow.Typ == token.EOF {
			p.raiseError(follow, "unterminated inline table")
		}
		switch follow.Typ {
		case token.RightCurlyBrace:
			p.getToken()
			break Loop
		case token.Key:
			if previous != nil && previous.Typ != token.Comma {
				p.raiseError(follow, "comma expected between fields in inline table")
			}
			key := p.getToken()
			p.assume(token.Equal)
			value := p.parseRvalue()
			tree.Set(key.Val, value)
		case token.Comma:
			if previous == nil {
				p.raiseError(follow, "inline table cannot start with a comma")
			}
			if previous.Typ == token.Comma {
				p.raiseError(follow, "need field between two commas in inline table")
			}
			p.getToken()
		default:
			p.raiseError(follow, "unexpected token type in inline table: %s", follow.Typ.String())
		}
		previous = follow
	}
	if previous != nil && previous.Typ == token.Comma {
		p.raiseError(previous, "trailing comma at the end of inline table")
	}
	return tree
}

func (p *tomlParser) parseArray() interface{} {
	var array []interface{}
	arrayType := reflect.TypeOf(nil)
	for {
		follow := p.peek()
		if follow == nil || follow.Typ == token.EOF {
			p.raiseError(follow, "unterminated array")
		}
		if follow.Typ == token.RightBracket {
			p.getToken()
			break
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
		if follow == nil || follow.Typ == token.EOF {
			p.raiseError(follow, "unterminated array")
		}
		if follow.Typ != token.RightBracket && follow.Typ != token.Comma {
			p.raiseError(follow, "missing comma")
		}
		if follow.Typ == token.Comma {
			p.getToken()
		}
	}
	// An array of TomlTrees is actually an array of inline
	// tables, which is a shorthand for a table array. If the
	// array was not converted from []interface{} to []*TomlTree,
	// the two notations would not be equivalent.
	if arrayType == reflect.TypeOf(newTomlTree()) {
		tomlArray := make([]*TomlTree, len(array))
		for i, v := range array {
			tomlArray[i] = v.(*TomlTree)
		}
		return tomlArray
	}
	return array
}

func parseToml(flow chan token.Token) *TomlTree {
	result := newTomlTree()
	result.position = token.Position{1, 1}
	parser := &tomlParser{
		flow:          flow,
		tree:          result,
		tokensBuffer:  make([]token.Token, 0),
		currentGroup:  make([]string, 0),
		seenGroupKeys: make([]string, 0),
	}
	parser.run()
	return result
}

func init() {
	numberUnderscoreInvalidRegexp = regexp.MustCompile(`([^\d]_|_[^\d]|_$|^_)`)
}
