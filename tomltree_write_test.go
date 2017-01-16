package toml

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestTomlTreeWriteExample(t *testing.T) {
	want, err := ioutil.ReadFile("example.toml")
	if err != nil {
		t.Fatal(err)
	}
	want = removeComments(want)
	tree, err := LoadReader(bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err = tree.WriteToToml(&buf, "", ""); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(want, buf.Bytes()) {
		t.Fatalf("Got:\n%s\n\nWant:\n%s", buf.Bytes(), want)
	}
}

// removeComments, dummy!
func removeComments(b []byte) []byte {
	for {
		i := bytes.IndexByte(b, '#')
		if i < 0 {
			return b
		}
		for i > 0 {
			if c := b[i-1]; !(c == ' ' || c == '\t' || c == '\n') {
				break
			}
			i--
			b[i] = ' '
		}
		j := bytes.IndexByte(b[i:], '\n')
		if j < 0 {
			return b[:i]
		}
		j += i
		if j < len(b) && (i == 0 || b[i-1] == '\n') {
			j++
		}
		b = append(b[:i], b[j:]...)
	}
}
