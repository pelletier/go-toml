package edit

import (
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2/unstable"
)

func TestSetArrayTables(t *testing.T) {
	tests := []struct {
		name  string
		doc   string
		key   []string
		value interface{}
		want  string
	}{
		{
			name:  "replace value in element",
			doc:   "[[s]]\na = 1\n\n[[s]]\na = 2\n",
			key:   []string{"s", "1", "a"},
			value: 3,
			want:  "[[s]]\na = 1\n\n[[s]]\na = 3\n",
		},
		{
			name:  "insert key into element",
			doc:   "[[s]]\na = 1\n\n[[s]]\na = 2\n",
			key:   []string{"s", "0", "b"},
			value: 4,
			want:  "[[s]]\na = 1\nb = 4\n\n[[s]]\na = 2\n",
		},
		{
			name:  "new table under non-last element uses dotted keys",
			doc:   "[[s]]\nx = 1\n\n[[s]]\nx = 2\n",
			key:   []string{"s", "0", "opts", "k"},
			value: 1,
			want:  "[[s]]\nx = 1\nopts.k = 1\n\n[[s]]\nx = 2\n",
		},
		{
			name:  "append element",
			doc:   "[[s]]\na = 1\n\n[[s]]\na = 2\n",
			key:   []string{"s", "2", "a"},
			value: 9,
			want:  "[[s]]\na = 1\n\n[[s]]\na = 2\n\n[[s]]\na = 9\n",
		},
		{
			name:  "append element with deep key",
			doc:   "[[s]]\na = 1\n",
			key:   []string{"s", "1", "o", "k"},
			value: 1,
			want:  "[[s]]\na = 1\n\n[[s]]\no.k = 1\n",
		},
		{
			name:  "append element to nested array of tables",
			doc:   "[[a.b]]\nx = 1\n",
			key:   []string{"a", "b", "1", "x"},
			value: 2,
			want:  "[[a.b]]\nx = 1\n\n[[a.b]]\nx = 2\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if err := d.Set(test.key, test.value); err != nil {
				t.Fatalf("Set(%q, %v): %v", test.key, test.value, err)
			}
			if got := d.String(); got != test.want {
				t.Errorf("Set(%q, %v):\ngot:  %q\nwant: %q", test.key, test.value, got, test.want)
			}
			if got, ok := d.Get(test.key); !ok || !reflect.DeepEqual(got, int64(test.value.(int))) {
				t.Errorf("Get(%q) after Set = %v, %v", test.key, got, ok)
			}
		})
	}
}

