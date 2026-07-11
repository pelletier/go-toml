package edit

import (
	"os"
	"reflect"
	"strconv"
	"testing"
)

// corpus is a set of valid documents exercising the constructs the index has
// to handle.
var corpus = []string{
	"",
	"# only a comment\n",
	"a = 1",
	"key = 'value'\n\n# section\n[table]\nk = 10 # trailing\n\n[table.sub]\narr = [1, 2, { x = 'y' }]\n",
	"[[products]]\nname = 'Hammer'\nsku = 738594937\n\n[[products]]\n\n[[products]]\nname = 'Nail'\n\n[products.extra]\nnote = 'last one'\n",
	"[[a.b]]\nx = 1\n\n[[a.b]]\nx = 2\n",
	"phys.color = 'orange'\nphys.shape = 'round'\n\n[sub]\ndot.ted.key = 1\n",
	"m = \"\"\"\nmulti \\\" line\nstring '' \"\"\ttab\n\"\"\"\nlit = '''\nliteral\n'''\n",
	"arr = [\n  1, # one\n  # standalone\n  2,\n]\n",
	"it = {\n  a = 1, # comment in inline table\n}\n",
	"crlf = 1\r\n[t]\r\nx = 'y'\r\n",
	"'quoted key' = 1\n\"another.one\" = 2\n\n[deep.'ta.ble']\nx = 1\n",
	"[ spaced . 'head er' ]\nx = 1\n\n[[ spaced . arr ]]\ny = 2\n",
	"d = 1979-05-27T07:32:00Z\nld = 1979-05-27\nlt = 07:32:00\nf = inf\nn = nan\nneg = -0.01\nhex = 0xDEADBEEF\n",
	"inline = {a = 1, b = [1, 2], c = {d = 'e'}}\n",
}

func mustParse(t *testing.T, doc string) *Document {
	t.Helper()
	d, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse(%q): %v", doc, err)
	}
	return d
}

func TestRoundTrip(t *testing.T) {
	for _, doc := range corpus {
		d := mustParse(t, doc)
		if got := d.String(); got != doc {
			t.Errorf("round trip mismatch:\nin:  %q\nout: %q", doc, got)
		}
	}
}

func TestRoundTripReferenceFile(t *testing.T) {
	b, err := os.ReadFile("../../benchmark/benchmark.toml")
	if err != nil {
		t.Skip("reference file not available:", err)
	}
	d, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Bytes(); string(got) != string(b) {
		t.Error("round trip mismatch on reference file")
	}
}

func TestParseInvalid(t *testing.T) {
	for _, doc := range []string{
		"a =",            // syntax error
		"a = 1\na = 2\n", // duplicate key
		"a = 1\n[a]\n",   // table redefines value
	} {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("Parse(%q): expected error", doc)
		}
	}
}

func TestGet(t *testing.T) {
	doc := `top = 'level'
'quoted key' = 1

[table]
num = 42
inline = {x = 'y'}
arr = [1, 2]

[[items]]
name = 'first'
`
	d := mustParse(t, doc)

	tests := []struct {
		key  []string
		want interface{}
		ok   bool
	}{
		{[]string{"top"}, "level", true},
		{[]string{"quoted key"}, int64(1), true},
		{[]string{"table", "num"}, int64(42), true},
		{[]string{"table", "inline", "x"}, "y", true},
		{[]string{"table", "arr"}, []interface{}{int64(1), int64(2)}, true},
		{[]string{"items"}, []interface{}{map[string]interface{}{"name": "first"}}, true},
		{[]string{"missing"}, nil, false},
		{[]string{"table", "missing"}, nil, false},
		{[]string{"top", "not-a-table"}, nil, false},
		{[]string{"items", "name"}, nil, false}, // cannot index arrays by name
	}
	for _, test := range tests {
		got, ok := d.Get(test.key)
		if ok != test.ok || (ok && !reflect.DeepEqual(got, test.want)) {
			t.Errorf("Get(%q) = (%v, %v), want (%v, %v)", test.key, got, ok, test.want, test.ok)
		}
		if d.Has(test.key) != test.ok {
			t.Errorf("Has(%q) = %v, want %v", test.key, !test.ok, test.ok)
		}
	}

	whole, ok := d.Get(nil)
	if !ok {
		t.Fatal("Get(nil) not ok")
	}
	if _, isMap := whole.(map[string]interface{}); !isMap {
		t.Errorf("Get(nil) = %T, want map", whole)
	}
}

