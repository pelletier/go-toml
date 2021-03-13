package unmarshaler

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

type parser struct {
	tree ast.Root
}

func (p *parser) parse(b []byte) error {
	b, err := p.parseExpression(b)
	if err != nil {
		return err
	}
	for len(b) > 0 {
		b, err = p.parseNewline(b)
		if err != nil {
			return err
		}

		b, err = p.parseExpression(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) parseNewline(b []byte) ([]byte, error) {
	if b[0] == '\n' {
		return b[1:], nil
	}
	if b[0] == '\r' {
		_, rest, err := scanWindowsNewline(b)
		return rest, err
	}
	return nil, fmt.Errorf("expected newline but got %#U", b[0])
}

func (p *parser) parseExpression(b []byte) ([]byte, error) {
	//expression =  ws [ comment ]
	//expression =/ ws keyval ws [ comment ]
	//expression =/ ws table ws [ comment ]

	b = p.parseWhitespace(b)

	if len(b) == 0 {
		return b, nil
	}

	if b[0] == '#' {
		_, rest, err := scanComment(b)
		return rest, err
	}
	if b[0] == '\n' || b[0] == '\r' {
		return b, nil
	}

	var err error
	var node ast.Node
	if b[0] == '[' {
		b, err = p.parseTable(b)
	} else {
		node, b, err = p.parseKeyval(b)
	}
	if err != nil {
		return nil, err
	}

	b = p.parseWhitespace(b)

	if len(b) > 0 && b[0] == '#' {
		_, rest, err := scanComment(b)
		return rest, err
	}

	p.tree = append(p.tree, node)

	return b, nil
}

func (p *parser) parseTable(b []byte) ([]byte, error) {
	//table = std-table / array-table
	if len(b) > 1 && b[1] == '[' {
		return p.parseArrayTable(b)
	}
	return p.parseStdTable(b)
}

func (p *parser) parseArrayTable(b []byte) ([]byte, error) {
	//array-table = array-table-open key array-table-close
	//array-table-open  = %x5B.5B ws  ; [[ Double left square bracket
	//array-table-close = ws %x5D.5D  ; ]] Double right square bracket

	// TODO
	//b = b[2:]
	//b = p.parseWhitespace(b)
	//b, err := p.parseKey(b)
	//if err != nil {
	//	return nil, err
	//}
	//b = p.parseWhitespace(b)
	//b, err = expect(']', b)
	//if err != nil {
	//	return nil, err
	//}
	//return expect(']', b)

	return nil, nil
}

func (p *parser) parseStdTable(b []byte) ([]byte, error) {
	//std-table = std-table-open key std-table-close
	//std-table-open  = %x5B ws     ; [ Left square bracket
	//std-table-close = ws %x5D     ; ] Right square bracket

	// TODO
	//b = b[1:]
	//b = p.parseWhitespace(b)
	//b, err := p.parseKey(b)
	//if err != nil {
	//	return nil, err
	//}
	//b = p.parseWhitespace(b)
	//
	//return expect(']', b)

	return nil, nil
}

func (p *parser) parseKeyval(b []byte) (ast.Node, []byte, error) {
	//keyval = key keyval-sep val

	node := ast.Node{
		Kind: ast.KeyValue,
	}

	key, b, err := p.parseKey(b)
	if err != nil {
		return ast.NoNode, nil, err
	}
	node.Children = append(node.Children, key...)

	//keyval-sep = ws %x3D ws ; =

	b = p.parseWhitespace(b)
	b, err = expect('=', b)
	if err != nil {
		return ast.NoNode, nil, err
	}
	b = p.parseWhitespace(b)

	valNode, b, err := p.parseVal(b)
	if err == nil {
		node.Children = append(node.Children, valNode)
	}
	return node, b, err
}

func (p *parser) parseVal(b []byte) (ast.Node, []byte, error) {
	// val = string / boolean / array / inline-table / date-time / float / integer
	if len(b) == 0 {
		return ast.NoNode, nil, fmt.Errorf("expected value, not eof")
	}

	node := ast.Node{}
	var err error
	c := b[0]

	switch c {
	// strings
	case '"':
		var v []byte
		if scanFollowsMultilineBasicStringDelimiter(b) {
			v, b, err = p.parseMultilineBasicString(b)
		} else {
			v, b, err = p.parseBasicString(b)
		}
		if err == nil {
			node.Kind = ast.String
			node.Data = v
		}
		return node, b, err
	case '\'':
		var v []byte
		if scanFollowsMultilineLiteralStringDelimiter(b) {
			v, b, err = p.parseMultilineLiteralString(b)
		} else {
			v, b, err = p.parseLiteralString(b)
		}
		if err == nil {
			// TODO
			v = v
		}
		return node, b, err
	case 't':
		if !scanFollowsTrue(b) {
			return node, nil, fmt.Errorf("expected 'true'")
		}
		// TODO
		return node, b[4:], nil
	case 'f':
		if !scanFollowsFalse(b) {
			return node, nil, fmt.Errorf("expected 'false'")
		}
		// TODO
		return node, b[5:], nil
	case '[':
		// TODO
		//return p.parseValArray(b)
	case '{':
		// TODO
		//return p.parseInlineTable(b)
	default:
		// TODO
		//return p.parseIntOrFloatOrDateTime(b)
	}
	panic("parseVal not finished yet")
	return ast.Node{}, nil, nil
}

func (p *parser) parseLiteralString(b []byte) ([]byte, []byte, error) {
	v, rest, err := scanLiteralString(b)
	if err != nil {
		return nil, nil, err
	}
	return v[1 : len(v)-1], rest, nil
}

func (p *parser) parseInlineTable(b []byte) ([]byte, error) {
	//inline-table = inline-table-open [ inline-table-keyvals ] inline-table-close
	//inline-table-open  = %x7B ws     ; {
	//inline-table-close = ws %x7D     ; }
	//inline-table-sep   = ws %x2C ws  ; , Comma
	//inline-table-keyvals = keyval [ inline-table-sep inline-table-keyvals ]

	// TODO
	//b = b[1:]
	//
	//first := true
	//var err error
	//for len(b) > 0 {
	//	b = p.parseWhitespace(b)
	//	if b[0] == '}' {
	//		break
	//	}
	//
	//	if !first {
	//		b, err = expect(',', b)
	//		if err != nil {
	//			return nil, err
	//		}
	//		b = p.parseWhitespace(b)
	//	}
	//	b, err = p.parseKeyval(b)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	first = false
	//}

	return expect('}', b)
}

func (p *parser) parseValArray(b []byte) ([]byte, error) {
	//array = array-open [ array-values ] ws-comment-newline array-close
	//array-open =  %x5B ; [
	//array-close = %x5D ; ]
	//array-values =  ws-comment-newline val ws-comment-newline array-sep array-values
	//array-values =/ ws-comment-newline val ws-comment-newline [ array-sep ]
	//array-sep = %x2C  ; , Comma
	//ws-comment-newline = *( wschar / [ comment ] newline )

	// TODO
	//b = b[1:]
	//
	//first := true
	//var err error
	//for len(b) > 0 {
	//	b, err = p.parseOptionalWhitespaceCommentNewline(b)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	if len(b) == 0 {
	//		return nil, unexpectedCharacter{b: b}
	//	}
	//
	//	if b[0] == ']' {
	//		break
	//	}
	//	if b[0] == ',' {
	//		if first {
	//			return nil, fmt.Errorf("array cannot start with comma")
	//		}
	//		b = b[1:]
	//		b, err = p.parseOptionalWhitespaceCommentNewline(b)
	//		if err != nil {
	//			return nil, err
	//		}
	//	}
	//
	//	b, err = p.parseVal(b)
	//	if err != nil {
	//		return nil, err
	//	}
	//	b, err = p.parseOptionalWhitespaceCommentNewline(b)
	//	if err != nil {
	//		return nil, err
	//	}
	//	first = false
	//}

	return expect(']', b)
}

func (p *parser) parseOptionalWhitespaceCommentNewline(b []byte) ([]byte, error) {
	var err error
	b = p.parseWhitespace(b)
	if len(b) > 0 && b[0] == '#' {
		_, b, err = scanComment(b)
		if err != nil {
			return nil, err
		}
	}
	if len(b) > 0 && (b[0] == '\n' || b[0] == '\r') {
		b, err = p.parseNewline(b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (p *parser) parseMultilineLiteralString(b []byte) ([]byte, []byte, error) {
	token, rest, err := scanMultilineLiteralString(b)
	if err != nil {
		return nil, nil, err
	}

	i := 3

	// skip the immediate new line
	if token[i] == '\n' {
		i++
	} else if token[i] == '\r' && token[i+1] == '\n' {
		i += 2
	}

	return token[i : len(b)-3], rest, err
}

func (p *parser) parseMultilineBasicString(b []byte) ([]byte, []byte, error) {
	//ml-basic-string = ml-basic-string-delim [ newline ] ml-basic-body
	//ml-basic-string-delim
	//ml-basic-string-delim = 3quotation-mark
	//ml-basic-body = *mlb-content *( mlb-quotes 1*mlb-content ) [ mlb-quotes ]
	//
	//mlb-content = mlb-char / newline / mlb-escaped-nl
	//mlb-char = mlb-unescaped / escaped
	//mlb-quotes = 1*2quotation-mark
	//mlb-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	//mlb-escaped-nl = escape ws newline *( wschar / newline )

	token, rest, err := scanMultilineBasicString(b)
	if err != nil {
		return nil, nil, err
	}
	var builder bytes.Buffer

	i := 3

	// skip the immediate new line
	if token[i] == '\n' {
		i++
	} else if token[i] == '\r' && token[i+1] == '\n' {
		i += 2
	}

	// The scanner ensures that the token starts and ends with quotes and that
	// escapes are balanced.
	for ; i < len(token)-3; i++ {
		c := token[i]
		if c == '\\' {
			// When the last non-whitespace character on a line is an unescaped \,
			// it will be trimmed along with all whitespace (including newlines) up
			// to the next non-whitespace character or closing delimiter.
			if token[i+1] == '\n' || (token[i+1] == '\r' && token[i+2] == '\n') {
				i++ // skip the \
				for ; i < len(token)-3; i++ {
					c := token[i]
					if !(c == '\n' || c == '\r' || c == ' ' || c == '\t') {
						break
					}
				}
				continue
			}

			// handle escaping
			i++
			c = token[i]
			switch c {
			case '"', '\\':
				builder.WriteByte(c)
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case 'u':
				x, err := hexToString(token[i+3:len(token)-3], 4)
				if err != nil {
					return nil, nil, err
				}
				builder.WriteString(x)
				i += 4
			case 'U':
				x, err := hexToString(token[i+3:len(token)-3], 8)
				if err != nil {
					return nil, nil, err
				}
				builder.WriteString(x)
				i += 8
			default:
				return nil, nil, fmt.Errorf("invalid escaped character: %#U", c)
			}
		} else {
			builder.WriteByte(c)
		}
	}

	return builder.Bytes(), rest, nil
}

func (p *parser) parseKey(b []byte) ([]ast.Node, []byte, error) {
	//key = simple-key / dotted-key
	//simple-key = quoted-key / unquoted-key
	//
	//unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	//quoted-key = basic-string / literal-string
	//dotted-key = simple-key 1*( dot-sep simple-key )
	//
	//dot-sep   = ws %x2E ws  ; . Period

	var nodes []ast.Node

	key, b, err := p.parseSimpleKey(b)
	if err != nil {
		return nodes, nil, err
	}

	nodes = append(nodes, ast.Node{
		Kind: ast.Key,
		Data: key,
	})

	for {
		b = p.parseWhitespace(b)
		if len(b) > 0 && b[0] == '.' {
			b, err = expect('.', b)
			if err != nil {
				return nodes, nil, err
			}
			b = p.parseWhitespace(b)
			key, b, err = p.parseSimpleKey(b)
			if err != nil {
				return nodes, nil, err
			}
			nodes = append(nodes, ast.Node{
				Kind: ast.Key,
				Data: key,
			})
		} else {
			break
		}
	}

	return nodes, b, nil
}

func (p *parser) parseSimpleKey(b []byte) (key, rest []byte, err error) {
	//simple-key = quoted-key / unquoted-key
	//unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	//quoted-key = basic-string / literal-string

	if len(b) == 0 {
		return nil, nil, unexpectedCharacter{b: b}
	}

	if b[0] == '\'' {
		key, rest, err = scanLiteralString(b)
	} else if b[0] == '"' {
		key, rest, err = p.parseBasicString(b)
	} else if isUnquotedKeyChar(b[0]) {
		key, rest, err = scanUnquotedKey(b)
	} else {
		err = unexpectedCharacter{b: b}
	}
	return
}

func (p *parser) parseBasicString(b []byte) ([]byte, []byte, error) {
	//basic-string = quotation-mark *basic-char quotation-mark
	//quotation-mark = %x22            ; "
	//basic-char = basic-unescaped / escaped
	//basic-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	//escaped = escape escape-seq-char
	//escape-seq-char =  %x22         ; "    quotation mark  U+0022
	//escape-seq-char =/ %x5C         ; \    reverse solidus U+005C
	//escape-seq-char =/ %x62         ; b    backspace       U+0008
	//escape-seq-char =/ %x66         ; f    form feed       U+000C
	//escape-seq-char =/ %x6E         ; n    line feed       U+000A
	//escape-seq-char =/ %x72         ; r    carriage return U+000D
	//escape-seq-char =/ %x74         ; t    tab             U+0009
	//escape-seq-char =/ %x75 4HEXDIG ; uXXXX                U+XXXX
	//escape-seq-char =/ %x55 8HEXDIG ; UXXXXXXXX            U+XXXXXXXX

	token, rest, err := scanBasicString(b)
	if err != nil {
		return nil, nil, err
	}
	var builder bytes.Buffer

	// The scanner ensures that the token starts and ends with quotes and that
	// escapes are balanced.
	for i := 1; i < len(token)-1; i++ {
		c := token[i]
		if c == '\\' {
			i++
			c = token[i]
			switch c {
			case '"', '\\':
				builder.WriteByte(c)
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case 'u':
				x, err := hexToString(token[i+1:len(token)-1], 4)
				if err != nil {
					return nil, nil, err
				}
				builder.WriteString(x)
				i += 4
			case 'U':
				x, err := hexToString(token[i+1:len(token)-1], 8)
				if err != nil {
					return nil, nil, err
				}
				builder.WriteString(x)
				i += 8
			default:
				return nil, nil, fmt.Errorf("invalid escaped character: %#U", c)
			}
		} else {
			builder.WriteByte(c)
		}
	}

	return builder.Bytes(), rest, nil
}

func hexToString(b []byte, length int) (string, error) {
	if len(b) < length {
		return "", fmt.Errorf("unicode point needs %d hex characters", length)
	}
	// TODO: slow
	b, err := hex.DecodeString(string(b[:length]))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *parser) parseWhitespace(b []byte) []byte {
	//ws = *wschar
	//wschar =  %x20  ; Space
	//wschar =/ %x09  ; Horizontal tab

	_, rest := scanWhitespace(b)
	return rest
}

func (p *parser) parseIntOrFloatOrDateTime(b []byte) ([]byte, error) {
	switch b[0] {
	case 'i':
		if !scanFollowsInf(b) {
			return nil, fmt.Errorf("expected 'inf'")
		}
		//p.builder.FloatValue(math.Inf(1))
		// TODO
		return b[3:], nil
	case 'n':
		if !scanFollowsNan(b) {
			return nil, fmt.Errorf("expected 'nan'")
		}
		//p.builder.FloatValue(math.NaN())
		// TODO
		return b[3:], nil
	case '+', '-':
		return p.parseIntOrFloat(b)
	}

	if len(b) < 3 {
		return p.parseIntOrFloat(b)
	}
	s := 5
	if len(b) < s {
		s = len(b)
	}
	for idx, c := range b[:s] {
		if c >= '0' && c <= '9' {
			continue
		}
		if idx == 2 && c == ':' {
			return p.parseDateTime(b)
		}
		if idx == 4 && c == '-' {
			return p.parseDateTime(b)
		}
	}
	return p.parseIntOrFloat(b)
}

func digitsToInt(b []byte) int {
	x := 0
	for _, d := range b {
		x *= 10
		x += int(d - '0')
	}
	return x
}

func (p *parser) parseDateTime(b []byte) ([]byte, error) {
	// we know the first 2 ar digits.
	if b[2] == ':' {
		return p.parseTime(b)
	}
	// This state accepts an offset date-time, a local date-time, or a local date.
	//
	//   v--- cursor
	// 1979-05-27T07:32:00Z
	// 1979-05-27T00:32:00-07:00
	// 1979-05-27T00:32:00.999999-07:00
	// 1979-05-27 07:32:00Z
	// 1979-05-27 00:32:00-07:00
	// 1979-05-27 00:32:00.999999-07:00
	// 1979-05-27T07:32:00
	// 1979-05-27T00:32:00.999999
	// 1979-05-27 07:32:00
	// 1979-05-27 00:32:00.999999
	// 1979-05-27

	// date

	idx := 4

	localDate := toml.LocalDate{
		Year: digitsToInt(b[:idx]),
	}

	for i := 0; i < 2; i++ {
		// month
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid month digit in date: %c", b[idx])
		}
		localDate.Month *= 10
		localDate.Month += time.Month(b[idx] - '0')
	}

	idx++
	if b[idx] != '-' {
		return nil, fmt.Errorf("expected - to separate month of a date, not %c", b[idx])
	}

	for i := 0; i < 2; i++ {
		// day
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid day digit in date: %c", b[idx])
		}
		localDate.Day *= 10
		localDate.Day += int(b[idx] - '0')
	}

	idx++

	if idx >= len(b) {
		//p.builder.LocalDateValue(localDate)
		// TODO
		return nil, nil
	} else if b[idx] != ' ' && b[idx] != 'T' {
		//p.builder.LocalDateValue(localDate)
		// TODO
		return b[idx:], nil
	}

	// check if there is a chance there is anything useful after
	if b[idx] == ' ' && (((idx + 2) >= len(b)) || !isDigit(b[idx+1]) || !isDigit(b[idx+2])) {
		//p.builder.LocalDateValue(localDate)
		// TODO
		return b[idx:], nil
	}

	//idx++ // skip the T or ' '

	// time
	localTime := toml.LocalTime{}

	for i := 0; i < 2; i++ {
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid hour digit in time: %c", b[idx])
		}
		localTime.Hour *= 10
		localTime.Hour += int(b[idx] - '0')
	}

	idx++
	if b[idx] != ':' {
		return nil, fmt.Errorf("time hour/minute separator should be :, not %c", b[idx])
	}

	for i := 0; i < 2; i++ {
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid minute digit in time: %c", b[idx])
		}
		localTime.Minute *= 10
		localTime.Minute += int(b[idx] - '0')
	}

	idx++
	if b[idx] != ':' {
		return nil, fmt.Errorf("time minute/second separator should be :, not %c", b[idx])
	}

	for i := 0; i < 2; i++ {
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid second digit in time: %c", b[idx])
		}
		localTime.Second *= 10
		localTime.Second += int(b[idx] - '0')
	}

	idx++
	if idx < len(b) && b[idx] == '.' {
		idx++
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("expected at least one digit in time's fraction, not %c", b[idx])
		}

		for {
			localTime.Nanosecond *= 10
			localTime.Nanosecond += int(b[idx] - '0')
			idx++

			if idx < len(b) {
				break
			}

			if !isDigit(b[idx]) {
				break
			}
		}
	}

	if idx >= len(b) || (b[idx] != 'Z' && b[idx] != '+' && b[idx] != '-') {
		dt := toml.LocalDateTime{
			Date: localDate,
			Time: localTime,
		}
		//p.builder.LocalDateTimeValue(dt)
		// TODO
		dt = dt
		return b[idx:], nil
	}

	loc := time.UTC

	if b[idx] == 'Z' {
		idx++
	} else {
		start := idx
		sign := 1
		if b[idx] == '-' {
			sign = -1
		}

		hours := 0
		for i := 0; i < 2; i++ {
			idx++
			if !isDigit(b[idx]) {
				return nil, fmt.Errorf("invalid hour digit in time offset: %c", b[idx])
			}
			hours *= 10
			hours += int(b[idx] - '0')
		}
		offset := hours * 60 * 60

		idx++
		if b[idx] != ':' {
			return nil, fmt.Errorf("time offset hour/minute separator should be :, not %c", b[idx])
		}

		minutes := 0
		for i := 0; i < 2; i++ {
			idx++
			if !isDigit(b[idx]) {
				return nil, fmt.Errorf("invalid minute digit in time offset: %c", b[idx])
			}
			minutes *= 10
			minutes += int(b[idx] - '0')
		}
		offset += minutes * 60
		offset *= sign
		idx++
		loc = time.FixedZone(string(b[start:idx]), offset)
	}
	dt := time.Date(localDate.Year, localDate.Month, localDate.Day, localTime.Hour, localTime.Minute, localTime.Second, localTime.Nanosecond, loc)
	//p.builder.DateTimeValue(dt)
	// TODO
	dt = dt
	return b[idx:], nil
}

