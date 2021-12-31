package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expectBufferEquality(t *testing.T, name string, buffer *bytes.Buffer, expected string) {
	t.Helper()
	output := buffer.String()
	assert.Equal(t, expected, output, fmt.Sprintf("%s does not match", name))
}

func expectProcessMainResults(t *testing.T, input string, args []string, exitCode int, expectedOutput string, expectedError string) {
	t.Helper()
	inputReader := strings.NewReader(input)
	outputBuffer := new(bytes.Buffer)
	errorBuffer := new(bytes.Buffer)

	returnCode := processMain(args, inputReader, outputBuffer, errorBuffer)

	expectBufferEquality(t, "stdout", outputBuffer, expectedOutput)
	expectBufferEquality(t, "stderr", errorBuffer, expectedError)

	require.Equal(t, exitCode, returnCode, "exit codes should match")
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
	expectedError := ``
	expectedExitCode := 0

	expectProcessMainResults(t, input, []string{}, expectedExitCode, expectedOutput, expectedError)
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

	expectProcessMainResults(t, ``, []string{tmpfile.Name()}, expectedExitCode, expectedOutput, expectedError)
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

	expectProcessMainResults(t, ``, []string{"/this/file/does/not/exist"}, -1, ``, expectedError)
}