func TestUnmarshal(t *testing.T) {
	d := mustParse(t, "[server]\nport = 8080\n")
	var cfg struct {
		Server struct{ Port int }
	}
	if err := d.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Server.Port)
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		name  string
		doc   string
		key   []string
		value interface{}
		want  string
	}{
		{
			name:  "replace scalar keeps trailing comment",
			doc:   "# Server config\n[server]\nhost = 'localhost' # the host\nport = 8080\n",
			key:   []string{"server", "host"},
			value: "example.com",
			want:  "# Server config\n[server]\nhost = 'example.com' # the host\nport = 8080\n",
		},
		{
			name:  "replace changes value type",
			doc:   "port = 8080\n",
			key:   []string{"port"},
			value: "auto",
			want:  "port = 'auto'\n",
		},
		{
			name:  "replace multiline array with scalar",
			doc:   "a = [\n  1,\n  2, # two\n]\nb = 1\n",
			key:   []string{"a"},
			value: 7,
			want:  "a = 7\nb = 1\n",
		},
		{
			name:  "replace with array value",
			doc:   "a = 1\n",
			key:   []string{"a"},
			value: []interface{}{1, "two"},
			want:  "a = [1, 'two']\n",
		},
		{
			name:  "replace with inline table value",
			doc:   "a = 1\n",
			key:   []string{"a"},
			value: map[string]interface{}{"x": 1},
			want:  "a = {x = 1}\n",
		},
		{
			name:  "replace value in inline table is atomic",
			doc:   "p = {x = 1, y = 2} # keep\n",
			key:   []string{"p"},
			value: map[string]interface{}{"x": 5},
			want:  "p = {x = 5} # keep\n",
		},
		{
			name:  "insert into table before next section comments",
			doc:   "[a]\nx = 1\n\n# next section\n[b]\ny = 2\n",
			key:   []string{"a", "z"},
			value: 3,
			want:  "[a]\nx = 1\nz = 3\n\n# next section\n[b]\ny = 2\n",
		},
		{
			name:  "insert into empty table",
			doc:   "[a]\n[b]\n",
			key:   []string{"a", "x"},
			value: 1,
			want:  "[a]\nx = 1\n[b]\n",
		},
		{
			name:  "insert at root after last top-level key",
			doc:   "a = 1\n\n[t]\nb = 2\n",
			key:   []string{"c"},
			value: true,
			want:  "a = 1\nc = true\n\n[t]\nb = 2\n",
		},
		{
			name:  "insert at root without top-level keys",
			doc:   "# about t\n[t]\nx = 1\n",
			key:   []string{"r"},
			value: 1,
			want:  "r = 1\n# about t\n[t]\nx = 1\n",
		},
		{
			name:  "insert into empty document",
			doc:   "",
			key:   []string{"a"},
			value: 1,
			want:  "a = 1\n",
		},
		{
			name:  "insert after document without trailing newline",
			doc:   "a = 1",
			key:   []string{"b"},
			value: 2,
			want:  "a = 1\nb = 2\n",
		},
		{
			name:  "create section in empty document",
			doc:   "",
			key:   []string{"srv", "port"},
			value: 8080,
			want:  "[srv]\nport = 8080\n",
		},
		{
			name:  "create section after root keys",
			doc:   "a = 1\n",
			key:   []string{"srv", "port"},
			value: 8080,
			want:  "a = 1\n\n[srv]\nport = 8080\n",
		},
		{
			name:  "create deep section with single header",
			doc:   "a = 1\n",
			key:   []string{"x", "y", "z"},
			value: 1,
			want:  "a = 1\n\n[x.y]\nz = 1\n",
		},
		{
			name:  "create section near its parent",
			doc:   "[a]\nx = 1\n\n[z]\ny = 2\n",
			key:   []string{"a", "b", "c"},
			value: 1,
			want:  "[a]\nx = 1\n\n[a.b]\nc = 1\n\n[z]\ny = 2\n",
		},
		{
			name:  "create header for implicit table",
			doc:   "[a.b]\nx = 1\n",
			key:   []string{"a", "y"},
			value: 2,
			want:  "[a.b]\nx = 1\n\n[a]\ny = 2\n",
		},
		{
			name:  "extend dotted table with dotted key",
			doc:   "fruit.apple = 1\n",
			key:   []string{"fruit", "banana"},
			value: 2,
			want:  "fruit.apple = 1\nfruit.banana = 2\n",
		},
		{
			name:  "extend dotted table deeply",
			doc:   "fruit.apple = 1\n",
			key:   []string{"fruit", "color", "hue"},
			value: 3,
			want:  "fruit.apple = 1\nfruit.color.hue = 3\n",
		},
		{
			name:  "extend dotted table inside section",
			doc:   "[t]\na.b = 1\n",
			key:   []string{"t", "a", "c"},
			value: 2,
			want:  "[t]\na.b = 1\na.c = 2\n",
		},
		{
			name:  "create section next to array of tables",
			doc:   "[[t.arr]]\nx = 1\n",
			key:   []string{"t", "new", "k"},
			value: 1,
			want:  "[[t.arr]]\nx = 1\n\n[t.new]\nk = 1\n",
		},
		{
			name:  "quoted key parts",
			doc:   "",
			key:   []string{"a b", "x.y"},
			value: 1,
			want:  "['a b']\n'x.y' = 1\n",
		},
		{
			name:  "insert key needing quotes",
			doc:   "a = 1\n",
			key:   []string{"needs quotes"},
			value: 1,
			want:  "a = 1\n'needs quotes' = 1\n",
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
			if got, ok := d.Get(test.key); !ok {
				t.Errorf("Get(%q) after Set: missing (got %v)", test.key, got)
			}
		})
	}
}