func TestSetInline(t *testing.T) {
	tests := []struct {
		name  string
		doc   string
		key   []string
		value interface{}
		want  string
	}{
		{
			name:  "replace in nested inline table",
			doc:   "p = {a = {b = 1}, c = 2} # keep\n",
			key:   []string{"p", "a", "b"},
			value: 5,
			want:  "p = {a = {b = 5}, c = 2} # keep\n",
		},
		{
			name:  "insert into inline table",
			doc:   "p = {a = 1, c = 2}\n",
			key:   []string{"p", "d"},
			value: 3,
			want:  "p = {a = 1, c = 2, d = 3}\n",
		},
		{
			name:  "insert into empty inline table",
			doc:   "e = {}\n",
			key:   []string{"e", "x"},
			value: 1,
			want:  "e = {x = 1}\n",
		},
		{
			name:  "insert reuses trailing comma",
			doc:   "e = {a = 1,}\n",
			key:   []string{"e", "b"},
			value: 2,
			want:  "e = {a = 1, b = 2}\n",
		},
		{
			name:  "insert into multi-line inline table",
			doc:   "m = {\n  a = 1, # one\n}\n",
			key:   []string{"m", "b"},
			value: 2,
			want:  "m = {\n  a = 1, b = 2 # one\n}\n",
		},
		{
			name:  "replace dotted key in inline table",
			doc:   "d = {a.b = 1}\n",
			key:   []string{"d", "a", "b"},
			value: 9,
			want:  "d = {a.b = 9}\n",
		},
		{
			name:  "extend dotted key in inline table",
			doc:   "d = {a.b = 1}\n",
			key:   []string{"d", "a", "c"},
			value: 2,
			want:  "d = {a.b = 1, a.c = 2}\n",
		},
		{
			name:  "replace array element",
			doc:   "arr = [1, 2, 3]\n",
			key:   []string{"arr", "1"},
			value: 9,
			want:  "arr = [1, 9, 3]\n",
		},
		{
			name:  "append to array",
			doc:   "arr = [1, 2, 3]\n",
			key:   []string{"arr", "3"},
			value: 4,
			want:  "arr = [1, 2, 3, 4]\n",
		},
		{
			name:  "append to empty array",
			doc:   "arr = []\n",
			key:   []string{"arr", "0"},
			value: 9,
			want:  "arr = [9]\n",
		},
		{
			name:  "append to multi-line array",
			doc:   "arr = [\n  1,\n  2,\n]\n",
			key:   []string{"arr", "2"},
			value: 3,
			want:  "arr = [\n  1,\n  2, 3\n]\n",
		},
		{
			name:  "replace in nested array",
			doc:   "aa = [[1], [2, 3]]\n",
			key:   []string{"aa", "1", "0"},
			value: 9,
			want:  "aa = [[1], [9, 3]]\n",
		},
		{
			name:  "replace in array of inline tables",
			doc:   "pts = [{x = 1}, {x = 2}]\n",
			key:   []string{"pts", "1", "x"},
			value: 5,
			want:  "pts = [{x = 1}, {x = 5}]\n",
		},
		{
			name:  "insert into inline table element",
			doc:   "pts = [{x = 1}, {x = 2}]\n",
			key:   []string{"pts", "0", "y"},
			value: 7,
			want:  "pts = [{x = 1, y = 7}, {x = 2}]\n",
		},
		{
			name:  "append wraps remaining keys in tables",
			doc:   "pts = [{x = 1}]\n",
			key:   []string{"pts", "1", "x"},
			value: 3,
			want:  "pts = [{x = 1}, {x = 3}]\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if err := d.Set(test.key, test.value); err != nil {
				t.Fatalf("Set(%q, %v): %v", test.key, test.value, err)
			}
			if got := d.String(); got != test.want {
				t.Errorf("Set(%q, %v):\ngot:  %q\nwant: %q", test.key, test.value, got, test.want)
			}
			if got, ok := d.Get(test.key); !ok || !reflect.DeepEqual(got, int64(test.value.(int))) {
				t.Errorf("Get(%q) after Set = %v, %v", test.key, got, ok)
			}
		})
	}
}

func TestDeleteArrayTables(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		key  []string
		want string
	}{
		{
			name: "middle element",
			doc:  "[[s]]\na = 1\n\n[[s]]\na = 2\n\n[[s]]\na = 3\n",
			key:  []string{"s", "1"},
			want: "[[s]]\na = 1\n\n[[s]]\na = 3\n",
		},
		{
			name: "last element",
			doc:  "[[s]]\na = 1\n\n[[s]]\na = 2\n",
			key:  []string{"s", "1"},
			want: "[[s]]\na = 1\n",
		},
		{
			name: "only element",
			doc:  "x = 1\n\n[[s]]\na = 1\n",
			key:  []string{"s", "0"},
			want: "x = 1\n",
		},
		{
			name: "element with sub-table section",
			doc:  "[[s]]\na = 1\n\n[s.opts]\nk = 1\n\n[[s]]\na = 2\n",
			key:  []string{"s", "0"},
			want: "[[s]]\na = 2\n",
		},
		{
			name: "key inside element",
			doc:  "[[s]]\na = 1\nb = 2\n",
			key:  []string{"s", "0", "a"},
			want: "[[s]]\nb = 2\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if !d.Delete(test.key) {
				t.Fatalf("Delete(%q) = false", test.key)
			}
			if got := d.String(); got != test.want {
				t.Errorf("Delete(%q):\ngot:  %q\nwant: %q", test.key, got, test.want)
			}
		})
	}
}

