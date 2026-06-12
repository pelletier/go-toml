package toml

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2/unstable"
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
			panic(fmt.Errorf("invalid base '%c', should have been checked by scanIntOrFloat", b[1]))
		}
	}
	return parseIntDec(b)
}

func parseIntHex(b []byte) (int64, error) {
	var v uint64
	for _, c := range b[2:] {
		if c == '_' {
			continue
		}
		var d byte
		switch {
		case c >= '0' && c <= '9':
			d = c - '0'
		case c >= 'a' && c <= 'f':
			d = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			d = c - 'A' + 10
		}
		if v > math.MaxInt64>>4 {
			return 0, unstable.NewParserError(b, "hexadecimal number is too large to fit in a 64-bit signed integer")
		}
		v = v<<4 | uint64(d)
	}
	return int64(v), nil
}

func parseIntOct(b []byte) (int64, error) {
	var v uint64
	for _, c := range b[2:] {
		if c == '_' {
			continue
		}
		if v > math.MaxInt64>>3 {
			return 0, unstable.NewParserError(b, "octal number is too large to fit in a 64-bit signed integer")
		}
		v = v<<3 | uint64(c-'0')
	}
	return int64(v), nil
}

func parseIntBin(b []byte) (int64, error) {
	var v uint64
	for _, c := range b[2:] {
		if c == '_' {
			continue
		}
		if v > math.MaxInt64>>1 {
			return 0, unstable.NewParserError(b, "binary number is too large to fit in a 64-bit signed integer")
		}
		v = v<<1 | uint64(c-'0')
	}
	return int64(v), nil
}

func parseIntDec(b []byte) (int64, error) {
	i := 0
	neg := false
	switch b[0] {
	case '-':
		neg = true
		i++
	case '+':
		i++
	}

	var limit uint64 = math.MaxInt64
	if neg {
		limit = math.MaxInt64 + 1
	}

	var v uint64
	for ; i < len(b); i++ {
		c := b[i]
		if c == '_' {
			continue
		}
		if v > limit/10 {
			return 0, unstable.NewParserError(b, "decimal number is too large to fit in a 64-bit signed integer")
		}
		v = v*10 + uint64(c-'0')
		if v > limit {
			return 0, unstable.NewParserError(b, "decimal number is too large to fit in a 64-bit signed integer")
		}
	}
	if neg {
		return -int64(v), nil //nolint:gosec // v <= MaxInt64+1, the conversion wraps to the intended negative value
	}
	return int64(v), nil //nolint:gosec // v <= MaxInt64
}

func parseFloat(b []byte) (float64, error) {
	i := 0
	if len(b) > 0 && (b[0] == '+' || b[0] == '-') {
		i = 1
	}
	if len(b) == i+3 {
		switch b[i] {
		case 'i':
			// inf
			if b[0] == '-' {
				return math.Inf(-1), nil
			}
			return math.Inf(1), nil
		case 'n':
			// nan
			return math.NaN(), nil
		}
	}

	// strconv.ParseFloat is the reference implementation for parsing
	// floating point numbers. The position of underscores has already been
	// validated by the parser; strip them so that they do not interfere with
	// Go's own underscore rules.
	cleaned := b
	if bytes.IndexByte(b, '_') >= 0 {
		cleaned = make([]byte, 0, len(b))
		for _, c := range b {
			if c != '_' {
				cleaned = append(cleaned, c)
			}
		}
	}

	f, err := strconv.ParseFloat(string(cleaned), 64)
	if err != nil {
		return 0, unstable.NewParserError(b, "unable to parse float: %s", err)
	}
	return f, nil
}

func isDecimalDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// parseLocalDate parses a date of the exact form YYYY-MM-DD and validates
// its components.
func parseLocalDate(b []byte) (LocalDate, error) {
	var date LocalDate

	if len(b) != 10 || b[4] != '-' || b[7] != '-' {
		return date, unstable.NewParserError(b, "dates are expected to have the format YYYY-MM-DD")
	}

	var err error
	date.Year, err = parseDecimalDigits(b[0:4])
	if err != nil {
		return date, err
	}
	date.Month, err = parseDecimalDigits(b[5:7])
	if err != nil {
		return date, err
	}
	date.Day, err = parseDecimalDigits(b[8:10])
	if err != nil {
		return date, err
	}

	if date.Month < 1 || date.Month > 12 {
		return date, unstable.NewParserError(b[5:7], "impossible date")
	}
	maxDay := daysIn(date.Month, date.Year)
	if date.Day < 1 || date.Day > maxDay {
		return date, unstable.NewParserError(b[8:10], "impossible date")
	}

	return date, nil
}