func TestSetErrors(t *testing.T) {
	tests := []struct {
		name  string
		doc   string
		key   []string
		value interface{}
	}{
		{"empty key", "a = 1\n", nil, 1},
		{"through value", "a = 1\n", []string{"a", "b"}, 1},
		{"non-index into array of tables", "[[x]]\nk = 1\n", []string{"x", "k"}, 2},
		{"onto array of tables", "[[x]]\nk = 1\n", []string{"x"}, 1},
		{"onto array-of-tables element", "[[x]]\nk = 1\n", []string{"x", "0"}, 1},
		{"array-of-tables index out of range", "[[x]]\nk = 1\n", []string{"x", "2", "k"}, 1},
		{"append element without keys", "[[x]]\nk = 1\n", []string{"x", "1"}, 1},
		{"append element with nil value", "[[x]]\nk = 1\n", []string{"x", "1", "k"}, nil},
		{"insert into inline table with nil value", "p = {a = 1}\n", []string{"p", "b"}, nil},
		{"append to array with nil value", "a = [1]\n", []string{"a", "1"}, nil},
		{"replace array element with nil", "a = [1]\n", []string{"a", "0"}, nil},
		{"through scalar in inline table", "a = {b = 1}\n", []string{"a", "b", "c"}, 2},
		{"non-index into array", "a = [1, 2]\n", []string{"a", "x"}, 1},
		{"array index out of range", "a = [1, 2]\n", []string{"a", "5"}, 1},
		{"onto implicit table in inline", "a = {b.c = 1}\n", []string{"a", "b"}, 1},
		{"replace inline value with nil", "p = {x = 1}\n", []string{"p", "x"}, nil},
		{"extend dotted table with nil", "a.b = 1\n", []string{"a", "c"}, nil},
		{"element sub-table with nil", "[[s]]\nx = 1\n", []string{"s", "0", "o", "k"}, nil},
		{"new section with nil", "", []string{"a", "b"}, nil},
		{"onto table", "[t]\nx = 1\n", []string{"t"}, 5},
		{"onto dotted table", "a.b = 1\n", []string{"a"}, 5},
		{"nil value", "", []string{"a"}, nil},
		{"unsupported value", "", []string{"a"}, func() {}},
		{"nil value on existing key", "a = 1\n", []string{"a"}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if err := d.Set(test.key, test.value); err == nil {
				t.Fatalf("Set(%q, %v): expected error, document is now %q", test.key, test.value, d.String())
			}
			if got := d.String(); got != test.doc {
				t.Errorf("document changed after failed Set:\ngot:  %q\nwant: %q", got, test.doc)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		key  []string
		want string
	}{
		{
			name: "key-value with attached comments",
			doc:  "# keep\n\n# about a\n# more about a\na = 1 # trail\nb = 2\n",
			key:  []string{"a"},
			want: "# keep\n\nb = 2\n",
		},
		{
			name: "key-value without trailing newline",
			doc:  "a = 1",
			key:  []string{"a"},
			want: "",
		},
		{
			name: "dotted key-value",
			doc:  "a.b = 1\nc = 2\n",
			key:  []string{"a", "b"},
			want: "c = 2\n",
		},
		{
			name: "dotted table deletes its lines",
			doc:  "a.b = 1\na.c = 2\nz = 1\n",
			key:  []string{"a"},
			want: "z = 1\n",
		},
		{
			name: "last key of table leaves empty section",
			doc:  "[a]\nx = 1\n",
			key:  []string{"a", "x"},
			want: "[a]\n",
		},
		{
			name: "table with attached header comment",
			doc:  "a = 1\n\n# about t\n[t]\nx = 1\n",
			key:  []string{"t"},
			want: "a = 1\n",
		},
		{
			name: "table with non-contiguous sub-table",
			doc:  "[a]\nx = 1\n\n[b]\ny = 2\n\n[a.sub]\nz = 3\n",
			key:  []string{"a"},
			want: "[b]\ny = 2\n",
		},
		{
			name: "sub-table only",
			doc:  "[a]\nx = 1\n\n[a.sub]\nz = 3\n\n[b]\ny = 2\n",
			key:  []string{"a", "sub"},
			want: "[a]\nx = 1\n\n[b]\ny = 2\n",
		},
		{
			name: "array of tables",
			doc:  "x = 5\n\n[[s]]\na = 1\n\n[[s]]\na = 2\n",
			key:  []string{"s"},
			want: "x = 5\n",
		},
		{
			name: "implicit parent disappears",
			doc:  "[a.b]\nx = 1\n",
			key:  []string{"a", "b"},
			want: "",
		},
		{
			name: "crlf table at end absorbs separator",
			doc:  "a = 1\r\n\r\n[t]\r\nx = 1\r\n",
			key:  []string{"t"},
			want: "a = 1\r\n",
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
			if d.Has(test.key) {
				t.Errorf("Has(%q) still true after Delete", test.key)
			}
		})
	}
}

func TestDeleteFalse(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		key  []string
	}{
		{"empty key", "a = 1\n", nil},
		{"missing", "a = 1\n", []string{"b"}},
		{"missing nested", "[t]\n", []string{"t", "x"}},
		{"through value", "a = 1\n", []string{"a", "b"}},
		{"missing in inline table", "a = {b = 1}\n", []string{"a", "c"}},
		{"non-index into array of tables", "[[x]]\nk = 1\n", []string{"x", "k"}},
		{"element index out of range", "[[x]]\nk = 1\n", []string{"x", "1"}},
		{"array index out of range", "a = [1]\n", []string{"a", "1"}},
		{"non-index into array", "a = [1]\n", []string{"a", "x"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := mustParse(t, test.doc)
			if d.Delete(test.key) {
				t.Fatalf("Delete(%q) = true, document is now %q", test.key, d.String())
			}
			if got := d.String(); got != test.doc {
				t.Errorf("document changed after Delete returning false:\ngot:  %q\nwant: %q", got, test.doc)
			}
		})
	}
}