func TestDeleteInline(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		key  []string
		want string
	}{
		{
			name: "first key of inline table",
			doc:  "p = {a = 1, b = 2, c = 3}\n",
			key:  []string{"p", "a"},
			want: "p = {b = 2, c = 3}\n",
		},
		{
			name: "last key of inline table",
			doc:  "p = {a = 1, b = 2, c = 3}\n",
			key:  []string{"p", "c"},
			want: "p = {a = 1, b = 2}\n",
		},
		{
			name: "only key of inline table",
			doc:  "o = {a = 1}\n",
			key:  []string{"o", "a"},
			want: "o = {}\n",
		},
		{
			name: "only key with trailing comma",
			doc:  "o = {a = 1,}\n",
			key:  []string{"o", "a"},
			want: "o = {}\n",
		},
		{
			name: "dotted key group",
			doc:  "g = {a.b = 1, a.c = 2, z = 3}\n",
			key:  []string{"g", "a"},
			want: "g = {z = 3}\n",
		},
		{
			name: "nested inline table key",
			doc:  "p = {a = {b = 1, c = 2}}\n",
			key:  []string{"p", "a", "b"},
			want: "p = {a = {c = 2}}\n",
		},
		{
			name: "first array element",
			doc:  "arr = [1, 2, 3]\n",
			key:  []string{"arr", "0"},
			want: "arr = [2, 3]\n",
		},
		{
			name: "last array element",
			doc:  "arr = [1, 2, 3]\n",
			key:  []string{"arr", "2"},
			want: "arr = [1, 2]\n",
		},
		{
			name: "only array element",
			doc:  "arr = [1]\n",
			key:  []string{"arr", "0"},
			want: "arr = []\n",
		},
		{
			name: "element of multi-line array",
			doc:  "arr = [\n  1, # one\n  2,\n]\n",
			key:  []string{"arr", "0"},
			want: "arr = [\n  2,\n]\n",
		},
		{
			name: "inline table element of array",
			doc:  "pts = [{x = 1}, {x = 2}]\n",
			key:  []string{"pts", "0"},
			want: "pts = [{x = 2}]\n",
		},
		{
			name: "key inside array element",
			doc:  "pts = [{x = 1}, {x = 2}]\n",
			key:  []string{"pts", "1", "x"},
			want: "pts = [{x = 1}, {}]\n",
		},
		{
			name: "nested array element",
			doc:  "aa = [[1], [2, 3]]\n",
			key:  []string{"aa", "1", "1"},
			want: "aa = [[1], [2]]\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if !d.Delete(test.key) {
				t.Fatalf("Delete(%q) = false", test.key)
			}
			if got := d.String(); got != test.want {
				t.Errorf("Delete(%q):\ngot:  %q\nwant: %q", test.key, got, test.want)
			}
			// Deleting an array element shifts the following ones, so an
			// index-ended path may legitimately still exist.
			if _, isIdx := parseIndex(test.key[len(test.key)-1]); !isIdx && d.Has(test.key) {
				t.Errorf("Has(%q) still true after Delete", test.key)
			}
		})
	}
}

func TestComment(t *testing.T) {
	doc := `# about a
# on two lines
a = 1 # trailing

[t] # table side
x = 1

[[s]]
y = 1
`
	d := mustParse(t, doc)

	if got, ok := d.Comment([]string{"a"}); !ok || got != "about a\non two lines" {
		t.Errorf("Comment(a) = %q, %v", got, ok)
	}
	if got, ok := d.TrailingComment([]string{"a"}); !ok || got != "trailing" {
		t.Errorf("TrailingComment(a) = %q, %v", got, ok)
	}
	if got, ok := d.Comment([]string{"t"}); !ok || got != "" {
		t.Errorf("Comment(t) = %q, %v", got, ok)
	}
	if got, ok := d.TrailingComment([]string{"t"}); !ok || got != "table side" {
		t.Errorf("TrailingComment(t) = %q, %v", got, ok)
	}
	if got, ok := d.Comment([]string{"s", "0"}); !ok || got != "" {
		t.Errorf("Comment(s.0) = %q, %v", got, ok)
	}
	if _, ok := d.Comment([]string{"missing"}); ok {
		t.Error("Comment(missing) should not be ok")
	}
	if _, ok := d.Comment([]string{"s"}); ok {
		t.Error("Comment on array of tables should not be ok")
	}
}

