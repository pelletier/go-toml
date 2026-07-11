package edit

import (
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2/unstable"
)

// parseExpr parses a single expression with a fresh parser. Feeding the
// resulting nodes to a Document holding different bytes exercises the
// defensive error paths of the span scanners, which are unreachable through
// the public API.
func parseExpr(t *testing.T, doc string) *unstable.Node {
	t.Helper()
	p := &unstable.Parser{}
	p.Reset([]byte(doc))
	if !p.NextExpression() {
		t.Fatalf("no expression in %q: %v", doc, p.Error())
	}
	return p.Expression()
}

func TestExprSpanErrors(t *testing.T) {
	tests := []struct {
		name string
		expr string
		data string
	}{
		{"missing open bracket", "[a]\n", "(a)\n"},
		{"missing close bracket", "[a]\n", "[a)\n"},
		{"missing second open bracket", "[[a]]\n", "([a]]\n"},
		{"missing second close bracket", "[[a]]\n", "[[a])\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := &Document{data: []byte(test.data)}
			if _, _, err := d.exprSpan(parseExpr(t, test.expr)); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestValueSpanError(t *testing.T) {
	d := &Document{data: []byte("a x 1\n")}
	if _, err := d.valueSpan(parseExpr(t, "a = 1\n")); err == nil {
		t.Error("expected error")
	}
}

func TestValueNodeEndErrors(t *testing.T) {
	t.Run("inline table close", func(t *testing.T) {
		d := &Document{data: []byte("k = {a = 1 x")}
		n := parseExpr(t, "k = {a = 1}\n").Value()
		if _, err := d.valueNodeEnd(n, 4); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("array close", func(t *testing.T) {
		d := &Document{data: []byte("k = [1 x")}
		n := parseExpr(t, "k = [1]\n").Value()
		if _, err := d.valueNodeEnd(n, 4); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("array element error propagates", func(t *testing.T) {
		d := &Document{data: []byte("k = [{a = 1 ]")}
		n := parseExpr(t, "k = [{a = 1}]\n").Value()
		if _, err := d.valueNodeEnd(n, 4); err == nil {
			t.Error("expected error")
		}
		if _, err := d.arrayElems(n, 4); err == nil {
			t.Error("expected error")
		}
	})
}

func TestLeafValueNodeError(t *testing.T) {
	d := mustParse(t, "a = 1\n")
	stale := &leaf{value: span{0, 999}}
	if _, err := d.leafValueNode(stale); err == nil {
		t.Error("leafValueNode: expected error")
	}
	if err := d.setInline(stale, []string{"a", "b"}, 1, 1); err == nil {
		t.Error("setInline: expected error")
	}
	if _, _, ok := d.deleteInline(stale, []string{"a", "b"}, 1); ok {
		t.Error("deleteInline: expected not ok")
	}
}

func TestReindexParseError(t *testing.T) {
	d := &Document{data: []byte("a =")}
	if err := d.reindex(); err == nil {
		t.Error("expected error")
	}
}

func TestSkipInlineTriviaEOF(t *testing.T) {
	d := mustParse(t, "a = 1\n")
	if got := d.skipInlineTrivia(5, true); got != 6 {
		t.Errorf("skipInlineTrivia = %d, want 6", got)
	}
}

func TestParseIndexLimits(t *testing.T) {
	if _, ok := parseIndex(strings.Repeat("9", 19)); ok {
		t.Error("oversized index should not parse")
	}
	if _, ok := parseIndex(""); ok {
		t.Error("empty index should not parse")
	}
}
