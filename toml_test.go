package toml

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var inputs = []string{
	`     #foo`,
	`#foo`,
	`#`,
	"\n\n\n",
	"#one\n   # two   \n",
	`a = false`,
	`abc = false`,
	`  abc  = false  # foo`,
	`'abc' = false`,
	`"foo bar" = false`,
	`"hello\tworld" = false`,
	`"hello \u1234 foo" = false`,
	`a.b.c = false`,
	`a."b".c = true`,
	`a = "foo"`,
	`b = 'sample thingy'`,
	`a = []`,
	`b = ["foo"]`,
	`c = [[[]]]`,
	`d = ["foo","bar"]`,
	`d = ["foo",    "test"]`,
	`d = {}`,
	`e = {f = "bar"}`,
	`[foo]`,
	`[   test   ]`,
	`[  "hello".world ]`,
	`[test]
a = false`,
	`[[foo]]`,
}

func TestParse(t *testing.T) {
	for i, data := range inputs {
		t.Run(fmt.Sprintf("example %d", i), func(t *testing.T) {
			fmt.Printf("input:\n\t`%s`\n", data)
			b := []byte(data)
			var token []byte
			var err error
			for len(b) > 0 {
				token, b, err = scan(b)
				require.NoError(t, err)
				fmt.Printf("token => '%s'\n", string(token))
			}
		})
	}
}

type noopParser struct {
}

func (n noopParser) ArrayTableBegin()       {}
func (n noopParser) ArrayTableEnd()         {}
func (n noopParser) StandardTableBegin()    {}
func (n noopParser) StandardTableEnd()      {}
func (n noopParser) InlineTableSeparator()  {}
func (n noopParser) InlineTableBegin()      {}
func (n noopParser) InlineTableEnd()        {}
func (n noopParser) ArraySeparator()        {}
func (n noopParser) ArrayBegin()            {}
func (n noopParser) ArrayEnd()              {}
func (n noopParser) Whitespace(b []byte)    {}
func (n noopParser) Comment(b []byte)       {}
func (n noopParser) UnquotedKey(b []byte)   {}
func (n noopParser) LiteralString(b []byte) {}
func (n noopParser) BasicString(b []byte)   {}
func (n noopParser) Dot(b []byte)           {}
func (n noopParser) Boolean(b []byte)       {}
func (n noopParser) Equal(b []byte)         {}

//
//func BenchmarkParseAll(b *testing.B) {
//	b.ReportAllocs()
//
//	for i := 0; i < b.N; i++ {
//		for _, data := range inputs {
//			p := noopParser{}
//			l := lexer{parser: &p, data: []byte(data)}
//			err := l.run()
//			if err != nil {
//				b.Fatalf("error: %s", err)
//			}
//		}
//	}
//}
