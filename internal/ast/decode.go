package ast

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

func parseFloat(b []byte) (float64, error) {
	// TODO: inefficient
	if len(b) == 4 && (b[0] == '+' || b[0] == '-') && b[1] == 'n' && b[2] == 'a' && b[3] == 'n' {
		return math.NaN(), nil
	}

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

func isHexDigit(r rune) bool {
	return isDigitRune(r) ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

var errInvalidUnderscore = errors.New("invalid use of _ in number")
var errInvalidUnderscoreHex = errors.New("invalid use of _ in hex number")
