package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

func processMain(args []string, input io.Reader, stdout, stderr io.Writer, f ConvertFn) int {
	p := Program{Fn: f}
	return p.main(args, input, stdout, stderr)
}

func TestProcessMainStdin(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	input := strings.NewReader("this is the input")

	exit := processMain([]string{}, input, stdout, stderr, func(io.Reader, io.Writer) error {
		return nil
	})

	assert.Equal(t, 0, exit)
	assert.Zero(t, stdout.String())
	assert.Zero(t, stderr.String())
}

func TestProcessMainStdinErr(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	input := strings.NewReader("this is the input")

	exit := processMain([]string{}, input, stdout, stderr, func(io.Reader, io.Writer) error {
		return errors.New("something bad")
	})

	assert.Equal(t, -1, exit)
	assert.Zero(t, stdout.String())
	assert.NotZero(t, stderr.String())
}

func TestProcessMainStdinDecodeErr(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	input := strings.NewReader("this is the input")

	exit := processMain([]string{}, input, stdout, stderr, func(io.Reader, io.Writer) error {
		var v interface{}
		return toml.Unmarshal([]byte(`qwe = 001`), &v)
	})

	assert.Equal(t, -1, exit)
	assert.Zero(t, stdout.String())
	assert.True(t, strings.Contains(stderr.String(), "error occurred at"))
}

func TestProcessMainFileExists(t *testing.T) {
	tmpfile, err := os.CreateTemp(t.TempDir(), "example")
	assert.NoError(t, err)
	_, err = tmpfile.WriteString(`some data`)
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	exit := processMain([]string{tmpfile.Name()}, nil, stdout, stderr, func(io.Reader, io.Writer) error {
		return nil
	})

	assert.Equal(t, 0, exit)
	assert.Zero(t, stdout.String())
	assert.Zero(t, stderr.String())
}

func TestProcessMainFileDoesNotExist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	exit := processMain([]string{"/lets/hope/this/does/not/exist"}, nil, stdout, stderr, func(io.Reader, io.Writer) error {
		return nil
	})

	assert.Equal(t, -1, exit)
	assert.Zero(t, stdout.String())
	assert.NotZero(t, stderr.String())
}

func TestProcessMainFilesInPlace(t *testing.T) {
	dir := t.TempDir()

	path1 := path.Join(dir, "file1")
	path2 := path.Join(dir, "file2")

	err := os.WriteFile(path1, []byte("content 1"), 0o600)
	assert.NoError(t, err)
	err = os.WriteFile(path2, []byte("content 2"), 0o600)
	assert.NoError(t, err)

	p := Program{
		Fn:      dummyFileFn,
		Inplace: true,
	}

	exit := p.main([]string{path1, path2}, os.Stdin, os.Stdout, os.Stderr)

	assert.Equal(t, 0, exit)

	v1, err := os.ReadFile(path1)
	assert.NoError(t, err)
	assert.Equal(t, "1", string(v1))

	v2, err := os.ReadFile(path2)
	assert.NoError(t, err)
	assert.Equal(t, "2", string(v2))
}

func TestProcessMainFilesInPlaceErrRead(t *testing.T) {
	p := Program{
		Fn:      dummyFileFn,
		Inplace: true,
	}

	exit := p.main([]string{"/this/path/is/invalid"}, os.Stdin, os.Stdout, os.Stderr)

	assert.Equal(t, -1, exit)
}

func TestProcessMainFilesInPlaceFailFn(t *testing.T) {
	dir := t.TempDir()

	path1 := path.Join(dir, "file1")

	err := os.WriteFile(path1, []byte("content 1"), 0o600)
	assert.NoError(t, err)

	p := Program{
		Fn:      func(io.Reader, io.Writer) error { return errors.New("oh no") },
		Inplace: true,
	}

	exit := p.main([]string{path1}, os.Stdin, os.Stdout, os.Stderr)

	assert.Equal(t, -1, exit)

	v1, err := os.ReadFile(path1)
	assert.NoError(t, err)
	assert.Equal(t, "content 1", string(v1))
}

func dummyFileFn(r io.Reader, w io.Writer) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	v := strings.SplitN(string(b), " ", 2)[1]
	_, err = w.Write([]byte(v))
	return err
}