func daysIn(month int, year int) int {
	switch month {
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	default:
		return 31
	}
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// parseDecimalDigits parses a sequence of digits as a decimal number.
func parseDecimalDigits(b []byte) (int, error) {
	v := 0
	for i, c := range b {
		if !isDecimalDigit(c) {
			return 0, unstable.NewParserError(b[i:i+1], "expected digit (0-9)")
		}
		v = v*10 + int(c-'0')
	}
	return v, nil
}

// parseLocalTime parses a time of the form HH:MM:SS with an optional
// fractional part. It returns the remaining bytes after the time.
func parseLocalTime(b []byte) (LocalTime, []byte, error) {
	var (
		nspow = [10]int{0, 1e8, 1e7, 1e6, 1e5, 1e4, 1e3, 1e2, 1e1, 1e0}
		t     LocalTime
	)

	if len(b) < 8 {
		return t, nil, unstable.NewParserError(b, "times are expected to have the format HH:MM:SS[.NNNNNN]")
	}

	var err error
	t.Hour, err = parseDecimalDigits(b[0:2])
	if err != nil {
		return t, nil, err
	}
	if t.Hour > 23 {
		return t, nil, unstable.NewParserError(b[0:2], "hour cannot be greater 23")
	}
	if b[2] != ':' {
		return t, nil, unstable.NewParserError(b[2:3], "expecting colon between hours and minutes")
	}

	t.Minute, err = parseDecimalDigits(b[3:5])
	if err != nil {
		return t, nil, err
	}
	if t.Minute > 59 {
		return t, nil, unstable.NewParserError(b[3:5], "minutes cannot be greater 59")
	}
	if b[5] != ':' {
		return t, nil, unstable.NewParserError(b[5:6], "expecting colon between minutes and seconds")
	}

	t.Second, err = parseDecimalDigits(b[6:8])
	if err != nil {
		return t, nil, err
	}
	if t.Second > 59 {
		return t, nil, unstable.NewParserError(b[6:8], "seconds cannot be greater than 59")
	}

	b = b[8:]

	if len(b) >= 1 && b[0] == '.' {
		frac := 0
		precision := 0
		digits := 0

		for i, c := range b[1:] {
			if !isDecimalDigit(c) {
				if i == 0 {
					return t, nil, unstable.NewParserError(b[0:1], "need at least one digit after fraction point")
				}
				break
			}
			digits++
			if i < 9 {
				frac = frac*10 + int(c-'0')
				precision++
			}
		}

		if digits == 0 {
			return t, nil, unstable.NewParserError(b[0:1], "need at least one digit after fraction point")
		}

		t.Nanosecond = frac * nspow[precision]
		t.Precision = precision

		return t, b[1+digits:], nil
	}
	return t, b, nil
}

// parseLocalDateTime parses a local date time of the form
// YYYY-MM-DD(T| )HH:MM:SS[.NNNNNN]. It returns the remaining bytes after the
// date-time.
func parseLocalDateTime(b []byte) (LocalDateTime, []byte, error) {
	var dt LocalDateTime

	const localDateTimeByteMinLen = 11
	if len(b) < localDateTimeByteMinLen {
		return dt, nil, unstable.NewParserError(b, "local datetimes are expected to have the format YYYY-MM-DDTHH:MM:SS[.NNNNNNNNN]")
	}

	date, err := parseLocalDate(b[:10])
	if err != nil {
		return dt, nil, err
	}
	dt.LocalDate = date

	sep := b[10]
	if sep != 'T' && sep != ' ' && sep != 't' {
		return dt, nil, unstable.NewParserError(b[10:11], "datetime separator is expected to be T or a space")
	}

	t, rest, err := parseLocalTime(b[11:])
	if err != nil {
		return dt, nil, err
	}
	dt.LocalTime = t

	return dt, rest, nil
}

// parseDateTime parses a date-time with a timezone offset (Z or +/-HH:MM).
func parseDateTime(b []byte) (time.Time, error) {
	dt, b, err := parseLocalDateTime(b)
	if err != nil {
		return time.Time{}, err
	}

	var zone *time.Location

	if len(b) == 0 {
		// parser should have checked that there is a timezone
		return time.Time{}, unstable.NewParserError(b, "date-time is missing timezone")
	}

	if b[0] == 'Z' || b[0] == 'z' {
		b = b[1:]
		zone = time.UTC
	} else {
		const dateTimeByteLen = 6
		if len(b) != dateTimeByteLen {
			return time.Time{}, unstable.NewParserError(b, "invalid date-time timezone")
		}
		var direction int
		switch b[0] {
		case '-':
			direction = -1
		case '+':
			direction = +1
		default:
			return time.Time{}, unstable.NewParserError(b[:1], "invalid timezone offset character")
		}

		if b[3] != ':' {
			return time.Time{}, unstable.NewParserError(b[3:4], "expected a : separator")
		}

		hours, err := parseDecimalDigits(b[1:3])
		if err != nil {
			return time.Time{}, err
		}
		if hours > 23 {
			return time.Time{}, unstable.NewParserError(b[1:3], "invalid timezone offset hours")
		}

		minutes, err := parseDecimalDigits(b[4:6])
		if err != nil {
			return time.Time{}, err
		}
		if minutes > 59 {
			return time.Time{}, unstable.NewParserError(b[4:6], "invalid timezone offset minutes")
		}

		seconds := direction * (hours*3600 + minutes*60)
		if seconds == 0 {
			zone = time.UTC
		} else {
			zone = time.FixedZone("", seconds)
		}
		b = b[dateTimeByteLen:]
	}

	if len(b) > 0 {
		return time.Time{}, unstable.NewParserError(b, "extra bytes at the end of the timezone")
	}

	t := time.Date(
		dt.Year,
		time.Month(dt.Month),
		dt.Day,
		dt.Hour,
		dt.Minute,
		dt.Second,
		dt.Nanosecond,
		zone)

	return t, nil
}
