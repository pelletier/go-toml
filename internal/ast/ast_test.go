package ast

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestNodeSize(t *testing.T) {
	require.LessOrEqual(t, int(unsafe.Sizeof(Node{})), int(64))
}
