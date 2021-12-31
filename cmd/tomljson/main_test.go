package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expectBufferEquality(t *testing.T, name string, buffer *bytes.Buffer, expected string) {
	t.Helper()
	output := buffer.String()
	assert.Equal(t, expected, output, fmt.Sprintf("%s does not match", name))
}

func expectProcessMainResults(t *testing.T, input io.Reader, args []string, exitCode int, expectedOutput string, expectedError string) {
	t.Helper()
	outputBuffer := new(bytes.Buffer)
	errorBuffer := new(bytes.Buffer)

	returnCode := processMain(args, input, outputBuffer, errorBuffer)

	expectBufferEquality(t, "stdout", outputBuffer, expectedOutput)
	expectBufferEquality(t, "stderr", errorBuffer, expectedError)

	require.Equal(t, exitCode, returnCode, "exit codes should match")
}

func expect(t *testing.T, input string, args []string, exitCode int, expectedOutput string, expectedError string) {
	t.Helper()
	r := strings.NewReader(input)
	expectProcessMainResults(t, r, args, exitCode, expectedOutput, expectedError)
}

func TestProcessMainReadFromStdin(t *testing.T) {
	input := `
		[mytoml]
		a = 42`
	expectedOutput := `{
  "mytoml": {
    "a": 42
  }
}
`
	expect(t, input, []string{}, 0, expectedOutput, ``)
}

func TestProcessMainReadInvalidTOML(t *testing.T) {
	input := `bad = []]`
	expectedError := `1| bad = []]
 |         ~ expected newline but got U+005D ']'
error occurred at row 1 column 9
`

	expect(t, input, []string{}, -1, ``, expectedError)
}

type badReader struct{}

func (r *badReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("reader failed on purpose")
}

func TestProcessMainProblemReadingFile(t *testing.T) {
	expectedError := `toml: reader failed on purpose
`
	input := &badReader{}

	expectProcessMainResults(t, input, []string{}, -1, ``, expectedError)
}

func TestProcessMainReadFromFile(t *testing.T) {
	input := `
		[mytoml]
		a = 42`

	tmpfile, err := ioutil.TempFile("", "example.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Write([]byte(input)); err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	expectedOutput := `{
  "mytoml": {
    "a": 42
  }
}
`
	expectedError := ``
	expectedExitCode := 0

	expect(t, ``, []string{tmpfile.Name()}, expectedExitCode, expectedOutput, expectedError)
}

func TestProcessMainReadFromMissingFile(t *testing.T) {
	var expectedError string
	if runtime.GOOS == "windows" {
		expectedError = `open /this/file/does/not/exist: The system cannot find the path specified.
`
	} else {
		expectedError = `open /this/file/does/not/exist: no such file or directory
`
	}

	expect(t, ``, []string{"/this/file/does/not/exist"}, -1, ``, expectedError)
}

func TestMainUsage(t *testing.T) {
	out := doAndCaptureStderr(usage)
	require.NotEmpty(t, out)
}

func doAndCaptureStderr(f func()) string {
	orig := os.Stderr
	defer func() { os.Stderr = orig }()

	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	b := new(bytes.Buffer)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(b, r)
		if err != nil {
			panic(err)
		}
	}()

	os.Stderr = w

	f()

	w.Close()
	wg.Wait()

	return b.String()
}
