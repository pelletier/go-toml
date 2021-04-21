package toml

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func parseInteger(b []byte) (int64, error) {
	if len(b) > 2 && b[0] == '0' {
		switch b[1] {
		case 'x':
			return parseIntHex(b)
		case 'b':
			return parseIntBin(b)
		case 'o':
			return parseIntOct(b)
		default:
			return 0, newDecodeError(b[1:2], "invalid base: '%c'", b[1])
		}
	}

	return parseIntDec(b)
}

func parseLocalDate(b []byte) (LocalDate, error) {
	// full-date      = date-fullyear "-" date-month "-" date-mday
	// date-fullyear  = 4DIGIT
	// date-month     = 2DIGIT  ; 01-12
	// date-mday      = 2DIGIT  ; 01-28, 01-29, 01-30, 01-31 based on month/year
	var date LocalDate

	if len(b) != 10 || b[4] != '-' || b[7] != '-' {
		return date, newDecodeError(b, "dates are expected to have the format YYYY-MM-DD")
	}

	var err error

	date.Year, err = parseDecimalDigits(b[0:4])
	if err != nil {
		return date, err
	}

	v, err := parseDecimalDigits(b[5:7])
	if err != nil {
		return date, err
	}

	date.Month = time.Month(v)

	date.Day, err = parseDecimalDigits(b[8:10])
	if err != nil {
		return date, err
	}

	return date, nil
}

var errNotDigit = errors.New("not a digit")

func parseDecimalDigits(b []byte) (int, error) {
	v := 0

	for _, c := range b {
		if !isDigit(c) {
			return 0, fmt.Errorf("%s: %w", b, errNotDigit)
		}

		v *= 10
		v += int(c - '0')
	}

	return v, nil
}

var errParseDateTimeMissingInfo = errors.New("date-time missing timezone information")

func parseDateTime(b []byte) (time.Time, error) {
	// offset-date-time = full-date time-delim full-time
	// full-time      = partial-time time-offset
	// time-offset    = "Z" / time-numoffset
	// time-numoffset = ( "+" / "-" ) time-hour ":" time-minute
	dt, b, err := parseLocalDateTime(b)
	if err != nil {
		return time.Time{}, err
	}

	var zone *time.Location

	if len(b) == 0 {
		return time.Time{}, errParseDateTimeMissingInfo
	}

	if b[0] == 'Z' {
		b = b[1:]
		zone = time.UTC
	} else {
		const dateTimeByteLen = 6
		if len(b) != dateTimeByteLen {
			return time.Time{}, newDecodeError(b, "invalid date-time timezone")
		}
		direction := 1
		switch b[0] {
		case '+':
		case '-':
			direction = -1
		default:
			return time.Time{}, newDecodeError(b[0:1], "invalid timezone offset character")
		}

		hours := digitsToInt(b[1:3])
		minutes := digitsToInt(b[4:6])
		seconds := direction * (hours*3600 + minutes*60)
		zone = time.FixedZone("", seconds)
	}

	if len(b) > 0 {
		return time.Time{}, newDecodeError(b, "extra bytes at the end of the timezone")
	}

	t := time.Date(
		dt.Date.Year,
		dt.Date.Month,
		dt.Date.Day,
		dt.Time.Hour,
		dt.Time.Minute,
		dt.Time.Second,
		dt.Time.Nanosecond,
		zone)

	return t, nil
}

var (
	errParseLocalDateTimeWrongLength = errors.New(
		"local datetimes are expected to have the format YYYY-MM-DDTHH:MM:SS[.NNNNNNNNN]",
	)
	errParseLocalDateTimeWrongSeparator = errors.New("datetime separator is expected to be T or a space")
)

func parseLocalDateTime(b []byte) (LocalDateTime, []byte, error) {
	var dt LocalDateTime

	const localDateTimeByteMinLen = 11
	if len(b) < localDateTimeByteMinLen {
		return dt, nil, errParseLocalDateTimeWrongLength
	}

	date, err := parseLocalDate(b[:10])
	if err != nil {
		return dt, nil, err
	}
	dt.Date = date

	sep := b[10]
	if sep != 'T' && sep != ' ' {
		return dt, nil, errParseLocalDateTimeWrongSeparator
	}

	t, rest, err := parseLocalTime(b[11:])
	if err != nil {
		return dt, nil, err
	}
	dt.Time = t

	return dt, rest, nil
}

