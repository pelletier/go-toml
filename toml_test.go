package toml

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {

	inputs := []string{
		`     #foo`,
		`#foo`,
		`#`,
		"\n\n\n",
		"#one\n   # two   \n",
		`a`,
		`abc`,
		`  abc   # foo`,
		`'abc'`,
		`"foo bar"`,
		`"hello\tworld"`,
		`"hello \u1234 foo"`,
	}

	for i, data := range inputs {
		t.Run(fmt.Sprintf("example %d", i), func(t *testing.T) {
			fmt.Printf("input:\n\t`%s`\n", data)
			doc, err := Parse([]byte(data))
			assert.NoError(t, err)
			fmt.Println(doc)
		})
	}
}
