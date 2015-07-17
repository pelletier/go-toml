// Parsing keys handling both bare and quoted keys.

package toml

import (
	"bytes"
	"fmt"
	"unicode"
)

func parseKey(key string) ([]string, error) {
	groups := []string{}
	var buffer bytes.Buffer
	inQuotes := false
	escapeNext := false
	for _, char := range key {
		if escapeNext {
			buffer.WriteRune(char)
			escapeNext = false
			continue
		}
		switch char {
		case '\\':
			escapeNext = true
			continue
		case '"':
			inQuotes = !inQuotes
		case '.':
			if inQuotes {
				buffer.WriteRune(char)
			} else {
				groups = append(groups, buffer.String())
				buffer.Reset()
			}
		default:
			if !inQuotes && !isValidBareChar(char) {
				return nil, fmt.Errorf("invalid bare character: %c", char)
			}
			buffer.WriteRune(char)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("mismatched quotes")
	}
	if escapeNext {
		return nil, fmt.Errorf("unfinished escape sequence")
	}
	if buffer.Len() > 0 {
		groups = append(groups, buffer.String())
	}
	return groups, nil
}

func isValidBareChar(r rune) bool {
	return isAlphanumeric(r) || r == '-' || unicode.IsNumber(r)
}
