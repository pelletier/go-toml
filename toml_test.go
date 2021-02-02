package toml

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
}

func TestParse(t *testing.T) {
	for i, data := range inputs {
		t.Run(fmt.Sprintf("example %d", i), func(t *testing.T) {
			fmt.Printf("input:\n\t`%s`\n", data)
			doc, err := Parse([]byte(data))
			assert.NoError(t, err)
			fmt.Println(doc)
		})
	}
}

type noopBuilder struct {
}

func (n noopBuilder) InlineTableSeparator()  {}
func (n noopBuilder) InlineTableBegin()      {}
func (n noopBuilder) InlineTableEnd()        {}
func (n noopBuilder) ArraySeparator()        {}
func (n noopBuilder) ArrayBegin()            {}
func (n noopBuilder) ArrayEnd()              {}
func (n noopBuilder) Whitespace(b []byte)    {}
func (n noopBuilder) Comment(b []byte)       {}
func (n noopBuilder) UnquotedKey(b []byte)   {}
func (n noopBuilder) LiteralString(b []byte) {}
func (n noopBuilder) BasicString(b []byte)   {}
func (n noopBuilder) Dot(b []byte)           {}
func (n noopBuilder) Boolean(b []byte)       {}
func (n noopBuilder) Equal(b []byte)         {}

func BenchmarkParseAll(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, data := range inputs {
			builder := noopBuilder{}
			p := parser{builder: &builder, data: []byte(data)}
			err := p.parse()
			if err != nil {
				b.Fatalf("error: %s", err)
			}
		}
	}
}
