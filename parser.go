package toml

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pelletier/go-toml/v2/internal/ast"
)

type parser struct {
	builder ast.Builder
}

func (p *parser) parse(b []byte) error {
	last, b, err := p.parseExpression(b)
	if err != nil {
		return err
	}
	for len(b) > 0 {
		b, err = p.parseNewline(b)
		if err != nil {
			return err
		}

		var next ast.Reference
		next, b, err = p.parseExpression(b)
		if err != nil {
			return err
		}
		if next.Valid() {
			p.builder.Chain(last, next)
			last = next
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

func (p *parser) parseExpression(b []byte) (ast.Reference, []byte, error) {
	//expression =  ws [ comment ]
	//expression =/ ws keyval ws [ comment ]
	//expression =/ ws table ws [ comment ]

	var ref ast.Reference

	b = p.parseWhitespace(b)

	if len(b) == 0 {
		return ref, b, nil
	}

	if b[0] == '#' {
		_, rest, err := scanComment(b)
		return ref, rest, err
	}
	if b[0] == '\n' || b[0] == '\r' {
		return ref, b, nil
	}

	var err error
	if b[0] == '[' {
		ref, b, err = p.parseTable(b)
	} else {
		ref, b, err = p.parseKeyval(b)
	}
	if err != nil {
		return ref, nil, err
	}

	b = p.parseWhitespace(b)

	if len(b) > 0 && b[0] == '#' {
		_, rest, err := scanComment(b)
		return ref, rest, err
	}

	return ref, b, nil
}

func (p *parser) parseTable(b []byte) (ast.Reference, []byte, error) {
	//table = std-table / array-table
	if len(b) > 1 && b[1] == '[' {
		return p.parseArrayTable(b)
	}
	return p.parseStdTable(b)
}

func (p *parser) parseArrayTable(b []byte) (ast.Reference, []byte, error) {
	//array-table = array-table-open key array-table-close
	//array-table-open  = %x5B.5B ws  ; [[ Double left square bracket
	//array-table-close = ws %x5D.5D  ; ]] Double right square bracket

	ref := p.builder.Push(ast.Node{
		Kind: ast.ArrayTable,
	})

	b = b[2:]
	b = p.parseWhitespace(b)
	k, b, err := p.parseKey(b)
	if err != nil {
		return ref, nil, err
	}
	p.builder.AttachChild(ref, k)
	b = p.parseWhitespace(b)
	b, err = expect(']', b)
	if err != nil {
		return ref, nil, err
	}
	b, err = expect(']', b)
	return ref, b, err
}

func (p *parser) parseStdTable(b []byte) (ast.Reference, []byte, error) {
	//std-table = std-table-open key std-table-close
	//std-table-open  = %x5B ws     ; [ Left square bracket
	//std-table-close = ws %x5D     ; ] Right square bracket

	ref := p.builder.Push(ast.Node{
		Kind: ast.Table,
	})

	b = b[1:]
	b = p.parseWhitespace(b)
	key, b, err := p.parseKey(b)
	if err != nil {
		return ref, nil, err
	}

	p.builder.AttachChild(ref, key)

	b = p.parseWhitespace(b)

	b, err = expect(']', b)

	return ref, b, err
}

func (p *parser) parseKeyval(b []byte) (ast.Reference, []byte, error) {
	//keyval = key keyval-sep val

	ref := p.builder.Push(ast.Node{
		Kind: ast.KeyValue,
	})

	key, b, err := p.parseKey(b)
	if err != nil {
		return ast.Reference{}, nil, err
	}

	//keyval-sep = ws %x3D ws ; =

	b = p.parseWhitespace(b)
	b, err = expect('=', b)
	if err != nil {
		return ast.Reference{}, nil, err
	}
	b = p.parseWhitespace(b)

	valRef, b, err := p.parseVal(b)
	if err != nil {
		return ref, b, err
	}
	p.builder.Chain(valRef, key)
	p.builder.AttachChild(ref, valRef)

	return ref, b, err
}

func (p *parser) parseVal(b []byte) (ast.Reference, []byte, error) {
	// val = string / boolean / array / inline-table / date-time / float / integer
	var ref ast.Reference

	if len(b) == 0 {
		return ref, nil, fmt.Errorf("expected value, not eof")
	}

	var err error
	c := b[0]

	switch c {
	case '"':
		var v []byte
		if scanFollowsMultilineBasicStringDelimiter(b) {
			v, b, err = p.parseMultilineBasicString(b)
		} else {
			v, b, err = p.parseBasicString(b)
		}
		if err == nil {
			ref = p.builder.Push(ast.Node{
				Kind: ast.String,
				Data: v,
			})
		}
		return ref, b, err
	case '\'':
		var v []byte
		if scanFollowsMultilineLiteralStringDelimiter(b) {
			v, b, err = p.parseMultilineLiteralString(b)
		} else {
			v, b, err = p.parseLiteralString(b)
		}
		if err == nil {
			ref = p.builder.Push(ast.Node{
				Kind: ast.String,
				Data: v,
			})
		}
		return ref, b, err
	case 't':
		if !scanFollowsTrue(b) {
			return ref, nil, fmt.Errorf("expected 'true'")
		}
		ref = p.builder.Push(ast.Node{
			Kind: ast.Bool,
			Data: b[:4],
		})
		return ref, b[4:], nil
	case 'f':
		if !scanFollowsFalse(b) {
			return ast.Reference{}, nil, fmt.Errorf("expected 'false'")
		}
		ref = p.builder.Push(ast.Node{
			Kind: ast.Bool,
			Data: b[:5],
		})
		return ref, b[5:], nil
	case '[':
		return p.parseValArray(b)
	case '{':
		return p.parseInlineTable(b)
	default:
		return p.parseIntOrFloatOrDateTime(b)
	}
}

func (p *parser) parseLiteralString(b []byte) ([]byte, []byte, error) {
	v, rest, err := scanLiteralString(b)
	if err != nil {
		return nil, nil, err
	}
	return v[1 : len(v)-1], rest, nil
}

func (p *parser) parseInlineTable(b []byte) (ast.Reference, []byte, error) {
	//inline-table = inline-table-open [ inline-table-keyvals ] inline-table-close
	//inline-table-open  = %x7B ws     ; {
	//inline-table-close = ws %x7D     ; }
	//inline-table-sep   = ws %x2C ws  ; , Comma
	//inline-table-keyvals = keyval [ inline-table-sep inline-table-keyvals ]

	parent := p.builder.Push(ast.Node{
		Kind: ast.InlineTable,
	})

	first := true
	var child ast.Reference

	b = b[1:]

	var err error
	for len(b) > 0 {
		b = p.parseWhitespace(b)
		if b[0] == '}' {
			break
		}

		if !first {
			b, err = expect(',', b)
			if err != nil {
				return parent, nil, err
			}
			b = p.parseWhitespace(b)
		}
		var kv ast.Reference
		kv, b, err = p.parseKeyval(b)
		if err != nil {
			return parent, nil, err
		}

		if first {
			p.builder.AttachChild(parent, kv)
			first = false
		} else {
			p.builder.Chain(child, kv)
		}
		child = kv

		first = false
	}

	rest, err := expect('}', b)
	return parent, rest, err
}

func (p *parser) parseValArray(b []byte) (ast.Reference, []byte, error) {
	//array = array-open [ array-values ] ws-comment-newline array-close
	//array-open =  %x5B ; [
	//array-close = %x5D ; ]
	//array-values =  ws-comment-newline val ws-comment-newline array-sep array-values
	//array-values =/ ws-comment-newline val ws-comment-newline [ array-sep ]
	//array-sep = %x2C  ; , Comma
	//ws-comment-newline = *( wschar / [ comment ] newline )

	b = b[1:]

	parent := p.builder.Push(ast.Node{
		Kind: ast.Array,
	})

	first := true
	var lastChild ast.Reference

	var err error
	for len(b) > 0 {
		b, err = p.parseOptionalWhitespaceCommentNewline(b)
		if err != nil {
			return parent, nil, err
		}

		if len(b) == 0 {
			return parent, nil, unexpectedCharacter{b: b}
		}

		if b[0] == ']' {
			break
		}
		if b[0] == ',' {
			if first {
				return parent, nil, fmt.Errorf("array cannot start with comma")
			}
			b = b[1:]
			b, err = p.parseOptionalWhitespaceCommentNewline(b)
			if err != nil {
				return parent, nil, err
			}
		}

		var valueRef ast.Reference
		valueRef, b, err = p.parseVal(b)
		if err != nil {
			return parent, nil, err
		}

		if first {
			p.builder.AttachChild(parent, valueRef)
			first = false
		} else {
			p.builder.Chain(lastChild, valueRef)
		}
		lastChild = valueRef

		b, err = p.parseOptionalWhitespaceCommentNewline(b)
		if err != nil {
			return parent, nil, err
		}
		first = false
	}

	rest, err := expect(']', b)
	return parent, rest, err
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

func (p *parser) parseKey(b []byte) (ast.Reference, []byte, error) {
	//key = simple-key / dotted-key
	//simple-key = quoted-key / unquoted-key
	//
	//unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	//quoted-key = basic-string / literal-string
	//dotted-key = simple-key 1*( dot-sep simple-key )
	//
	//dot-sep   = ws %x2E ws  ; . Period

	key, b, err := p.parseSimpleKey(b)
	if err != nil {
		return ast.Reference{}, nil, err
	}

	ref := p.builder.Push(ast.Node{
		Kind: ast.Key,
		Data: key,
	})

	for {
		b = p.parseWhitespace(b)
		if len(b) > 0 && b[0] == '.' {
			b, err = expect('.', b)
			if err != nil {
				return ref, nil, err
			}
			b = p.parseWhitespace(b)
			key, b, err = p.parseSimpleKey(b)
			if err != nil {
				return ref, nil, err
			}
			p.builder.PushAndChain(ast.Node{
				Kind: ast.Key,
				Data: key,
			})
		} else {
			break
		}
	}

	return ref, b, nil
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

func (p *parser) parseIntOrFloatOrDateTime(b []byte) (ast.Reference, []byte, error) {
	switch b[0] {
	case 'i':
		if !scanFollowsInf(b) {
			return ast.Reference{}, nil, fmt.Errorf("expected 'inf'")
		}
		return p.builder.Push(ast.Node{
			Kind: ast.Float,
			Data: b[:3],
		}), b[3:], nil
	case 'n':
		if !scanFollowsNan(b) {
			return ast.Reference{}, nil, fmt.Errorf("expected 'nan'")
		}
		return p.builder.Push(ast.Node{
			Kind: ast.Float,
			Data: b[:3],
		}), b[3:], nil
	case '+', '-':
		return p.scanIntOrFloat(b)
	}

	if len(b) < 3 {
		return p.scanIntOrFloat(b)
	}
	s := 5
	if len(b) < s {
		s = len(b)
	}
	for idx, c := range b[:s] {
		if isDigit(c) {
			continue
		}
		if idx == 2 && c == ':' || (idx == 4 && c == '-') {
			return p.scanDateTime(b)
		}
	}
	return p.scanIntOrFloat(b)
}

func digitsToInt(b []byte) int {
	x := 0
	for _, d := range b {
		x *= 10
		x += int(d - '0')
	}
	return x
}

func (p *parser) scanDateTime(b []byte) (ast.Reference, []byte, error) {
	// scans for contiguous characters in [0-9T:Z.+-], and up to one space if
	// followed by a digit.

	hasTime := false
	hasTz := false
	seenSpace := false

	i := 0
	for ; i < len(b); i++ {
		c := b[i]
		if isDigit(c) || c == '-' {
		} else if c == 'T' || c == ':' || c == '.' {
			hasTime = true
			continue
		} else if c == '+' || c == '-' || c == 'Z' {
			hasTz = true
		} else if c == ' ' {
			if !seenSpace && i+1 < len(b) && isDigit(b[i+1]) {
				i += 2
				seenSpace = true
				hasTime = true
			} else {
				break
			}
		} else {
			break
		}
	}

	var kind ast.Kind

	if hasTime {
		if hasTz {
			kind = ast.DateTime
		} else {
			kind = ast.LocalDateTime
		}
	} else {
		if hasTz {
			return ast.Reference{}, nil, fmt.Errorf("possible DateTime cannot have a timezone but no time component")
		}
		kind = ast.LocalDate
	}

	return p.builder.Push(ast.Node{
		Kind: kind,
		Data: b[:i],
	}), b[i:], nil
}

func (p *parser) parseDateTime(b []byte) ([]byte, error) {
	// we know the first 2 are digits.
	if b[2] == ':' {
		return p.parseTime(b)
	}
	// This state accepts an offset date-time, a local date-time, or a local date.
	//
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

	localDate := LocalDate{
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
	localTime := LocalTime{}

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
		dt := LocalDateTime{
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
	localTime := LocalTime{}

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

func (p *parser) scanIntOrFloat(b []byte) (ast.Reference, []byte, error) {
	i := 0

	if len(b) > 2 && b[0] == '0' {
		var isValidRune validRuneFn
		switch b[1] {
		case 'x':
			isValidRune = isValidHexRune
		case 'o':
			isValidRune = isValidOctalRune
		case 'b':
			isValidRune = isValidBinaryRune
		default:
			i++
		}

		if isValidRune != nil {
			i += 2
			for ; i < len(b); i++ {
				if !isValidRune(b[i]) {
					break
				}
			}
		}

		return p.builder.Push(ast.Node{
			Kind: ast.Integer,
			Data: b[:i],
		}), b[i:], nil
	}

	isFloat := false

	for ; i < len(b); i++ {
		c := b[i]

		if c >= '0' && c <= '9' || c == '+' || c == '-' || c == '_' {
			continue
		}

		if c == '.' || c == 'e' || c == 'E' {
			isFloat = true
			continue
		}

		if c == 'i' {
			if scanFollowsInf(b[i:]) {
				return p.builder.Push(ast.Node{
					Kind: ast.Float,
					Data: b[:i+3],
				}), b[i+3:], nil
			}
			return ast.Reference{}, nil, fmt.Errorf("unexpected character i while scanning for a number")
		}
		if c == 'n' {
			if scanFollowsNan(b[i:]) {
				return p.builder.Push(ast.Node{
					Kind: ast.Float,
					Data: b[:i+3],
				}), b[i+3:], nil
			}
			return ast.Reference{}, nil, fmt.Errorf("unexpected character n while scanning for a number")
		}

		break
	}

	kind := ast.Integer

	if isFloat {
		kind = ast.Float
	}

	return p.builder.Push(ast.Node{
		Kind: kind,
		Data: b[:i],
	}), b[i:], nil
}

func isDigit(r byte) bool {
	return r >= '0' && r <= '9'
}

type validRuneFn func(r byte) bool

func isValidHexRune(r byte) bool {
	return r >= 'a' && r <= 'f' ||
		r >= 'A' && r <= 'F' ||
		r >= '0' && r <= '9' ||
		r == '_'
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
