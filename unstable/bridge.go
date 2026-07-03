package unstable

import "github.com/pelletier/go-toml/v2/internal/parserbridge"

// Expose the non-AST scanners to the root toml package without committing to
// them in the public API. See internal/parserbridge for the rationale.
//
//nolint:gochecknoinits // load-time wiring of an internal bridge (see internal/parserbridge)
func init() {
	parserbridge.ScanScalar = func(p any, b []byte) (kind int, raw, value, rest []byte, err error) {
		k, raw, value, rest, err := p.(*Parser).scanScalar(b)
		return int(k), raw, value, rest, err
	}
	parserbridge.ScanKey = func(p any, b []byte, dst [][]byte) (parts [][]byte, raw, rest []byte, err error) {
		return p.(*Parser).scanKey(b, dst)
	}
	parserbridge.ScanKeyRaws = func(p any, b []byte, dst, raws [][]byte) (parts, rawsOut [][]byte, raw, rest []byte, err error) {
		return p.(*Parser).scanKeyRaws(b, dst, raws)
	}
	parserbridge.ScanComment = scanComment
	parserbridge.ParseValue = func(p any, b []byte) (node any, rest []byte, err error) {
		return p.(*Parser).parseValue(b)
	}
	parserbridge.SetCursor = func(p any, rest []byte) {
		p.(*Parser).left = rest
	}
	parserbridge.Cursor = func(p any) []byte {
		return p.(*Parser).left
	}
}