func (p *parser) parseTime(b []byte) ([]byte, error) {
	localTime := toml.LocalTime{}

	idx := 0

	for i := 0; i < 2; i++ {
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid hour digit in time: %c", b[idx])
		}
		localTime.Hour *= 10
		localTime.Hour += int(b[idx] - '0')
	}

	idx++
	if b[idx] != ':' {
		return nil, fmt.Errorf("time hour/minute separator should be :, not %c", b[idx])
	}

	for i := 0; i < 2; i++ {
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid minute digit in time: %c", b[idx])
		}
		localTime.Minute *= 10
		localTime.Minute += int(b[idx] - '0')
	}

	idx++
	if b[idx] != ':' {
		return nil, fmt.Errorf("time minute/second separator should be :, not %c", b[idx])
	}

	for i := 0; i < 2; i++ {
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("invalid second digit in time: %c", b[idx])
		}
		localTime.Second *= 10
		localTime.Second += int(b[idx] - '0')
	}

	idx++
	if idx < len(b) && b[idx] == '.' {
		idx++
		idx++
		if !isDigit(b[idx]) {
			return nil, fmt.Errorf("expected at least one digit in time's fraction, not %c", b[idx])
		}

		for {
			localTime.Nanosecond *= 10
			localTime.Nanosecond += int(b[idx] - '0')
			idx++
			if !isDigit(b[idx]) {
				break
			}
		}
	}

	//p.builder.LocalTimeValue(localTime)
	// TODO
	return b[idx:], nil
}