var errParseLocalTimeWrongLength = errors.New("times are expected to have the format HH:MM:SS[.NNNNNN]")

// parseLocalTime is a bit different because it also returns the remaining
// []byte that is didn't need. This is to allow parseDateTime to parse those
// remaining bytes as a timezone.
//nolint:cyclop,funlen
func parseLocalTime(b []byte) (LocalTime, []byte, error) {
	var (
		nspow = [10]int{0, 1e8, 1e7, 1e6, 1e5, 1e4, 1e3, 1e2, 1e1, 1e0}
		t     LocalTime
	)

	const localTimeByteLen = 8
	if len(b) < localTimeByteLen {
		return t, nil, errParseLocalTimeWrongLength
	}

	var err error

	t.Hour, err = parseDecimalDigits(b[0:2])
	if err != nil {
		return t, nil, err
	}

	if b[2] != ':' {
		return t, nil, newDecodeError(b[2:3], "expecting colon between hours and minutes")
	}

	t.Minute, err = parseDecimalDigits(b[3:5])
	if err != nil {
		return t, nil, err
	}

	if b[5] != ':' {
		return t, nil, newDecodeError(b[5:6], "expecting colon between minutes and seconds")
	}

	t.Second, err = parseDecimalDigits(b[6:8])
	if err != nil {
		return t, nil, err
	}

	if len(b) >= 9 && b[8] == '.' {
		frac := 0
		digits := 0

		for i, c := range b[9:] {
			if !isDigit(c) {
				if i == 0 {
					return t, nil, newDecodeError(b[i:i+1], "need at least one digit after fraction point")
				}

				break
			}

			if i >= 9 {
				return t, nil, newDecodeError(b[i:i+1], "maximum precision for date time is nanosecond")
			}

			frac *= 10
			frac += int(c - '0')
			digits++
		}

		t.Nanosecond = frac * nspow[digits]

		return t, b[9+digits:], nil
	}

	return t, b[8:], nil
}

var (
	errParseFloatStartDot = errors.New("float cannot start with a dot")
	errParseFloatEndDot   = errors.New("float cannot end with a dot")
)

//nolint:cyclop
func parseFloat(b []byte) (float64, error) {
	//nolint:godox
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
	if cleanedVal[0] == '.' {
		return 0, errParseFloatStartDot
	}

	if cleanedVal[len(cleanedVal)-1] == '.' {
		return 0, errParseFloatEndDot
	}

	f, err := strconv.ParseFloat(cleanedVal, 64)
	if err != nil {
		return 0, fmt.Errorf("coudn't ParseFloat %w", err)
	}

	return f, nil
}

func parseIntHex(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)

	err := hexNumberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(cleanedVal[2:], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("coudn't ParseIntHex %w", err)
	}

	return i, nil
}

func parseIntOct(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)

	err := numberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(cleanedVal[2:], 8, 64)
	if err != nil {
		return 0, fmt.Errorf("coudn't ParseIntOct %w", err)
	}

	return i, nil
}

func parseIntBin(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)

	err := numberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(cleanedVal[2:], 2, 64)
	if err != nil {
		return 0, fmt.Errorf("coudn't ParseIntBin %w", err)
	}

	return i, nil
}

func parseIntDec(b []byte) (int64, error) {
	tok := string(b)
	cleanedVal := cleanupNumberToken(tok)

	err := numberContainsInvalidUnderscore(cleanedVal)
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(cleanedVal, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("coudn't parseIntDec %w", err)
	}

	return i, nil
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
	cleanedVal := strings.ReplaceAll(value, "_", "")

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

var (
	errInvalidUnderscore    = errors.New("invalid use of _ in number")
	errInvalidUnderscoreHex = errors.New("invalid use of _ in hex number")
)
