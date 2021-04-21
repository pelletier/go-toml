package toml

import (
	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/pelletier/go-toml/v2/internal/tracker"
)

type strict struct {
	Enabled bool

	// Tracks the current key being processed.
	key tracker.KeyTracker

	missing []decodeError
}

func (s *strict) EnterTable(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.key.UpdateTable(node)
}

func (s *strict) EnterArrayTable(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.key.UpdateArrayTable(node)
}

func (s *strict) EnterKeyValue(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.key.Push(node)
}

func (s *strict) ExitKeyValue(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.key.Pop(node)
}

func (s *strict) MissingTable(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.missing = append(s.missing, decodeError{
		highlight: keyLocation(node),
		message:   "missing table",
		key:       s.key.Key(),
	})
}

func (s *strict) MissingField(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.missing = append(s.missing, decodeError{
		highlight: keyLocation(node),
		message:   "missing field",
		key:       s.key.Key(),
	})
}

func (s *strict) Error(doc []byte) error {
	if !s.Enabled || len(s.missing) == 0 {
		return nil
	}

	err := &StrictMissingError{
		Errors: make([]DecodeError, 0, len(s.missing)),
	}
	for _, derr := range s.missing {
		err.Errors = append(err.Errors, *wrapDecodeError(doc, &derr))
	}
	return err
}
