package toml

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2/internal/unsafe"
)

// DecodeError represents an error encountered during the parsing or decoding
// of a TOML document.
//
// In addition to the error message, it contains the position in the document
// where it happened, as well as a human-readable representation that shows
// where the error occurred in the document.
type DecodeError struct {
	message string
	line    int
	column  int

	human string
}

// internal version of DecodeError that is used as the base to create a
// DecodeError with full context.
type decodeError struct {
	highlight []byte
	message   string
}

func (de *decodeError) Error() string {
	return de.message
}

func newDecodeError(highlight []byte, format string, args ...interface{}) error {
	return &decodeError{
		highlight: highlight,
		message:   fmt.Sprintf(format, args...),
	}
}

// Error returns the error message contained in the DecodeError.
func (e *DecodeError) Error() string {
	return e.message
}

// String returns the human-readable contextualized error. This string is multi-line.
func (e *DecodeError) String() string {
	return e.human
}

/// Position returns the (line, column) pair indicating where the error
// occurred in the document. Positions are 1-indexed.
func (e *DecodeError) Position() (row int, column int) {
	return e.line, e.column
}

// decodeErrorFromHighlight creates a DecodeError referencing to a highlighted
// range of bytes from document.
//
// highlight needs to be a sub-slice of document, or this function panics.
//
// The function copies all bytes used in DecodeError, so that document and
// highlight can be freely deallocated.
func wrapDecodeError(document []byte, de *decodeError) error {
	if de == nil {
		return nil
	}
	err := &DecodeError{
		message: de.message,
	}

	offset := unsafe.SubsliceOffset(document, de.highlight)

	err.line, err.column = positionAtEnd(document[:offset])
	before, after := linesOfContext(document, de.highlight, offset, 3)

	var buf strings.Builder

	maxLine := err.line + len(after) - 1
	lineColumnWidth := len(strconv.Itoa(maxLine))

	for i := len(before) - 1; i > 0; i-- {
		line := err.line - i
		buf.WriteString(formatLineNumber(line, lineColumnWidth))
		buf.WriteString("| ")
		buf.Write(before[i])
		buf.WriteRune('\n')
	}

	buf.WriteString(formatLineNumber(err.line, lineColumnWidth))
	buf.WriteString("| ")

	if len(before) > 0 {
		buf.Write(before[0])
	}
	buf.Write(de.highlight)
	if len(after) > 0 {
		buf.Write(after[0])
	}
	buf.WriteRune('\n')
	buf.WriteString(strings.Repeat(" ", lineColumnWidth))
	buf.WriteString("| ")
	if len(before) > 0 {
		buf.WriteString(strings.Repeat(" ", len(before[0])))
	}
	buf.WriteString(strings.Repeat("~", len(de.highlight)))
	buf.WriteString(" ")
	buf.WriteString(err.message)

	for i := 1; i < len(after); i++ {
		buf.WriteRune('\n')
		line := err.line + i
		buf.WriteString(formatLineNumber(line, lineColumnWidth))
		buf.WriteString("| ")
		buf.Write(after[i])
	}

	err.human = buf.String()
	return err
}

func formatLineNumber(line int, width int) string {
	format := "%" + strconv.Itoa(width) + "d"
	return fmt.Sprintf(format, line)
}

func linesOfContext(document []byte, highlight []byte, offset int, linesAround int) ([][]byte, [][]byte) {
	var beforeLines [][]byte
	for beforeOffset, lastOffset := offset, offset; beforeOffset >= 0 && len(beforeLines) <= linesAround; beforeOffset-- {
		if document[beforeOffset] == '\n' {
			beforeLines = append(beforeLines, document[beforeOffset+1:lastOffset])
			lastOffset = beforeOffset
		} else if beforeOffset == 0 && beforeOffset != lastOffset {
			beforeLines = append(beforeLines, document[beforeOffset:lastOffset])
		}
	}

	var afterLines [][]byte

	document = document[offset+len(highlight):]
	for afterOffset, lastOffset := 0, 0; afterOffset < len(document) && len(afterLines) <= linesAround; afterOffset++ {
		if document[afterOffset] == '\n' {
			afterLines = append(afterLines, document[lastOffset:afterOffset])
			afterOffset++ // skip \n
			lastOffset = afterOffset
		} else if afterOffset == len(document)-1 && lastOffset != afterOffset+1 {
			afterLines = append(afterLines, document[lastOffset:afterOffset+1])
		}
	}
	return beforeLines, afterLines
}

func positionAtEnd(b []byte) (row int, column int) {
	row = 1
	column = 1

	for _, c := range b {
		if c == '\n' {
			row++
			column = 1
		} else {
			column++
		}
	}
	return
}
