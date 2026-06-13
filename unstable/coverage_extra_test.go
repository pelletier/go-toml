package unstable

import (
	"fmt"
	"testing"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

func TestKindStringAll(t *testing.T) {
	expected := map[Kind]string{
		Invalid:       "Invalid",
		Comment:       "Comment",
		Key:           "Key",
		Table:         "Table",
		ArrayTable:    "ArrayTable",
		KeyValue:      "KeyValue",
		Array:         "Array",
		InlineTable:   "InlineTable",
		String:        "String",
		Bool:          "Bool",
		Float:         "Float",
		Integer:       "Integer",
		LocalDate:     "LocalDate",
		LocalTime:     "LocalTime",
		LocalDateTime: "LocalDateTime",
		DateTime:      "DateTime",
	}
	for k, s := range expected {
		assert.Equal(t, s, k.String())
	}
}

func TestKindStringUnknownPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = Kind(0xFF).String()
	})
}

func TestParserErrorError(t *testing.T) {
	err := &ParserError{Message: "boom"}
	assert.Equal(t, "boom", err.Error())
}

func TestParserRangeNotSubslicePanics(t *testing.T) {
	p := &Parser{}
	p.Reset([]byte("a = 1"))
	// A slice with a capacity larger than the document cannot be one of its
	// subslices.
	other := make([]byte, 2, 64)
	assert.Panics(t, func() {
		_ = p.Range(other)
	})
}

func TestParserExpressionEmpty(t *testing.T) {
	p := &Parser{}
	p.Reset([]byte(""))
	assert.True(t, p.Expression() == nil)
}

func TestNodeKeyPanicsOnUnsupportedKind(t *testing.T) {
	n := &Node{Kind: Integer}
	assert.Panics(t, func() {
		_ = n.Key()
	})
}

func parseAll(doc string) error {
	p := &Parser{}
	p.Reset([]byte(doc))
	for p.NextExpression() {
	}
	return p.Error()
}

func TestParserScanErrors(t *testing.T) {
	invalid := []string{
		"a = [1\rx]",
		"# comment\rx",
		"# comment\x00",
		"a = '''\rx'''",
		"a = \"aaaaaaaaaaaaaaaaaaaaaaaa",
		"a = \"\\u0041\nx\"",
		"a = \"\\u0041\xffx\"",
		"a = \"\\e\"",
		`a = """x""""""`,
		"a = \"\"\"\\\rx\"\"\"",
		"a = \"\"\"x\rz\"\"\"",
		"a = \"\"\"\xffx\"\"\"",
		"a = '\xc3\x28'",
		"a = '\xe0\x80\x80'",
		"a = '\xed\xa0\x80'",
		"a = '\xe1\x80\x28'",
		"a = '\xf0\x80\x80\x80'",
		"a = '\xf4\x90\x80\x80'",
		"a = '\xf1\x28\x80\x80'",
		"a = '\xf0\x9f'",
		"a = '\xc3'",
		"a = '\xfe'",
	}
	for _, doc := range invalid {
		t.Run(fmt.Sprintf("%q", doc), func(t *testing.T) {
			assert.Error(t, parseAll(doc))
		})
	}

	valid := []string{
		"a = \"\"\"a\r\nb\"\"\"",
		"a = \"\"\"\\\n  \r\n  x\"\"\"",
		"a = '\xc3\xa9'",
		"a = '\xe2\x82\xac'",
		"a = '\xe0\xa4\x84'",
		"a = '\xed\x9f\xbf'",
		"a = '\xf0\x90\x80\x80'",
		"a = '\xf4\x8f\xbf\xbf'",
		"a = '\xf1\x80\x80\x80'",
		"# comment ends in unicode \xc3\xa9",
	}
	for _, doc := range valid {
		t.Run(fmt.Sprintf("%q", doc), func(t *testing.T) {
			assert.NoError(t, parseAll(doc))
		})
	}
}

func TestParserNextExpressionAfterError(t *testing.T) {
	p := &Parser{}
	p.Reset([]byte("a = \"\n\"\nb = 1"))
	for p.NextExpression() {
	}
	assert.Error(t, p.Error())
	assert.False(t, p.NextExpression())
}

func TestParserScanEdgeCases(t *testing.T) {
	invalid := []string{
		"a = {,}",
		"a = \"\"\"\\t a\rb\"\"\"",
		"a = \"\"\"\\t \xffx\"\"\"",
		"a = \"\"\"\\t x\"\"\"\"\"\"",
		"a = \"\"\"\\\n\r x\"\"\"",
		"a = '\xe1\x28\x80'",
	}
	for _, doc := range invalid {
		t.Run(fmt.Sprintf("%q", doc), func(t *testing.T) {
			assert.Error(t, parseAll(doc))
		})
	}

	valid := []string{
		"a = [1,\r\n2]",
		"# comment\r\nb = 1",
		"# comment\twith tab",
		"a = '''x\r\ny'''",
		"a = \"\\u0041\tb\"",
		"a = \"\"\"\\t a\r\nb\"\"\"",
		"a = \"\"\"\\t x\"\"\"\"",
		"a = \"\"\"\\t x\"\"\"\"\"",
	}
	for _, doc := range valid {
		t.Run(fmt.Sprintf("%q", doc), func(t *testing.T) {
			assert.NoError(t, parseAll(doc))
		})
	}
}

func TestNodeKeyPanicsOnEmptyKeyValue(t *testing.T) {
	n := &Node{Kind: KeyValue}
	assert.Panics(t, func() {
		_ = n.Key()
	})
}
