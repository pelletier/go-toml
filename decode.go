package toml

import (
	"fmt"
	"math"
	"strconv"
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
			panic(fmt.Errorf("invalid base '%c', should have been checked by scanIntOrFloat", b[1]))
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

	date.Year = parseDecimalDigits(b[0:4])

	v := parseDecimalDigits(b[5:7])

	date.Month = time.Month(v)

	date.Day = parseDecimalDigits(b[8:10])

	return date, nil
}

func parseDecimalDigits(b []byte) int {
	v := 0

	for _, c := range b {
		v *= 10
		v += int(c - '0')
	}

	return v
}

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
		// parser should have checked that when assigning the date time node
		panic("date time should have a timezone")
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
		if b[0] == '-' {
			direction = -1
		}

		hours := digitsToInt(b[1:3])
		minutes := digitsToInt(b[4:6])
		seconds := direction * (hours*3600 + minutes*60)
		zone = time.FixedZone("", seconds)
		b = b[dateTimeByteLen:]
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

func parseLocalDateTime(b []byte) (LocalDateTime, []byte, error) {
	var dt LocalDateTime

	const localDateTimeByteMinLen = 11
	if len(b) < localDateTimeByteMinLen {
		return dt, nil, newDecodeError(b, "local datetimes are expected to have the format YYYY-MM-DDTHH:MM:SS[.NNNNNNNNN]")
	}

	date, err := parseLocalDate(b[:10])
	if err != nil {
		return dt, nil, err
	}
	dt.Date = date

	sep := b[10]
	if sep != 'T' && sep != ' ' {
		return dt, nil, newDecodeError(b[10:11], "datetime separator is expected to be T or a space")
	}

	t, rest, err := parseLocalTime(b[11:])
	if err != nil {
		return dt, nil, err
	}
	dt.Time = t

	return dt, rest, nil
}

// parseLocalTime is a bit different because it also returns the remaining
// []byte that is didn't need. This is to allow parseDateTime to parse those
// remaining bytes as a timezone.
func parseLocalTime(b []byte) (LocalTime, []byte, error) {
	var (
		nspow = [10]int{0, 1e8, 1e7, 1e6, 1e5, 1e4, 1e3, 1e2, 1e1, 1e0}
		t     LocalTime
	)

	const localTimeByteLen = 8
	if len(b) < localTimeByteLen {
		return t, nil, newDecodeError(b, "times are expected to have the format HH:MM:SS[.NNNNNN]")
	}

	t.Hour = parseDecimalDigits(b[0:2])
	if b[2] != ':' {
		return t, nil, newDecodeError(b[2:3], "expecting colon between hours and minutes")
	}

	t.Minute = parseDecimalDigits(b[3:5])
	if b[5] != ':' {
		return t, nil, newDecodeError(b[5:6], "expecting colon between minutes and seconds")
	}

	t.Second = parseDecimalDigits(b[6:8])

	const minLengthWithFrac = 9
	if len(b) >= minLengthWithFrac && b[minLengthWithFrac-1] == '.' {
		frac := 0
		digits := 0

		for i, c := range b[minLengthWithFrac:] {
			if !isDigit(c) {
				if i == 0 {
					return t, nil, newDecodeError(b[i:i+1], "need at least one digit after fraction point")
				}

				break
			}

			const maxFracPrecision = 9
			if i >= maxFracPrecision {
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

//nolint:cyclop
func parseFloat(b []byte) (float64, error) {
	if len(b) == 4 && (b[0] == '+' || b[0] == '-') && b[1] == 'n' && b[2] == 'a' && b[3] == 'n' {
		return math.NaN(), nil
	}

	cleaned, err := checkAndRemoveUnderscores(b)
	if err != nil {
		return 0, err
	}

	if cleaned[0] == '.' {
		return 0, newDecodeError(b, "float cannot start with a dot")
	}

	if cleaned[len(cleaned)-1] == '.' {
		return 0, newDecodeError(b, "float cannot end with a dot")
	}

	f, err := strconv.ParseFloat(string(cleaned), 64)
	if err != nil {
		return 0, newDecodeError(b, "unable to parse float: %w", err)
	}

	return f, nil
}

func parseIntHex(b []byte) (int64, error) {
	cleaned, err := checkAndRemoveUnderscores(b[2:])
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(string(cleaned), 16, 64)
	if err != nil {
		return 0, newDecodeError(b, "couldn't parse hexadecimal number: %w", err)
	}

	return i, nil
}

func parseIntOct(b []byte) (int64, error) {
	cleaned, err := checkAndRemoveUnderscores(b[2:])
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(string(cleaned), 8, 64)
	if err != nil {
		return 0, newDecodeError(b, "couldn't parse octal number: %w", err)
	}

	return i, nil
}

func parseIntBin(b []byte) (int64, error) {
	cleaned, err := checkAndRemoveUnderscores(b[2:])
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(string(cleaned), 2, 64)
	if err != nil {
		return 0, newDecodeError(b, "couldn't parse binary number: %w", err)
	}

	return i, nil
}

func parseIntDec(b []byte) (int64, error) {
	cleaned, err := checkAndRemoveUnderscores(b)
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(string(cleaned), 10, 64)
	if err != nil {
		return 0, newDecodeError(b, "couldn't parse decimal number: %w", err)
	}

	return i, nil
}

func checkAndRemoveUnderscores(b []byte) ([]byte, error) {
	if b[0] == '_' {
		return nil, newDecodeError(b[0:1], "number cannot start with underscore")
	}

	if b[len(b)-1] == '_' {
		return nil, newDecodeError(b[len(b)-1:], "number cannot end with underscore")
	}

	// fast path
	i := 0
	for ; i < len(b); i++ {
		if b[i] == '_' {
			break
		}
	}
	if i == len(b) {
		return b, nil
	}

	before := false
	cleaned := make([]byte, i, len(b))
	copy(cleaned, b)

	for i++; i < len(b); i++ {
		c := b[i]
		if c == '_' {
			if !before {
				return nil, newDecodeError(b[i-1:i+1], "number must have at least one digit between underscores")
			}
			before = false
		} else {
			before = true
			cleaned = append(cleaned, c)
		}
	}

	return cleaned, nil
}