func (p *parser) parseIntOrFloat(b []byte) ([]byte, error) {
	i := 0
	r := b[0]
	if r == '0' {
		if len(b) >= 2 {
			var isValidRune validRuneFn
			var parseFn func([]byte) (int64, error)
			switch b[1] {
			case 'x':
				isValidRune = isValidHexRune
				parseFn = parseIntHex
			case 'o':
				isValidRune = isValidOctalRune
				parseFn = parseIntOct
			case 'b':
				isValidRune = isValidBinaryRune
				parseFn = parseIntBin
			default:
				if b[1] >= 'a' && b[1] <= 'z' || b[1] >= 'A' && b[1] <= 'Z' {
					return nil, fmt.Errorf("unknown number base: %s. possible options are x (hex) o (octal) b (binary)", string(b[1]))
				}
				parseFn = parseIntDec
			}

			if isValidRune != nil {
				i = 2
				digitSeen := false
				for {
					if !isValidRune(b[i]) {
						break
					}
					digitSeen = true
					i++
				}

				if !digitSeen {
					return nil, fmt.Errorf("number needs at least one digit")
				}

				v, err := parseFn(b[:i])
				if err != nil {
					return nil, err
				}
				//p.builder.IntValue(v)
				// TODO
				v = v
				return b[i:], nil
			}
		}
	}

	if r == '+' || r == '-' {
		b = b[1:]
		if scanFollowsInf(b) {
			if r == '+' {
				//p.builder.FloatValue(plusInf)
				// TODO
			} else {
				//p.builder.FloatValue(minusInf)
				// TODO
			}
			return b, nil
		}
		if scanFollowsNan(b) {
			//p.builder.FloatValue(nan)
			// TODO
			return b, nil
		}
	}

	pointSeen := false
	expSeen := false
	digitSeen := false
	for i < len(b) {
		next := b[i]
		if next == '.' {
			if pointSeen {
				return nil, fmt.Errorf("cannot have two dots in one float")
			}
			i++
			if i < len(b) && !isDigit(b[i]) {
				return nil, fmt.Errorf("float cannot end with a dot")
			}
			pointSeen = true
		} else if next == 'e' || next == 'E' {
			expSeen = true
			i++
			if i >= len(b) {
				break
			}
			if b[i] == '+' || b[i] == '-' {
				i++
			}
		} else if isDigit(next) {
			digitSeen = true
			i++
		} else if next == '_' {
			i++
		} else {
			break
		}
		if pointSeen && !digitSeen {
			return nil, fmt.Errorf("cannot start float with a dot")
		}
	}

	if !digitSeen {
		return nil, fmt.Errorf("no digit in that number")
	}
	if pointSeen || expSeen {
		f, err := parseFloat(b[:i])
		if err != nil {
			return nil, err
		}
		//p.builder.FloatValue(f)
		// TODO
		f = f
	} else {
		v, err := parseIntDec(b[:i])
		if err != nil {
			return nil, err
		}
		//p.builder.IntValue(v)
		// TODO
		v = v
	}
	return b[i:], nil
}