func TestSetComment(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		key  []string
		text string
		want string
	}{
		{
			name: "add to key-value",
			doc:  "a = 1\n",
			key:  []string{"a"},
			text: "hi",
			want: "# hi\na = 1\n",
		},
		{
			name: "replace block",
			doc:  "# keep\n\n# old\n# block\na = 1\nb = 2\n",
			key:  []string{"a"},
			text: "new",
			want: "# keep\n\n# new\na = 1\nb = 2\n",
		},
		{
			name: "remove block",
			doc:  "# old\na = 1\n",
			key:  []string{"a"},
			text: "",
			want: "a = 1\n",
		},
		{
			name: "multi-line text",
			doc:  "a = 1\n",
			key:  []string{"a"},
			text: "l1\n\nl3",
			want: "# l1\n#\n# l3\na = 1\n",
		},
		{
			name: "table header",
			doc:  "[t]\nx = 1\n",
			key:  []string{"t"},
			text: "about t",
			want: "# about t\n[t]\nx = 1\n",
		},
		{
			name: "array-of-tables element",
			doc:  "[[s]]\nx = 1\n\n[[s]]\nx = 2\n",
			key:  []string{"s", "1"},
			text: "second",
			want: "[[s]]\nx = 1\n\n# second\n[[s]]\nx = 2\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if err := d.SetComment(test.key, test.text); err != nil {
				t.Fatalf("SetComment(%q, %q): %v", test.key, test.text, err)
			}
			if got := d.String(); got != test.want {
				t.Errorf("SetComment(%q, %q):\ngot:  %q\nwant: %q", test.key, test.text, got, test.want)
			}
			if got, _ := d.Comment(test.key); got != test.text {
				t.Errorf("Comment(%q) after SetComment = %q, want %q", test.key, got, test.text)
			}
		})
	}
}

func TestSetTrailingComment(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		key  []string
		text string
		want string
	}{
		{
			name: "add to key-value",
			doc:  "a = 1\n",
			key:  []string{"a"},
			text: "hi",
			want: "a = 1 # hi\n",
		},
		{
			name: "replace",
			doc:  "a = 1 # old\n",
			key:  []string{"a"},
			text: "new",
			want: "a = 1 # new\n",
		},
		{
			name: "remove",
			doc:  "a = 1 # old\n",
			key:  []string{"a"},
			text: "",
			want: "a = 1\n",
		},
		{
			name: "table header",
			doc:  "[t]\nx = 1\n",
			key:  []string{"t"},
			text: "side",
			want: "[t] # side\nx = 1\n",
		},
		{
			name: "after multi-line value",
			doc:  "arr = [\n  1,\n] # end\n",
			key:  []string{"arr"},
			text: "x",
			want: "arr = [\n  1,\n] # x\n",
		},
		{
			name: "key-value without trailing newline",
			doc:  "a = 1",
			key:  []string{"a"},
			text: "hi",
			want: "a = 1 # hi",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if err := d.SetTrailingComment(test.key, test.text); err != nil {
				t.Fatalf("SetTrailingComment(%q, %q): %v", test.key, test.text, err)
			}
			if got := d.String(); got != test.want {
				t.Errorf("SetTrailingComment(%q, %q):\ngot:  %q\nwant: %q", test.key, test.text, got, test.want)
			}
			if got, _ := d.TrailingComment(test.key); got != test.text {
				t.Errorf("TrailingComment(%q) = %q, want %q", test.key, got, test.text)
			}
		})
	}
}

