package unstable

import (
	"bytes"
	"testing"
)

var (
	valid10ASCII = []byte("1234567890")
	valid10Utf8  = []byte("日本語a")
	valid1kUtf8  = bytes.Repeat([]byte("0123456789日本語日本語日本語日abcdefghijklmnopqrstuvwx"), 16)
	valid1MUtf8  = bytes.Repeat(valid1kUtf8, 1024)
	valid1kASCII = bytes.Repeat([]byte("012345678998jhjklasDJKLAAdjdfjsdklfjdslkabcdefghijklmnopqrstuvwx"), 16)
	valid1MASCII = bytes.Repeat(valid1kASCII, 1024)
)

func BenchmarkScanComments(b *testing.B) {
	wrap := func(x []byte) []byte {
		return []byte("# " + string(x) + "\n")
	}

	inputs := map[string][]byte{
		"10Valid":     wrap(valid10ASCII),
		"1kValid":     wrap(valid1kASCII),
		"1MValid":     wrap(valid1MASCII),
		"10ValidUtf8": wrap(valid10Utf8),
		"1kValidUtf8": wrap(valid1kUtf8),
		"1MValidUtf8": wrap(valid1MUtf8),
	}

	for name, input := range inputs {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, _ = scanComment(input)
			}
		})
	}
}

func BenchmarkParseLiteralStringValid(b *testing.B) {
	wrap := func(x []byte) []byte {
		return []byte("'" + string(x) + "'")
	}

	inputs := map[string][]byte{
		"10Valid":     wrap(valid10ASCII),
		"1kValid":     wrap(valid1kASCII),
		"1MValid":     wrap(valid1MASCII),
		"10ValidUtf8": wrap(valid10Utf8),
		"1kValidUtf8": wrap(valid1kUtf8),
		"1MValidUtf8": wrap(valid1MUtf8),
	}

	for name, input := range inputs {
		b.Run(name, func(b *testing.B) {
			p := Parser{}
			b.SetBytes(int64(len(input)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, _, err := p.parseLiteralString(input)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}