func parseFloat(b []byte) (float64, error) {
	// TODO: inefficient
	tok := string(b)
	err := numberContainsInvalidUnderscore(tok)
	if err != nil {
		return 0, err
	}
	cleanedVal := cleanupNumberToken(tok)
	return strconv.ParseFloat(cleanedVal, 64)
}

func parseIntHex(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)
	err := hexNumberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, nil
	}
	return strconv.ParseInt(cleanedVal[2:], 16, 64)
}

func parseIntOct(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)
	err := numberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(cleanedVal[2:], 8, 64)
}

func parseIntBin(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)
	err := numberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(cleanedVal[2:], 2, 64)
}

func parseIntDec(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)
	err := numberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(cleanedVal, 10, 64)
}

func numberContainsInvalidUnderscore(value string) error {
	// For large numbers, you may use underscores between digits to enhance
	// readability. Each underscore must be surrounded by at least one digit on
	// each side.

	hasBefore := false
	for idx, r := range value {
		if r == '_' {
			if !hasBefore || idx+1 >= len(value) {
				// can't end with an underscore
				return errInvalidUnderscore
			}
		}
		hasBefore = isDigitRune(r)
	}
	return nil
}

func hexNumberContainsInvalidUnderscore(value string) error {
	hasBefore := false
	for idx, r := range value {
		if r == '_' {
			if !hasBefore || idx+1 >= len(value) {
				// can't end with an underscore
				return errInvalidUnderscoreHex
			}
		}
		hasBefore = isHexDigit(r)
	}
	return nil
}

