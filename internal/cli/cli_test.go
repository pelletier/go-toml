package cli

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessMainStdin(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	input := strings.NewReader("this is the input")

	exit := processMain([]string{}, input, stdout, stderr, func(r io.Reader, w io.Writer) error {
		return nil
	})

	assert.Equal(t, 0, exit)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestProcessMainStdinErr(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	input := strings.NewReader("this is the input")

	exit := processMain([]string{}, input, stdout, stderr, func(r io.Reader, w io.Writer) error {
		return fmt.Errorf("something bad")
	})

	assert.Equal(t, -1, exit)
	assert.Empty(t, stdout.String())
	assert.NotEmpty(t, stderr.String())
}

func TestProcessMainFileExists(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write([]byte(`some data`))
	require.NoError(t, err)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	exit := processMain([]string{tmpfile.Name()}, nil, stdout, stderr, func(r io.Reader, w io.Writer) error {
		return nil
	})

	assert.Equal(t, 0, exit)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestProcessMainFileDoesNotExist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	exit := processMain([]string{"/lets/hope/this/does/not/exist"}, nil, stdout, stderr, func(r io.Reader, w io.Writer) error {
		return nil
	})

	assert.Equal(t, -1, exit)
	assert.Empty(t, stdout.String())
	assert.NotEmpty(t, stderr.String())
}