func TestCRLF(t *testing.T) {
	d := mustParse(t, "a = 1\r\n[t]\r\nb = 2\r\n")
	if err := d.Set([]string{"t", "c"}, 3); err != nil {
		t.Fatal(err)
	}
	want := "a = 1\r\n[t]\r\nb = 2\r\nc = 3\r\n"
	if got := d.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if err := d.Set([]string{"n", "k"}, 1); err != nil {
		t.Fatal(err)
	}
	want = "a = 1\r\n[t]\r\nb = 2\r\nc = 3\r\n\r\n[n]\r\nk = 1\r\n"
	if got := d.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// collectPaths gathers the key paths of the leaves and tables of the index
// that are addressable by Set and Delete (that is, not inside an array of
// tables).
func collectPaths(t *table, prefix []string, leaves, tables *[][]string) {
	for name, it := range t.items {
		p := append(append([]string{}, prefix...), name)
		switch {
		case it.leaf != nil:
			*leaves = append(*leaves, p)
		case it.tbl != nil:
			*tables = append(*tables, p)
			collectPaths(it.tbl, p, leaves, tables)
		case it.arr != nil:
			*tables = append(*tables, p)
			for idx, elem := range it.arr {
				ep := append(append([]string{}, p...), strconv.Itoa(idx))
				*tables = append(*tables, ep)
				collectPaths(elem, ep, leaves, tables)
			}
		}
	}
}

// TestEveryPathEditable checks, for every addressable path of every corpus
// document, that Set and Delete apply cleanly and that the result decodes as
// expected.
func TestEveryPathEditable(t *testing.T) {
	for _, doc := range corpus {
		var leaves, tables [][]string
		collectPaths(mustParse(t, doc).root, nil, &leaves, &tables)

		for _, path := range leaves {
			d := mustParse(t, doc)
			if err := d.Set(path, "edited"); err != nil {
				t.Errorf("doc %q: Set(%q): %v", doc, path, err)
				continue
			}
			if v, _ := d.Get(path); v != "edited" {
				t.Errorf("doc %q: Get(%q) after Set = %v", doc, path, v)
			}
		}
		for _, path := range append(leaves, tables...) {
			d := mustParse(t, doc)
			if !d.Delete(path) {
				t.Errorf("doc %q: Delete(%q) = false", doc, path)
				continue
			}
			// Deleting an array element shifts the following ones, so an
			// index-ended path may legitimately still exist.
			if _, isIdx := parseIndex(path[len(path)-1]); !isIdx && d.Has(path) {
				t.Errorf("doc %q: Has(%q) after Delete", doc, path)
			}
		}
	}
}

func TestReindexInvalidDocuments(t *testing.T) {
	// reindex is only ever called on pre-validated documents, but it must
	// fail cleanly rather than build a broken index if that invariant is
	// ever violated.
	for _, doc := range []string{
		"a = 1\na = 2\n",      // duplicate key
		"a = 1\na.b = 2\n",    // dotted key crosses value
		"a = 1\n[a]\n",        // table header redefines value
		"a = 1\n[a.b]\n",      // table header crosses value
		"a = 1\n[[a]]\n",      // array table redefines value
		"a = 1\n[[a.b]]\n",    // array table crosses value
		"[t]\nx = 1\nx = 2\n", // duplicate key in table
	} {
		d := &Document{data: []byte(doc)}
		if err := d.reindex(); err == nil {
			t.Errorf("reindex(%q): expected error", doc)
		}
	}
}

func TestApplyOverlap(t *testing.T) {
	d := mustParse(t, "abc = 1\n")
	err := d.apply(splice{span{0, 5}, nil}, splice{span{3, 7}, nil})
	if err == nil {
		t.Fatal("expected error on overlapping edits")
	}
}