func cleanupNumberToken(value string) string {
	cleanedVal := strings.Replace(value, "_", "", -1)
	return cleanedVal
}

func isDigit(r byte) bool {
	return r >= '0' && r <= '9'
}

func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

var plusInf = math.Inf(1)
var minusInf = math.Inf(-1)
var nan = math.NaN()

type validRuneFn func(r byte) bool

func isValidHexRune(r byte) bool {
	return r >= 'a' && r <= 'f' ||
		r >= 'A' && r <= 'F' ||
		r >= '0' && r <= '9' ||
		r == '_'
}

func isHexDigit(r rune) bool {
	return isDigitRune(r) ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

func isValidOctalRune(r byte) bool {
	return r >= '0' && r <= '7' || r == '_'
}

func isValidBinaryRune(r byte) bool {
	return r == '0' || r == '1' || r == '_'
}

func expect(x byte, b []byte) ([]byte, error) {
	if len(b) == 0 || b[0] != x {
		return nil, unexpectedCharacter{r: x, b: b}
	}
	return b[1:], nil
}

type unexpectedCharacter struct {
	r byte
	b []byte
}

func (u unexpectedCharacter) Error() string {
	if len(u.b) == 0 {
		return fmt.Sprintf("expected %#U, not EOF", u.r)

	}
	return fmt.Sprintf("expected %#U, not %#U", u.r, u.b[0])
}

var errInvalidUnderscore = errors.New("invalid use of _ in number")
var errInvalidUnderscoreHex = errors.New("invalid use of _ in hex number")