func TestCommentErrors(t *testing.T) {
	d := mustParse(t, "a.b = 1\np = {x = 1}\n\n[[s]]\ny = 1\n")
	for _, key := range [][]string{
		nil,                 // empty key
		{"missing"},         // does not exist
		{"a"},               // dotted table: no header line
		{"p", "x"},          // inside an inline value
		{"s"},               // array of tables, no index
		{"s", "5"},          // element out of range
		{"s", "not-an-int"}, // not an index
	} {
		if err := d.SetComment(key, "c"); err == nil {
			t.Errorf("SetComment(%q): expected error", key)
		}
		if err := d.SetTrailingComment(key, "c"); err == nil {
			t.Errorf("SetTrailingComment(%q): expected error", key)
		}
		if _, ok := d.Comment(key); ok {
			t.Errorf("Comment(%q): expected not ok", key)
		}
		if _, ok := d.TrailingComment(key); ok {
			t.Errorf("TrailingComment(%q): expected not ok", key)
		}
	}
	if err := d.SetTrailingComment([]string{"p"}, "two\nlines"); err == nil {
		t.Error("SetTrailingComment with newline: expected error")
	}
	if got := d.String(); got != "a.b = 1\np = {x = 1}\n\n[[s]]\ny = 1\n" {
		t.Errorf("document changed by failed comment edits: %q", got)
	}
}

func TestCommentCRLF(t *testing.T) {
	d := mustParse(t, "a = 1\r\n")
	if err := d.SetComment([]string{"a"}, "hi\nthere"); err != nil {
		t.Fatal(err)
	}
	want := "# hi\r\n# there\r\na = 1\r\n"
	if got := d.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, _ := d.Comment([]string{"a"}); got != "hi\nthere" {
		t.Errorf("Comment = %q", got)
	}

	d = mustParse(t, "a = 1 # old\r\n")
	if got, _ := d.TrailingComment([]string{"a"}); got != "old" {
		t.Errorf("TrailingComment = %q", got)
	}
	if err := d.SetTrailingComment([]string{"a"}, "new"); err != nil {
		t.Fatal(err)
	}
	if got, want := d.String(), "a = 1 # new\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRawMessage(t *testing.T) {
	d := mustParse(t, "a = 1\n")
	if err := d.Set([]string{"a"}, unstable.RawMessage("0x10")); err != nil {
		t.Fatal(err)
	}
	if got, want := d.String(), "a = 0x10\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// Raw values keep their exact representation, including multi-line
	// strings that the regular rendering never produces.
	if err := d.Set([]string{"m"}, unstable.RawMessage("\"\"\"\nhi\n\"\"\"")); err != nil {
		t.Fatal(err)
	}
	if got, want := d.String(), "a = 0x10\nm = \"\"\"\nhi\n\"\"\"\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if v, _ := d.Get([]string{"m"}); v != "hi\n" {
		t.Errorf("Get(m) = %q", v)
	}

	// Inside an inline value.
	if err := d.Set([]string{"p", "x"}, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.Set([]string{"p", "x"}, unstable.RawMessage("'raw'")); err != nil {
		t.Fatal(err)
	}
	if v, _ := d.Get([]string{"p", "x"}); v != "raw" {
		t.Errorf("Get(p.x) = %q", v)
	}

	// Invalid and empty raw values are rejected and leave the document
	// unchanged.
	before := d.String()
	if err := d.Set([]string{"bad"}, unstable.RawMessage("= not toml")); err == nil {
		t.Error("expected error for invalid raw value")
	}
	if err := d.Set([]string{"bad"}, unstable.RawMessage("  ")); err == nil {
		t.Error("expected error for empty raw value")
	}
	if got := d.String(); got != before {
		t.Errorf("document changed by failed raw edits: %q", got)
	}
}

func TestGetIndexes(t *testing.T) {
	d := mustParse(t, "arr = [1, [2, 3]]\n\n[[s]]\nv = 'a'\n\n[[s]]\nv = 'b'\n")
	tests := []struct {
		key  []string
		want interface{}
		ok   bool
	}{
		{[]string{"arr", "0"}, int64(1), true},
		{[]string{"arr", "1", "1"}, int64(3), true},
		{[]string{"s", "1", "v"}, "b", true},
		{[]string{"arr", "2"}, nil, false},
		{[]string{"arr", "x"}, nil, false},
		{[]string{"s", "2", "v"}, nil, false},
	}
	for _, test := range tests {
		got, ok := d.Get(test.key)
		if ok != test.ok || (ok && !reflect.DeepEqual(got, test.want)) {
			t.Errorf("Get(%q) = (%v, %v), want (%v, %v)", test.key, got, ok, test.want, test.ok)
		}
	}
}
